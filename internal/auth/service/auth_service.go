package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/EduGoGroup/edugo-api-iam-platform/internal/auth/dto"
	"github.com/EduGoGroup/edugo-api-iam-platform/internal/auth/model"
	authrepo "github.com/EduGoGroup/edugo-api-iam-platform/internal/auth/repository"
	"github.com/EduGoGroup/edugo-api-iam-platform/internal/domain/repository"
	"github.com/EduGoGroup/edugo-infrastructure/postgres/entities"
	"github.com/EduGoGroup/edugo-shared/audit"
	"github.com/EduGoGroup/edugo-shared/auth"
	"github.com/EduGoGroup/edugo-shared/logger"
	sharedrepo "github.com/EduGoGroup/edugo-shared/repository"
	"github.com/google/uuid"
)

// Sentinel errors for auth operations
var (
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrUserNotFound          = errors.New("user not found")
	ErrUserInactive          = errors.New("user inactive")
	ErrInvalidRefreshToken   = errors.New("invalid refresh token")
	ErrNoMembership          = errors.New("no active membership in target school")
	ErrInvalidSchoolID       = errors.New("invalid school_id")
	ErrTooManyLoginAttempts  = errors.New("too many login attempts, try again later")
)

// AuthService defines the authentication service interface
type AuthService interface {
	Login(ctx context.Context, email, password, clientIP, userAgent string) (*dto.LoginResponse, error)
	Logout(ctx context.Context, accessToken string) error
	RefreshToken(ctx context.Context, refreshToken string) (*dto.RefreshResponse, error)
	SwitchContext(ctx context.Context, userID, targetSchoolID string) (*dto.SwitchContextResponse, error)
	GetAvailableContexts(ctx context.Context, userID string, currentContext *auth.UserContext) (*dto.AvailableContextsResponse, error)
}

type authService struct {
	userRepo          sharedrepo.UserRepository
	userRoleRepo      repository.UserRoleRepository
	roleRepo          repository.RoleRepository
	membershipRepo    sharedrepo.MembershipRepository
	schoolRepo        sharedrepo.SchoolRepository
	tokenService      *TokenService
	logger            logger.Logger
	auditLogger       audit.AuditLogger
	loginAttemptRepo  authrepo.LoginAttemptRepository
}

// NewAuthService creates a new auth service
func NewAuthService(
	userRepo sharedrepo.UserRepository,
	userRoleRepo repository.UserRoleRepository,
	roleRepo repository.RoleRepository,
	membershipRepo sharedrepo.MembershipRepository,
	schoolRepo sharedrepo.SchoolRepository,
	tokenService *TokenService,
	logger logger.Logger,
	auditLogger audit.AuditLogger,
	loginAttemptRepo authrepo.LoginAttemptRepository,
) AuthService {
	return &authService{
		userRepo:         userRepo,
		userRoleRepo:     userRoleRepo,
		roleRepo:         roleRepo,
		membershipRepo:   membershipRepo,
		schoolRepo:       schoolRepo,
		tokenService:     tokenService,
		logger:           logger,
		auditLogger:      auditLogger,
		loginAttemptRepo: loginAttemptRepo,
	}
}

// Login validates credentials and returns JWT tokens
func (s *authService) Login(ctx context.Context, email, password, clientIP, userAgent string) (*dto.LoginResponse, error) {
	// Normalize email to prevent case/whitespace bypass on rate limiting
	email = strings.ToLower(strings.TrimSpace(email))

	// Helper to record login attempt. Accepts explicit context so callers in
	// background goroutines can pass context.Background() instead of the
	// already-canceled request context.
	recordAttempt := func(c context.Context, success bool) {
		var ua, ip *string
		if userAgent != "" {
			ua = &userAgent
		}
		if clientIP != "" {
			ip = &clientIP
		}
		attempt := &model.LoginAttempt{
			Identifier:  email,
			AttemptType: "email",
			Successful:  success,
			UserAgent:   ua,
			IPAddress:   ip,
			AttemptedAt: time.Now(),
		}
		if err := s.loginAttemptRepo.Create(c, attempt); err != nil {
			s.logger.Warn("error recording login attempt", "email", email, "error", err)
		}
	}

	// Phase 1: Rate limit check runs in background while we look up the user.
	// Both are independent DB queries — overlap them.
	var failedCount int
	var rateLimitErr error
	var phase1 sync.WaitGroup
	phase1.Add(1)
	go func() {
		defer phase1.Done()
		fc, err := s.loginAttemptRepo.CountFailedSince(ctx, email, time.Now().Add(-15*time.Minute))
		failedCount = int(fc)
		rateLimitErr = err
	}()

	// 1. Find user by email (runs concurrently with rate limit check)
	user, err := s.userRepo.FindByEmail(ctx, email)

	// Wait for rate limit check to complete
	phase1.Wait()

	// Check rate limit
	if rateLimitErr != nil {
		s.logger.Warn("error checking login rate limit", "email", email, "error", rateLimitErr)
	}
	if failedCount >= 5 {
		s.logger.Warn("login rate limited", "email", email, "failed_count", failedCount)
		return nil, ErrTooManyLoginAttempts
	}

	// Check user lookup result
	if err != nil {
		if errors.Is(err, sharedrepo.ErrNotFound) {
			recordAttempt(ctx, false)
			_ = s.auditLogger.Log(ctx, audit.AuditEvent{
				ActorEmail:   email,
				ActorIP:      clientIP,
				Action:       "login_failed",
				ResourceType: "session",
				ErrorMessage: "user not found",
				Severity:     audit.SeverityWarning,
				Category:     audit.CategoryAuth,
			})
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("error finding user: %w", err)
	}
	if user == nil {
		s.logger.Warn("login attempt with non-existent email", "email", email)
		recordAttempt(ctx, false)
		_ = s.auditLogger.Log(ctx, audit.AuditEvent{
			ActorEmail:   email,
			ActorIP:      clientIP,
			Action:       "login_failed",
			ResourceType: "session",
			ErrorMessage: "user not found",
			Severity:     audit.SeverityWarning,
			Category:     audit.CategoryAuth,
		})
		return nil, ErrInvalidCredentials
	}

	// 2. Verify user is active
	if !user.IsActive {
		s.logger.Warn("login attempt with inactive user", "email", email, "user_id", user.ID.String())
		recordAttempt(ctx, false)
		_ = s.auditLogger.Log(ctx, audit.AuditEvent{
			ActorID:      user.ID.String(),
			ActorEmail:   email,
			ActorIP:      clientIP,
			Action:       "login_failed",
			ResourceType: "session",
			ErrorMessage: "user inactive",
			Severity:     audit.SeverityWarning,
			Category:     audit.CategoryAuth,
		})
		return nil, ErrUserInactive
	}

	// 3. Verify password
	if err := auth.VerifyPassword(user.PasswordHash, password); err != nil {
		s.logger.Warn("incorrect password", "email", email)
		recordAttempt(ctx, false)
		_ = s.auditLogger.Log(ctx, audit.AuditEvent{
			ActorID:      user.ID.String(),
			ActorEmail:   email,
			ActorIP:      clientIP,
			Action:       "login_failed",
			ResourceType: "session",
			ErrorMessage: "invalid password",
			Severity:     audit.SeverityWarning,
			Category:     audit.CategoryAuth,
		})
		return nil, ErrInvalidCredentials
	}

	// 4+5. Get schools + build RBAC context in PARALLEL.
	// Try global role first (superadmin). If found, skip school-scoped lookup.
	var schools []dto.SchoolInfo
	var firstSchoolID *uuid.UUID
	var globalContext *auth.UserContext
	var phase2 sync.WaitGroup
	phase2.Add(2)
	go func() {
		defer phase2.Done()
		schools, firstSchoolID = s.getUserSchools(ctx, user.ID)
	}()
	go func() {
		defer phase2.Done()
		globalContext = s.buildUserContext(ctx, user.ID, nil)
	}()
	phase2.Wait()

	var activeContext *auth.UserContext
	if globalContext != nil {
		// User has a global role (e.g. super_admin) — pin school if available
		if firstSchoolID != nil {
			globalContext.SchoolID = firstSchoolID.String()
			school, err := s.schoolRepo.FindByID(ctx, *firstSchoolID)
			if err == nil && school != nil {
				globalContext.SchoolName = school.Name
			}
		}
		activeContext = globalContext
	} else if firstSchoolID != nil {
		// No global role — build school-scoped context
		activeContext = s.buildUserContext(ctx, user.ID, firstSchoolID)
	}

	if activeContext == nil {
		s.logger.Error("no RBAC context found for user", "user_id", user.ID.String(), "email", user.Email)
		recordAttempt(ctx, false)
		return nil, fmt.Errorf("user has no assigned roles")
	}

	// 6. Generate tokens
	tokenResponse, err := s.tokenService.GenerateTokenPairWithContext(user.ID.String(), user.Email, activeContext)
	if err != nil {
		return nil, fmt.Errorf("error generating tokens: %w", err)
	}

	// 7. Build response
	schoolID := ""
	if firstSchoolID != nil {
		schoolID = firstSchoolID.String()
	}

	tokenResponse.User = &dto.UserInfo{
		ID:        user.ID.String(),
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		FullName:  user.FirstName + " " + user.LastName,
		SchoolID:  schoolID,
	}
	tokenResponse.Schools = schools
	tokenResponse.ActiveContext = &dto.UserContextDTO{
		RoleID:      activeContext.RoleID,
		RoleName:    activeContext.RoleName,
		SchoolID:    activeContext.SchoolID,
		SchoolName:  activeContext.SchoolName,
		Permissions: activeContext.Permissions,
	}

	s.logger.Info("user logged in",
		"entity_type", "auth_session",
		"user_id", user.ID.String(),
		"email", user.Email,
		"role", activeContext.RoleName,
		"school_id", schoolID,
	)

	// Record successful attempt, audit, and update last login (fire and forget).
	// These are post-response bookkeeping — the user already has valid tokens.
	// NOTE: recordAttempt(ctx, false) calls above remain synchronous so rate limiting works.
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		recordAttempt(bgCtx, true)
		_ = s.auditLogger.Log(bgCtx, audit.AuditEvent{
			ActorID:      user.ID.String(),
			ActorEmail:   user.Email,
			ActorRole:    activeContext.RoleName,
			ActorIP:      clientIP,
			Action:       "login",
			ResourceType: "session",
			Severity:     audit.SeverityInfo,
			Category:     audit.CategoryAuth,
			Metadata:     map[string]interface{}{"school_id": schoolID},
		})
		user.UpdatedAt = time.Now()
		if err := s.userRepo.Update(bgCtx, user); err != nil {
			s.logger.Warn("error updating last login", "error", err)
		}
	}()

	return tokenResponse, nil
}

// Logout invalidates the access token
func (s *authService) Logout(ctx context.Context, _ string) error {
	s.logger.Info("user logged out", "entity_type", "auth_session")
	_ = s.auditLogger.Log(ctx, audit.AuditEvent{
		Action:       "logout",
		ResourceType: "session",
		Severity:     audit.SeverityInfo,
		Category:     audit.CategoryAuth,
	})
	return nil
}

// RefreshToken validates a refresh token JWT and generates new access + refresh tokens
func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*dto.RefreshResponse, error) {
	// 1. Validate refresh token JWT
	userID, _, schoolIDFromToken, err := s.tokenService.ValidateRefreshJWT(refreshToken)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	// 2. Find user and verify active
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}
	user, err := s.userRepo.FindByID(ctx, userUUID)
	if err != nil {
		return nil, fmt.Errorf("error finding user: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	if !user.IsActive {
		return nil, ErrUserInactive
	}

	// 3. Determine target school: prefer schoolID embedded in the refresh token
	// (set at login or switchContext) to preserve the school the user selected.
	// Fall back to firstSchoolID from memberships only for legacy tokens without schoolID.
	var targetSchoolID *uuid.UUID
	if schoolIDFromToken != "" {
		sid, parseErr := uuid.Parse(schoolIDFromToken)
		if parseErr != nil {
			// Signed JWT contains a non-parseable schoolID — reject to avoid silent context switch.
			s.logger.Warn("invalid schoolID in refresh token", "user_id", userID, "school_id", schoolIDFromToken)
			return nil, ErrInvalidRefreshToken
		}
		targetSchoolID = &sid
	}
	if targetSchoolID == nil {
		_, targetSchoolID = s.getUserSchools(ctx, userUUID)
	}

	// 4. Rebuild RBAC context.
	// Try global context first (covers super_admin and other global roles),
	// then fall back to school-scoped context.
	var activeContext *auth.UserContext
	globalContext := s.buildUserContext(ctx, userUUID, nil)
	if globalContext != nil {
		// User has a global role — pin the target school if available
		if targetSchoolID != nil {
			globalContext.SchoolID = targetSchoolID.String()
			school, err := s.schoolRepo.FindByID(ctx, *targetSchoolID)
			if err == nil && school != nil {
				globalContext.SchoolName = school.Name
			}
		}
		activeContext = globalContext
	} else if targetSchoolID != nil {
		// No global role — try school-scoped context
		activeContext = s.buildUserContext(ctx, userUUID, targetSchoolID)
	}
	if activeContext == nil {
		return nil, fmt.Errorf("user has no assigned roles")
	}

	// 5. Generate new access token (use DB email, not claim email)
	resp, err := s.tokenService.GenerateAccessTokenWithContext(userID, user.Email, activeContext)
	if err != nil {
		return nil, fmt.Errorf("error generating access token: %w", err)
	}

	// 6. Rotate refresh token preserving the resolved schoolID
	newRefreshJWT, _, err := s.tokenService.GenerateRefreshJWT(userID, user.Email, activeContext.SchoolID)
	if err != nil {
		return nil, fmt.Errorf("error generating refresh token: %w", err)
	}

	resp.RefreshToken = newRefreshJWT
	resp.ActiveContext = &dto.UserContextDTO{
		RoleID:      activeContext.RoleID,
		RoleName:    activeContext.RoleName,
		SchoolID:    activeContext.SchoolID,
		SchoolName:  activeContext.SchoolName,
		Permissions: activeContext.Permissions,
	}

	s.logger.Info("token refreshed", "user_id", userID, "email", user.Email)

	return resp, nil
}

// getUserSchools devuelve las escuelas activas del usuario desde memberships.
// School lookups run in parallel using goroutines.
func (s *authService) getUserSchools(ctx context.Context, userID uuid.UUID) ([]dto.SchoolInfo, *uuid.UUID) {
	active := true
	memberships, _, err := s.membershipRepo.FindByUser(ctx, userID, sharedrepo.ListFilters{IsActive: &active})
	if err != nil {
		s.logger.Warn("error fetching memberships for user", "user_id", userID.String(), "error", err)
		return []dto.SchoolInfo{}, nil
	}

	// Deduplicate school IDs
	seen := make(map[uuid.UUID]struct{})
	var uniqueSchoolIDs []uuid.UUID
	for _, m := range memberships {
		if !m.IsActive {
			continue
		}
		if _, exists := seen[m.SchoolID]; !exists {
			seen[m.SchoolID] = struct{}{}
			uniqueSchoolIDs = append(uniqueSchoolIDs, m.SchoolID)
		}
	}

	if len(uniqueSchoolIDs) == 0 {
		return []dto.SchoolInfo{}, nil
	}

	// Parallel school lookups
	type schoolResult struct {
		info dto.SchoolInfo
		ok   bool
	}
	results := make([]schoolResult, len(uniqueSchoolIDs))
	var wg sync.WaitGroup
	for i, sid := range uniqueSchoolIDs {
		wg.Add(1)
		go func(idx int, schoolID uuid.UUID) {
			defer wg.Done()
			school, err := s.schoolRepo.FindByID(ctx, schoolID)
			if err != nil || school == nil {
				return
			}
			results[idx] = schoolResult{
				info: dto.SchoolInfo{ID: schoolID.String(), Name: school.Name},
				ok:   true,
			}
		}(i, sid)
	}
	wg.Wait()

	var schools []dto.SchoolInfo
	var firstSchoolID *uuid.UUID
	for i, r := range results {
		if r.ok {
			schools = append(schools, r.info)
			if firstSchoolID == nil {
				sid := uniqueSchoolIDs[i]
				firstSchoolID = &sid
			}
		}
	}
	if schools == nil {
		schools = []dto.SchoolInfo{}
	}
	return schools, firstSchoolID
}

// SwitchContext switches the active school context for the user
func (s *authService) SwitchContext(ctx context.Context, userID, targetSchoolID string) (*dto.SwitchContextResponse, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id: %w", err)
	}

	schoolUUID, err := uuid.Parse(targetSchoolID)
	if err != nil {
		return nil, ErrInvalidSchoolID
	}

	user, err := s.userRepo.FindByID(ctx, userUUID)
	if err != nil {
		return nil, fmt.Errorf("error finding user: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	if !user.IsActive {
		return nil, ErrUserInactive
	}

	membership, err := s.membershipRepo.FindByUserAndSchool(ctx, userUUID, schoolUUID)
	if err != nil {
		if !errors.Is(err, sharedrepo.ErrNotFound) {
			return nil, fmt.Errorf("error checking membership: %w", err)
		}
		membership = nil
	}

	var activeContext *auth.UserContext

	if membership == nil {
		// No school-specific membership: check for global role (e.g. super_admin)
		globalContext := s.buildUserContext(ctx, userUUID, nil)
		if globalContext == nil {
			s.logger.Warn("switch-context attempt without membership",
				"user_id", userID,
				"target_school_id", targetSchoolID,
			)
			return nil, ErrNoMembership
		}
		// Verify the target school exists
		school, err := s.schoolRepo.FindByID(ctx, schoolUUID)
		if err != nil {
			return nil, fmt.Errorf("error verifying target school: %w", err)
		}
		if school == nil {
			return nil, ErrInvalidSchoolID
		}
		globalContext.SchoolID = targetSchoolID
		globalContext.SchoolName = school.Name
		activeContext = globalContext
	} else {
		activeContext = s.buildUserContext(ctx, userUUID, &schoolUUID)
		if activeContext == nil {
			s.logger.Error("no RBAC context found for switch-context",
				"user_id", userID,
				"target_school_id", targetSchoolID,
			)
			return nil, fmt.Errorf("user has no assigned roles in target school")
		}
	}

	tokenResponse, err := s.tokenService.GenerateTokenPairWithContext(
		user.ID.String(),
		user.Email,
		activeContext,
	)
	if err != nil {
		return nil, fmt.Errorf("error generating tokens: %w", err)
	}

	_ = s.auditLogger.Log(ctx, audit.AuditEvent{
		ActorID:      userID,
		ActorEmail:   user.Email,
		ActorRole:    activeContext.RoleName,
		Action:       "switch_context",
		ResourceType: "session",
		Severity:     audit.SeverityInfo,
		Category:     audit.CategoryAuth,
		Metadata:     map[string]interface{}{"new_school_id": targetSchoolID},
	})

	s.logger.Info("context switched",
		"entity_type", "auth_context",
		"user_id", userID,
		"new_school_id", targetSchoolID,
		"new_role", activeContext.RoleName,
	)

	return &dto.SwitchContextResponse{
		AccessToken:  tokenResponse.AccessToken,
		RefreshToken: tokenResponse.RefreshToken,
		ExpiresIn:    tokenResponse.ExpiresIn,
		TokenType:    tokenResponse.TokenType,
		Context: &dto.ContextInfo{
			SchoolID:   targetSchoolID,
			SchoolName: activeContext.SchoolName,
			Role:       activeContext.RoleName,
			UserID:     userID,
			Email:      user.Email,
		},
	}, nil
}

// GetAvailableContexts returns all available contexts (roles/schools) for the user
func (s *authService) GetAvailableContexts(ctx context.Context, userID string, currentContext *auth.UserContext) (*dto.AvailableContextsResponse, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id: %w", err)
	}

	userRoles, err := s.userRoleRepo.FindByUser(ctx, userUUID)
	if err != nil {
		return nil, fmt.Errorf("error fetching roles: %w", err)
	}

	roleCache := make(map[string]string)
	schoolCache := make(map[string]string)

	var available []*dto.UserContextDTO
	for _, ur := range userRoles {
		roleID := ur.RoleID.String()

		roleName, ok := roleCache[roleID]
		if !ok {
			role, err := s.roleRepo.FindByID(ctx, ur.RoleID)
			if err != nil {
				s.logger.Warn("error fetching role", "role_id", roleID, "error", err)
				continue
			}
			if role == nil {
				s.logger.Warn("role not found", "role_id", roleID)
				continue
			}
			roleName = role.Name
			roleCache[roleID] = roleName
		}

		schoolID := ""
		schoolName := ""
		if ur.SchoolID != nil {
			schoolID = ur.SchoolID.String()
			if cached, ok := schoolCache[schoolID]; ok {
				schoolName = cached
			} else {
				school, err := s.schoolRepo.FindByID(ctx, *ur.SchoolID)
				if err == nil && school != nil {
					schoolName = school.Name
					schoolCache[schoolID] = schoolName
				}
			}
		}

		permissions, err := s.userRoleRepo.GetUserPermissions(ctx, userUUID, ur.SchoolID, ur.AcademicUnitID)
		if err != nil {
			s.logger.Warn("error fetching user permissions", "user_id", userUUID.String(), "role_id", roleID, "error", err)
		}

		item := &dto.UserContextDTO{
			RoleID:      roleID,
			RoleName:    roleName,
			SchoolID:    schoolID,
			SchoolName:  schoolName,
			Permissions: permissions,
		}

		if ur.AcademicUnitID != nil {
			item.AcademicUnitID = ur.AcademicUnitID.String()
		}

		available = append(available, item)
	}

	var current *dto.UserContextDTO
	if currentContext != nil {
		current = &dto.UserContextDTO{
			RoleID:           currentContext.RoleID,
			RoleName:         currentContext.RoleName,
			SchoolID:         currentContext.SchoolID,
			SchoolName:       currentContext.SchoolName,
			AcademicUnitID:   currentContext.AcademicUnitID,
			AcademicUnitName: currentContext.AcademicUnitName,
			Permissions:      currentContext.Permissions,
		}
	}

	s.logger.Info("available contexts fetched",
		"user_id", userID,
		"contexts_count", len(available),
	)

	return &dto.AvailableContextsResponse{
		Current:   current,
		Available: available,
	}, nil
}

// buildUserContext constructs the RBAC UserContext for the JWT.
// Queries that only depend on userID+schoolID run in parallel.
func (s *authService) buildUserContext(ctx context.Context, userID uuid.UUID, schoolID *uuid.UUID) *auth.UserContext {
	// Phase A: These 3 queries are independent — run in parallel.
	// - FindByUserInContext: needs userID + schoolID
	// - GetUserPermissions: needs userID + schoolID (independent from user_roles)
	// - FindByID(school): needs schoolID only
	var userRoles []*entities.UserRole
	var userRolesErr error
	var permissions []string
	var permErr error
	var schoolName string

	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		userRoles, userRolesErr = s.userRoleRepo.FindByUserInContext(ctx, userID, schoolID, nil)
	}()
	go func() {
		defer wg.Done()
		p, err := s.userRoleRepo.GetUserPermissions(ctx, userID, schoolID, nil)
		permissions = p
		permErr = err
	}()

	if schoolID != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			school, err := s.schoolRepo.FindByID(ctx, *schoolID)
			if err == nil && school != nil {
				schoolName = school.Name
			}
		}()
	}
	wg.Wait()

	// Check user_roles result
	if userRolesErr != nil {
		s.logger.Warn("error obtaining user roles for RBAC context",
			"user_id", userID.String(),
			"error", userRolesErr,
		)
		return nil
	}
	if len(userRoles) == 0 {
		return nil
	}

	// Phase B: Get role details (depends on FindByUserInContext result)
	firstRole := userRoles[0]
	role, err := s.roleRepo.FindByID(ctx, firstRole.RoleID)
	if err != nil {
		s.logger.Warn("error obtaining role for RBAC context",
			"user_id", userID.String(),
			"role_id", firstRole.RoleID.String(),
			"error", err,
		)
		return nil
	}

	if permErr != nil {
		s.logger.Warn("error obtaining user permissions",
			"user_id", userID.String(),
			"error", permErr,
		)
		permissions = []string{}
	}

	uc := &auth.UserContext{
		RoleID:      role.ID.String(),
		RoleName:    role.Name,
		Permissions: permissions,
	}

	if schoolID != nil {
		uc.SchoolID = schoolID.String()
		uc.SchoolName = schoolName
	}

	return uc
}

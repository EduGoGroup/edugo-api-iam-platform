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
	"github.com/EduGoGroup/edugo-shared/metrics"
	sharedrepo "github.com/EduGoGroup/edugo-shared/repository"
	"github.com/google/uuid"
)

// authMetrics is a package-level metrics instance used by auth services.
// Uses NoOp recorder by default; a real recorder can be swapped in later.
var authMetrics = metrics.New("edugo-api-iam-platform")

// Sentinel errors for auth operations
var (
	ErrInvalidCredentials         = errors.New("invalid credentials")
	ErrUserNotFound               = errors.New("user not found")
	ErrUserInactive               = errors.New("user inactive")
	ErrInvalidRefreshToken        = errors.New("invalid refresh token")
	ErrNoMembership               = errors.New("no active membership in target school")
	ErrInvalidSchoolID            = errors.New("invalid school_id")
	ErrUnauthorizedUnit           = errors.New("user has no active membership in the requested academic unit")
	ErrTooManyLoginAttempts       = errors.New("too many login attempts, try again later")
	ErrAcademicUnitNotFound       = errors.New("academic unit not found")
	ErrAcademicUnitSchoolMismatch = errors.New("academic unit does not belong to target school")
)

// AuthService defines the authentication service interface
type AuthService interface {
	Login(ctx context.Context, email, password, clientIP, userAgent string) (*dto.LoginResponse, error)
	Logout(ctx context.Context, accessToken string) error
	RefreshToken(ctx context.Context, refreshToken string) (*dto.RefreshResponse, error)
	SwitchContext(ctx context.Context, userID, targetSchoolID, academicUnitID string) (*dto.SwitchContextResponse, error)
	GetAvailableContexts(ctx context.Context, userID string, currentContext *auth.UserContext) (*dto.AvailableContextsResponse, error)
	GetSchoolUnits(ctx context.Context, schoolID string) (*dto.SchoolUnitsResponse, error)
}

type authService struct {
	userRepo         sharedrepo.UserRepository
	userRoleRepo     repository.UserRoleRepository
	roleRepo         repository.RoleRepository
	membershipRepo   sharedrepo.MembershipRepository
	schoolRepo       sharedrepo.SchoolRepository
	academicUnitRepo sharedrepo.AcademicUnitRepository
	tokenService     *TokenService
	logger           logger.Logger
	auditLogger      audit.AuditLogger
	loginAttemptRepo authrepo.LoginAttemptRepository
	blacklist        auth.TokenBlacklist
}

// NewAuthService creates a new auth service
func NewAuthService(
	userRepo sharedrepo.UserRepository,
	userRoleRepo repository.UserRoleRepository,
	roleRepo repository.RoleRepository,
	membershipRepo sharedrepo.MembershipRepository,
	schoolRepo sharedrepo.SchoolRepository,
	academicUnitRepo sharedrepo.AcademicUnitRepository,
	tokenService *TokenService,
	logger logger.Logger,
	auditLogger audit.AuditLogger,
	loginAttemptRepo authrepo.LoginAttemptRepository,
	blacklist auth.TokenBlacklist,
) AuthService {
	return &authService{
		userRepo:         userRepo,
		userRoleRepo:     userRoleRepo,
		roleRepo:         roleRepo,
		membershipRepo:   membershipRepo,
		schoolRepo:       schoolRepo,
		academicUnitRepo: academicUnitRepo,
		tokenService:     tokenService,
		logger:           logger,
		auditLogger:      auditLogger,
		loginAttemptRepo: loginAttemptRepo,
		blacklist:        blacklist,
	}
}

// Login validates credentials and returns JWT tokens
func (s *authService) Login(ctx context.Context, email, password, clientIP, userAgent string) (*dto.LoginResponse, error) {
	log := logger.FromContext(ctx)
	start := time.Now()

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
			log.Warn("error recording login attempt", "email", email, "error", err)
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
		log.Warn("error checking login rate limit", "email", email, "error", rateLimitErr)
	}
	if failedCount >= 5 {
		log.Warn("login rate limited", "email", email, "failed_count", failedCount, "ip", clientIP)
		authMetrics.RecordRateLimitHit("login")
		authMetrics.RecordLogin(false, time.Since(start))
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
			authMetrics.RecordLogin(false, time.Since(start))
			return nil, ErrInvalidCredentials
		}
		authMetrics.RecordLogin(false, time.Since(start))
		return nil, fmt.Errorf("error finding user: %w", err)
	}
	if user == nil {
		log.Warn("login attempt with non-existent email", "email", email)
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
		authMetrics.RecordLogin(false, time.Since(start))
		return nil, ErrInvalidCredentials
	}

	// 2. Verify user is active
	if !user.IsActive {
		log.Warn("login attempt with inactive user", "email", email, "user_id", user.ID.String())
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
		authMetrics.RecordLogin(false, time.Since(start))
		return nil, ErrUserInactive
	}

	// 3. Verify password
	if err := auth.VerifyPassword(user.PasswordHash, password); err != nil {
		log.Warn("incorrect password", "email", email)
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
		authMetrics.RecordLogin(false, time.Since(start))
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
			// Auto-populate AcademicUnitID for users with exactly 1 unit
			s.autoPopulateUnit(ctx, user.ID, firstSchoolID, globalContext)
		}
		activeContext = globalContext
	} else if firstSchoolID != nil {
		// No global role — build school-scoped context
		activeContext = s.buildUserContext(ctx, user.ID, firstSchoolID)
	}

	if activeContext == nil {
		log.Error("no RBAC context found for user", "user_id", user.ID.String(), "email", user.Email)
		recordAttempt(ctx, false)
		authMetrics.RecordLogin(false, time.Since(start))
		return nil, fmt.Errorf("user has no assigned roles")
	}

	// 6. Generate tokens
	tokenResponse, err := s.tokenService.GenerateTokenPairWithContext(user.ID.String(), user.Email, activeContext)
	if err != nil {
		authMetrics.RecordLogin(false, time.Since(start))
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
		RoleID:           activeContext.RoleID,
		RoleName:         activeContext.RoleName,
		SchoolID:         activeContext.SchoolID,
		SchoolName:       activeContext.SchoolName,
		AcademicUnitID:   activeContext.AcademicUnitID,
		AcademicUnitName: activeContext.AcademicUnitName,
		Permissions:      activeContext.Permissions,
	}

	log.Info("user logged in",
		"entity_type", "auth_session",
		"user_id", user.ID.String(),
		"email", user.Email,
		"role", activeContext.RoleName,
		"school_id", schoolID,
		"ip", clientIP,
	)

	// Record successful attempt and audit log synchronously (required for rate
	// limiting accuracy and compliance/security audit trails).
	recordAttempt(ctx, true)
	_ = s.auditLogger.Log(ctx, audit.AuditEvent{
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

	authMetrics.RecordLogin(true, time.Since(start))

	// Update last-login timestamp in background (non-critical).
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		user.UpdatedAt = time.Now()
		if err := s.userRepo.Update(bgCtx, user); err != nil {
			log.Warn("error updating last login", "error", err)
		}
	}()

	return tokenResponse, nil
}

// Logout invalidates the access token
func (s *authService) Logout(ctx context.Context, tokenString string) error {
	// Parse token to get JTI for revocation
	claims, err := s.tokenService.ValidateAccessToken(tokenString)
	if err != nil {
		// Token might be expired/invalid, still consider logout successful
		s.logger.Warn("logout with invalid token", "error", err.Error())
	} else if claims.ID != "" {
		s.blacklist.Revoke(claims.ID, claims.ExpiresAt.Time)
		s.logger.Info("token revoked", "jti", claims.ID, "entity_type", "auth_session")
	}

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
	log := logger.FromContext(ctx)
	start := time.Now()

	// 1. Validate refresh token JWT
	userID, _, schoolIDFromToken, err := s.tokenService.ValidateRefreshJWT(refreshToken)
	if err != nil {
		authMetrics.RecordTokenRefresh(false, time.Since(start))
		return nil, ErrInvalidRefreshToken
	}

	// 2. Find user and verify active
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		authMetrics.RecordTokenRefresh(false, time.Since(start))
		return nil, ErrInvalidRefreshToken
	}
	user, err := s.userRepo.FindByID(ctx, userUUID)
	if err != nil {
		authMetrics.RecordTokenRefresh(false, time.Since(start))
		return nil, fmt.Errorf("error finding user: %w", err)
	}
	if user == nil {
		authMetrics.RecordTokenRefresh(false, time.Since(start))
		return nil, ErrUserNotFound
	}
	if !user.IsActive {
		authMetrics.RecordTokenRefresh(false, time.Since(start))
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
			log.Warn("invalid schoolID in refresh token", "user_id", userID, "school_id", schoolIDFromToken)
			authMetrics.RecordTokenRefresh(false, time.Since(start))
			return nil, ErrInvalidRefreshToken
		}
		targetSchoolID = &sid
	}
	if targetSchoolID == nil {
		_, targetSchoolID = s.getUserSchools(ctx, userUUID)
	}

	// 4. Rebuild RBAC context.
	// Use a single FindByUser query to determine role scope in-memory,
	// then call buildUserContext only for the correct scope (global or school).
	var activeContext *auth.UserContext

	allRoles, rolesErr := s.userRoleRepo.FindByUser(ctx, userUUID)
	if rolesErr != nil {
		authMetrics.RecordTokenRefresh(false, time.Since(start))
		return nil, fmt.Errorf("error fetching user roles: %w", rolesErr)
	}

	hasGlobalRole := false
	for _, r := range allRoles {
		if r.SchoolID == nil {
			hasGlobalRole = true
			break
		}
	}

	if hasGlobalRole {
		globalContext := s.buildUserContext(ctx, userUUID, nil)
		if globalContext != nil {
			if targetSchoolID != nil {
				globalContext.SchoolID = targetSchoolID.String()
				school, err := s.schoolRepo.FindByID(ctx, *targetSchoolID)
				if err == nil && school != nil {
					globalContext.SchoolName = school.Name
				}
			}
			activeContext = globalContext
		} else if targetSchoolID != nil {
			// Fallback: global context build failed (transient error, missing role record),
			// try school-scoped context so refresh doesn't fail for users with dual roles.
			activeContext = s.buildUserContext(ctx, userUUID, targetSchoolID)
		}
	} else if targetSchoolID != nil {
		activeContext = s.buildUserContext(ctx, userUUID, targetSchoolID)
	}

	if activeContext == nil {
		authMetrics.RecordTokenRefresh(false, time.Since(start))
		return nil, fmt.Errorf("user has no assigned roles")
	}

	// 5. Generate new access token (use DB email, not claim email)
	resp, err := s.tokenService.GenerateAccessTokenWithContext(userID, user.Email, activeContext)
	if err != nil {
		authMetrics.RecordTokenRefresh(false, time.Since(start))
		return nil, fmt.Errorf("error generating access token: %w", err)
	}

	// 6. Rotate refresh token preserving the resolved schoolID
	newRefreshJWT, _, err := s.tokenService.GenerateRefreshJWT(userID, user.Email, activeContext.SchoolID)
	if err != nil {
		authMetrics.RecordTokenRefresh(false, time.Since(start))
		return nil, fmt.Errorf("error generating refresh token: %w", err)
	}

	resp.RefreshToken = newRefreshJWT
	resp.ActiveContext = &dto.UserContextDTO{
		RoleID:           activeContext.RoleID,
		RoleName:         activeContext.RoleName,
		SchoolID:         activeContext.SchoolID,
		SchoolName:       activeContext.SchoolName,
		AcademicUnitID:   activeContext.AcademicUnitID,
		AcademicUnitName: activeContext.AcademicUnitName,
		Permissions:      activeContext.Permissions,
	}

	log.Info("token refreshed", "user_id", userID, "email", user.Email)

	authMetrics.RecordTokenRefresh(true, time.Since(start))
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

	// Parallel school lookups with bounded concurrency
	const maxConcurrent = 5
	type schoolResult struct {
		info dto.SchoolInfo
		ok   bool
	}
	results := make([]schoolResult, len(uniqueSchoolIDs))
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrent)
	for i, sid := range uniqueSchoolIDs {
		wg.Add(1)
		go func(idx int, schoolID uuid.UUID) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
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
func (s *authService) SwitchContext(ctx context.Context, userID, targetSchoolID, academicUnitID string) (*dto.SwitchContextResponse, error) {
	log := logger.FromContext(ctx)
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

	// Set academic unit if provided in request, validating:
	// 1. UUID is valid (already enforced by DTO binding:"omitempty,uuid")
	// 2. Unit exists and belongs to the target school
	// 3. User has an active membership in that unit
	if academicUnitID != "" {
		unitUUID, err := uuid.Parse(academicUnitID)
		if err != nil {
			return nil, fmt.Errorf("invalid academic_unit_id: %w", err)
		}
		unit, err := s.academicUnitRepo.FindByID(ctx, unitUUID)
		if err != nil || unit == nil {
			return nil, ErrAcademicUnitNotFound
		}
		if unit.SchoolID.String() != targetSchoolID {
			return nil, ErrAcademicUnitSchoolMismatch
		}

		// Security: verify the user has an active membership in this unit
		active := true
		userMemberships, _, err := s.membershipRepo.FindByUser(ctx, userUUID, sharedrepo.ListFilters{IsActive: &active})
		if err != nil {
			return nil, fmt.Errorf("error verifying unit membership: %w", err)
		}
		hasUnitAccess := false
		for _, m := range userMemberships {
			if m.SchoolID == schoolUUID && m.IsActive {
				if m.AcademicUnitID == nil {
					// School-level membership grants access to all units in the school
					hasUnitAccess = true
					break
				}
				if *m.AcademicUnitID == unitUUID {
					hasUnitAccess = true
					break
				}
			}
		}
		// Global-role users (e.g. super_admin) may not have a membership — allow them through
		if !hasUnitAccess && membership != nil {
			s.logger.Warn("switch-context: user has no membership in requested unit",
				"user_id", userID,
				"school_id", targetSchoolID,
				"academic_unit_id", academicUnitID,
			)
			return nil, ErrUnauthorizedUnit
		}

		activeContext.AcademicUnitID = academicUnitID
		activeContext.AcademicUnitName = unit.Name

		// Recompute permissions with unit context only for membership-based users.
		// Global roles (e.g. super_admin) have no membership — their permissions
		// were already resolved without school/unit filter and must not be overwritten.
		if membership != nil {
			updatedPerms, err := s.userRoleRepo.GetUserPermissions(ctx, userUUID, &schoolUUID, &unitUUID)
			if err == nil {
				activeContext.Permissions = updatedPerms
			}
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
		Metadata:     map[string]interface{}{"new_school_id": targetSchoolID, "academic_unit_id": academicUnitID},
	})

	log.Info("context switched",
		"entity_type", "auth_context",
		"user_id", userID,
		"new_school_id", targetSchoolID,
		"academic_unit_id", academicUnitID,
		"new_role", activeContext.RoleName,
	)

	return &dto.SwitchContextResponse{
		AccessToken:  tokenResponse.AccessToken,
		RefreshToken: tokenResponse.RefreshToken,
		ExpiresIn:    tokenResponse.ExpiresIn,
		TokenType:    tokenResponse.TokenType,
		Context: &dto.ContextInfo{
			SchoolID:         targetSchoolID,
			SchoolName:       activeContext.SchoolName,
			AcademicUnitID:   activeContext.AcademicUnitID,
			AcademicUnitName: activeContext.AcademicUnitName,
			Role:             activeContext.RoleName,
			UserID:           userID,
			Email:            user.Email,
		},
	}, nil
}

// GetAvailableContexts returns all available contexts (roles/schools/units) for the user.
// It merges data from iam.user_roles (for RBAC roles) and academic.memberships (for unit assignments).
func (s *authService) GetAvailableContexts(ctx context.Context, userID string, currentContext *auth.UserContext) (*dto.AvailableContextsResponse, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id: %w", err)
	}

	// Phase 1: Fetch user_roles and memberships in parallel
	var userRoles []*entities.UserRole
	var userRolesErr error
	var memberships []*entities.Membership
	var membershipsErr error

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		userRoles, userRolesErr = s.userRoleRepo.FindByUser(ctx, userUUID)
	}()
	go func() {
		defer wg.Done()
		memberships, _, membershipsErr = s.membershipRepo.FindByUser(ctx, userUUID, sharedrepo.ListFilters{})
	}()
	wg.Wait()

	if userRolesErr != nil {
		return nil, fmt.Errorf("error fetching roles: %w", userRolesErr)
	}
	if membershipsErr != nil {
		s.logger.Warn("error fetching memberships for available contexts", "user_id", userID, "error", membershipsErr)
		// Continue without memberships — roles are still available
	}

	// Phase 2: Build caches for role, school, and unit names
	roleCache := make(map[string]string)   // roleID -> roleName
	schoolCache := make(map[string]string) // schoolID -> schoolName
	unitCache := make(map[string]string)   // unitID -> unitName

	resolveRole := func(roleID uuid.UUID) string {
		key := roleID.String()
		if name, ok := roleCache[key]; ok {
			return name
		}
		role, err := s.roleRepo.FindByID(ctx, roleID)
		if err != nil || role == nil {
			return ""
		}
		roleCache[key] = role.Name
		return role.Name
	}

	resolveSchool := func(schoolID uuid.UUID) string {
		key := schoolID.String()
		if name, ok := schoolCache[key]; ok {
			return name
		}
		school, err := s.schoolRepo.FindByID(ctx, schoolID)
		if err != nil || school == nil {
			return ""
		}
		schoolCache[key] = school.Name
		return school.Name
	}

	resolveUnit := func(unitID uuid.UUID) string {
		key := unitID.String()
		if name, ok := unitCache[key]; ok {
			return name
		}
		unit, err := s.academicUnitRepo.FindByID(ctx, unitID)
		if err != nil || unit == nil {
			return ""
		}
		unitCache[key] = unit.Name
		return unit.Name
	}

	// Phase 3: Build available contexts from user_roles (base RBAC entries)
	var available []*dto.UserContextDTO

	// Track which (school, unit) combinations come from memberships to avoid duplicates
	type schoolUnitKey struct{ schoolID, unitID string }
	membershipKeys := make(map[schoolUnitKey]bool)

	// First, add membership-based contexts (these have unit info)
	for _, m := range memberships {
		if !m.IsActive {
			continue
		}
		schoolID := m.SchoolID.String()
		schoolName := resolveSchool(m.SchoolID)

		unitID := ""
		unitName := ""
		if m.AcademicUnitID != nil {
			unitID = m.AcademicUnitID.String()
			unitName = resolveUnit(*m.AcademicUnitID)
		}

		membershipKeys[schoolUnitKey{schoolID, unitID}] = true

		// Find matching role from user_roles — try (school, unit) first, then school-only, then global
		roleName := ""
		roleID := ""
		if m.AcademicUnitID != nil {
			for _, ur := range userRoles {
				if ur.SchoolID != nil && ur.SchoolID.String() == schoolID &&
					ur.AcademicUnitID != nil && ur.AcademicUnitID.String() == unitID {
					roleID = ur.RoleID.String()
					roleName = resolveRole(ur.RoleID)
					break
				}
			}
		}
		if roleName == "" {
			for _, ur := range userRoles {
				if ur.SchoolID != nil && ur.SchoolID.String() == schoolID && ur.AcademicUnitID == nil {
					roleID = ur.RoleID.String()
					roleName = resolveRole(ur.RoleID)
					break
				}
			}
		}
		// If no school-specific role, use global role
		if roleName == "" {
			for _, ur := range userRoles {
				if ur.SchoolID == nil {
					roleID = ur.RoleID.String()
					roleName = resolveRole(ur.RoleID)
					break
				}
			}
		}

		// Skip entries where no role was found
		if roleName == "" {
			continue
		}

		// Populate permissions for membership-based contexts
		var schoolUUID *uuid.UUID
		sid := m.SchoolID
		schoolUUID = &sid

		var unitUUID *uuid.UUID
		if m.AcademicUnitID != nil {
			unitUUID = m.AcademicUnitID
		}

		perms, err := s.userRoleRepo.GetUserPermissions(ctx, userUUID, schoolUUID, unitUUID)
		if err != nil {
			s.logger.Warn("error fetching permissions for membership context",
				"user_id", userUUID.String(), "school_id", schoolID, "error", err)
		}
		if perms == nil {
			perms = []string{}
		}

		available = append(available, &dto.UserContextDTO{
			RoleID:           roleID,
			RoleName:         roleName,
			SchoolID:         schoolID,
			SchoolName:       schoolName,
			AcademicUnitID:   unitID,
			AcademicUnitName: unitName,
			Permissions:      perms,
		})
	}

	// Then, add user_role entries that don't overlap with memberships (global roles, school-level admins)
	for _, ur := range userRoles {
		schoolID := ""
		if ur.SchoolID != nil {
			schoolID = ur.SchoolID.String()
		}
		unitID := ""
		if ur.AcademicUnitID != nil {
			unitID = ur.AcademicUnitID.String()
		}

		key := schoolUnitKey{schoolID, unitID}
		if membershipKeys[key] {
			continue // Already covered by a membership entry
		}

		// Check if this school+empty-unit is already covered by membership entries with units
		if unitID == "" && schoolID != "" {
			alreadyCovered := false
			for mk := range membershipKeys {
				if mk.schoolID == schoolID {
					alreadyCovered = true
					break
				}
			}
			if alreadyCovered {
				continue
			}
		}

		roleName := resolveRole(ur.RoleID)
		if roleName == "" {
			continue
		}

		schoolName := ""
		if ur.SchoolID != nil {
			schoolName = resolveSchool(*ur.SchoolID)
		}

		unitName := ""
		if ur.AcademicUnitID != nil {
			unitName = resolveUnit(*ur.AcademicUnitID)
		}

		permissions, err := s.userRoleRepo.GetUserPermissions(ctx, userUUID, ur.SchoolID, ur.AcademicUnitID)
		if err != nil {
			s.logger.Warn("error fetching user permissions", "user_id", userUUID.String(), "error", err)
		}
		if permissions == nil {
			permissions = []string{}
		}

		available = append(available, &dto.UserContextDTO{
			RoleID:           ur.RoleID.String(),
			RoleName:         roleName,
			SchoolID:         schoolID,
			SchoolName:       schoolName,
			AcademicUnitID:   unitID,
			AcademicUnitName: unitName,
			Permissions:      permissions,
		})
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
			if err != nil {
				s.logger.Warn("error fetching school for RBAC context",
					"school_id", schoolID.String(),
					"error", err,
				)
			} else if school != nil {
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

	// Auto-populate AcademicUnitID if user has exactly 1 unit in this school
	if schoolID != nil {
		s.autoPopulateUnit(ctx, userID, schoolID, uc)
	}

	return uc
}

// GetSchoolUnits returns all active academic units for a given school.
// Used by users with context:browse_units permission to select a unit.
func (s *authService) GetSchoolUnits(ctx context.Context, schoolID string) (*dto.SchoolUnitsResponse, error) {
	schoolUUID, err := uuid.Parse(schoolID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSchoolID, err)
	}

	// Filter only active units. The repository already hardcodes is_active = true,
	// but we pass it explicitly for clarity and defense-in-depth.
	active := true
	units, _, err := s.academicUnitRepo.FindBySchoolID(ctx, schoolUUID, sharedrepo.ListFilters{IsActive: &active})
	if err != nil {
		return nil, fmt.Errorf("error fetching units: %w", err)
	}

	result := make([]dto.UnitInfoDTO, 0, len(units))
	for _, u := range units {
		// Defense-in-depth: skip inactive units even if the repo returned them
		if !u.IsActive {
			continue
		}
		result = append(result, dto.UnitInfoDTO{
			ID:   u.ID.String(),
			Name: u.Name,
			Type: u.Type,
		})
	}

	return &dto.SchoolUnitsResponse{
		Units: result,
		Total: int64(len(result)),
	}, nil
}

// autoPopulateUnit sets AcademicUnitID on the context if the user has exactly 1 active unit in the school.
func (s *authService) autoPopulateUnit(ctx context.Context, userID uuid.UUID, schoolID *uuid.UUID, uc *auth.UserContext) {
	if schoolID == nil || uc == nil {
		return
	}
	memberships, _, err := s.membershipRepo.FindByUser(ctx, userID, sharedrepo.ListFilters{})
	if err != nil {
		s.logger.Warn("error fetching memberships for unit auto-select",
			"user_id", userID.String(), "error", err)
		return
	}
	seen := make(map[uuid.UUID]struct{})
	var unitIDs []uuid.UUID
	for _, m := range memberships {
		if m.SchoolID == *schoolID && m.AcademicUnitID != nil && m.IsActive {
			uid := *m.AcademicUnitID
			if _, exists := seen[uid]; !exists {
				seen[uid] = struct{}{}
				unitIDs = append(unitIDs, uid)
			}
		}
	}
	if len(unitIDs) == 1 {
		uc.AcademicUnitID = unitIDs[0].String()
		if unit, err := s.academicUnitRepo.FindByID(ctx, unitIDs[0]); err == nil && unit != nil {
			uc.AcademicUnitName = unit.Name
		}
	}
}

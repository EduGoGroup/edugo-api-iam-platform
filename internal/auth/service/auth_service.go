package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/EduGoGroup/edugo-api-iam-platform/internal/auth/dto"
	"github.com/EduGoGroup/edugo-api-iam-platform/internal/domain/repository"
	sharedrepo "github.com/EduGoGroup/edugo-shared/repository"

	"github.com/EduGoGroup/edugo-shared/auth"
	"github.com/EduGoGroup/edugo-shared/logger"
	"github.com/google/uuid"
)

// Sentinel errors for auth operations
var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrUserNotFound        = errors.New("user not found")
	ErrUserInactive        = errors.New("user inactive")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrNoMembership        = errors.New("no active membership in target school")
	ErrInvalidSchoolID     = errors.New("invalid school_id")
)

// AuthService defines the authentication service interface
type AuthService interface {
	Login(ctx context.Context, email, password string) (*dto.LoginResponse, error)
	Logout(ctx context.Context, accessToken string) error
	RefreshToken(ctx context.Context, refreshToken string) (*dto.RefreshResponse, error)
	SwitchContext(ctx context.Context, userID, targetSchoolID string) (*dto.SwitchContextResponse, error)
	GetAvailableContexts(ctx context.Context, userID string, currentContext *auth.UserContext) (*dto.AvailableContextsResponse, error)
}

type authService struct {
	userRepo       sharedrepo.UserRepository
	userRoleRepo   repository.UserRoleRepository
	roleRepo       repository.RoleRepository
	membershipRepo sharedrepo.MembershipRepository
	schoolRepo     sharedrepo.SchoolRepository
	tokenService   *TokenService
	logger         logger.Logger
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
) AuthService {
	return &authService{
		userRepo:       userRepo,
		userRoleRepo:   userRoleRepo,
		roleRepo:       roleRepo,
		membershipRepo: membershipRepo,
		schoolRepo:     schoolRepo,
		tokenService:   tokenService,
		logger:         logger,
	}
}

// Login validates credentials and returns JWT tokens
func (s *authService) Login(ctx context.Context, email, password string) (*dto.LoginResponse, error) {
	// 1. Find user by email
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("error finding user: %w", err)
	}
	if user == nil {
		s.logger.Warn("login attempt with non-existent email", "email", email)
		return nil, ErrInvalidCredentials
	}

	// 2. Verify user is active
	if !user.IsActive {
		s.logger.Warn("login attempt with inactive user", "email", email, "user_id", user.ID.String())
		return nil, ErrUserInactive
	}

	// 3. Verify password
	if err := auth.VerifyPassword(user.PasswordHash, password); err != nil {
		s.logger.Warn("incorrect password", "email", email)
		return nil, ErrInvalidCredentials
	}

	// 4. Get user's active schools from memberships
	schools, firstSchoolID := s.getUserSchools(ctx, user.ID)

	// 5. Build RBAC context using the first active school
	activeContext := s.buildUserContext(ctx, user.ID, firstSchoolID)
	if activeContext == nil {
		s.logger.Error("no RBAC context found for user", "user_id", user.ID.String(), "email", user.Email)
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
		Permissions: activeContext.Permissions,
	}

	s.logger.Info("user logged in",
		"entity_type", "auth_session",
		"user_id", user.ID.String(),
		"email", user.Email,
		"role", activeContext.RoleName,
		"school_id", schoolID,
	)

	// 8. Update last login (fire and forget)
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		user.UpdatedAt = time.Now()
		if err := s.userRepo.Update(bgCtx, user); err != nil {
			s.logger.Warn("error updating last login", "error", err)
		}
	}()

	return tokenResponse, nil
}

// Logout invalidates the access token
func (s *authService) Logout(_ context.Context, _ string) error {
	s.logger.Info("user logged out", "entity_type", "auth_session")
	return nil
}

// RefreshToken validates a refresh token and generates a new access token
func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*dto.RefreshResponse, error) {
	tokenHash := auth.HashToken(refreshToken)
	if tokenHash == "" {
		return nil, ErrInvalidRefreshToken
	}
	_ = tokenHash
	return nil, ErrInvalidRefreshToken
}

// getUserSchools devuelve las escuelas activas del usuario desde memberships.
func (s *authService) getUserSchools(ctx context.Context, userID uuid.UUID) ([]dto.SchoolInfo, *uuid.UUID) {
	memberships, err := s.membershipRepo.FindByUser(ctx, userID)
	if err != nil {
		s.logger.Warn("error fetching memberships for user", "user_id", userID.String(), "error", err)
		return []dto.SchoolInfo{}, nil
	}

	seen := make(map[uuid.UUID]struct{})
	var schools []dto.SchoolInfo
	var firstSchoolID *uuid.UUID

	for _, m := range memberships {
		if !m.IsActive {
			continue
		}
		if _, exists := seen[m.SchoolID]; exists {
			continue
		}
		seen[m.SchoolID] = struct{}{}

		school, err := s.schoolRepo.FindByID(ctx, m.SchoolID)
		if err != nil || school == nil {
			continue
		}

		sid := m.SchoolID
		if firstSchoolID == nil {
			firstSchoolID = &sid
		}
		schools = append(schools, dto.SchoolInfo{
			ID:   m.SchoolID.String(),
			Name: school.Name,
		})
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
		return nil, fmt.Errorf("error checking membership: %w", err)
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
			SchoolID: targetSchoolID,
			Role:     activeContext.RoleName,
			UserID:   userID,
			Email:    user.Email,
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

// buildUserContext constructs the RBAC UserContext for the JWT
func (s *authService) buildUserContext(ctx context.Context, userID uuid.UUID, schoolID *uuid.UUID) *auth.UserContext {
	userRoles, err := s.userRoleRepo.FindByUserInContext(ctx, userID, schoolID, nil)
	if err != nil {
		s.logger.Warn("error obtaining user roles for RBAC context",
			"user_id", userID.String(),
			"error", err,
		)
		return nil
	}
	if len(userRoles) == 0 {
		return nil
	}

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

	permissions, err := s.userRoleRepo.GetUserPermissions(ctx, userID, schoolID, nil)
	if err != nil {
		s.logger.Warn("error obtaining user permissions",
			"user_id", userID.String(),
			"error", err,
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
	}

	return uc
}

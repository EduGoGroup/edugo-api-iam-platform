package service

import (
	"context"
	"time"

	"github.com/EduGoGroup/edugo-api-iam-platform/internal/application/dto"
	"github.com/EduGoGroup/edugo-api-iam-platform/internal/domain/repository"
	"github.com/EduGoGroup/edugo-infrastructure/postgres/entities"
	"github.com/EduGoGroup/edugo-shared/common/errors"
	"github.com/EduGoGroup/edugo-shared/logger"
	sharedrepo "github.com/EduGoGroup/edugo-shared/repository"
	"github.com/google/uuid"
)

// RoleService defines the role service interface
type RoleService interface {
	GetRoles(ctx context.Context, scope string, filters sharedrepo.ListFilters) (*dto.RolesResponse, error)
	GetRole(ctx context.Context, id string) (*dto.RoleDTO, error)
	GetRolePermissions(ctx context.Context, roleID string) (*dto.PermissionsResponse, error)
	GetUserRoles(ctx context.Context, userID string) (*dto.UserRolesResponse, error)
	GrantRoleToUser(ctx context.Context, userID string, req *dto.GrantRoleRequest, grantedBy string) (*dto.GrantRoleResponse, error)
	RevokeRoleFromUser(ctx context.Context, userID, roleID string) error
}

type roleService struct {
	roleRepo       repository.RoleRepository
	permissionRepo repository.PermissionRepository
	userRoleRepo   repository.UserRoleRepository
	logger         logger.Logger
}

// NewRoleService creates a new role service
func NewRoleService(roleRepo repository.RoleRepository, permissionRepo repository.PermissionRepository, userRoleRepo repository.UserRoleRepository, logger logger.Logger) RoleService {
	return &roleService{roleRepo: roleRepo, permissionRepo: permissionRepo, userRoleRepo: userRoleRepo, logger: logger}
}

func (s *roleService) GetRoles(ctx context.Context, scope string, filters sharedrepo.ListFilters) (*dto.RolesResponse, error) {
	var roles []*entities.Role
	var err error
	if scope != "" {
		roles, err = s.roleRepo.FindByScope(ctx, scope, filters)
	} else {
		roles, err = s.roleRepo.FindAll(ctx, filters)
	}
	if err != nil {
		return nil, errors.NewDatabaseError("list roles", err)
	}
	return &dto.RolesResponse{Roles: dto.ToRoleDTOList(roles)}, nil
}

func (s *roleService) GetRole(ctx context.Context, id string) (*dto.RoleDTO, error) {
	roleID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.NewValidationError("invalid role ID")
	}
	role, err := s.roleRepo.FindByID(ctx, roleID)
	if err != nil {
		return nil, errors.NewDatabaseError("find role", err)
	}
	if role == nil {
		return nil, errors.NewNotFoundError("role")
	}
	return dto.ToRoleDTO(role), nil
}

func (s *roleService) GetRolePermissions(ctx context.Context, roleID string) (*dto.PermissionsResponse, error) {
	id, err := uuid.Parse(roleID)
	if err != nil {
		return nil, errors.NewValidationError("invalid role ID")
	}
	perms, err := s.permissionRepo.FindByRole(ctx, id)
	if err != nil {
		return nil, errors.NewDatabaseError("find role permissions", err)
	}
	return &dto.PermissionsResponse{Permissions: dto.ToPermissionDTOList(perms)}, nil
}

func (s *roleService) GetUserRoles(ctx context.Context, userID string) (*dto.UserRolesResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.NewValidationError("invalid user ID")
	}
	userRoles, err := s.userRoleRepo.FindByUser(ctx, uid)
	if err != nil {
		return nil, errors.NewDatabaseError("find user roles", err)
	}

	roleCache := make(map[uuid.UUID]*entities.Role)
	for _, ur := range userRoles {
		if _, exists := roleCache[ur.RoleID]; !exists {
			role, err := s.roleRepo.FindByID(ctx, ur.RoleID)
			if err == nil && role != nil {
				roleCache[ur.RoleID] = role
			}
		}
	}

	dtos := make([]*dto.UserRoleDTO, len(userRoles))
	for i, ur := range userRoles {
		d := &dto.UserRoleDTO{
			ID:        ur.ID.String(),
			UserID:    ur.UserID.String(),
			RoleID:    ur.RoleID.String(),
			IsActive:  ur.IsActive,
			GrantedAt: ur.GrantedAt.Format(time.RFC3339),
		}
		if ur.SchoolID != nil {
			sid := ur.SchoolID.String()
			d.SchoolID = &sid
		}
		if ur.AcademicUnitID != nil {
			aid := ur.AcademicUnitID.String()
			d.AcademicUnitID = &aid
		}
		if role, exists := roleCache[ur.RoleID]; exists {
			d.RoleName = role.Name
		}
		dtos[i] = d
	}

	return &dto.UserRolesResponse{UserRoles: dtos}, nil
}

func (s *roleService) GrantRoleToUser(ctx context.Context, userID string, req *dto.GrantRoleRequest, grantedBy string) (*dto.GrantRoleResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.NewValidationError("invalid user ID")
	}
	roleID, err := uuid.Parse(req.RoleID)
	if err != nil {
		return nil, errors.NewValidationError("invalid role ID")
	}

	role, err := s.roleRepo.FindByID(ctx, roleID)
	if err != nil || role == nil {
		return nil, errors.NewValidationError("role not found")
	}

	var schoolID *uuid.UUID
	if req.SchoolID != nil && *req.SchoolID != "" {
		sid, err := uuid.Parse(*req.SchoolID)
		if err != nil {
			return nil, errors.NewValidationError("invalid school_id")
		}
		schoolID = &sid
	}

	var unitID *uuid.UUID
	if req.AcademicUnitID != nil && *req.AcademicUnitID != "" {
		aid, err := uuid.Parse(*req.AcademicUnitID)
		if err != nil {
			return nil, errors.NewValidationError("invalid academic_unit_id")
		}
		unitID = &aid
	}

	hasRole, err := s.userRoleRepo.UserHasRole(ctx, uid, roleID, schoolID, unitID)
	if err != nil {
		return nil, errors.NewDatabaseError("check user role", err)
	}
	if hasRole {
		return nil, errors.NewAlreadyExistsError("user_role")
	}

	var grantedByUUID *uuid.UUID
	if grantedBy != "" {
		gid, err := uuid.Parse(grantedBy)
		if err == nil {
			grantedByUUID = &gid
		}
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			return nil, errors.NewValidationError("invalid expires_at format, use RFC3339")
		}
		expiresAt = &t
	}

	now := time.Now()
	userRole := &entities.UserRole{
		ID:             uuid.New(),
		UserID:         uid,
		RoleID:         roleID,
		SchoolID:       schoolID,
		AcademicUnitID: unitID,
		IsActive:       true,
		GrantedBy:      grantedByUUID,
		GrantedAt:      now,
		ExpiresAt:      expiresAt,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.userRoleRepo.Grant(ctx, userRole); err != nil {
		return nil, errors.NewDatabaseError("grant role", err)
	}

	s.logger.Info("role granted", "entity_type", "user_role", "user_id", userID, "role_id", req.RoleID, "role_name", role.Name)

	d := &dto.UserRoleDTO{
		ID:        userRole.ID.String(),
		UserID:    userRole.UserID.String(),
		RoleID:    userRole.RoleID.String(),
		RoleName:  role.Name,
		IsActive:  userRole.IsActive,
		GrantedAt: userRole.GrantedAt.Format(time.RFC3339),
	}
	if userRole.SchoolID != nil {
		sid := userRole.SchoolID.String()
		d.SchoolID = &sid
	}
	if userRole.AcademicUnitID != nil {
		aid := userRole.AcademicUnitID.String()
		d.AcademicUnitID = &aid
	}

	return &dto.GrantRoleResponse{UserRole: d}, nil
}

func (s *roleService) RevokeRoleFromUser(ctx context.Context, userID, roleID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return errors.NewValidationError("invalid user ID")
	}
	rid, err := uuid.Parse(roleID)
	if err != nil {
		return errors.NewValidationError("invalid role ID")
	}
	if err := s.userRoleRepo.RevokeByUserAndRole(ctx, uid, rid, nil, nil); err != nil {
		return errors.NewDatabaseError("revoke role", err)
	}
	s.logger.Info("role revoked", "entity_type", "user_role", "user_id", userID, "role_id", roleID)
	return nil
}

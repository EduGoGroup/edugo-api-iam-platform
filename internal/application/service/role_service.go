package service

import (
	"context"
	"time"

	"github.com/EduGoGroup/edugo-api-iam-platform/internal/application/dto"
	"github.com/EduGoGroup/edugo-api-iam-platform/internal/domain/repository"
	"github.com/EduGoGroup/edugo-infrastructure/postgres/entities"
	"github.com/EduGoGroup/edugo-shared/audit"
	"github.com/EduGoGroup/edugo-shared/common/errors"
	"github.com/EduGoGroup/edugo-shared/logger"
	sharedrepo "github.com/EduGoGroup/edugo-shared/repository"
	"github.com/google/uuid"
)

var validScopes = map[string]bool{"system": true, "school": true, "unit": true, "platform": true}

// RoleService defines the role service interface
type RoleService interface {
	GetRoles(ctx context.Context, scope string, filters sharedrepo.ListFilters) (*dto.RolesResponse, error)
	GetRole(ctx context.Context, id string) (*dto.RoleDTO, error)
	CreateRole(ctx context.Context, req *dto.CreateRoleRequest) (*dto.RoleDTO, error)
	UpdateRole(ctx context.Context, id string, req *dto.UpdateRoleRequest) (*dto.RoleDTO, error)
	DeleteRole(ctx context.Context, id string) error
	GetRolePermissions(ctx context.Context, roleID string) (*dto.PermissionsResponse, error)
	AssignPermission(ctx context.Context, roleID string, req *dto.AssignPermissionRequest) (*dto.RolePermissionResponse, error)
	RevokePermission(ctx context.Context, roleID, permissionID string) error
	BulkReplacePermissions(ctx context.Context, roleID string, req *dto.BulkPermissionsRequest) (*dto.PermissionsResponse, error)
	GetUserRoles(ctx context.Context, userID string) (*dto.UserRolesResponse, error)
	GrantRoleToUser(ctx context.Context, userID string, req *dto.GrantRoleRequest, grantedBy string) (*dto.GrantRoleResponse, error)
	RevokeRoleFromUser(ctx context.Context, userID, roleID string) error
}

type roleService struct {
	roleRepo       repository.RoleRepository
	permissionRepo repository.PermissionRepository
	userRoleRepo   repository.UserRoleRepository
	rolePermRepo   repository.RolePermissionRepository
	logger         logger.Logger
	auditLogger    audit.AuditLogger
}

// NewRoleService creates a new role service
func NewRoleService(roleRepo repository.RoleRepository, permissionRepo repository.PermissionRepository, userRoleRepo repository.UserRoleRepository, rolePermRepo repository.RolePermissionRepository, logger logger.Logger, auditLogger audit.AuditLogger) RoleService {
	return &roleService{roleRepo: roleRepo, permissionRepo: permissionRepo, userRoleRepo: userRoleRepo, rolePermRepo: rolePermRepo, logger: logger, auditLogger: auditLogger}
}

func (s *roleService) GetRoles(ctx context.Context, scope string, filters sharedrepo.ListFilters) (*dto.RolesResponse, error) {
	var roles []*entities.Role
	var total int
	var err error
	if scope != "" {
		roles, total, err = s.roleRepo.FindByScope(ctx, scope, filters)
	} else {
		roles, total, err = s.roleRepo.FindAll(ctx, filters)
	}
	if err != nil {
		return nil, errors.NewDatabaseError("list roles", err)
	}
	page := filters.Page
	if page == 0 {
		page = 1
	}
	limit := filters.Limit
	if filters.Page > 0 && filters.Limit == 0 {
		limit = 50
	} else if limit == 0 {
		limit = total
	}
	return &dto.RolesResponse{
		Roles: dto.ToRoleDTOList(roles),
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
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

func (s *roleService) CreateRole(ctx context.Context, req *dto.CreateRoleRequest) (*dto.RoleDTO, error) {
	if !validScopes[req.Scope] {
		return nil, errors.NewValidationError("scope must be system, school, or unit")
	}

	now := time.Now()
	role := &entities.Role{
		ID:          uuid.New(),
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Scope:       req.Scope,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if req.Description != "" {
		role.Description = &req.Description
	}

	if err := s.roleRepo.Create(ctx, role); err != nil {
		return nil, errors.NewDatabaseError("create role", err)
	}

	_ = s.auditLogger.Log(ctx, audit.AuditEvent{
		Action:       "create",
		ResourceType: "role",
		ResourceID:   role.ID.String(),
		Severity:     audit.SeverityCritical,
		Category:     audit.CategoryAdmin,
		Metadata:     map[string]interface{}{"role_name": role.Name, "scope": role.Scope},
	})
	s.logger.Info("entity created", "entity_type", "role", "entity_id", role.ID.String(), "name", role.Name)
	return dto.ToRoleDTO(role), nil
}

func (s *roleService) UpdateRole(ctx context.Context, id string, req *dto.UpdateRoleRequest) (*dto.RoleDTO, error) {
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

	if req.Name != nil {
		role.Name = *req.Name
	}
	if req.DisplayName != nil {
		role.DisplayName = *req.DisplayName
	}
	if req.Description != nil {
		role.Description = req.Description
	}
	if req.Scope != nil {
		if !validScopes[*req.Scope] {
			return nil, errors.NewValidationError("scope must be system, school, or unit")
		}
		role.Scope = *req.Scope
	}

	role.UpdatedAt = time.Now()
	if err := s.roleRepo.Update(ctx, role); err != nil {
		return nil, errors.NewDatabaseError("update role", err)
	}

	s.logger.Info("entity updated", "entity_type", "role", "entity_id", id)
	return dto.ToRoleDTO(role), nil
}

func (s *roleService) DeleteRole(ctx context.Context, id string) error {
	roleID, err := uuid.Parse(id)
	if err != nil {
		return errors.NewValidationError("invalid role ID")
	}
	role, err := s.roleRepo.FindByID(ctx, roleID)
	if err != nil {
		return errors.NewDatabaseError("find role", err)
	}
	if role == nil {
		return errors.NewNotFoundError("role")
	}

	hasActive, err := s.roleRepo.HasActiveUserRoles(ctx, roleID)
	if err != nil {
		return errors.NewDatabaseError("check active user roles", err)
	}
	if hasActive {
		return errors.NewConflictError("cannot delete role with active user assignments")
	}

	if err := s.roleRepo.SoftDelete(ctx, roleID); err != nil {
		return errors.NewDatabaseError("delete role", err)
	}

	_ = s.auditLogger.Log(ctx, audit.AuditEvent{
		Action:       "delete",
		ResourceType: "role",
		ResourceID:   id,
		Severity:     audit.SeverityCritical,
		Category:     audit.CategoryAdmin,
		Metadata:     map[string]interface{}{"role_name": role.Name},
	})
	s.logger.Info("entity deleted", "entity_type", "role", "entity_id", id)
	return nil
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

func (s *roleService) AssignPermission(ctx context.Context, roleID string, req *dto.AssignPermissionRequest) (*dto.RolePermissionResponse, error) {
	rid, err := uuid.Parse(roleID)
	if err != nil {
		return nil, errors.NewValidationError("invalid role ID")
	}
	pid, err := uuid.Parse(req.PermissionID)
	if err != nil {
		return nil, errors.NewValidationError("invalid permission ID")
	}

	role, err := s.roleRepo.FindByID(ctx, rid)
	if err != nil {
		return nil, errors.NewDatabaseError("find role", err)
	}
	if role == nil {
		return nil, errors.NewNotFoundError("role")
	}

	perm, err := s.permissionRepo.FindByID(ctx, pid)
	if err != nil {
		return nil, errors.NewDatabaseError("find permission", err)
	}
	if perm == nil {
		return nil, errors.NewNotFoundError("permission")
	}

	exists, err := s.rolePermRepo.Exists(ctx, rid, pid)
	if err != nil {
		return nil, errors.NewDatabaseError("check role permission", err)
	}
	if exists {
		return nil, errors.NewAlreadyExistsError("role_permission")
	}

	rp := &entities.RolePermission{
		ID:           uuid.New(),
		RoleID:       rid,
		PermissionID: pid,
	}
	if err := s.rolePermRepo.Assign(ctx, rp); err != nil {
		return nil, errors.NewDatabaseError("assign permission", err)
	}

	s.logger.Info("permission assigned to role", "role_id", roleID, "permission_id", req.PermissionID)
	return &dto.RolePermissionResponse{RoleID: roleID, PermissionID: req.PermissionID}, nil
}

func (s *roleService) RevokePermission(ctx context.Context, roleID, permissionID string) error {
	rid, err := uuid.Parse(roleID)
	if err != nil {
		return errors.NewValidationError("invalid role ID")
	}
	pid, err := uuid.Parse(permissionID)
	if err != nil {
		return errors.NewValidationError("invalid permission ID")
	}

	if err := s.rolePermRepo.Revoke(ctx, rid, pid); err != nil {
		return errors.NewDatabaseError("revoke permission", err)
	}

	s.logger.Info("permission revoked from role", "role_id", roleID, "permission_id", permissionID)
	return nil
}

func (s *roleService) BulkReplacePermissions(ctx context.Context, roleID string, req *dto.BulkPermissionsRequest) (*dto.PermissionsResponse, error) {
	rid, err := uuid.Parse(roleID)
	if err != nil {
		return nil, errors.NewValidationError("invalid role ID")
	}

	role, err := s.roleRepo.FindByID(ctx, rid)
	if err != nil {
		return nil, errors.NewDatabaseError("find role", err)
	}
	if role == nil {
		return nil, errors.NewNotFoundError("role")
	}

	permIDs := make([]uuid.UUID, len(req.PermissionIDs))
	for i, pidStr := range req.PermissionIDs {
		pid, err := uuid.Parse(pidStr)
		if err != nil {
			return nil, errors.NewValidationError("invalid permission ID: " + pidStr)
		}
		perm, err := s.permissionRepo.FindByID(ctx, pid)
		if err != nil {
			return nil, errors.NewDatabaseError("find permission", err)
		}
		if perm == nil {
			return nil, errors.NewNotFoundError("permission " + pidStr)
		}
		permIDs[i] = pid
	}

	if err := s.rolePermRepo.BulkReplace(ctx, rid, permIDs); err != nil {
		return nil, errors.NewDatabaseError("bulk replace permissions", err)
	}

	s.logger.Info("permissions bulk replaced", "role_id", roleID, "count", len(permIDs))

	perms, err := s.permissionRepo.FindByRole(ctx, rid)
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

	_ = s.auditLogger.Log(ctx, audit.AuditEvent{
		Action:       "assign",
		ResourceType: "user_role",
		ResourceID:   userRole.ID.String(),
		Severity:     audit.SeverityCritical,
		Category:     audit.CategoryAdmin,
		Metadata:     map[string]interface{}{"user_id": userID, "role_id": req.RoleID, "role_name": role.Name},
	})
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
	_ = s.auditLogger.Log(ctx, audit.AuditEvent{
		Action:       "revoke",
		ResourceType: "user_role",
		Severity:     audit.SeverityCritical,
		Category:     audit.CategoryAdmin,
		Metadata:     map[string]interface{}{"user_id": userID, "role_id": roleID},
	})
	s.logger.Info("role revoked", "entity_type", "user_role", "user_id", userID, "role_id", roleID)
	return nil
}

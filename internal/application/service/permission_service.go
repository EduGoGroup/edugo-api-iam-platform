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

// PermissionService defines the permission service interface
type PermissionService interface {
	ListPermissions(ctx context.Context, filters sharedrepo.ListFilters) (*dto.PermissionsResponse, error)
	GetPermission(ctx context.Context, id string) (*dto.PermissionDTO, error)
	CreatePermission(ctx context.Context, req *dto.CreatePermissionRequest) (*dto.PermissionDTO, error)
	UpdatePermission(ctx context.Context, id string, req *dto.UpdatePermissionRequest) (*dto.PermissionDTO, error)
	DeletePermission(ctx context.Context, id string) error
}

type permissionService struct {
	permissionRepo repository.PermissionRepository
	resourceRepo   repository.ResourceRepository
	logger         logger.Logger
}

// NewPermissionService creates a new permission service
func NewPermissionService(permissionRepo repository.PermissionRepository, resourceRepo repository.ResourceRepository, logger logger.Logger) PermissionService {
	return &permissionService{permissionRepo: permissionRepo, resourceRepo: resourceRepo, logger: logger}
}

func (s *permissionService) ListPermissions(ctx context.Context, filters sharedrepo.ListFilters) (*dto.PermissionsResponse, error) {
	perms, err := s.permissionRepo.FindAll(ctx, filters)
	if err != nil {
		return nil, errors.NewDatabaseError("list permissions", err)
	}
	return &dto.PermissionsResponse{Permissions: dto.ToPermissionDTOList(perms)}, nil
}

func (s *permissionService) GetPermission(ctx context.Context, id string) (*dto.PermissionDTO, error) {
	pid, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.NewValidationError("invalid permission ID")
	}
	perm, err := s.permissionRepo.FindByID(ctx, pid)
	if err != nil {
		return nil, errors.NewDatabaseError("find permission", err)
	}
	if perm == nil {
		return nil, errors.NewNotFoundError("permission")
	}
	return dto.ToPermissionDTO(perm), nil
}

func (s *permissionService) CreatePermission(ctx context.Context, req *dto.CreatePermissionRequest) (*dto.PermissionDTO, error) {
	if !permissionNameRegex.MatchString(req.Name) {
		return nil, errors.NewValidationError("name must match format resource:action (e.g. users:read)")
	}

	resourceID, err := uuid.Parse(req.ResourceID)
	if err != nil {
		return nil, errors.NewValidationError("invalid resource_id")
	}

	resource, err := s.resourceRepo.FindByID(ctx, resourceID)
	if err != nil {
		return nil, errors.NewDatabaseError("find resource", err)
	}
	if resource == nil {
		return nil, errors.NewNotFoundError("resource")
	}

	now := time.Now()
	perm := &entities.Permission{
		ID:          uuid.New(),
		Name:        req.Name,
		DisplayName: req.DisplayName,
		ResourceID:  resourceID,
		ResourceKey: resource.Key,
		Action:      req.Action,
		Scope:       req.Scope,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if req.Description != "" {
		perm.Description = &req.Description
	}

	if err := s.permissionRepo.Create(ctx, perm); err != nil {
		return nil, errors.NewDatabaseError("create permission", err)
	}

	s.logger.Info("entity created", "entity_type", "permission", "entity_id", perm.ID.String(), "name", perm.Name)
	return dto.ToPermissionDTO(perm), nil
}

func (s *permissionService) UpdatePermission(ctx context.Context, id string, req *dto.UpdatePermissionRequest) (*dto.PermissionDTO, error) {
	pid, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.NewValidationError("invalid permission ID")
	}
	perm, err := s.permissionRepo.FindByID(ctx, pid)
	if err != nil {
		return nil, errors.NewDatabaseError("find permission", err)
	}
	if perm == nil {
		return nil, errors.NewNotFoundError("permission")
	}

	if req.DisplayName != nil {
		perm.DisplayName = *req.DisplayName
	}
	if req.Description != nil {
		perm.Description = req.Description
	}
	if req.Scope != nil {
		perm.Scope = *req.Scope
	}

	perm.UpdatedAt = time.Now()
	if err := s.permissionRepo.Update(ctx, perm); err != nil {
		return nil, errors.NewDatabaseError("update permission", err)
	}

	s.logger.Info("entity updated", "entity_type", "permission", "entity_id", id)
	return dto.ToPermissionDTO(perm), nil
}

func (s *permissionService) DeletePermission(ctx context.Context, id string) error {
	pid, err := uuid.Parse(id)
	if err != nil {
		return errors.NewValidationError("invalid permission ID")
	}
	perm, err := s.permissionRepo.FindByID(ctx, pid)
	if err != nil {
		return errors.NewDatabaseError("find permission", err)
	}
	if perm == nil {
		return errors.NewNotFoundError("permission")
	}

	hasActive, err := s.permissionRepo.HasActiveRolePermissions(ctx, pid)
	if err != nil {
		return errors.NewDatabaseError("check active role permissions", err)
	}
	if hasActive {
		return errors.NewBusinessRuleError("cannot delete permission with active role assignments")
	}

	if err := s.permissionRepo.SoftDelete(ctx, pid); err != nil {
		return errors.NewDatabaseError("delete permission", err)
	}

	s.logger.Info("entity deleted", "entity_type", "permission", "entity_id", id)
	return nil
}

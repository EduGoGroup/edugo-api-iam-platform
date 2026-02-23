package service

import (
	"context"

	"github.com/EduGoGroup/edugo-api-iam-platform/internal/application/dto"
	"github.com/EduGoGroup/edugo-api-iam-platform/internal/domain/repository"
	"github.com/EduGoGroup/edugo-shared/common/errors"
	"github.com/EduGoGroup/edugo-shared/logger"
	"github.com/google/uuid"
)

// PermissionService defines the permission service interface
type PermissionService interface {
	ListPermissions(ctx context.Context) (*dto.PermissionsResponse, error)
	GetPermission(ctx context.Context, id string) (*dto.PermissionDTO, error)
}

type permissionService struct {
	permissionRepo repository.PermissionRepository
	logger         logger.Logger
}

// NewPermissionService creates a new permission service
func NewPermissionService(permissionRepo repository.PermissionRepository, logger logger.Logger) PermissionService {
	return &permissionService{permissionRepo: permissionRepo, logger: logger}
}

func (s *permissionService) ListPermissions(ctx context.Context) (*dto.PermissionsResponse, error) {
	perms, err := s.permissionRepo.FindAll(ctx)
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

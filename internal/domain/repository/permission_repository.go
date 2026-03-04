package repository

import (
	"context"

	"github.com/EduGoGroup/edugo-infrastructure/postgres/entities"
	sharedrepo "github.com/EduGoGroup/edugo-shared/repository"
	"github.com/google/uuid"
)

type PermissionRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*entities.Permission, error)
	FindAll(ctx context.Context, filters sharedrepo.ListFilters) ([]*entities.Permission, int, error)
	FindByRole(ctx context.Context, roleID uuid.UUID) ([]*entities.Permission, error)
	Create(ctx context.Context, perm *entities.Permission) error
	Update(ctx context.Context, perm *entities.Permission) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	HasActiveRolePermissions(ctx context.Context, permissionID uuid.UUID) (bool, error)
}

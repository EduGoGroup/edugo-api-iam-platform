package repository

import (
	"context"

	"github.com/EduGoGroup/edugo-infrastructure/postgres/entities"
	"github.com/google/uuid"
)

type RolePermissionRepository interface {
	Assign(ctx context.Context, rp *entities.RolePermission) error
	Revoke(ctx context.Context, roleID, permissionID uuid.UUID) error
	BulkReplace(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error
	FindByRole(ctx context.Context, roleID uuid.UUID) ([]*entities.RolePermission, error)
	Exists(ctx context.Context, roleID, permissionID uuid.UUID) (bool, error)
}

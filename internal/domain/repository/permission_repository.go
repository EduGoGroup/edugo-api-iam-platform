package repository

import (
	"context"

	"github.com/EduGoGroup/edugo-infrastructure/postgres/entities"
	"github.com/google/uuid"
)

type PermissionRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*entities.Permission, error)
	FindAll(ctx context.Context) ([]*entities.Permission, error)
	FindByRole(ctx context.Context, roleID uuid.UUID) ([]*entities.Permission, error)
}

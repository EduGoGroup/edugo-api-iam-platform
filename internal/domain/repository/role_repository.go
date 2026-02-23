package repository

import (
	"context"

	"github.com/EduGoGroup/edugo-infrastructure/postgres/entities"
	"github.com/google/uuid"
)

type RoleRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*entities.Role, error)
	FindAll(ctx context.Context) ([]*entities.Role, error)
	FindByScope(ctx context.Context, scope string) ([]*entities.Role, error)
}

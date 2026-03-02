package repository

import (
	"context"

	"github.com/EduGoGroup/edugo-infrastructure/postgres/entities"
	sharedrepo "github.com/EduGoGroup/edugo-shared/repository"
	"github.com/google/uuid"
)

type RoleRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*entities.Role, error)
	FindAll(ctx context.Context, filters sharedrepo.ListFilters) ([]*entities.Role, error)
	FindByScope(ctx context.Context, scope string, filters sharedrepo.ListFilters) ([]*entities.Role, error)
	Create(ctx context.Context, role *entities.Role) error
	Update(ctx context.Context, role *entities.Role) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	HasActiveUserRoles(ctx context.Context, roleID uuid.UUID) (bool, error)
}

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
}

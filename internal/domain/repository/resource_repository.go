package repository

import (
	"context"

	"github.com/EduGoGroup/edugo-infrastructure/postgres/entities"
	"github.com/google/uuid"
)

type ResourceRepository interface {
	FindAll(ctx context.Context) ([]*entities.Resource, error)
	FindByID(ctx context.Context, id uuid.UUID) (*entities.Resource, error)
	FindMenuVisible(ctx context.Context) ([]*entities.Resource, error)
	Create(ctx context.Context, resource *entities.Resource) error
	Update(ctx context.Context, resource *entities.Resource) error
}

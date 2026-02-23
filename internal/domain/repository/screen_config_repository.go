package repository

import (
	"context"

	"github.com/EduGoGroup/edugo-infrastructure/postgres/entities"
	"github.com/google/uuid"
)

type ScreenTemplateFilter struct {
	Pattern string
	Offset  int
	Limit   int
}

type ScreenInstanceFilter struct {
	TemplateID *string
	Offset     int
	Limit      int
}

type ScreenTemplateRepository interface {
	Create(ctx context.Context, template *entities.ScreenTemplate) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.ScreenTemplate, error)
	List(ctx context.Context, filter ScreenTemplateFilter) ([]*entities.ScreenTemplate, int, error)
	Update(ctx context.Context, template *entities.ScreenTemplate) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type ScreenInstanceRepository interface {
	Create(ctx context.Context, instance *entities.ScreenInstance) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.ScreenInstance, error)
	GetByScreenKey(ctx context.Context, key string) (*entities.ScreenInstance, error)
	List(ctx context.Context, filter ScreenInstanceFilter) ([]*entities.ScreenInstance, int, error)
	Update(ctx context.Context, instance *entities.ScreenInstance) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type ResourceScreenRepository interface {
	Create(ctx context.Context, rs *entities.ResourceScreen) error
	GetByResourceID(ctx context.Context, resourceID uuid.UUID) ([]*entities.ResourceScreen, error)
	GetByResourceKey(ctx context.Context, key string) ([]*entities.ResourceScreen, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

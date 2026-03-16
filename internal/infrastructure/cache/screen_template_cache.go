package cache

import (
	"context"
	"sync"

	"github.com/EduGoGroup/edugo-api-iam-platform/internal/domain/repository"
	"github.com/EduGoGroup/edugo-infrastructure/postgres/entities"
	"github.com/google/uuid"
)

// CachedScreenTemplateRepository wraps a ScreenTemplateRepository with an
// in-memory cache for GetByID lookups. Templates change very rarely, so
// caching eliminates ~270ms of DB latency per resolve/key request.
type CachedScreenTemplateRepository struct {
	inner repository.ScreenTemplateRepository
	cache sync.Map // uuid.UUID -> *entities.ScreenTemplate
}

func NewCachedScreenTemplateRepository(inner repository.ScreenTemplateRepository) *CachedScreenTemplateRepository {
	return &CachedScreenTemplateRepository{inner: inner}
}

func (r *CachedScreenTemplateRepository) Create(ctx context.Context, t *entities.ScreenTemplate) error {
	err := r.inner.Create(ctx, t)
	if err == nil {
		r.cache.Store(t.ID, t)
	}
	return err
}

func (r *CachedScreenTemplateRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.ScreenTemplate, error) {
	if cached, ok := r.cache.Load(id); ok {
		return cached.(*entities.ScreenTemplate), nil
	}
	t, err := r.inner.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t != nil {
		r.cache.Store(id, t)
	}
	return t, nil
}

func (r *CachedScreenTemplateRepository) List(ctx context.Context, filter repository.ScreenTemplateFilter) ([]*entities.ScreenTemplate, int, error) {
	// List always goes to DB — pagination and filters make caching impractical
	return r.inner.List(ctx, filter)
}

func (r *CachedScreenTemplateRepository) Update(ctx context.Context, t *entities.ScreenTemplate) error {
	err := r.inner.Update(ctx, t)
	if err == nil {
		r.cache.Store(t.ID, t)
	}
	return err
}

func (r *CachedScreenTemplateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.inner.Delete(ctx, id)
	if err == nil {
		r.cache.Delete(id)
	}
	return err
}

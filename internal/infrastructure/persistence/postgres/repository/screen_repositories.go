package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/EduGoGroup/edugo-api-iam-platform/internal/domain/repository"
	"github.com/EduGoGroup/edugo-infrastructure/postgres/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ==================== ScreenTemplate ====================

type postgresScreenTemplateRepository struct{ db *gorm.DB }

func NewPostgresScreenTemplateRepository(db *gorm.DB) repository.ScreenTemplateRepository {
	return &postgresScreenTemplateRepository{db: db}
}

func (r *postgresScreenTemplateRepository) Create(ctx context.Context, t *entities.ScreenTemplate) error {
	return r.db.WithContext(ctx).Table("ui_config.screen_templates").Create(t).Error
}

func (r *postgresScreenTemplateRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.ScreenTemplate, error) {
	var t entities.ScreenTemplate
	if err := r.db.WithContext(ctx).Table("ui_config.screen_templates").First(&t, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("screen template not found")
		}
		return nil, err
	}
	return &t, nil
}

func (r *postgresScreenTemplateRepository) List(ctx context.Context, filter repository.ScreenTemplateFilter) ([]*entities.ScreenTemplate, int, error) {
	baseQuery := r.db.WithContext(ctx).Table("ui_config.screen_templates").Where("is_active = true")
	if filter.Pattern != "" {
		baseQuery = baseQuery.Where("pattern = ?", filter.Pattern)
	}

	var total int64
	baseQuery.Count(&total)

	query := r.db.WithContext(ctx).Table("ui_config.screen_templates").Where("is_active = true")
	if filter.Pattern != "" {
		query = query.Where("pattern = ?", filter.Pattern)
	}
	query = query.Order("created_at DESC")
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	var templates []*entities.ScreenTemplate
	err := query.Find(&templates).Error
	return templates, int(total), err
}

func (r *postgresScreenTemplateRepository) Update(ctx context.Context, t *entities.ScreenTemplate) error {
	return r.db.WithContext(ctx).Table("ui_config.screen_templates").Save(t).Error
}

func (r *postgresScreenTemplateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Table("ui_config.screen_templates").Where("id = ?", id).
		Updates(map[string]interface{}{"is_active": false, "updated_at": time.Now()}).Error
}

// ==================== ScreenInstance ====================

type postgresScreenInstanceRepository struct{ db *gorm.DB }

func NewPostgresScreenInstanceRepository(db *gorm.DB) repository.ScreenInstanceRepository {
	return &postgresScreenInstanceRepository{db: db}
}

func (r *postgresScreenInstanceRepository) Create(ctx context.Context, i *entities.ScreenInstance) error {
	return r.db.WithContext(ctx).Table("ui_config.screen_instances").Create(i).Error
}

func (r *postgresScreenInstanceRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.ScreenInstance, error) {
	var i entities.ScreenInstance
	if err := r.db.WithContext(ctx).Table("ui_config.screen_instances").First(&i, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("screen instance not found")
		}
		return nil, err
	}
	return &i, nil
}

func (r *postgresScreenInstanceRepository) GetByScreenKey(ctx context.Context, key string) (*entities.ScreenInstance, error) {
	var i entities.ScreenInstance
	if err := r.db.WithContext(ctx).Table("ui_config.screen_instances").Where("screen_key = ? AND is_active = true", key).First(&i).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("screen instance not found for key: %s", key)
		}
		return nil, err
	}
	return &i, nil
}

func (r *postgresScreenInstanceRepository) List(ctx context.Context, filter repository.ScreenInstanceFilter) ([]*entities.ScreenInstance, int, error) {
	baseQuery := r.db.WithContext(ctx).Table("ui_config.screen_instances").Where("is_active = true")
	if filter.TemplateID != nil {
		baseQuery = baseQuery.Where("template_id = ?", *filter.TemplateID)
	}

	var total int64
	baseQuery.Count(&total)

	query := r.db.WithContext(ctx).Table("ui_config.screen_instances").Where("is_active = true")
	if filter.TemplateID != nil {
		query = query.Where("template_id = ?", *filter.TemplateID)
	}
	query = query.Order("created_at DESC")
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	var instances []*entities.ScreenInstance
	err := query.Find(&instances).Error
	return instances, int(total), err
}

func (r *postgresScreenInstanceRepository) Update(ctx context.Context, i *entities.ScreenInstance) error {
	return r.db.WithContext(ctx).Table("ui_config.screen_instances").Save(i).Error
}

func (r *postgresScreenInstanceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Table("ui_config.screen_instances").Where("id = ?", id).
		Updates(map[string]interface{}{"is_active": false, "updated_at": time.Now()}).Error
}

// ==================== ResourceScreen ====================

type postgresResourceScreenRepository struct{ db *gorm.DB }

func NewPostgresResourceScreenRepository(db *gorm.DB) repository.ResourceScreenRepository {
	return &postgresResourceScreenRepository{db: db}
}

func (r *postgresResourceScreenRepository) Create(ctx context.Context, rs *entities.ResourceScreen) error {
	return r.db.WithContext(ctx).Table("ui_config.resource_screens").Create(rs).Error
}

func (r *postgresResourceScreenRepository) GetByResourceID(ctx context.Context, resourceID uuid.UUID) ([]*entities.ResourceScreen, error) {
	var result []*entities.ResourceScreen
	err := r.db.WithContext(ctx).Table("ui_config.resource_screens").
		Where("resource_id = ? AND is_active = true", resourceID).Order("sort_order").Find(&result).Error
	return result, err
}

func (r *postgresResourceScreenRepository) GetByResourceKey(ctx context.Context, key string) ([]*entities.ResourceScreen, error) {
	var result []*entities.ResourceScreen
	err := r.db.WithContext(ctx).Table("ui_config.resource_screens").
		Where("resource_key = ? AND is_active = true", key).Order("sort_order").Find(&result).Error
	return result, err
}

func (r *postgresResourceScreenRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Table("ui_config.resource_screens").Where("id = ?", id).Delete(&entities.ResourceScreen{}).Error
}

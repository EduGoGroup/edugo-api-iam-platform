package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/EduGoGroup/edugo-api-iam-platform/internal/audit/model"
	"gorm.io/gorm"
)

// AuditRepository handles audit event queries
type AuditRepository interface {
	List(ctx context.Context, filters model.AuditFilters, page, pageSize int) ([]model.AuditEvent, int64, error)
	GetByID(ctx context.Context, id string) (*model.AuditEvent, error)
	GetByUserID(ctx context.Context, userID string, page, pageSize int) ([]model.AuditEvent, int64, error)
	GetByResource(ctx context.Context, resourceType, resourceID string, page, pageSize int) ([]model.AuditEvent, int64, error)
	CountByField(ctx context.Context, field string, from, to time.Time) (map[string]int64, error)
	CountTotal(ctx context.Context, from, to time.Time) (int64, error)
}

type postgresAuditRepository struct {
	db *gorm.DB
}

// NewPostgresAuditRepository creates a new audit repository
func NewPostgresAuditRepository(db *gorm.DB) AuditRepository {
	return &postgresAuditRepository{db: db}
}

func (r *postgresAuditRepository) List(ctx context.Context, filters model.AuditFilters, page, pageSize int) ([]model.AuditEvent, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.AuditEvent{})

	if filters.Action != "" {
		query = query.Where("action = ?", filters.Action)
	}
	if filters.ResourceType != "" {
		query = query.Where("resource_type = ?", filters.ResourceType)
	}
	if filters.Severity != "" {
		query = query.Where("severity = ?", filters.Severity)
	}
	if filters.Category != "" {
		query = query.Where("category = ?", filters.Category)
	}
	if filters.ActorID != "" {
		query = query.Where("actor_id = ?", filters.ActorID)
	}
	if filters.ServiceName != "" {
		query = query.Where("service_name = ?", filters.ServiceName)
	}
	if filters.From != nil {
		query = query.Where("created_at >= ?", *filters.From)
	}
	if filters.To != nil {
		query = query.Where("created_at <= ?", *filters.To)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var events []model.AuditEvent
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&events).Error; err != nil {
		return nil, 0, err
	}

	return events, total, nil
}

func (r *postgresAuditRepository) GetByID(ctx context.Context, id string) (*model.AuditEvent, error) {
	var event model.AuditEvent
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&event).Error; err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *postgresAuditRepository) GetByUserID(ctx context.Context, userID string, page, pageSize int) ([]model.AuditEvent, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&model.AuditEvent{}).Where("actor_id = ?", userID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var events []model.AuditEvent
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&events).Error; err != nil {
		return nil, 0, err
	}

	return events, total, nil
}

func (r *postgresAuditRepository) GetByResource(ctx context.Context, resourceType, resourceID string, page, pageSize int) ([]model.AuditEvent, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&model.AuditEvent{}).
		Where("resource_type = ? AND resource_id = ?", resourceType, resourceID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var events []model.AuditEvent
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&events).Error; err != nil {
		return nil, 0, err
	}

	return events, total, nil
}

// allowedCountFields is the whitelist of columns allowed for CountByField aggregation.
var allowedCountFields = map[string]bool{
	"action":        true,
	"severity":      true,
	"category":      true,
	"resource_type": true,
	"service_name":  true,
}

func (r *postgresAuditRepository) CountByField(ctx context.Context, field string, from, to time.Time) (map[string]int64, error) {
	if !allowedCountFields[field] {
		return nil, fmt.Errorf("invalid aggregation field: %s", field)
	}

	type result struct {
		Value string `gorm:"column:value"`
		Count int64  `gorm:"column:count"`
	}

	var results []result
	err := r.db.WithContext(ctx).
		Model(&model.AuditEvent{}).
		Select(field+" as value, count(*) as count").
		Where("created_at >= ? AND created_at <= ?", from, to).
		Group(field).
		Find(&results).Error
	if err != nil {
		return nil, err
	}

	m := make(map[string]int64)
	for _, r := range results {
		m[r.Value] = r.Count
	}
	return m, nil
}

func (r *postgresAuditRepository) CountTotal(ctx context.Context, from, to time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.AuditEvent{}).
		Where("created_at >= ? AND created_at <= ?", from, to).
		Count(&count).Error
	return count, err
}

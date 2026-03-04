package repository

import (
	"context"
	"time"

	"github.com/EduGoGroup/edugo-api-iam-platform/internal/auth/model"
	"gorm.io/gorm"
)

// LoginAttemptRepository handles login attempt persistence
type LoginAttemptRepository interface {
	Create(ctx context.Context, attempt *model.LoginAttempt) error
	CountFailedSince(ctx context.Context, identifier string, since time.Time) (int64, error)
}

type postgresLoginAttemptRepository struct {
	db *gorm.DB
}

// NewPostgresLoginAttemptRepository creates a new login attempt repository
func NewPostgresLoginAttemptRepository(db *gorm.DB) LoginAttemptRepository {
	return &postgresLoginAttemptRepository{db: db}
}

func (r *postgresLoginAttemptRepository) Create(ctx context.Context, attempt *model.LoginAttempt) error {
	return r.db.WithContext(ctx).Create(attempt).Error
}

func (r *postgresLoginAttemptRepository) CountFailedSince(ctx context.Context, identifier string, since time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.LoginAttempt{}).
		Where("identifier = ? AND successful = false AND attempted_at >= ?", identifier, since).
		Count(&count).Error
	return count, err
}

package repository

import (
	"context"

	"github.com/EduGoGroup/edugo-api-iam-platform/internal/domain/repository"
	"github.com/EduGoGroup/edugo-infrastructure/postgres/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type postgresSchoolConceptRepository struct{ db *gorm.DB }

func NewPostgresSchoolConceptRepository(db *gorm.DB) repository.SchoolConceptRepository {
	return &postgresSchoolConceptRepository{db: db}
}

func (r *postgresSchoolConceptRepository) FindBySchoolID(ctx context.Context, schoolID uuid.UUID) ([]*entities.SchoolConcept, error) {
	var concepts []*entities.SchoolConcept
	err := r.db.WithContext(ctx).Table("academic.school_concepts").
		Where("school_id = ?", schoolID).
		Order("category ASC, term_key ASC").
		Find(&concepts).Error
	return concepts, err
}

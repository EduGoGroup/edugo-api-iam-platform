package repository

import (
	"context"

	"github.com/EduGoGroup/edugo-infrastructure/postgres/entities"
	"github.com/google/uuid"
)

type SchoolConceptRepository interface {
	FindBySchoolID(ctx context.Context, schoolID uuid.UUID) ([]*entities.SchoolConcept, error)
}

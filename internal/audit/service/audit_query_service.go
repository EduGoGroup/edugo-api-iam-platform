package service

import (
	"context"
	"time"

	"github.com/EduGoGroup/edugo-api-iam-platform/internal/audit/model"
	"github.com/EduGoGroup/edugo-api-iam-platform/internal/audit/repository"
)

// AuditQueryService provides read access to audit events
type AuditQueryService interface {
	List(ctx context.Context, filters model.AuditFilters, page, pageSize int) ([]model.AuditEvent, int64, error)
	GetByID(ctx context.Context, id string) (*model.AuditEvent, error)
	GetByUserID(ctx context.Context, userID string, page, pageSize int) ([]model.AuditEvent, int64, error)
	GetByResource(ctx context.Context, resourceType, resourceID string, page, pageSize int) ([]model.AuditEvent, int64, error)
	Summary(ctx context.Context, from, to time.Time) (*model.AuditSummary, error)
}

type auditQueryService struct {
	repo repository.AuditRepository
}

// NewAuditQueryService creates a new audit query service
func NewAuditQueryService(repo repository.AuditRepository) AuditQueryService {
	return &auditQueryService{repo: repo}
}

func (s *auditQueryService) List(ctx context.Context, filters model.AuditFilters, page, pageSize int) ([]model.AuditEvent, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}
	return s.repo.List(ctx, filters, page, pageSize)
}

func (s *auditQueryService) GetByID(ctx context.Context, id string) (*model.AuditEvent, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *auditQueryService) GetByUserID(ctx context.Context, userID string, page, pageSize int) ([]model.AuditEvent, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}
	return s.repo.GetByUserID(ctx, userID, page, pageSize)
}

func (s *auditQueryService) GetByResource(ctx context.Context, resourceType, resourceID string, page, pageSize int) ([]model.AuditEvent, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}
	return s.repo.GetByResource(ctx, resourceType, resourceID, page, pageSize)
}

func (s *auditQueryService) Summary(ctx context.Context, from, to time.Time) (*model.AuditSummary, error) {
	total, err := s.repo.CountTotal(ctx, from, to)
	if err != nil {
		return nil, err
	}

	byAction, err := s.repo.CountByField(ctx, "action", from, to)
	if err != nil {
		return nil, err
	}

	bySeverity, err := s.repo.CountByField(ctx, "severity", from, to)
	if err != nil {
		return nil, err
	}

	byCategory, err := s.repo.CountByField(ctx, "category", from, to)
	if err != nil {
		return nil, err
	}

	byResourceType, err := s.repo.CountByField(ctx, "resource_type", from, to)
	if err != nil {
		return nil, err
	}

	return &model.AuditSummary{
		TotalEvents:    total,
		ByAction:       byAction,
		BySeverity:     bySeverity,
		ByCategory:     byCategory,
		ByResourceType: byResourceType,
	}, nil
}

package service

import (
	"context"
	"time"

	"github.com/EduGoGroup/edugo-api-iam-platform/internal/application/dto"
	"github.com/EduGoGroup/edugo-api-iam-platform/internal/domain/repository"
	"github.com/EduGoGroup/edugo-infrastructure/postgres/entities"
	"github.com/EduGoGroup/edugo-shared/common/errors"
	"github.com/EduGoGroup/edugo-shared/logger"
	"github.com/google/uuid"
)

// ResourceService defines the resource service interface
type ResourceService interface {
	ListResources(ctx context.Context) (*dto.ResourcesResponse, error)
	GetResource(ctx context.Context, id string) (*dto.ResourceDTO, error)
	CreateResource(ctx context.Context, req dto.CreateResourceRequest) (*dto.ResourceDTO, error)
	UpdateResource(ctx context.Context, id string, req dto.UpdateResourceRequest) (*dto.ResourceDTO, error)
}

type resourceService struct {
	resourceRepo repository.ResourceRepository
	logger       logger.Logger
}

// NewResourceService creates a new resource service
func NewResourceService(resourceRepo repository.ResourceRepository, logger logger.Logger) ResourceService {
	return &resourceService{resourceRepo: resourceRepo, logger: logger}
}

func (s *resourceService) ListResources(ctx context.Context) (*dto.ResourcesResponse, error) {
	resources, err := s.resourceRepo.FindAll(ctx)
	if err != nil {
		return nil, errors.NewDatabaseError("list resources", err)
	}
	return &dto.ResourcesResponse{
		Resources: dto.ToResourceDTOList(resources),
		Total:     len(resources),
	}, nil
}

func (s *resourceService) GetResource(ctx context.Context, id string) (*dto.ResourceDTO, error) {
	rid, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.NewValidationError("invalid resource ID")
	}
	resource, err := s.resourceRepo.FindByID(ctx, rid)
	if err != nil {
		return nil, errors.NewDatabaseError("find resource", err)
	}
	if resource == nil {
		return nil, errors.NewNotFoundError("resource")
	}
	d := dto.ToResourceDTO(resource)
	return &d, nil
}

func (s *resourceService) CreateResource(ctx context.Context, req dto.CreateResourceRequest) (*dto.ResourceDTO, error) {
	now := time.Now()
	resource := &entities.Resource{
		ID:            uuid.New(),
		Key:           req.Key,
		DisplayName:   req.DisplayName,
		SortOrder:     req.SortOrder,
		IsMenuVisible: req.IsMenuVisible,
		Scope:         req.Scope,
		IsActive:      true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if req.Description != "" {
		resource.Description = &req.Description
	}
	if req.Icon != "" {
		resource.Icon = &req.Icon
	}
	if req.ParentID != nil && *req.ParentID != "" {
		pid, err := uuid.Parse(*req.ParentID)
		if err != nil {
			return nil, errors.NewValidationError("invalid parent_id")
		}
		resource.ParentID = &pid
	}

	if err := s.resourceRepo.Create(ctx, resource); err != nil {
		return nil, errors.NewDatabaseError("create resource", err)
	}

	s.logger.Info("entity created", "entity_type", "resource", "entity_id", resource.ID.String(), "key", resource.Key)
	d := dto.ToResourceDTO(resource)
	return &d, nil
}

func (s *resourceService) UpdateResource(ctx context.Context, id string, req dto.UpdateResourceRequest) (*dto.ResourceDTO, error) {
	rid, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.NewValidationError("invalid resource ID")
	}
	resource, err := s.resourceRepo.FindByID(ctx, rid)
	if err != nil {
		return nil, errors.NewDatabaseError("find resource", err)
	}
	if resource == nil {
		return nil, errors.NewNotFoundError("resource")
	}

	if req.DisplayName != nil {
		resource.DisplayName = *req.DisplayName
	}
	if req.Description != nil {
		resource.Description = req.Description
	}
	if req.Icon != nil {
		resource.Icon = req.Icon
	}
	if req.ParentID != nil {
		if *req.ParentID == "" {
			resource.ParentID = nil
		} else {
			pid, err := uuid.Parse(*req.ParentID)
			if err != nil {
				return nil, errors.NewValidationError("invalid parent_id")
			}
			resource.ParentID = &pid
		}
	}
	if req.SortOrder != nil {
		resource.SortOrder = *req.SortOrder
	}
	if req.IsMenuVisible != nil {
		resource.IsMenuVisible = *req.IsMenuVisible
	}
	if req.Scope != nil {
		resource.Scope = *req.Scope
	}
	if req.IsActive != nil {
		resource.IsActive = *req.IsActive
	}

	resource.UpdatedAt = time.Now()
	if err := s.resourceRepo.Update(ctx, resource); err != nil {
		return nil, errors.NewDatabaseError("update resource", err)
	}

	s.logger.Info("entity updated", "entity_type", "resource", "entity_id", id)
	d := dto.ToResourceDTO(resource)
	return &d, nil
}

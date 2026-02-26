package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/EduGoGroup/edugo-api-iam-platform/internal/domain/repository"
	"github.com/EduGoGroup/edugo-infrastructure/postgres/entities"
	"github.com/EduGoGroup/edugo-shared/common/errors"
	"github.com/EduGoGroup/edugo-shared/logger"
	"github.com/google/uuid"
)

// ScreenConfigService defines the screen configuration service interface
type ScreenConfigService interface {
	CreateTemplate(ctx context.Context, req *CreateTemplateRequest) (*ScreenTemplateDTO, error)
	GetTemplate(ctx context.Context, id string) (*ScreenTemplateDTO, error)
	ListTemplates(ctx context.Context, filter TemplateFilter) ([]*ScreenTemplateDTO, int, error)
	UpdateTemplate(ctx context.Context, id string, req *UpdateTemplateRequest) (*ScreenTemplateDTO, error)
	DeleteTemplate(ctx context.Context, id string) error
	CreateInstance(ctx context.Context, req *CreateInstanceRequest) (*ScreenInstanceDTO, error)
	GetInstance(ctx context.Context, id string) (*ScreenInstanceDTO, error)
	GetInstanceByKey(ctx context.Context, key string) (*ScreenInstanceDTO, error)
	ListInstances(ctx context.Context, filter InstanceFilter) ([]*ScreenInstanceDTO, int, error)
	UpdateInstance(ctx context.Context, id string, req *UpdateInstanceRequest) (*ScreenInstanceDTO, error)
	DeleteInstance(ctx context.Context, id string) error
	ResolveScreenByKey(ctx context.Context, key string) (*CombinedScreenDTO, error)
	GetScreenVersion(ctx context.Context, key string) (*ScreenVersionDTO, error)
	LinkScreenToResource(ctx context.Context, req *LinkScreenRequest) (*ResourceScreenDTO, error)
	GetScreensForResource(ctx context.Context, resourceID string) ([]*ResourceScreenDTO, error)
	UnlinkScreen(ctx context.Context, id string) error
}

// Request/Response types for screen config

type CreateTemplateRequest struct {
	Pattern     string          `json:"pattern" binding:"required"`
	Name        string          `json:"name" binding:"required"`
	Description string          `json:"description"`
	Definition  json.RawMessage `json:"definition" binding:"required"`
}

type UpdateTemplateRequest struct {
	Pattern     *string          `json:"pattern"`
	Name        *string          `json:"name"`
	Description *string          `json:"description"`
	Definition  *json.RawMessage `json:"definition"`
}

type TemplateFilter struct {
	Pattern string `form:"pattern"`
	Page    int    `form:"page"`
	PerPage int    `form:"per_page"`
}

type CreateInstanceRequest struct {
	ScreenKey          string          `json:"screen_key" binding:"required"`
	TemplateID         string          `json:"template_id" binding:"required"`
	Name               string          `json:"name" binding:"required"`
	Description        string          `json:"description"`
	SlotData           json.RawMessage `json:"slot_data"`
	Scope              string          `json:"scope"`
	RequiredPermission string          `json:"required_permission"`
	HandlerKey         *string         `json:"handler_key,omitempty"`
}

type UpdateInstanceRequest struct {
	ScreenKey          *string          `json:"screen_key"`
	TemplateID         *string          `json:"template_id"`
	Name               *string          `json:"name"`
	Description        *string          `json:"description"`
	SlotData           *json.RawMessage `json:"slot_data"`
	Scope              *string          `json:"scope"`
	RequiredPermission *string          `json:"required_permission"`
	HandlerKey         *string          `json:"handler_key,omitempty"`
}

type InstanceFilter struct {
	TemplateID string `form:"template_id"`
	Page       int    `form:"page"`
	PerPage    int    `form:"per_page"`
}

type LinkScreenRequest struct {
	ResourceID  string `json:"resource_id" binding:"required"`
	ResourceKey string `json:"resource_key" binding:"required"`
	ScreenKey   string `json:"screen_key" binding:"required"`
	ScreenType  string `json:"screen_type" binding:"required"`
	IsDefault   bool   `json:"is_default"`
}

// DTO types

type ScreenTemplateDTO struct {
	ID          string          `json:"id"`
	Pattern     string          `json:"pattern"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Version     int             `json:"version"`
	Definition  json.RawMessage `json:"definition"`
	IsActive    bool            `json:"is_active"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type ScreenInstanceDTO struct {
	ID                 string          `json:"id"`
	ScreenKey          string          `json:"screen_key"`
	TemplateID         string          `json:"template_id"`
	Name               string          `json:"name"`
	Description        string          `json:"description,omitempty"`
	SlotData           json.RawMessage `json:"slot_data"`
	Scope              string          `json:"scope"`
	RequiredPermission string          `json:"required_permission,omitempty"`
	HandlerKey         *string         `json:"handler_key,omitempty"`
	IsActive           bool            `json:"is_active"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

type CombinedScreenDTO struct {
	ScreenID   string          `json:"screen_id"`
	ScreenKey  string          `json:"screen_key"`
	ScreenName string          `json:"screen_name"`
	Pattern    string          `json:"pattern"`
	Version    int             `json:"version"`
	Template   json.RawMessage `json:"template"`
	SlotData   json.RawMessage `json:"slot_data"`
	HandlerKey *string         `json:"handler_key,omitempty"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

type ResourceScreenDTO struct {
	ResourceID  string `json:"resource_id"`
	ResourceKey string `json:"resource_key"`
	ScreenKey   string `json:"screen_key"`
	ScreenType  string `json:"screen_type"`
	IsDefault   bool   `json:"is_default"`
}

type ScreenVersionDTO struct {
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Implementation

type screenConfigService struct {
	templateRepo       repository.ScreenTemplateRepository
	instanceRepo       repository.ScreenInstanceRepository
	resourceScreenRepo repository.ResourceScreenRepository
	logger             logger.Logger
}

func NewScreenConfigService(
	templateRepo repository.ScreenTemplateRepository,
	instanceRepo repository.ScreenInstanceRepository,
	resourceScreenRepo repository.ResourceScreenRepository,
	logger logger.Logger,
) ScreenConfigService {
	return &screenConfigService{templateRepo: templateRepo, instanceRepo: instanceRepo, resourceScreenRepo: resourceScreenRepo, logger: logger}
}

func (s *screenConfigService) CreateTemplate(ctx context.Context, req *CreateTemplateRequest) (*ScreenTemplateDTO, error) {
	now := time.Now()
	template := &entities.ScreenTemplate{
		ID: uuid.New(), Pattern: req.Pattern, Name: req.Name, Version: 1,
		Definition: req.Definition, IsActive: true, CreatedAt: now, UpdatedAt: now,
	}
	if req.Description != "" {
		template.Description = &req.Description
	}
	if err := s.templateRepo.Create(ctx, template); err != nil {
		return nil, errors.NewDatabaseError("create screen template", err)
	}
	s.logger.Info("entity created", "entity_type", "screen_template", "entity_id", template.ID.String())
	return toTemplateDTO(template), nil
}

func (s *screenConfigService) GetTemplate(ctx context.Context, id string) (*ScreenTemplateDTO, error) {
	tid, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.NewValidationError("invalid template ID")
	}
	template, err := s.templateRepo.GetByID(ctx, tid)
	if err != nil {
		return nil, err
	}
	return toTemplateDTO(template), nil
}

func (s *screenConfigService) ListTemplates(ctx context.Context, filter TemplateFilter) ([]*ScreenTemplateDTO, int, error) {
	if filter.PerPage <= 0 {
		filter.PerPage = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.PerPage
	repoFilter := repository.ScreenTemplateFilter{Pattern: filter.Pattern, Offset: offset, Limit: filter.PerPage}
	templates, total, err := s.templateRepo.List(ctx, repoFilter)
	if err != nil {
		return nil, 0, errors.NewDatabaseError("list screen templates", err)
	}
	dtos := make([]*ScreenTemplateDTO, len(templates))
	for i, t := range templates {
		dtos[i] = toTemplateDTO(t)
	}
	return dtos, total, nil
}

func (s *screenConfigService) UpdateTemplate(ctx context.Context, id string, req *UpdateTemplateRequest) (*ScreenTemplateDTO, error) {
	tid, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.NewValidationError("invalid template ID")
	}
	template, err := s.templateRepo.GetByID(ctx, tid)
	if err != nil {
		return nil, err
	}
	if req.Pattern != nil {
		template.Pattern = *req.Pattern
	}
	if req.Name != nil {
		template.Name = *req.Name
	}
	if req.Description != nil {
		template.Description = req.Description
	}
	if req.Definition != nil {
		template.Definition = *req.Definition
		template.Version++
	}
	template.UpdatedAt = time.Now()
	if err := s.templateRepo.Update(ctx, template); err != nil {
		return nil, errors.NewDatabaseError("update screen template", err)
	}
	s.logger.Info("entity updated", "entity_type", "screen_template", "entity_id", id)
	return toTemplateDTO(template), nil
}

func (s *screenConfigService) DeleteTemplate(ctx context.Context, id string) error {
	tid, err := uuid.Parse(id)
	if err != nil {
		return errors.NewValidationError("invalid template ID")
	}
	if _, err := s.templateRepo.GetByID(ctx, tid); err != nil {
		return err
	}
	if err := s.templateRepo.Delete(ctx, tid); err != nil {
		return errors.NewDatabaseError("delete screen template", err)
	}
	s.logger.Info("entity deleted", "entity_type", "screen_template", "entity_id", id)
	return nil
}

func (s *screenConfigService) CreateInstance(ctx context.Context, req *CreateInstanceRequest) (*ScreenInstanceDTO, error) {
	templateID, err := uuid.Parse(req.TemplateID)
	if err != nil {
		return nil, errors.NewValidationError("invalid template_id")
	}
	if _, err := s.templateRepo.GetByID(ctx, templateID); err != nil {
		return nil, errors.NewValidationError("template not found")
	}

	now := time.Now()
	instance := &entities.ScreenInstance{
		ID: uuid.New(), ScreenKey: req.ScreenKey, TemplateID: templateID, Name: req.Name,
		SlotData: req.SlotData,
		Scope:    "system", IsActive: true, CreatedAt: now, UpdatedAt: now,
	}
	if req.Description != "" {
		instance.Description = &req.Description
	}
	if req.Scope != "" {
		instance.Scope = req.Scope
	}
	if req.RequiredPermission != "" {
		instance.RequiredPermission = &req.RequiredPermission
	}
	if req.HandlerKey != nil {
		instance.HandlerKey = req.HandlerKey
	}
	if instance.SlotData == nil {
		instance.SlotData = json.RawMessage(`{}`)
	}

	if err := s.instanceRepo.Create(ctx, instance); err != nil {
		return nil, errors.NewDatabaseError("create screen instance", err)
	}
	s.logger.Info("entity created", "entity_type", "screen_instance", "entity_id", instance.ID.String())
	return toInstanceDTO(instance), nil
}

func (s *screenConfigService) GetInstance(ctx context.Context, id string) (*ScreenInstanceDTO, error) {
	iid, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.NewValidationError("invalid instance ID")
	}
	instance, err := s.instanceRepo.GetByID(ctx, iid)
	if err != nil {
		return nil, err
	}
	return toInstanceDTO(instance), nil
}

func (s *screenConfigService) GetInstanceByKey(ctx context.Context, key string) (*ScreenInstanceDTO, error) {
	instance, err := s.instanceRepo.GetByScreenKey(ctx, key)
	if err != nil {
		return nil, err
	}
	return toInstanceDTO(instance), nil
}

func (s *screenConfigService) ListInstances(ctx context.Context, filter InstanceFilter) ([]*ScreenInstanceDTO, int, error) {
	if filter.PerPage <= 0 {
		filter.PerPage = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.PerPage
	repoFilter := repository.ScreenInstanceFilter{Offset: offset, Limit: filter.PerPage}
	if filter.TemplateID != "" {
		repoFilter.TemplateID = &filter.TemplateID
	}
	instances, total, err := s.instanceRepo.List(ctx, repoFilter)
	if err != nil {
		return nil, 0, errors.NewDatabaseError("list screen instances", err)
	}
	dtos := make([]*ScreenInstanceDTO, len(instances))
	for i, inst := range instances {
		dtos[i] = toInstanceDTO(inst)
	}
	return dtos, total, nil
}

func (s *screenConfigService) UpdateInstance(ctx context.Context, id string, req *UpdateInstanceRequest) (*ScreenInstanceDTO, error) {
	iid, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.NewValidationError("invalid instance ID")
	}
	instance, err := s.instanceRepo.GetByID(ctx, iid)
	if err != nil {
		return nil, err
	}
	if req.ScreenKey != nil {
		instance.ScreenKey = *req.ScreenKey
	}
	if req.TemplateID != nil {
		tid, err := uuid.Parse(*req.TemplateID)
		if err != nil {
			return nil, errors.NewValidationError("invalid template_id")
		}
		instance.TemplateID = tid
	}
	if req.Name != nil {
		instance.Name = *req.Name
	}
	if req.Description != nil {
		instance.Description = req.Description
	}
	if req.SlotData != nil {
		instance.SlotData = *req.SlotData
	}
	if req.Scope != nil {
		instance.Scope = *req.Scope
	}
	if req.RequiredPermission != nil {
		instance.RequiredPermission = req.RequiredPermission
	}
	if req.HandlerKey != nil {
		instance.HandlerKey = req.HandlerKey
	}
	instance.UpdatedAt = time.Now()
	if err := s.instanceRepo.Update(ctx, instance); err != nil {
		return nil, errors.NewDatabaseError("update screen instance", err)
	}
	s.logger.Info("entity updated", "entity_type", "screen_instance", "entity_id", id)
	return toInstanceDTO(instance), nil
}

func (s *screenConfigService) DeleteInstance(ctx context.Context, id string) error {
	iid, err := uuid.Parse(id)
	if err != nil {
		return errors.NewValidationError("invalid instance ID")
	}
	if _, err := s.instanceRepo.GetByID(ctx, iid); err != nil {
		return err
	}
	if err := s.instanceRepo.Delete(ctx, iid); err != nil {
		return errors.NewDatabaseError("delete screen instance", err)
	}
	s.logger.Info("entity deleted", "entity_type", "screen_instance", "entity_id", id)
	return nil
}

func (s *screenConfigService) ResolveScreenByKey(ctx context.Context, key string) (*CombinedScreenDTO, error) {
	instance, err := s.instanceRepo.GetByScreenKey(ctx, key)
	if err != nil {
		return nil, err
	}
	template, err := s.templateRepo.GetByID(ctx, instance.TemplateID)
	if err != nil {
		return nil, errors.NewDatabaseError("get template for screen instance", err)
	}
	combined := &CombinedScreenDTO{
		ScreenID: instance.ID.String(), ScreenKey: instance.ScreenKey, ScreenName: instance.Name,
		Pattern: template.Pattern, Version: template.Version, Template: template.Definition,
		SlotData: instance.SlotData, UpdatedAt: instance.UpdatedAt,
	}
	if instance.HandlerKey != nil {
		combined.HandlerKey = instance.HandlerKey
	}
	return combined, nil
}

func (s *screenConfigService) GetScreenVersion(ctx context.Context, key string) (*ScreenVersionDTO, error) {
	instance, err := s.instanceRepo.GetByScreenKey(ctx, key)
	if err != nil {
		return nil, err
	}
	template, err := s.templateRepo.GetByID(ctx, instance.TemplateID)
	if err != nil {
		return nil, errors.NewDatabaseError("get template for screen version", err)
	}
	return &ScreenVersionDTO{
		Version:   template.Version,
		UpdatedAt: instance.UpdatedAt,
	}, nil
}

func (s *screenConfigService) LinkScreenToResource(ctx context.Context, req *LinkScreenRequest) (*ResourceScreenDTO, error) {
	resourceID, err := uuid.Parse(req.ResourceID)
	if err != nil {
		return nil, errors.NewValidationError("invalid resource_id")
	}
	now := time.Now()
	rs := &entities.ResourceScreen{
		ID: uuid.New(), ResourceID: resourceID, ResourceKey: req.ResourceKey,
		ScreenKey: req.ScreenKey, ScreenType: req.ScreenType, IsDefault: req.IsDefault,
		IsActive: true, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.resourceScreenRepo.Create(ctx, rs); err != nil {
		return nil, errors.NewDatabaseError("link screen to resource", err)
	}
	s.logger.Info("screen linked", "resource_key", req.ResourceKey, "screen_key", req.ScreenKey)
	return toResourceScreenDTO(rs), nil
}

func (s *screenConfigService) GetScreensForResource(ctx context.Context, resourceID string) ([]*ResourceScreenDTO, error) {
	resID, err := uuid.Parse(resourceID)
	if err != nil {
		return nil, errors.NewValidationError("invalid resource_id")
	}
	items, err := s.resourceScreenRepo.GetByResourceID(ctx, resID)
	if err != nil {
		return nil, errors.NewDatabaseError("get screens for resource", err)
	}
	dtos := make([]*ResourceScreenDTO, len(items))
	for i, rs := range items {
		dtos[i] = toResourceScreenDTO(rs)
	}
	return dtos, nil
}

func (s *screenConfigService) UnlinkScreen(ctx context.Context, id string) error {
	rsID, err := uuid.Parse(id)
	if err != nil {
		return errors.NewValidationError("invalid resource_screen ID")
	}
	if err := s.resourceScreenRepo.Delete(ctx, rsID); err != nil {
		return err
	}
	s.logger.Info("screen unlinked", "resource_screen_id", id)
	return nil
}

// Conversion helpers

func toTemplateDTO(t *entities.ScreenTemplate) *ScreenTemplateDTO {
	d := &ScreenTemplateDTO{
		ID: t.ID.String(), Pattern: t.Pattern, Name: t.Name, Version: t.Version,
		Definition: t.Definition, IsActive: t.IsActive, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
	if t.Description != nil {
		d.Description = *t.Description
	}
	return d
}

func toInstanceDTO(inst *entities.ScreenInstance) *ScreenInstanceDTO {
	d := &ScreenInstanceDTO{
		ID: inst.ID.String(), ScreenKey: inst.ScreenKey, TemplateID: inst.TemplateID.String(),
		Name: inst.Name, SlotData: inst.SlotData,
		Scope: inst.Scope, IsActive: inst.IsActive, CreatedAt: inst.CreatedAt, UpdatedAt: inst.UpdatedAt,
	}
	if inst.Description != nil {
		d.Description = *inst.Description
	}
	if inst.RequiredPermission != nil {
		d.RequiredPermission = *inst.RequiredPermission
	}
	if inst.HandlerKey != nil {
		d.HandlerKey = inst.HandlerKey
	}
	return d
}

func toResourceScreenDTO(rs *entities.ResourceScreen) *ResourceScreenDTO {
	return &ResourceScreenDTO{
		ResourceID: rs.ResourceID.String(), ResourceKey: rs.ResourceKey,
		ScreenKey: rs.ScreenKey, ScreenType: rs.ScreenType, IsDefault: rs.IsDefault,
	}
}

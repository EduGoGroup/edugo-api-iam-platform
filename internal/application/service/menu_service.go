package service

import (
	"context"
	"strings"

	"github.com/EduGoGroup/edugo-api-iam-platform/internal/application/dto"
	"github.com/EduGoGroup/edugo-api-iam-platform/internal/domain/repository"
	"github.com/EduGoGroup/edugo-infrastructure/postgres/entities"
	"github.com/EduGoGroup/edugo-shared/common/errors"
	"github.com/EduGoGroup/edugo-shared/logger"
	"github.com/google/uuid"
)

// MenuService defines the menu service interface
type MenuService interface {
	GetMenuForUser(ctx context.Context, permissions []string) (*dto.MenuResponse, error)
	GetFullMenu(ctx context.Context) (*dto.MenuResponse, error)
}

type menuService struct {
	resourceRepo       repository.ResourceRepository
	resourceScreenRepo repository.ResourceScreenRepository
	logger             logger.Logger
}

// NewMenuService creates a new menu service
func NewMenuService(resourceRepo repository.ResourceRepository, resourceScreenRepo repository.ResourceScreenRepository, logger logger.Logger) MenuService {
	return &menuService{resourceRepo: resourceRepo, resourceScreenRepo: resourceScreenRepo, logger: logger}
}

func (s *menuService) GetMenuForUser(ctx context.Context, permissions []string) (*dto.MenuResponse, error) {
	resourceKeys := extractResourceKeys(permissions)
	if len(resourceKeys) == 0 {
		return &dto.MenuResponse{Items: []dto.MenuItemDTO{}}, nil
	}

	allResources, err := s.resourceRepo.FindMenuVisible(ctx)
	if err != nil {
		return nil, errors.NewDatabaseError("find menu resources", err)
	}

	resourceByKey := make(map[string]*entities.Resource)
	resourceByID := make(map[uuid.UUID]*entities.Resource)
	for _, r := range allResources {
		resourceByKey[r.Key] = r
		resourceByID[r.ID] = r
	}

	userResourceKeys := make(map[string]bool)
	for _, key := range resourceKeys {
		userResourceKeys[key] = true
	}

	visibleKeys := make(map[string]bool)
	for key := range userResourceKeys {
		visibleKeys[key] = true
		res := resourceByKey[key]
		for res != nil && res.ParentID != nil {
			parent := resourceByID[*res.ParentID]
			if parent != nil {
				visibleKeys[parent.Key] = true
				res = parent
			} else {
				break
			}
		}
	}

	permsByResource := make(map[string][]string)
	for _, perm := range permissions {
		parts := strings.SplitN(perm, ":", 2)
		if len(parts) >= 2 {
			permsByResource[parts[0]] = append(permsByResource[parts[0]], perm)
		}
	}

	screensByResource := s.loadScreenMappings(ctx, allResources)
	items := buildMenuTree(allResources, visibleKeys, permsByResource, screensByResource, nil)

	return &dto.MenuResponse{Items: items}, nil
}

func (s *menuService) GetFullMenu(ctx context.Context) (*dto.MenuResponse, error) {
	allResources, err := s.resourceRepo.FindMenuVisible(ctx)
	if err != nil {
		return nil, errors.NewDatabaseError("find menu resources", err)
	}

	allKeys := make(map[string]bool)
	for _, r := range allResources {
		allKeys[r.Key] = true
	}

	screensByResource := s.loadScreenMappings(ctx, allResources)
	items := buildMenuTree(allResources, allKeys, nil, screensByResource, nil)

	return &dto.MenuResponse{Items: items}, nil
}

func (s *menuService) loadScreenMappings(ctx context.Context, resources []*entities.Resource) map[string]map[string]string {
	result := make(map[string]map[string]string)
	if s.resourceScreenRepo == nil {
		return result
	}
	for _, r := range resources {
		screens, err := s.resourceScreenRepo.GetByResourceKey(ctx, r.Key)
		if err != nil || len(screens) == 0 {
			continue
		}
		screenMap := make(map[string]string)
		for _, sc := range screens {
			screenMap[sc.ScreenType] = sc.ScreenKey
		}
		result[r.Key] = screenMap
	}
	return result
}

func extractResourceKeys(permissions []string) []string {
	seen := make(map[string]bool)
	var keys []string
	for _, perm := range permissions {
		parts := strings.SplitN(perm, ":", 2)
		if len(parts) >= 2 && !seen[parts[0]] {
			seen[parts[0]] = true
			keys = append(keys, parts[0])
		}
	}
	return keys
}

func buildMenuTree(resources []*entities.Resource, visibleKeys map[string]bool, permsByResource map[string][]string, screensByResource map[string]map[string]string, parentID *uuid.UUID) []dto.MenuItemDTO {
	var items []dto.MenuItemDTO
	for _, r := range resources {
		if !visibleKeys[r.Key] {
			continue
		}

		if parentID == nil && r.ParentID != nil {
			continue
		}
		if parentID != nil && (r.ParentID == nil || *r.ParentID != *parentID) {
			continue
		}

		item := dto.MenuItemDTO{
			Key:         r.Key,
			DisplayName: r.DisplayName,
			Scope:       r.Scope,
			SortOrder:   r.SortOrder,
		}
		if r.Icon != nil {
			item.Icon = *r.Icon
		}
		if permsByResource != nil {
			item.Permissions = permsByResource[r.Key]
		}
		if screens, ok := screensByResource[r.Key]; ok && len(screens) > 0 {
			item.Screens = screens
		}

		item.Children = buildMenuTree(resources, visibleKeys, permsByResource, screensByResource, &r.ID)
		if item.Children == nil {
			item.Children = []dto.MenuItemDTO{}
		}

		items = append(items, item)
	}

	if items == nil {
		items = []dto.MenuItemDTO{}
	}
	return items
}

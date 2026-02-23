package dto

import (
	"github.com/EduGoGroup/edugo-infrastructure/postgres/entities"
)

// ResourceDTO represents a resource in API responses
type ResourceDTO struct {
	ID            string  `json:"id"`
	Key           string  `json:"key"`
	DisplayName   string  `json:"display_name"`
	Description   string  `json:"description,omitempty"`
	Icon          string  `json:"icon,omitempty"`
	ParentID      *string `json:"parent_id,omitempty"`
	SortOrder     int     `json:"sort_order"`
	IsMenuVisible bool    `json:"is_menu_visible"`
	Scope         string  `json:"scope"`
	IsActive      bool    `json:"is_active"`
}

// ResourcesResponse wraps a list of resources
type ResourcesResponse struct {
	Resources []ResourceDTO `json:"resources"`
	Total     int           `json:"total"`
}

// CreateResourceRequest represents the request to create a resource
type CreateResourceRequest struct {
	Key           string  `json:"key" binding:"required"`
	DisplayName   string  `json:"display_name" binding:"required"`
	Description   string  `json:"description"`
	Icon          string  `json:"icon"`
	ParentID      *string `json:"parent_id"`
	SortOrder     int     `json:"sort_order"`
	IsMenuVisible bool    `json:"is_menu_visible"`
	Scope         string  `json:"scope" binding:"required"`
}

// UpdateResourceRequest represents the request to update a resource
type UpdateResourceRequest struct {
	DisplayName   *string `json:"display_name"`
	Description   *string `json:"description"`
	Icon          *string `json:"icon"`
	ParentID      *string `json:"parent_id"`
	SortOrder     *int    `json:"sort_order"`
	IsMenuVisible *bool   `json:"is_menu_visible"`
	Scope         *string `json:"scope"`
	IsActive      *bool   `json:"is_active"`
}

// ToResourceDTO converts a Resource entity to ResourceDTO
func ToResourceDTO(resource *entities.Resource) ResourceDTO {
	d := ResourceDTO{
		ID:            resource.ID.String(),
		Key:           resource.Key,
		DisplayName:   resource.DisplayName,
		SortOrder:     resource.SortOrder,
		IsMenuVisible: resource.IsMenuVisible,
		Scope:         resource.Scope,
		IsActive:      resource.IsActive,
	}
	if resource.Description != nil {
		d.Description = *resource.Description
	}
	if resource.Icon != nil {
		d.Icon = *resource.Icon
	}
	if resource.ParentID != nil {
		pid := resource.ParentID.String()
		d.ParentID = &pid
	}
	return d
}

// ToResourceDTOList converts a slice of Resource entities to DTOs
func ToResourceDTOList(resources []*entities.Resource) []ResourceDTO {
	dtos := make([]ResourceDTO, len(resources))
	for i, r := range resources {
		dtos[i] = ToResourceDTO(r)
	}
	return dtos
}

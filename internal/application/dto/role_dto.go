package dto

import (
	"github.com/EduGoGroup/edugo-infrastructure/postgres/entities"
)

// RoleDTO represents a role in API responses
type RoleDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description,omitempty"`
	Scope       string `json:"scope"`
	IsActive    bool   `json:"is_active"`
}

// RolesResponse wraps a list of roles
type RolesResponse struct {
	Roles []*RoleDTO `json:"roles"`
}

// PermissionDTO represents a permission in API responses
type PermissionDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description,omitempty"`
	ResourceID  string `json:"resource_id"`
	ResourceKey string `json:"resource_key"`
	Action      string `json:"action"`
	Scope       string `json:"scope"`
}

// PermissionsResponse wraps a list of permissions
type PermissionsResponse struct {
	Permissions []*PermissionDTO `json:"permissions"`
}

// UserRoleDTO represents a user-role assignment
type UserRoleDTO struct {
	ID             string  `json:"id"`
	UserID         string  `json:"user_id"`
	RoleID         string  `json:"role_id"`
	RoleName       string  `json:"role_name"`
	SchoolID       *string `json:"school_id,omitempty"`
	AcademicUnitID *string `json:"academic_unit_id,omitempty"`
	IsActive       bool    `json:"is_active"`
	GrantedAt      string  `json:"granted_at"`
}

// UserRolesResponse wraps a list of user roles
type UserRolesResponse struct {
	UserRoles []*UserRoleDTO `json:"user_roles"`
}

// GrantRoleRequest represents the request to grant a role
type GrantRoleRequest struct {
	RoleID         string  `json:"role_id" binding:"required"`
	SchoolID       *string `json:"school_id,omitempty"`
	AcademicUnitID *string `json:"academic_unit_id,omitempty"`
	ExpiresAt      *string `json:"expires_at,omitempty"`
}

// GrantRoleResponse wraps the granted user role
type GrantRoleResponse struct {
	UserRole *UserRoleDTO `json:"user_role"`
}

// ToRoleDTO converts a Role entity to RoleDTO
func ToRoleDTO(role *entities.Role) *RoleDTO {
	d := &RoleDTO{
		ID:          role.ID.String(),
		Name:        role.Name,
		DisplayName: role.DisplayName,
		Scope:       role.Scope,
		IsActive:    role.IsActive,
	}
	if role.Description != nil {
		d.Description = *role.Description
	}
	return d
}

// ToRoleDTOList converts a slice of Role entities to DTOs
func ToRoleDTOList(roles []*entities.Role) []*RoleDTO {
	dtos := make([]*RoleDTO, len(roles))
	for i, role := range roles {
		dtos[i] = ToRoleDTO(role)
	}
	return dtos
}

// ToPermissionDTO converts a Permission entity to PermissionDTO
func ToPermissionDTO(perm *entities.Permission) *PermissionDTO {
	d := &PermissionDTO{
		ID:          perm.ID.String(),
		Name:        perm.Name,
		DisplayName: perm.DisplayName,
		ResourceID:  perm.ResourceID.String(),
		ResourceKey: perm.ResourceKey,
		Action:      perm.Action,
		Scope:       perm.Scope,
	}
	if perm.Description != nil {
		d.Description = *perm.Description
	}
	return d
}

// ToPermissionDTOList converts a slice of Permission entities to DTOs
func ToPermissionDTOList(perms []*entities.Permission) []*PermissionDTO {
	dtos := make([]*PermissionDTO, len(perms))
	for i, perm := range perms {
		dtos[i] = ToPermissionDTO(perm)
	}
	return dtos
}

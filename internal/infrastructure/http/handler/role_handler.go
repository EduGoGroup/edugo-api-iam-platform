package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/EduGoGroup/edugo-api-iam-platform/internal/application/dto"
	"github.com/EduGoGroup/edugo-api-iam-platform/internal/application/service"
	"github.com/EduGoGroup/edugo-shared/logger"
	sharedrepo "github.com/EduGoGroup/edugo-shared/repository"
)

type RoleHandler struct {
	roleService service.RoleService
	logger      logger.Logger
}

func NewRoleHandler(roleService service.RoleService, logger logger.Logger) *RoleHandler {
	return &RoleHandler{roleService: roleService, logger: logger}
}

// ListRoles lists all roles
// @Summary List roles
// @Description Get all roles, optionally filtered by scope
// @Tags Roles
// @Produce json
// @Security BearerAuth
// @Param scope query string false "Filter by scope (e.g. platform, school)"
// @Param search query string false "Search term (ILIKE)"
// @Param search_fields query string false "Comma-separated fields to search"
// @Param page query int false "Page number (1-based)" minimum(1)
// @Param limit query int false "Items per page" minimum(1) maximum(200)
// @Success 200 {object} dto.RolesResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /roles [get]
func (h *RoleHandler) ListRoles(c *gin.Context) {
	scope := c.Query("scope")
	var filters sharedrepo.ListFilters
	if search := c.Query("search"); search != "" {
		filters.Search = search
		if fields := c.Query("search_fields"); fields != "" {
			rawFields := strings.Split(fields, ",")
			cleanFields := make([]string, 0, len(rawFields))
			for _, f := range rawFields {
				if f = strings.TrimSpace(f); f != "" {
					cleanFields = append(cleanFields, f)
				}
			}
			if len(cleanFields) > 0 {
				filters.SearchFields = cleanFields
			}
		}
	}
	if pageStr := c.Query("page"); pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err != nil || page <= 0 {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "page must be a positive integer", Code: "INVALID_REQUEST"})
			return
		}
		filters.Page = page
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "limit must be a positive integer", Code: "INVALID_REQUEST"})
			return
		}
		filters.Limit = limit
	}
	roles, err := h.roleService.GetRoles(c.Request.Context(), scope, filters)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, roles)
}

// GetRole gets a role by ID
// @Summary Get role by ID
// @Description Get a single role by its ID
// @Tags Roles
// @Produce json
// @Security BearerAuth
// @Param id path string true "Role ID"
// @Success 200 {object} dto.RoleDTO
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /roles/{id} [get]
func (h *RoleHandler) GetRole(c *gin.Context) {
	id := c.Param("id")
	role, err := h.roleService.GetRole(c.Request.Context(), id)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, role)
}

// CreateRole creates a new role
// @Summary Create role
// @Description Create a new role
// @Tags Roles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.CreateRoleRequest true "Role data"
// @Success 201 {object} dto.RoleDTO
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /roles [post]
func (h *RoleHandler) CreateRole(c *gin.Context) {
	var req dto.CreateRoleRequest
	if err := bindJSON(c, &req); err != nil {
		_ = c.Error(err)
		return
	}
	role, err := h.roleService.CreateRole(c.Request.Context(), &req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, role)
}

// UpdateRole updates a role
// @Summary Update role
// @Description Update an existing role
// @Tags Roles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Role ID"
// @Param request body dto.UpdateRoleRequest true "Updated role data"
// @Success 200 {object} dto.RoleDTO
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /roles/{id} [put]
func (h *RoleHandler) UpdateRole(c *gin.Context) {
	id := c.Param("id")
	var req dto.UpdateRoleRequest
	if err := bindJSON(c, &req); err != nil {
		_ = c.Error(err)
		return
	}
	role, err := h.roleService.UpdateRole(c.Request.Context(), id, &req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, role)
}

// DeleteRole soft-deletes a role
// @Summary Delete role
// @Description Soft delete a role (set is_active=false)
// @Tags Roles
// @Security BearerAuth
// @Param id path string true "Role ID"
// @Success 204
// @Failure 404 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /roles/{id} [delete]
func (h *RoleHandler) DeleteRole(c *gin.Context) {
	id := c.Param("id")
	if err := h.roleService.DeleteRole(c.Request.Context(), id); err != nil {
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusNoContent)
}

// GetRolePermissions gets permissions for a role
// @Summary Get role permissions
// @Description Get all permissions assigned to a role
// @Tags Roles
// @Produce json
// @Security BearerAuth
// @Param id path string true "Role ID"
// @Success 200 {object} dto.PermissionsResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /roles/{id}/permissions [get]
func (h *RoleHandler) GetRolePermissions(c *gin.Context) {
	id := c.Param("id")
	perms, err := h.roleService.GetRolePermissions(c.Request.Context(), id)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, perms)
}

// AssignPermission assigns a permission to a role
// @Summary Assign permission to role
// @Description Assign a permission to a role
// @Tags Roles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Role ID"
// @Param request body dto.AssignPermissionRequest true "Permission assignment"
// @Success 201 {object} dto.RolePermissionResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /roles/{id}/permissions [post]
func (h *RoleHandler) AssignPermission(c *gin.Context) {
	id := c.Param("id")
	var req dto.AssignPermissionRequest
	if err := bindJSON(c, &req); err != nil {
		_ = c.Error(err)
		return
	}
	result, err := h.roleService.AssignPermission(c.Request.Context(), id, &req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, result)
}

// RevokePermission revokes a permission from a role
// @Summary Revoke permission from role
// @Description Remove a permission assignment from a role
// @Tags Roles
// @Security BearerAuth
// @Param id path string true "Role ID"
// @Param perm_id path string true "Permission ID"
// @Success 204
// @Failure 500 {object} dto.ErrorResponse
// @Router /roles/{id}/permissions/{perm_id} [delete]
func (h *RoleHandler) RevokePermission(c *gin.Context) {
	roleID := c.Param("id")
	permID := c.Param("perm_id")
	if err := h.roleService.RevokePermission(c.Request.Context(), roleID, permID); err != nil {
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusNoContent)
}

// BulkReplacePermissions replaces all permissions for a role
// @Summary Bulk replace role permissions
// @Description Replace all permissions assigned to a role
// @Tags Roles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Role ID"
// @Param request body dto.BulkPermissionsRequest true "Permission IDs"
// @Success 200 {object} dto.PermissionsResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /roles/{id}/permissions/bulk [put]
func (h *RoleHandler) BulkReplacePermissions(c *gin.Context) {
	id := c.Param("id")
	var req dto.BulkPermissionsRequest
	if err := bindJSON(c, &req); err != nil {
		_ = c.Error(err)
		return
	}
	result, err := h.roleService.BulkReplacePermissions(c.Request.Context(), id, &req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// GetUserRoles gets roles assigned to a user
// @Summary Get user roles
// @Description Get all roles assigned to a user
// @Tags Roles
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "User ID"
// @Success 200 {object} dto.RolesResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /users/{user_id}/roles [get]
func (h *RoleHandler) GetUserRoles(c *gin.Context) {
	userID := c.Param("user_id")
	roles, err := h.roleService.GetUserRoles(c.Request.Context(), userID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, roles)
}

// GrantRole grants a role to a user
// @Summary Grant role to user
// @Description Assign a role to a user
// @Tags Roles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "User ID"
// @Param request body dto.GrantRoleRequest true "Role grant request"
// @Success 201 {object} dto.GrantRoleResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /users/{user_id}/roles [post]
func (h *RoleHandler) GrantRole(c *gin.Context) {
	userID := c.Param("user_id")
	var req dto.GrantRoleRequest
	if err := bindJSON(c, &req); err != nil {
		_ = c.Error(err)
		return
	}
	grantedBy, _ := c.Get("user_id")
	grantedByStr := ""
	if grantedBy != nil {
		grantedByStr, _ = grantedBy.(string)
	}
	result, err := h.roleService.GrantRoleToUser(c.Request.Context(), userID, &req, grantedByStr)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, result)
}

// RevokeRole revokes a role from a user
// @Summary Revoke role from user
// @Description Remove a role assignment from a user
// @Tags Roles
// @Security BearerAuth
// @Param user_id path string true "User ID"
// @Param role_id path string true "Role ID"
// @Success 204
// @Failure 500 {object} dto.ErrorResponse
// @Router /users/{user_id}/roles/{role_id} [delete]
func (h *RoleHandler) RevokeRole(c *gin.Context) {
	userID := c.Param("user_id")
	roleID := c.Param("role_id")
	if err := h.roleService.RevokeRoleFromUser(c.Request.Context(), userID, roleID); err != nil {
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusNoContent)
}

package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/EduGoGroup/edugo-api-iam-platform/internal/application/dto"
	"github.com/EduGoGroup/edugo-api-iam-platform/internal/application/service"
	"github.com/EduGoGroup/edugo-shared/logger"
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
// @Success 200 {object} dto.RolesResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /roles [get]
func (h *RoleHandler) ListRoles(c *gin.Context) {
	scope := c.Query("scope")
	roles, err := h.roleService.GetRoles(c.Request.Context(), scope)
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
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "invalid request body", Code: "INVALID_REQUEST"})
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

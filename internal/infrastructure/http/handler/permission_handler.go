package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/EduGoGroup/edugo-api-iam-platform/internal/application/dto"
	"github.com/EduGoGroup/edugo-api-iam-platform/internal/application/service"
	"github.com/EduGoGroup/edugo-shared/logger"
)

// ensure dto import for swag annotations
var _ dto.ErrorResponse

type PermissionHandler struct {
	permissionService service.PermissionService
	logger            logger.Logger
}

func NewPermissionHandler(permissionService service.PermissionService, logger logger.Logger) *PermissionHandler {
	return &PermissionHandler{permissionService: permissionService, logger: logger}
}

// ListPermissions lists all permissions
// @Summary List permissions
// @Description Get all available permissions
// @Tags Permissions
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dto.PermissionsResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /permissions [get]
func (h *PermissionHandler) ListPermissions(c *gin.Context) {
	perms, err := h.permissionService.ListPermissions(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, perms)
}

// GetPermission gets a permission by ID
// @Summary Get permission by ID
// @Description Get a single permission by its ID
// @Tags Permissions
// @Produce json
// @Security BearerAuth
// @Param id path string true "Permission ID"
// @Success 200 {object} dto.PermissionDTO
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /permissions/{id} [get]
func (h *PermissionHandler) GetPermission(c *gin.Context) {
	id := c.Param("id")
	perm, err := h.permissionService.GetPermission(c.Request.Context(), id)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, perm)
}

package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/EduGoGroup/edugo-api-iam-platform/internal/application/dto"
	"github.com/EduGoGroup/edugo-api-iam-platform/internal/application/service"
	"github.com/EduGoGroup/edugo-shared/logger"
	sharedrepo "github.com/EduGoGroup/edugo-shared/repository"
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
// @Param search query string false "Search term (ILIKE)"
// @Param search_fields query string false "Comma-separated fields to search"
// @Success 200 {object} dto.PermissionsResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /permissions [get]
func (h *PermissionHandler) ListPermissions(c *gin.Context) {
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
	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		val := isActiveStr == "true"
		filters.IsActive = &val
	}
	perms, err := h.permissionService.ListPermissions(c.Request.Context(), filters)
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

// CreatePermission creates a new permission
// @Summary Create permission
// @Description Create a new permission
// @Tags Permissions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.CreatePermissionRequest true "Permission data"
// @Success 201 {object} dto.PermissionDTO
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /permissions [post]
func (h *PermissionHandler) CreatePermission(c *gin.Context) {
	var req dto.CreatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "invalid request body", Code: "INVALID_REQUEST"})
		return
	}
	perm, err := h.permissionService.CreatePermission(c.Request.Context(), &req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, perm)
}

// UpdatePermission updates a permission
// @Summary Update permission
// @Description Update an existing permission
// @Tags Permissions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Permission ID"
// @Param request body dto.UpdatePermissionRequest true "Updated permission data"
// @Success 200 {object} dto.PermissionDTO
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /permissions/{id} [put]
func (h *PermissionHandler) UpdatePermission(c *gin.Context) {
	id := c.Param("id")
	var req dto.UpdatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "invalid request body", Code: "INVALID_REQUEST"})
		return
	}
	perm, err := h.permissionService.UpdatePermission(c.Request.Context(), id, &req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, perm)
}

// DeletePermission soft-deletes a permission
// @Summary Delete permission
// @Description Soft delete a permission (set is_active=false)
// @Tags Permissions
// @Security BearerAuth
// @Param id path string true "Permission ID"
// @Success 204
// @Failure 404 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /permissions/{id} [delete]
func (h *PermissionHandler) DeletePermission(c *gin.Context) {
	id := c.Param("id")
	if err := h.permissionService.DeletePermission(c.Request.Context(), id); err != nil {
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusNoContent)
}

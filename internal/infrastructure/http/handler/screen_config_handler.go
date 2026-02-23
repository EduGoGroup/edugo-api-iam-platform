package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/EduGoGroup/edugo-api-iam-platform/internal/application/dto"
	"github.com/EduGoGroup/edugo-api-iam-platform/internal/application/service"
	"github.com/EduGoGroup/edugo-shared/logger"
)

type ScreenConfigHandler struct {
	screenService service.ScreenConfigService
	logger        logger.Logger
}

func NewScreenConfigHandler(screenService service.ScreenConfigService, logger logger.Logger) *ScreenConfigHandler {
	return &ScreenConfigHandler{screenService: screenService, logger: logger}
}

// Templates

// CreateTemplate creates a new screen template
// @Summary Create screen template
// @Description Create a new screen configuration template
// @Tags Screen Config
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body service.CreateTemplateRequest true "Template data"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /screen-config/templates [post]
func (h *ScreenConfigHandler) CreateTemplate(c *gin.Context) {
	var req service.CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "invalid request body", Code: "INVALID_REQUEST"})
		return
	}
	template, err := h.screenService.CreateTemplate(c.Request.Context(), &req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, template)
}

// ListTemplates lists screen templates
// @Summary List screen templates
// @Description Get all screen templates with optional filtering
// @Tags Screen Config
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} dto.ErrorResponse
// @Router /screen-config/templates [get]
func (h *ScreenConfigHandler) ListTemplates(c *gin.Context) {
	var filter service.TemplateFilter
	_ = c.ShouldBindQuery(&filter)
	templates, total, err := h.screenService.ListTemplates(c.Request.Context(), filter)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": templates, "total": total})
}

// GetTemplate gets a screen template by ID
// @Summary Get screen template
// @Description Get a screen template by its ID
// @Tags Screen Config
// @Produce json
// @Security BearerAuth
// @Param id path string true "Template ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /screen-config/templates/{id} [get]
func (h *ScreenConfigHandler) GetTemplate(c *gin.Context) {
	id := c.Param("id")
	template, err := h.screenService.GetTemplate(c.Request.Context(), id)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, template)
}

// UpdateTemplate updates a screen template
// @Summary Update screen template
// @Description Update an existing screen template
// @Tags Screen Config
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Template ID"
// @Param request body service.UpdateTemplateRequest true "Updated template data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /screen-config/templates/{id} [put]
func (h *ScreenConfigHandler) UpdateTemplate(c *gin.Context) {
	id := c.Param("id")
	var req service.UpdateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "invalid request body", Code: "INVALID_REQUEST"})
		return
	}
	template, err := h.screenService.UpdateTemplate(c.Request.Context(), id, &req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, template)
}

// DeleteTemplate deletes a screen template
// @Summary Delete screen template
// @Description Delete a screen template by its ID
// @Tags Screen Config
// @Security BearerAuth
// @Param id path string true "Template ID"
// @Success 204
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /screen-config/templates/{id} [delete]
func (h *ScreenConfigHandler) DeleteTemplate(c *gin.Context) {
	id := c.Param("id")
	if err := h.screenService.DeleteTemplate(c.Request.Context(), id); err != nil {
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Instances

// CreateInstance creates a new screen instance
// @Summary Create screen instance
// @Description Create a new screen configuration instance
// @Tags Screen Config
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body service.CreateInstanceRequest true "Instance data"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /screen-config/instances [post]
func (h *ScreenConfigHandler) CreateInstance(c *gin.Context) {
	var req service.CreateInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "invalid request body", Code: "INVALID_REQUEST"})
		return
	}
	instance, err := h.screenService.CreateInstance(c.Request.Context(), &req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, instance)
}

// ListInstances lists screen instances
// @Summary List screen instances
// @Description Get all screen instances with optional filtering
// @Tags Screen Config
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} dto.ErrorResponse
// @Router /screen-config/instances [get]
func (h *ScreenConfigHandler) ListInstances(c *gin.Context) {
	var filter service.InstanceFilter
	_ = c.ShouldBindQuery(&filter)
	instances, total, err := h.screenService.ListInstances(c.Request.Context(), filter)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": instances, "total": total})
}

// GetInstance gets a screen instance by ID
// @Summary Get screen instance
// @Description Get a screen instance by its ID
// @Tags Screen Config
// @Produce json
// @Security BearerAuth
// @Param id path string true "Instance ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /screen-config/instances/{id} [get]
func (h *ScreenConfigHandler) GetInstance(c *gin.Context) {
	id := c.Param("id")
	instance, err := h.screenService.GetInstance(c.Request.Context(), id)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, instance)
}

// GetInstanceByKey gets a screen instance by key
// @Summary Get screen instance by key
// @Description Get a screen instance by its unique key
// @Tags Screen Config
// @Produce json
// @Security BearerAuth
// @Param key path string true "Instance key"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /screen-config/instances/key/{key} [get]
func (h *ScreenConfigHandler) GetInstanceByKey(c *gin.Context) {
	key := c.Param("key")
	instance, err := h.screenService.GetInstanceByKey(c.Request.Context(), key)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, instance)
}

// UpdateInstance updates a screen instance
// @Summary Update screen instance
// @Description Update an existing screen instance
// @Tags Screen Config
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Instance ID"
// @Param request body service.UpdateInstanceRequest true "Updated instance data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /screen-config/instances/{id} [put]
func (h *ScreenConfigHandler) UpdateInstance(c *gin.Context) {
	id := c.Param("id")
	var req service.UpdateInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "invalid request body", Code: "INVALID_REQUEST"})
		return
	}
	instance, err := h.screenService.UpdateInstance(c.Request.Context(), id, &req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, instance)
}

// DeleteInstance deletes a screen instance
// @Summary Delete screen instance
// @Description Delete a screen instance by its ID
// @Tags Screen Config
// @Security BearerAuth
// @Param id path string true "Instance ID"
// @Success 204
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /screen-config/instances/{id} [delete]
func (h *ScreenConfigHandler) DeleteInstance(c *gin.Context) {
	id := c.Param("id")
	if err := h.screenService.DeleteInstance(c.Request.Context(), id); err != nil {
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Resolve

// ResolveScreenByKey resolves a screen configuration by key
// @Summary Resolve screen by key
// @Description Resolve and return the combined screen configuration for a given key
// @Tags Screen Config
// @Produce json
// @Security BearerAuth
// @Param key path string true "Screen key"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /screen-config/resolve/key/{key} [get]
func (h *ScreenConfigHandler) ResolveScreenByKey(c *gin.Context) {
	key := c.Param("key")
	combined, err := h.screenService.ResolveScreenByKey(c.Request.Context(), key)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, combined)
}

// Resource-Screens

// LinkScreenToResource links a screen to a resource
// @Summary Link screen to resource
// @Description Create a link between a screen instance and a resource
// @Tags Screen Config
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body service.LinkScreenRequest true "Link data"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /screen-config/resource-screens [post]
func (h *ScreenConfigHandler) LinkScreenToResource(c *gin.Context) {
	var req service.LinkScreenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "invalid request body", Code: "INVALID_REQUEST"})
		return
	}
	rs, err := h.screenService.LinkScreenToResource(c.Request.Context(), &req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, rs)
}

// GetScreensForResource gets screens linked to a resource
// @Summary Get screens for resource
// @Description Get all screen instances linked to a resource
// @Tags Screen Config
// @Produce json
// @Security BearerAuth
// @Param resourceId path string true "Resource ID"
// @Success 200 {array} map[string]interface{}
// @Failure 500 {object} dto.ErrorResponse
// @Router /screen-config/resource-screens/{resourceId} [get]
func (h *ScreenConfigHandler) GetScreensForResource(c *gin.Context) {
	resourceID := c.Param("resourceId")
	screens, err := h.screenService.GetScreensForResource(c.Request.Context(), resourceID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, screens)
}

// UnlinkScreen removes a screen-resource link
// @Summary Unlink screen from resource
// @Description Remove the link between a screen instance and a resource
// @Tags Screen Config
// @Security BearerAuth
// @Param id path string true "Resource-Screen link ID"
// @Success 204
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /screen-config/resource-screens/{id} [delete]
func (h *ScreenConfigHandler) UnlinkScreen(c *gin.Context) {
	id := c.Param("id")
	if err := h.screenService.UnlinkScreen(c.Request.Context(), id); err != nil {
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusNoContent)
}

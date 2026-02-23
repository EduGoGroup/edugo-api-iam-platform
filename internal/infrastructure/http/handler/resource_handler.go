package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/EduGoGroup/edugo-api-iam-platform/internal/application/dto"
	"github.com/EduGoGroup/edugo-api-iam-platform/internal/application/service"
	"github.com/EduGoGroup/edugo-shared/logger"
)

type ResourceHandler struct {
	resourceService service.ResourceService
	logger          logger.Logger
}

func NewResourceHandler(resourceService service.ResourceService, logger logger.Logger) *ResourceHandler {
	return &ResourceHandler{resourceService: resourceService, logger: logger}
}

// ListResources lists all resources
// @Summary List resources
// @Description Get all registered resources
// @Tags Resources
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dto.ResourcesResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /resources [get]
func (h *ResourceHandler) ListResources(c *gin.Context) {
	resources, err := h.resourceService.ListResources(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, resources)
}

// GetResource gets a resource by ID
// @Summary Get resource by ID
// @Description Get a single resource by its ID
// @Tags Resources
// @Produce json
// @Security BearerAuth
// @Param id path string true "Resource ID"
// @Success 200 {object} dto.ResourceDTO
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /resources/{id} [get]
func (h *ResourceHandler) GetResource(c *gin.Context) {
	id := c.Param("id")
	resource, err := h.resourceService.GetResource(c.Request.Context(), id)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, resource)
}

// CreateResource creates a new resource
// @Summary Create resource
// @Description Create a new resource entry
// @Tags Resources
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.CreateResourceRequest true "Resource data"
// @Success 201 {object} dto.ResourceDTO
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /resources [post]
func (h *ResourceHandler) CreateResource(c *gin.Context) {
	var req dto.CreateResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "invalid request body", Code: "INVALID_REQUEST"})
		return
	}
	resource, err := h.resourceService.CreateResource(c.Request.Context(), req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, resource)
}

// UpdateResource updates a resource
// @Summary Update resource
// @Description Update an existing resource
// @Tags Resources
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Resource ID"
// @Param request body dto.UpdateResourceRequest true "Updated resource data"
// @Success 200 {object} dto.ResourceDTO
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /resources/{id} [put]
func (h *ResourceHandler) UpdateResource(c *gin.Context) {
	id := c.Param("id")
	var req dto.UpdateResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "invalid request body", Code: "INVALID_REQUEST"})
		return
	}
	resource, err := h.resourceService.UpdateResource(c.Request.Context(), id, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, resource)
}

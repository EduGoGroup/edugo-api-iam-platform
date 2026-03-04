package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/EduGoGroup/edugo-api-iam-platform/internal/audit/model"
	"github.com/EduGoGroup/edugo-api-iam-platform/internal/audit/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AuditHandler handles audit query endpoints
type AuditHandler struct {
	service service.AuditQueryService
}

// NewAuditHandler creates a new audit handler
func NewAuditHandler(service service.AuditQueryService) *AuditHandler {
	return &AuditHandler{service: service}
}

// List godoc
// @Summary List audit events
// @Tags Audit
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(50)
// @Param action query string false "Filter by action"
// @Param resource_type query string false "Filter by resource type"
// @Param severity query string false "Filter by severity"
// @Param category query string false "Filter by category"
// @Param actor_id query string false "Filter by actor ID"
// @Param service_name query string false "Filter by service name"
// @Param from query string false "From date (RFC3339)"
// @Param to query string false "To date (RFC3339)"
// @Success 200 {object} map[string]interface{}
// @Router /audit/events [get]
func (h *AuditHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	filters := model.AuditFilters{
		Action:       c.Query("action"),
		ResourceType: c.Query("resource_type"),
		Severity:     c.Query("severity"),
		Category:     c.Query("category"),
		ActorID:      c.Query("actor_id"),
		ServiceName:  c.Query("service_name"),
	}

	if from := c.Query("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			filters.From = &t
		}
	}
	if to := c.Query("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			filters.To = &t
		}
	}

	events, total, err := h.service.List(c.Request.Context(), filters, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list audit events"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events":    events,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetByID godoc
// @Summary Get audit event by ID
// @Tags Audit
// @Produce json
// @Security BearerAuth
// @Param id path string true "Event ID"
// @Success 200 {object} model.AuditEvent
// @Router /audit/events/{id} [get]
func (h *AuditHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	event, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "audit event not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get audit event"})
		return
	}
	c.JSON(http.StatusOK, event)
}

// GetByUserID godoc
// @Summary Get audit events by user ID
// @Tags Audit
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "User ID"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(50)
// @Success 200 {object} map[string]interface{}
// @Router /audit/events/user/{user_id} [get]
func (h *AuditHandler) GetByUserID(c *gin.Context) {
	userID := c.Param("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	events, total, err := h.service.GetByUserID(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get audit events"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events":    events,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetByResource godoc
// @Summary Get audit events by resource
// @Tags Audit
// @Produce json
// @Security BearerAuth
// @Param type path string true "Resource type"
// @Param id path string true "Resource ID"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(50)
// @Success 200 {object} map[string]interface{}
// @Router /audit/events/resource/{type}/{id} [get]
func (h *AuditHandler) GetByResource(c *gin.Context) {
	resourceType := c.Param("type")
	resourceID := c.Param("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	events, total, err := h.service.GetByResource(c.Request.Context(), resourceType, resourceID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get audit events"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events":    events,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// Summary godoc
// @Summary Get audit summary
// @Tags Audit
// @Produce json
// @Security BearerAuth
// @Param from query string false "From date (RFC3339)"
// @Param to query string false "To date (RFC3339)"
// @Success 200 {object} model.AuditSummary
// @Router /audit/summary [get]
func (h *AuditHandler) Summary(c *gin.Context) {
	now := time.Now()
	from := now.AddDate(0, 0, -30) // default last 30 days
	to := now

	if f := c.Query("from"); f != "" {
		if t, err := time.Parse(time.RFC3339, f); err == nil {
			from = t
		}
	}
	if t := c.Query("to"); t != "" {
		if parsed, err := time.Parse(time.RFC3339, t); err == nil {
			to = parsed
		}
	}

	summary, err := h.service.Summary(c.Request.Context(), from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get audit summary"})
		return
	}

	c.JSON(http.StatusOK, summary)
}

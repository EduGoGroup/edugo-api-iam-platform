package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type HealthHandler struct {
	db      *gorm.DB
	version string
}

func NewHealthHandler(db *gorm.DB, version string) *HealthHandler {
	return &HealthHandler{db: db, version: version}
}

// Health returns the service health status
// @Summary Health check
// @Description Check the health status of the IAM Platform service
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /health [get]
func (h *HealthHandler) Health(c *gin.Context) {
	status := "ok"
	checks := map[string]string{}

	if h.db != nil {
		sqlDB, err := h.db.DB()
		if err != nil {
			status = "degraded"
			checks["postgres"] = "unhealthy"
		} else if err := sqlDB.PingContext(c.Request.Context()); err != nil {
			status = "degraded"
			checks["postgres"] = "unhealthy"
		} else {
			checks["postgres"] = "healthy"
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  status,
		"service": "edugo-api-iam-platform",
		"version": h.version,
		"checks":  checks,
	})
}

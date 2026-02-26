package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/EduGoGroup/edugo-api-iam-platform/internal/application/dto"
	"github.com/EduGoGroup/edugo-api-iam-platform/internal/application/service"
	"github.com/EduGoGroup/edugo-shared/auth"
	"github.com/EduGoGroup/edugo-shared/logger"
	ginmiddleware "github.com/EduGoGroup/edugo-shared/middleware/gin"
)

// SyncHandler handles sync endpoints
type SyncHandler struct {
	syncService service.SyncService
	logger      logger.Logger
}

// NewSyncHandler creates a new SyncHandler
func NewSyncHandler(syncService service.SyncService, logger logger.Logger) *SyncHandler {
	return &SyncHandler{syncService: syncService, logger: logger}
}

// GetBundle returns the full sync bundle for the authenticated user
func (h *SyncHandler) GetBundle(c *gin.Context) {
	userID, activeContext, ok := h.extractAuth(c)
	if !ok {
		return
	}

	bundle, err := h.syncService.GetFullBundle(c.Request.Context(), userID, activeContext)
	if err != nil {
		h.logger.Error("error building sync bundle", "user_id", userID, "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "internal_error",
			Code:  "SYNC_BUNDLE_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, bundle)
}

// DeltaSync returns only changed buckets based on client hashes
func (h *SyncHandler) DeltaSync(c *gin.Context) {
	userID, activeContext, ok := h.extractAuth(c)
	if !ok {
		return
	}

	var req dto.DeltaSyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "bad_request",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	delta, err := h.syncService.GetDeltaSync(c.Request.Context(), userID, activeContext, req.Hashes)
	if err != nil {
		h.logger.Error("error computing delta sync", "user_id", userID, "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "internal_error",
			Code:  "SYNC_DELTA_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, delta)
}

// extractAuth extracts userID and activeContext from JWT claims
func (h *SyncHandler) extractAuth(c *gin.Context) (string, *auth.UserContext, bool) {
	userID, err := ginmiddleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: "unauthorized",
			Code:  "NOT_AUTHENTICATED",
		})
		return "", nil, false
	}

	claims, err := ginmiddleware.GetClaims(c)
	if err != nil || claims == nil || claims.ActiveContext == nil {
		c.JSON(http.StatusForbidden, dto.ErrorResponse{
			Error: "forbidden",
			Code:  "NO_ACTIVE_CONTEXT",
		})
		return "", nil, false
	}

	return userID, claims.ActiveContext, true
}

package handler

import (
	"net/http"
	"strings"

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
// @Summary Get full sync bundle
// @Description Returns the sync bundle. Use ?buckets=menu,permissions,available_contexts,screens to load specific buckets only.
// @Tags Sync
// @Produce json
// @Security BearerAuth
// @Param buckets query string false "Comma-separated bucket names to load (menu,permissions,available_contexts,screens). Empty = all."
// @Success 200 {object} dto.SyncBundleResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /sync/bundle [get]
func (h *SyncHandler) GetBundle(c *gin.Context) {
	userID, activeContext, ok := h.extractAuth(c)
	if !ok {
		return
	}

	// Parse optional buckets filter
	var buckets []string
	if bucketsParam := c.Query("buckets"); bucketsParam != "" {
		for _, b := range strings.Split(bucketsParam, ",") {
			b = strings.TrimSpace(b)
			if b != "" {
				buckets = append(buckets, b)
			}
		}
	}

	bundle, err := h.syncService.GetFullBundle(c.Request.Context(), userID, activeContext, buckets)
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
// @Summary Get delta sync
// @Description Compares client hashes with server state and returns only changed buckets
// @Tags Sync
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.DeltaSyncRequest true "Client hashes for comparison"
// @Success 200 {object} dto.DeltaSyncResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /sync/delta [post]
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

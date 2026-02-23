package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/EduGoGroup/edugo-api-iam-platform/internal/auth/dto"
	"github.com/EduGoGroup/edugo-api-iam-platform/internal/auth/service"
)

// VerifyHandler handles token verification endpoints
type VerifyHandler struct {
	tokenService *service.TokenService
}

// NewVerifyHandler creates a new VerifyHandler
func NewVerifyHandler(tokenService *service.TokenService) *VerifyHandler {
	return &VerifyHandler{tokenService: tokenService}
}

// VerifyToken verifies a JWT token
// @Summary Verify JWT token
// @Description Validate a JWT token and return its claims
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body dto.VerifyTokenRequest true "Token to verify"
// @Success 200 {object} dto.VerifyTokenResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /auth/verify [post]
func (h *VerifyHandler) VerifyToken(c *gin.Context) {
	startTime := time.Now()

	var req dto.VerifyTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "bad_request",
			Message: "Token is required",
			Code:    "INVALID_REQUEST",
		})
		return
	}

	token := strings.TrimPrefix(req.Token, "Bearer ")
	token = strings.TrimSpace(token)

	if token == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "bad_request",
			Message: "Empty token",
			Code:    "EMPTY_TOKEN",
		})
		return
	}

	response, err := h.tokenService.VerifyToken(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "internal_error",
			Message: "Error verifying token",
			Code:    "VERIFICATION_ERROR",
		})
		return
	}

	duration := time.Since(startTime)
	c.Header("X-Response-Time", duration.String())
	c.JSON(http.StatusOK, response)
}

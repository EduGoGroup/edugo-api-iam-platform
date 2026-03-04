package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/EduGoGroup/edugo-api-iam-platform/internal/application/dto"
	"github.com/EduGoGroup/edugo-shared/common/errors"
	"github.com/EduGoGroup/edugo-shared/logger"
)

func ErrorHandler(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			handleError(c, log, err)
		}
	}
}

func handleError(c *gin.Context, log logger.Logger, err error) {
	if appErr, ok := errors.GetAppError(err); ok {
		log.Error("request failed",
			"path", c.Request.URL.Path,
			"method", c.Request.Method,
			"error", appErr.Message,
			"code", appErr.Code,
			"status", appErr.StatusCode,
		)
		response := dto.ErrorResponse{
			Error: appErr.Message,
			Code:  string(appErr.Code),
		}
		if len(appErr.Fields) > 0 {
			details := make(map[string]string, len(appErr.Fields))
			for k, v := range appErr.Fields {
				details[k] = fmt.Sprintf("%v", v)
			}
			response.Details = details
		}
		c.JSON(appErr.StatusCode, response)
		return
	}

	log.Error("unexpected error",
		"path", c.Request.URL.Path,
		"method", c.Request.Method,
		"error", err.Error(),
	)
	c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
		Error: "internal server error",
		Code:  "INTERNAL_ERROR",
	})
}

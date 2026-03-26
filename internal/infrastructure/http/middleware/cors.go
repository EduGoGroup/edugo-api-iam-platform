package middleware

import (
	"log"
	"strings"

	"github.com/EduGoGroup/edugo-api-iam-platform/internal/config"
	"github.com/gin-gonic/gin"
)

// CORSMiddleware configures CORS based on configuration.
// In non-development environments, wildcard (*) origins are rejected at startup.
func CORSMiddleware(cfg *config.CORSConfig, environment string) gin.HandlerFunc {
	allowedOrigins := parseCSV(cfg.AllowedOrigins)

	hasWildcard := false
	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			hasWildcard = true
			break
		}
	}

	// Block wildcard CORS in non-development environments
	if hasWildcard && environment != "" && environment != "development" && environment != "local" {
		log.Fatalf("CORS wildcard (*) is not allowed in %s environment. Set CORS_ALLOWED_ORIGINS explicitly.", environment)
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		if isOriginAllowed(origin, allowedOrigins) {
			if hasWildcard {
				c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			}
		}

		if c.Request.Method == "OPTIONS" {
			c.Writer.Header().Set("Access-Control-Allow-Methods", cfg.AllowedMethods)
			c.Writer.Header().Set("Access-Control-Allow-Headers", cfg.AllowedHeaders)
			c.Writer.Header().Set("Access-Control-Max-Age", "86400")
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func parseCSV(csv string) []string {
	if csv == "" {
		return []string{}
	}
	parts := strings.Split(csv, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func isOriginAllowed(origin string, allowedOrigins []string) bool {
	if origin == "" {
		return false
	}
	for _, allowed := range allowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

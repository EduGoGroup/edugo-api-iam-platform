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

	// Block wildcard CORS in non-development environments (fail closed: empty env is treated as non-development)
	normalizedEnv := strings.ToLower(strings.TrimSpace(environment))

	if hasWildcard && normalizedEnv != "development" && normalizedEnv != "local" {
		envForLog := environment
		if strings.TrimSpace(environment) == "" {
			envForLog = "non-development"
		}
		log.Fatalf("CORS wildcard (*) is not allowed in %s environment. Set CORS_ALLOWED_ORIGINS explicitly.", envForLog)
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		originAllowed := isOriginAllowed(origin, allowedOrigins)
		if originAllowed {
			if hasWildcard {
				c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length,ETag,X-Request-ID,X-Correlation-ID")
		}

		if c.Request.Method == "OPTIONS" {
			if originAllowed {
				c.Writer.Header().Set("Access-Control-Allow-Methods", cfg.AllowedMethods)
				c.Writer.Header().Set("Access-Control-Allow-Headers", cfg.AllowedHeaders)
				c.Writer.Header().Set("Access-Control-Max-Age", "86400")
			}
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

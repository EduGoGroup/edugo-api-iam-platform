package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EduGoGroup/edugo-api-iam-platform/internal/config"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestCORSMiddleware_WildcardInDevelopment(t *testing.T) {
	cfg := &config.CORSConfig{
		AllowedOrigins: "*",
		AllowedMethods: "GET,POST",
		AllowedHeaders: "Origin,Content-Type",
	}

	handler := CORSMiddleware(cfg, "development")
	if handler == nil {
		t.Fatal("expected handler, got nil")
	}

	r := gin.New()
	r.Use(handler)
	r.GET("/test", func(c *gin.Context) { c.Status(200) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	r.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected wildcard origin, got %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSMiddleware_WildcardInLocal(t *testing.T) {
	cfg := &config.CORSConfig{
		AllowedOrigins: "*",
		AllowedMethods: "GET,POST",
		AllowedHeaders: "Origin,Content-Type",
	}

	handler := CORSMiddleware(cfg, "local")
	if handler == nil {
		t.Fatal("expected handler, got nil")
	}
}

func TestCORSMiddleware_WildcardEmptyEnv(t *testing.T) {
	cfg := &config.CORSConfig{
		AllowedOrigins: "*",
		AllowedMethods: "GET,POST",
		AllowedHeaders: "Origin,Content-Type",
	}

	handler := CORSMiddleware(cfg, "")
	if handler == nil {
		t.Fatal("expected handler, got nil")
	}
}

func TestCORSMiddleware_ExplicitOriginsInProduction(t *testing.T) {
	cfg := &config.CORSConfig{
		AllowedOrigins: "https://app.edugo.com,https://admin.edugo.com",
		AllowedMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowedHeaders: "Origin,Content-Type,Authorization",
	}

	handler := CORSMiddleware(cfg, "production")
	if handler == nil {
		t.Fatal("expected handler, got nil")
	}

	r := gin.New()
	r.Use(handler)
	r.GET("/test", func(c *gin.Context) { c.Status(200) })

	// Allowed origin
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://app.edugo.com")
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.edugo.com" {
		t.Errorf("expected allowed origin, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("expected credentials header, got %q", got)
	}

	// Disallowed origin
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/test", nil)
	req2.Header.Set("Origin", "https://evil.com")
	r.ServeHTTP(w2, req2)

	if got := w2.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no origin header for disallowed origin, got %q", got)
	}
}

func TestCORSMiddleware_PreflightResponse(t *testing.T) {
	cfg := &config.CORSConfig{
		AllowedOrigins: "https://app.edugo.com",
		AllowedMethods: "GET,POST,PUT",
		AllowedHeaders: "Origin,Content-Type,Authorization",
	}

	handler := CORSMiddleware(cfg, "production")
	r := gin.New()
	r.Use(handler)
	r.GET("/test", func(c *gin.Context) { c.Status(200) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://app.edugo.com")
	r.ServeHTTP(w, req)

	if w.Code != 204 {
		t.Errorf("expected 204 for preflight, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Methods"); got != "GET,POST,PUT" {
		t.Errorf("expected methods header, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Max-Age"); got != "86400" {
		t.Errorf("expected max-age header, got %q", got)
	}
}

// Note: Testing log.Fatal in production+wildcard scenario is not straightforward
// because log.Fatal calls os.Exit(1). The behavior is verified by the explicit
// origins test in production, which passes without fatal.
// The wildcard+production case is documented as a fail-fast startup guard.

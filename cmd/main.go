package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"

	"github.com/EduGoGroup/edugo-api-iam-platform/docs"
	"github.com/EduGoGroup/edugo-api-iam-platform/internal/config"
	"github.com/EduGoGroup/edugo-api-iam-platform/internal/container"
	"github.com/EduGoGroup/edugo-api-iam-platform/internal/infrastructure/http/middleware"
	"github.com/EduGoGroup/edugo-shared/audit"
	"github.com/EduGoGroup/edugo-shared/common/types/enum"
	"github.com/EduGoGroup/edugo-shared/logger"
	ginmiddleware "github.com/EduGoGroup/edugo-shared/middleware/gin"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

// @title EduGo API IAM Platform
// @version 1.0
// @description IAM Platform API for EduGo - authentication, roles, permissions, resources, menu and screen configuration
// @host localhost:8070
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and the JWT token. Example: "Bearer eyJhbGci..."
func main() {
	log.Printf("EduGo API IAM Platform starting... (Version: %s, Build: %s)", Version, BuildTime)

	// 1. Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	// 2. Connect to PostgreSQL via GORM
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s search_path=auth,iam,academic,ui_config,public",
		cfg.Database.Postgres.Host, cfg.Database.Postgres.Port, cfg.Database.Postgres.User,
		cfg.Database.Postgres.Password, cfg.Database.Postgres.Database, cfg.Database.Postgres.SSLMode)

	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Info),
	})
	if err != nil {
		log.Fatalf("Error connecting to PostgreSQL via GORM: %v", err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		log.Fatalf("Error getting underlying sql.DB: %v", err)
	}

	sqlDB.SetMaxOpenConns(cfg.Database.Postgres.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Database.Postgres.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		log.Fatalf("Error pinging PostgreSQL: %v", err)
	}
	log.Println("PostgreSQL connected successfully via GORM")

	// 3. Initialize logger
	appLogger := newSimpleLogger()

	// 4. Create dependency container
	c := container.NewContainer(gormDB, appLogger, cfg)
	defer func() { _ = c.Close() }()

	// 5. Configure Swagger host dynamically
	docs.SwaggerInfo.Host = fmt.Sprintf("localhost:%d", cfg.Server.Port)

	// 6. Configure Gin
	r := gin.Default()

	// CORS middleware
	r.Use(middleware.CORSMiddleware(&cfg.CORS))

	// Error handler middleware
	r.Use(middleware.ErrorHandler(appLogger))

	// Health check
	r.GET("/health", c.HealthHandler.Health)

	// Swagger UI
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// ==================== PUBLIC ROUTES ====================

	v1Public := r.Group("/api/v1")
	{
		v1Public.GET("/health", c.HealthHandler.Health)

		authGroup := v1Public.Group("/auth")
		{
			authGroup.POST("/login", c.AuthHandler.Login)
			authGroup.POST("/refresh", c.AuthHandler.Refresh)
			authGroup.POST("/verify", c.VerifyHandler.VerifyToken)
		}
	}

	// ==================== PROTECTED ROUTES (JWT required) ====================
	auditLogger := audit.NewPostgresAuditLogger(gormDB, "iam-platform")

	v1 := r.Group("/api/v1")
	v1.Use(ginmiddleware.JWTAuthMiddleware(c.JWTManager))
	v1.Use(ginmiddleware.AuditMiddleware(auditLogger))
	{
		// Auth (protected)
		v1.POST("/auth/logout", c.AuthHandler.Logout)
		v1.POST("/auth/switch-context", c.AuthHandler.SwitchContext)
		v1.GET("/auth/contexts", c.AuthHandler.GetAvailableContexts)

		// Roles
		roles := v1.Group("/roles")
		{
			roles.GET("", ginmiddleware.RequirePermission(enum.PermissionRolesRead), c.RoleHandler.ListRoles)
			roles.GET("/:id", ginmiddleware.RequirePermission(enum.PermissionRolesRead), c.RoleHandler.GetRole)
			roles.POST("", ginmiddleware.RequirePermission(enum.PermissionRolesCreate), c.RoleHandler.CreateRole)
			roles.PUT("/:id", ginmiddleware.RequirePermission(enum.PermissionRolesUpdate), c.RoleHandler.UpdateRole)
			roles.DELETE("/:id", ginmiddleware.RequirePermission(enum.PermissionRolesDelete), c.RoleHandler.DeleteRole)
			roles.GET("/:id/permissions", ginmiddleware.RequirePermission(enum.PermissionRolesRead), c.RoleHandler.GetRolePermissions)
			roles.POST("/:id/permissions", ginmiddleware.RequirePermission(enum.PermissionRolesUpdate), c.RoleHandler.AssignPermission)
			roles.DELETE("/:id/permissions/:perm_id", ginmiddleware.RequirePermission(enum.PermissionRolesUpdate), c.RoleHandler.RevokePermission)
			roles.PUT("/:id/permissions/bulk", ginmiddleware.RequirePermission(enum.PermissionRolesUpdate), c.RoleHandler.BulkReplacePermissions)
		}

		// Permissions
		permissions := v1.Group("/permissions")
		{
			permissions.GET("", ginmiddleware.RequirePermission(enum.PermissionPermissionsMgmtRead), c.PermissionHandler.ListPermissions)
			permissions.GET("/:id", ginmiddleware.RequirePermission(enum.PermissionPermissionsMgmtRead), c.PermissionHandler.GetPermission)
			permissions.POST("", ginmiddleware.RequirePermission(enum.PermissionPermissionsMgmtCreate), c.PermissionHandler.CreatePermission)
			permissions.PUT("/:id", ginmiddleware.RequirePermission(enum.PermissionPermissionsMgmtUpdate), c.PermissionHandler.UpdatePermission)
			permissions.DELETE("/:id", ginmiddleware.RequirePermission(enum.PermissionPermissionsMgmtDelete), c.PermissionHandler.DeletePermission)
		}

		// Resources
		resources := v1.Group("/resources")
		{
			resources.GET("", ginmiddleware.RequirePermission(enum.PermissionPermissionsMgmtRead), c.ResourceHandler.ListResources)
			resources.GET("/:id", ginmiddleware.RequirePermission(enum.PermissionPermissionsMgmtRead), c.ResourceHandler.GetResource)
			resources.POST("", ginmiddleware.RequirePermission(enum.PermissionPermissionsMgmtCreate), c.ResourceHandler.CreateResource)
			resources.PUT("/:id", ginmiddleware.RequirePermission(enum.PermissionPermissionsMgmtUpdate), c.ResourceHandler.UpdateResource)
		}

		// Menu
		v1.GET("/menu", c.MenuHandler.GetUserMenu)
		v1.GET("/menu/full", c.MenuHandler.GetFullMenu)

		// User Roles
		users := v1.Group("/users")
		{
			users.GET("/:user_id/roles", ginmiddleware.RequirePermission(enum.PermissionUsersRead), c.RoleHandler.GetUserRoles)
			users.POST("/:user_id/roles", ginmiddleware.RequirePermission(enum.PermissionUsersUpdate), c.RoleHandler.GrantRole)
			users.DELETE("/:user_id/roles/:role_id", ginmiddleware.RequirePermission(enum.PermissionUsersUpdate), c.RoleHandler.RevokeRole)
		}

		// Sync
		syncGroup := v1.Group("/sync")
		{
			syncGroup.GET("/bundle", c.SyncHandler.GetBundle)
			syncGroup.POST("/delta", c.SyncHandler.DeltaSync)
		}

		// Screen Config
		screenConfig := v1.Group("/screen-config")
		{
			templates := screenConfig.Group("/templates")
			{
				templates.POST("", ginmiddleware.RequirePermission(enum.PermissionScreenTemplatesCreate), c.ScreenConfigHandler.CreateTemplate)
				templates.GET("", ginmiddleware.RequirePermission(enum.PermissionScreenTemplatesRead), c.ScreenConfigHandler.ListTemplates)
				templates.GET("/:id", ginmiddleware.RequirePermission(enum.PermissionScreenTemplatesRead), c.ScreenConfigHandler.GetTemplate)
				templates.PUT("/:id", ginmiddleware.RequirePermission(enum.PermissionScreenTemplatesUpdate), c.ScreenConfigHandler.UpdateTemplate)
				templates.DELETE("/:id", ginmiddleware.RequirePermission(enum.PermissionScreenTemplatesDelete), c.ScreenConfigHandler.DeleteTemplate)
			}
			instances := screenConfig.Group("/instances")
			{
				instances.POST("", ginmiddleware.RequirePermission(enum.PermissionScreenInstancesCreate), c.ScreenConfigHandler.CreateInstance)
				instances.GET("", ginmiddleware.RequirePermission(enum.PermissionScreenInstancesRead), c.ScreenConfigHandler.ListInstances)
				instances.GET("/:id", ginmiddleware.RequirePermission(enum.PermissionScreenInstancesRead), c.ScreenConfigHandler.GetInstance)
				instances.GET("/key/:key", ginmiddleware.RequirePermission(enum.PermissionScreenInstancesRead), c.ScreenConfigHandler.GetInstanceByKey)
				instances.PUT("/:id", ginmiddleware.RequirePermission(enum.PermissionScreenInstancesUpdate), c.ScreenConfigHandler.UpdateInstance)
				instances.DELETE("/:id", ginmiddleware.RequirePermission(enum.PermissionScreenInstancesDelete), c.ScreenConfigHandler.DeleteInstance)
			}
			screenConfig.GET("/version/:key", ginmiddleware.RequirePermission(enum.PermissionScreensRead), c.ScreenConfigHandler.GetScreenVersion)
			resolve := screenConfig.Group("/resolve")
			{
				resolve.GET("/key/:key", ginmiddleware.RequirePermission(enum.PermissionScreensRead), c.ScreenConfigHandler.ResolveScreenByKey)
			}
			resourceScreens := screenConfig.Group("/resource-screens")
			{
				resourceScreens.POST("", ginmiddleware.RequirePermission(enum.PermissionScreenTemplatesCreate), c.ScreenConfigHandler.LinkScreenToResource)
				resourceScreens.GET("/:resourceId", ginmiddleware.RequirePermission(enum.PermissionScreenTemplatesRead), c.ScreenConfigHandler.GetScreensForResource)
				resourceScreens.DELETE("/:id", ginmiddleware.RequirePermission(enum.PermissionScreenTemplatesDelete), c.ScreenConfigHandler.UnlinkScreen)
			}
		}

		// Audit endpoints
		auditGroup := v1.Group("/audit")
		auditGroup.Use(ginmiddleware.RequirePermission(enum.PermissionAuditRead))
		{
			auditGroup.GET("/events", c.AuditHandler.List)
			auditGroup.GET("/events/:id", c.AuditHandler.GetByID)
			auditGroup.GET("/events/user/:user_id", c.AuditHandler.GetByUserID)
			auditGroup.GET("/events/resource/:type/:id", c.AuditHandler.GetByResource)
			auditGroup.GET("/summary", c.AuditHandler.Summary)
		}
	}

	// 6. Start HTTP server with graceful shutdown
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		log.Printf("Server listening on port %d", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}

// simpleLogger adapts standard log to the logger.Logger interface
type simpleLogger struct{}

func newSimpleLogger() *simpleLogger { return &simpleLogger{} }

func (l *simpleLogger) Debug(msg string, keysAndValues ...interface{}) {
	log.Printf("[DEBUG] %s %v", msg, keysAndValues)
}
func (l *simpleLogger) Info(msg string, keysAndValues ...interface{}) {
	log.Printf("[INFO] %s %v", msg, keysAndValues)
}
func (l *simpleLogger) Warn(msg string, keysAndValues ...interface{}) {
	log.Printf("[WARN] %s %v", msg, keysAndValues)
}
func (l *simpleLogger) Error(msg string, keysAndValues ...interface{}) {
	log.Printf("[ERROR] %s %v", msg, keysAndValues)
}
func (l *simpleLogger) Fatal(msg string, keysAndValues ...interface{}) {
	log.Fatalf("[FATAL] %s %v", msg, keysAndValues)
}
func (l *simpleLogger) With(_ ...interface{}) logger.Logger {
	return l
}
func (l *simpleLogger) Sync() error { return nil }

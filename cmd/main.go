package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	gormlogger "gorm.io/gorm/logger"

	"github.com/EduGoGroup/edugo-shared/bootstrap"
	pgbootstrap "github.com/EduGoGroup/edugo-shared/bootstrap/postgres"

	"github.com/EduGoGroup/edugo-api-iam-platform/docs"
	"github.com/EduGoGroup/edugo-api-iam-platform/internal/config"
	"github.com/EduGoGroup/edugo-api-iam-platform/internal/container"
	"github.com/EduGoGroup/edugo-api-iam-platform/internal/infrastructure/http/middleware"
	auditpostgres "github.com/EduGoGroup/edugo-shared/audit/postgres"
	"github.com/EduGoGroup/edugo-shared/auth"
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
	slog.Info("EduGo API IAM Platform starting...", "version", Version, "build", BuildTime)

	// 1. Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Error loading configuration", "error", err)
		os.Exit(1)
	}

	// 2. Initialize logger (early, so all subsequent logs use the configured handler)
	slogLogger := logger.NewSlogProvider(logger.SlogConfig{
		Level:   cfg.Logging.Level,
		Format:  cfg.Logging.Format,
		Service: "edugo-api-iam-platform",
		Env:     cfg.Environment,
		Version: Version,
	})
	slog.SetDefault(slogLogger)
	appLogger := logger.NewSlogAdapter(slogLogger)

	// 3. Connect to PostgreSQL via GORM (using bootstrap/postgres factory)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pgFactory := pgbootstrap.NewFactory()
	gormDB, err := pgFactory.CreateGORMConnection(ctx, bootstrap.PostgreSQLConfig{
		Host:            cfg.Database.Postgres.Host,
		Port:            cfg.Database.Postgres.Port,
		User:            cfg.Database.Postgres.User,
		Password:        cfg.Database.Postgres.Password,
		Database:        cfg.Database.Postgres.Database,
		SSLMode:         cfg.Database.Postgres.SSLMode,
		SearchPath:      "auth,iam,academic,ui_config,public",
		MaxOpenConns:    cfg.Database.Postgres.MaxOpenConns,
		MaxIdleConns:    cfg.Database.Postgres.MaxIdleConns,
		ConnMaxLifetime: time.Hour,
	},
		bootstrap.WithGORMLogger(gormlogger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			gormlogger.Config{
				SlowThreshold:             500 * time.Millisecond,
				LogLevel:                  gormlogger.Info,
				IgnoreRecordNotFoundError: true,
				Colorful:                  true,
			},
		)),
	)
	if err != nil {
		slog.Error("Error connecting to PostgreSQL via GORM", "error", err)
		os.Exit(1)
	}
	slog.Info("PostgreSQL connected successfully via GORM")

	// 4. Create token blacklist (in-memory, TTL cleanup via background goroutine)
	blacklistCtx, blacklistCancel := context.WithCancel(context.Background())
	defer blacklistCancel()
	blacklist := auth.NewInMemoryBlacklist(blacklistCtx)

	// 5. Create dependency container
	c := container.NewContainer(gormDB, appLogger, cfg, blacklist)
	defer func() { _ = c.Close() }()

	// 6. Configure Swagger host dynamically
	docs.SwaggerInfo.Host = fmt.Sprintf("localhost:%d", cfg.Server.Port)

	// 7. Configure Gin
	r := gin.New()
	r.Use(gin.Recovery())

	// CORS middleware
	r.Use(middleware.CORSMiddleware(&cfg.CORS, cfg.Environment))

	// Request logging middleware (request_id, structured logging)
	r.Use(ginmiddleware.RequestLogging(slogLogger))

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
	auditLogger := auditpostgres.NewPostgresAuditLogger(gormDB, "iam-platform")

	v1 := r.Group("/api/v1")
	v1.Use(ginmiddleware.JWTAuthMiddlewareWithBlacklist(c.JWTManager, blacklist))
	v1.Use(ginmiddleware.PostAuthLogging())
	v1.Use(ginmiddleware.AuditMiddleware(auditLogger))
	{
		// Auth (protected)
		v1.POST("/auth/logout", c.AuthHandler.Logout)
		v1.POST("/auth/switch-context", c.AuthHandler.SwitchContext)
		v1.GET("/auth/contexts", c.AuthHandler.GetAvailableContexts)
		v1.GET("/auth/contexts/schools/:school_id/units", ginmiddleware.RequirePermission(enum.PermissionContextBrowseUnits), c.AuthHandler.GetSchoolUnits)

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

	// 8. Start HTTP server with graceful shutdown
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		slogLogger.Info("Server listening", "port", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slogLogger.Error("Server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slogLogger.Info("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slogLogger.Error("Server shutdown error", "error", err)
	}

	slogLogger.Info("Server stopped")
}

package container

import (
	"github.com/EduGoGroup/edugo-api-iam-platform/internal/application/service"
	auditHandler "github.com/EduGoGroup/edugo-api-iam-platform/internal/audit/handler"
	auditRepo "github.com/EduGoGroup/edugo-api-iam-platform/internal/audit/repository"
	auditService "github.com/EduGoGroup/edugo-api-iam-platform/internal/audit/service"
	authHandler "github.com/EduGoGroup/edugo-api-iam-platform/internal/auth/handler"
	authrepo "github.com/EduGoGroup/edugo-api-iam-platform/internal/auth/repository"
	authService "github.com/EduGoGroup/edugo-api-iam-platform/internal/auth/service"
	"github.com/EduGoGroup/edugo-api-iam-platform/internal/config"
	"github.com/EduGoGroup/edugo-api-iam-platform/internal/infrastructure/cache"
	"github.com/EduGoGroup/edugo-api-iam-platform/internal/infrastructure/http/handler"
	pgRepo "github.com/EduGoGroup/edugo-api-iam-platform/internal/infrastructure/persistence/postgres/repository"
	auditpostgres "github.com/EduGoGroup/edugo-shared/audit/postgres"
	"github.com/EduGoGroup/edugo-shared/auth"
	"github.com/EduGoGroup/edugo-shared/logger"
	"github.com/EduGoGroup/edugo-shared/metrics"
	sharedPgRepo "github.com/EduGoGroup/edugo-shared/repository"
	"gorm.io/gorm"
)

// Container is the dependency injection container
type Container struct {
	DB         *gorm.DB
	Logger     logger.Logger
	Metrics    *metrics.Metrics
	JWTManager *auth.JWTManager
	Blacklist  auth.TokenBlacklist

	// Auth
	TokenService  *authService.TokenService
	AuthService   authService.AuthService
	AuthHandler   *authHandler.AuthHandler
	VerifyHandler *authHandler.VerifyHandler

	// Handlers
	RoleHandler         *handler.RoleHandler
	ResourceHandler     *handler.ResourceHandler
	MenuHandler         *handler.MenuHandler
	PermissionHandler   *handler.PermissionHandler
	ScreenConfigHandler *handler.ScreenConfigHandler
	SyncHandler         *handler.SyncHandler
	HealthHandler       *handler.HealthHandler
	AuditHandler        *auditHandler.AuditHandler
}

// NewContainer creates a new container and initializes all dependencies
func NewContainer(db *gorm.DB, log logger.Logger, cfg *config.Config, blacklist auth.TokenBlacklist) *Container {
	c := &Container{
		DB:         db,
		Logger:     log,
		Metrics:    metrics.New("edugo-api-iam-platform"),
		JWTManager: auth.NewJWTManager(cfg.Auth.JWT.Secret, cfg.Auth.JWT.Issuer),
		Blacklist:  blacklist,
	}

	// Audit logger
	auditLogger := auditpostgres.NewPostgresAuditLogger(db, "iam-platform")

	// Shared Repositories (from edugo-shared/repository)
	userRepo := sharedPgRepo.NewPostgresUserRepository(db)
	membershipRepo := sharedPgRepo.NewPostgresMembershipRepository(db)
	schoolRepo := sharedPgRepo.NewPostgresSchoolRepository(db)
	academicUnitRepo := sharedPgRepo.NewPostgresAcademicUnitRepository(db)

	// IAM Repositories (local)
	roleRepo := pgRepo.NewPostgresRoleRepository(db)
	permissionRepo := pgRepo.NewPostgresPermissionRepository(db)
	userRoleRepo := pgRepo.NewPostgresUserRoleRepository(db)
	resourceRepo := pgRepo.NewPostgresResourceRepository(db)
	rolePermRepo := pgRepo.NewPostgresRolePermissionRepository(db)
	screenTemplateRepo := pgRepo.NewPostgresScreenTemplateRepository(db)
	cachedTemplateRepo := cache.NewCachedScreenTemplateRepository(screenTemplateRepo)
	screenInstanceRepo := pgRepo.NewPostgresScreenInstanceRepository(db)
	resourceScreenRepo := pgRepo.NewPostgresResourceScreenRepository(db)
	schoolConceptRepo := pgRepo.NewPostgresSchoolConceptRepository(db)

	// Login attempt repository
	loginAttemptRepo := authrepo.NewPostgresLoginAttemptRepository(db)

	// Auth
	c.TokenService = authService.NewTokenService(c.JWTManager, cfg.Auth.JWT.AccessTokenDuration, cfg.Auth.JWT.RefreshTokenDuration)
	c.AuthService = authService.NewAuthService(userRepo, userRoleRepo, roleRepo, membershipRepo, schoolRepo, academicUnitRepo, c.TokenService, log, auditLogger, loginAttemptRepo, blacklist)
	c.AuthHandler = authHandler.NewAuthHandler(c.AuthService, log)
	c.VerifyHandler = authHandler.NewVerifyHandler(c.TokenService)

	// Services
	roleService := service.NewRoleService(roleRepo, permissionRepo, userRoleRepo, rolePermRepo, log, auditLogger)
	resourceService := service.NewResourceService(resourceRepo, log)
	menuService := service.NewMenuService(resourceRepo, resourceScreenRepo, log)
	permissionService := service.NewPermissionService(permissionRepo, resourceRepo, log, auditLogger)
	screenConfigService := service.NewScreenConfigService(cachedTemplateRepo, screenInstanceRepo, resourceScreenRepo, log)

	// Sync
	syncService := service.NewSyncService(menuService, screenConfigService, c.AuthService, screenInstanceRepo, schoolConceptRepo, log)

	// Audit query
	auditRepository := auditRepo.NewPostgresAuditRepository(db)
	auditQueryService := auditService.NewAuditQueryService(auditRepository)
	c.AuditHandler = auditHandler.NewAuditHandler(auditQueryService)

	// Handlers
	c.RoleHandler = handler.NewRoleHandler(roleService, log)
	c.ResourceHandler = handler.NewResourceHandler(resourceService, log)
	c.MenuHandler = handler.NewMenuHandler(menuService, log)
	c.PermissionHandler = handler.NewPermissionHandler(permissionService, log)
	c.ScreenConfigHandler = handler.NewScreenConfigHandler(screenConfigService, log)
	c.SyncHandler = handler.NewSyncHandler(syncService, log)
	c.HealthHandler = handler.NewHealthHandler(db, "dev")

	return c
}

// Close releases container resources
func (c *Container) Close() error {
	if c.DB != nil {
		sqlDB, err := c.DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

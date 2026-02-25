package service

import (
	"context"

	"github.com/EduGoGroup/edugo-infrastructure/postgres/entities"
	"github.com/EduGoGroup/edugo-shared/logger"
	domainrepo "github.com/EduGoGroup/edugo-api-iam-platform/internal/domain/repository"
	"github.com/google/uuid"
)

// ─── Logger mock ─────────────────────────────────────────────────────────────

type mockLogger struct{}

func (m *mockLogger) Debug(msg string, fields ...interface{})  {}
func (m *mockLogger) Info(msg string, fields ...interface{})   {}
func (m *mockLogger) Warn(msg string, fields ...interface{})   {}
func (m *mockLogger) Error(msg string, fields ...interface{})  {}
func (m *mockLogger) Fatal(msg string, fields ...interface{})  {}
func (m *mockLogger) Sync() error                              { return nil }
func (m *mockLogger) With(fields ...interface{}) logger.Logger { return m }

// ─── RoleRepository mock ─────────────────────────────────────────────────────

type mockRoleRepo struct {
	findByIDFn    func(ctx context.Context, id uuid.UUID) (*entities.Role, error)
	findAllFn     func(ctx context.Context) ([]*entities.Role, error)
	findByScopeFn func(ctx context.Context, scope string) ([]*entities.Role, error)
}

func (m *mockRoleRepo) FindByID(ctx context.Context, id uuid.UUID) (*entities.Role, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockRoleRepo) FindAll(ctx context.Context) ([]*entities.Role, error) {
	if m.findAllFn != nil {
		return m.findAllFn(ctx)
	}
	return nil, nil
}
func (m *mockRoleRepo) FindByScope(ctx context.Context, scope string) ([]*entities.Role, error) {
	if m.findByScopeFn != nil {
		return m.findByScopeFn(ctx, scope)
	}
	return nil, nil
}

// ─── PermissionRepository mock ───────────────────────────────────────────────

type mockPermissionRepo struct {
	findByIDFn   func(ctx context.Context, id uuid.UUID) (*entities.Permission, error)
	findAllFn    func(ctx context.Context) ([]*entities.Permission, error)
	findByRoleFn func(ctx context.Context, roleID uuid.UUID) ([]*entities.Permission, error)
}

func (m *mockPermissionRepo) FindByID(ctx context.Context, id uuid.UUID) (*entities.Permission, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockPermissionRepo) FindAll(ctx context.Context) ([]*entities.Permission, error) {
	if m.findAllFn != nil {
		return m.findAllFn(ctx)
	}
	return nil, nil
}
func (m *mockPermissionRepo) FindByRole(ctx context.Context, roleID uuid.UUID) ([]*entities.Permission, error) {
	if m.findByRoleFn != nil {
		return m.findByRoleFn(ctx, roleID)
	}
	return nil, nil
}

// ─── UserRoleRepository mock ─────────────────────────────────────────────────

type mockUserRoleRepo struct {
	findByUserFn          func(ctx context.Context, userID uuid.UUID) ([]*entities.UserRole, error)
	findByUserInContextFn func(ctx context.Context, userID uuid.UUID, schoolID *uuid.UUID, unitID *uuid.UUID) ([]*entities.UserRole, error)
	grantFn               func(ctx context.Context, userRole *entities.UserRole) error
	revokeFn              func(ctx context.Context, id uuid.UUID) error
	revokeByUserAndRoleFn func(ctx context.Context, userID, roleID uuid.UUID, schoolID, unitID *uuid.UUID) error
	userHasRoleFn         func(ctx context.Context, userID, roleID uuid.UUID, schoolID, unitID *uuid.UUID) (bool, error)
	getUserPermissionsFn  func(ctx context.Context, userID uuid.UUID, schoolID, unitID *uuid.UUID) ([]string, error)
}

func (m *mockUserRoleRepo) FindByUser(ctx context.Context, userID uuid.UUID) ([]*entities.UserRole, error) {
	if m.findByUserFn != nil {
		return m.findByUserFn(ctx, userID)
	}
	return nil, nil
}
func (m *mockUserRoleRepo) FindByUserInContext(ctx context.Context, userID uuid.UUID, schoolID *uuid.UUID, unitID *uuid.UUID) ([]*entities.UserRole, error) {
	if m.findByUserInContextFn != nil {
		return m.findByUserInContextFn(ctx, userID, schoolID, unitID)
	}
	return nil, nil
}
func (m *mockUserRoleRepo) Grant(ctx context.Context, userRole *entities.UserRole) error {
	if m.grantFn != nil {
		return m.grantFn(ctx, userRole)
	}
	return nil
}
func (m *mockUserRoleRepo) Revoke(ctx context.Context, id uuid.UUID) error {
	if m.revokeFn != nil {
		return m.revokeFn(ctx, id)
	}
	return nil
}
func (m *mockUserRoleRepo) RevokeByUserAndRole(ctx context.Context, userID, roleID uuid.UUID, schoolID, unitID *uuid.UUID) error {
	if m.revokeByUserAndRoleFn != nil {
		return m.revokeByUserAndRoleFn(ctx, userID, roleID, schoolID, unitID)
	}
	return nil
}
func (m *mockUserRoleRepo) UserHasRole(ctx context.Context, userID, roleID uuid.UUID, schoolID, unitID *uuid.UUID) (bool, error) {
	if m.userHasRoleFn != nil {
		return m.userHasRoleFn(ctx, userID, roleID, schoolID, unitID)
	}
	return false, nil
}
func (m *mockUserRoleRepo) GetUserPermissions(ctx context.Context, userID uuid.UUID, schoolID, unitID *uuid.UUID) ([]string, error) {
	if m.getUserPermissionsFn != nil {
		return m.getUserPermissionsFn(ctx, userID, schoolID, unitID)
	}
	return nil, nil
}

// ─── ResourceRepository mock ─────────────────────────────────────────────────

type mockResourceRepo struct {
	findAllFn         func(ctx context.Context) ([]*entities.Resource, error)
	findByIDFn        func(ctx context.Context, id uuid.UUID) (*entities.Resource, error)
	findMenuVisibleFn func(ctx context.Context) ([]*entities.Resource, error)
	createFn          func(ctx context.Context, resource *entities.Resource) error
	updateFn          func(ctx context.Context, resource *entities.Resource) error
}

func (m *mockResourceRepo) FindAll(ctx context.Context) ([]*entities.Resource, error) {
	if m.findAllFn != nil {
		return m.findAllFn(ctx)
	}
	return nil, nil
}
func (m *mockResourceRepo) FindByID(ctx context.Context, id uuid.UUID) (*entities.Resource, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockResourceRepo) FindMenuVisible(ctx context.Context) ([]*entities.Resource, error) {
	if m.findMenuVisibleFn != nil {
		return m.findMenuVisibleFn(ctx)
	}
	return nil, nil
}
func (m *mockResourceRepo) Create(ctx context.Context, resource *entities.Resource) error {
	if m.createFn != nil {
		return m.createFn(ctx, resource)
	}
	return nil
}
func (m *mockResourceRepo) Update(ctx context.Context, resource *entities.Resource) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, resource)
	}
	return nil
}

// ─── ScreenTemplateRepository mock ───────────────────────────────────────────

type mockScreenTemplateRepo struct {
	createFn  func(ctx context.Context, template *entities.ScreenTemplate) error
	getByIDFn func(ctx context.Context, id uuid.UUID) (*entities.ScreenTemplate, error)
	listFn    func(ctx context.Context, filter domainrepo.ScreenTemplateFilter) ([]*entities.ScreenTemplate, int, error)
	updateFn  func(ctx context.Context, template *entities.ScreenTemplate) error
	deleteFn  func(ctx context.Context, id uuid.UUID) error
}

func (m *mockScreenTemplateRepo) Create(ctx context.Context, template *entities.ScreenTemplate) error {
	if m.createFn != nil {
		return m.createFn(ctx, template)
	}
	return nil
}
func (m *mockScreenTemplateRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.ScreenTemplate, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockScreenTemplateRepo) List(ctx context.Context, filter domainrepo.ScreenTemplateFilter) ([]*entities.ScreenTemplate, int, error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return nil, 0, nil
}
func (m *mockScreenTemplateRepo) Update(ctx context.Context, template *entities.ScreenTemplate) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, template)
	}
	return nil
}
func (m *mockScreenTemplateRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

// ─── ScreenInstanceRepository mock ───────────────────────────────────────────

type mockScreenInstanceRepo struct {
	createFn         func(ctx context.Context, instance *entities.ScreenInstance) error
	getByIDFn        func(ctx context.Context, id uuid.UUID) (*entities.ScreenInstance, error)
	getByScreenKeyFn func(ctx context.Context, key string) (*entities.ScreenInstance, error)
	listFn           func(ctx context.Context, filter domainrepo.ScreenInstanceFilter) ([]*entities.ScreenInstance, int, error)
	updateFn         func(ctx context.Context, instance *entities.ScreenInstance) error
	deleteFn         func(ctx context.Context, id uuid.UUID) error
}

func (m *mockScreenInstanceRepo) Create(ctx context.Context, instance *entities.ScreenInstance) error {
	if m.createFn != nil {
		return m.createFn(ctx, instance)
	}
	return nil
}
func (m *mockScreenInstanceRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.ScreenInstance, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockScreenInstanceRepo) GetByScreenKey(ctx context.Context, key string) (*entities.ScreenInstance, error) {
	if m.getByScreenKeyFn != nil {
		return m.getByScreenKeyFn(ctx, key)
	}
	return nil, nil
}
func (m *mockScreenInstanceRepo) List(ctx context.Context, filter domainrepo.ScreenInstanceFilter) ([]*entities.ScreenInstance, int, error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return nil, 0, nil
}
func (m *mockScreenInstanceRepo) Update(ctx context.Context, instance *entities.ScreenInstance) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, instance)
	}
	return nil
}
func (m *mockScreenInstanceRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

// ─── ResourceScreenRepository mock ───────────────────────────────────────────

type mockResourceScreenRepo struct {
	createFn           func(ctx context.Context, rs *entities.ResourceScreen) error
	getByResourceIDFn  func(ctx context.Context, resourceID uuid.UUID) ([]*entities.ResourceScreen, error)
	getByResourceKeyFn func(ctx context.Context, key string) ([]*entities.ResourceScreen, error)
	deleteFn           func(ctx context.Context, id uuid.UUID) error
}

func (m *mockResourceScreenRepo) Create(ctx context.Context, rs *entities.ResourceScreen) error {
	if m.createFn != nil {
		return m.createFn(ctx, rs)
	}
	return nil
}
func (m *mockResourceScreenRepo) GetByResourceID(ctx context.Context, resourceID uuid.UUID) ([]*entities.ResourceScreen, error) {
	if m.getByResourceIDFn != nil {
		return m.getByResourceIDFn(ctx, resourceID)
	}
	return nil, nil
}
func (m *mockResourceScreenRepo) GetByResourceKey(ctx context.Context, key string) ([]*entities.ResourceScreen, error) {
	if m.getByResourceKeyFn != nil {
		return m.getByResourceKeyFn(ctx, key)
	}
	return nil, nil
}
func (m *mockResourceScreenRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

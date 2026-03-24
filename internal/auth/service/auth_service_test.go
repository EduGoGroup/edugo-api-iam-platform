package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/EduGoGroup/edugo-api-iam-platform/internal/auth/dto"
	"github.com/EduGoGroup/edugo-api-iam-platform/internal/auth/model"
	"github.com/EduGoGroup/edugo-api-iam-platform/internal/domain/repository"
	"github.com/EduGoGroup/edugo-infrastructure/postgres/entities"
	"github.com/EduGoGroup/edugo-shared/audit"
	"github.com/EduGoGroup/edugo-shared/auth"
	"github.com/EduGoGroup/edugo-shared/logger"
	sharedrepo "github.com/EduGoGroup/edugo-shared/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Mocks ──────────────────────────────────────────────────────────────────

type mockAuditLog struct {
	logFn func(ctx context.Context, event audit.AuditEvent) error
}

func (m *mockAuditLog) Log(ctx context.Context, event audit.AuditEvent) error {
	if m.logFn != nil {
		return m.logFn(ctx, event)
	}
	return nil
}

type mockLog struct{}

func (m *mockLog) Debug(_ string, _ ...interface{})    {}
func (m *mockLog) Info(_ string, _ ...interface{})     {}
func (m *mockLog) Warn(_ string, _ ...interface{})     {}
func (m *mockLog) Error(_ string, _ ...interface{})    {}
func (m *mockLog) Fatal(_ string, _ ...interface{})    {}
func (m *mockLog) Sync() error                         { return nil }
func (m *mockLog) With(_ ...interface{}) logger.Logger { return m }

type mockUserRepo struct {
	findByEmailFn func(ctx context.Context, email string) (*entities.User, error)
	findByIDFn    func(ctx context.Context, id uuid.UUID) (*entities.User, error)
	updateFn      func(ctx context.Context, user *entities.User) error
	createFn      func(ctx context.Context, user *entities.User) error
	existsByEmail func(ctx context.Context, email string) (bool, error)
	deleteFn      func(ctx context.Context, id uuid.UUID) error
	listFn        func(ctx context.Context, filters sharedrepo.ListFilters) ([]*entities.User, int64, error)
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*entities.User, error) {
	if m.findByEmailFn != nil {
		return m.findByEmailFn(ctx, email)
	}
	return nil, nil
}
func (m *mockUserRepo) FindByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockUserRepo) Update(ctx context.Context, user *entities.User) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, user)
	}
	return nil
}
func (m *mockUserRepo) Create(ctx context.Context, user *entities.User) error {
	if m.createFn != nil {
		return m.createFn(ctx, user)
	}
	return nil
}
func (m *mockUserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	if m.existsByEmail != nil {
		return m.existsByEmail(ctx, email)
	}
	return false, nil
}
func (m *mockUserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}
func (m *mockUserRepo) List(ctx context.Context, filters sharedrepo.ListFilters) ([]*entities.User, int64, error) {
	if m.listFn != nil {
		return m.listFn(ctx, filters)
	}
	return nil, 0, nil
}

type mockUserRoleRepo struct {
	findByUserFn          func(ctx context.Context, userID uuid.UUID) ([]*entities.UserRole, error)
	findByUserInContextFn func(ctx context.Context, userID uuid.UUID, schoolID *uuid.UUID, unitID *uuid.UUID) ([]*entities.UserRole, error)
	getUserPermissionsFn  func(ctx context.Context, userID uuid.UUID, schoolID, unitID *uuid.UUID) ([]string, error)
	grantFn               func(ctx context.Context, userRole *entities.UserRole) error
	revokeFn              func(ctx context.Context, id uuid.UUID) error
	revokeByUserAndRoleFn func(ctx context.Context, userID, roleID uuid.UUID, schoolID, unitID *uuid.UUID) error
	userHasRoleFn         func(ctx context.Context, userID, roleID uuid.UUID, schoolID, unitID *uuid.UUID) (bool, error)
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
func (m *mockUserRoleRepo) GetUserPermissions(ctx context.Context, userID uuid.UUID, schoolID, unitID *uuid.UUID) ([]string, error) {
	if m.getUserPermissionsFn != nil {
		return m.getUserPermissionsFn(ctx, userID, schoolID, unitID)
	}
	return []string{}, nil
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

type mockRoleRepository struct {
	findByIDFn func(ctx context.Context, id uuid.UUID) (*entities.Role, error)
}

func (m *mockRoleRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.Role, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockRoleRepository) FindAll(_ context.Context, _ sharedrepo.ListFilters) ([]*entities.Role, int, error) {
	return nil, 0, nil
}
func (m *mockRoleRepository) FindByScope(_ context.Context, _ string, _ sharedrepo.ListFilters) ([]*entities.Role, int, error) {
	return nil, 0, nil
}
func (m *mockRoleRepository) Create(_ context.Context, _ *entities.Role) error { return nil }
func (m *mockRoleRepository) Update(_ context.Context, _ *entities.Role) error { return nil }
func (m *mockRoleRepository) SoftDelete(_ context.Context, _ uuid.UUID) error  { return nil }
func (m *mockRoleRepository) HasActiveUserRoles(_ context.Context, _ uuid.UUID) (bool, error) {
	return false, nil
}

type mockMembershipRepo struct {
	findByUserFn          func(ctx context.Context, userID uuid.UUID, filters sharedrepo.ListFilters) ([]*entities.Membership, int64, error)
	findByUserAndSchoolFn func(ctx context.Context, userID, schoolID uuid.UUID) (*entities.Membership, error)
}

func (m *mockMembershipRepo) FindByUser(ctx context.Context, userID uuid.UUID, filters sharedrepo.ListFilters) ([]*entities.Membership, int64, error) {
	if m.findByUserFn != nil {
		return m.findByUserFn(ctx, userID, filters)
	}
	return nil, 0, nil
}
func (m *mockMembershipRepo) FindByUserAndSchool(ctx context.Context, userID, schoolID uuid.UUID) (*entities.Membership, error) {
	if m.findByUserAndSchoolFn != nil {
		return m.findByUserAndSchoolFn(ctx, userID, schoolID)
	}
	return nil, sharedrepo.ErrNotFound
}
func (m *mockMembershipRepo) Create(_ context.Context, _ *entities.Membership) error { return nil }
func (m *mockMembershipRepo) FindByID(_ context.Context, _ uuid.UUID) (*entities.Membership, error) {
	return nil, nil
}
func (m *mockMembershipRepo) FindByUnit(_ context.Context, _ uuid.UUID, _ sharedrepo.ListFilters) ([]*entities.Membership, int64, error) {
	return nil, 0, nil
}
func (m *mockMembershipRepo) FindByUnitAndRole(_ context.Context, _ uuid.UUID, _ string, _ bool, _ sharedrepo.ListFilters) ([]*entities.Membership, int64, error) {
	return nil, 0, nil
}
func (m *mockMembershipRepo) Update(_ context.Context, _ *entities.Membership) error { return nil }
func (m *mockMembershipRepo) Delete(_ context.Context, _ uuid.UUID) error            { return nil }

type mockSchoolRepo struct {
	findByIDFn func(ctx context.Context, id uuid.UUID) (*entities.School, error)
}

func (m *mockSchoolRepo) FindByID(ctx context.Context, id uuid.UUID) (*entities.School, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockSchoolRepo) Create(_ context.Context, _ *entities.School) error { return nil }
func (m *mockSchoolRepo) FindByCode(_ context.Context, _ string) (*entities.School, error) {
	return nil, nil
}
func (m *mockSchoolRepo) Update(_ context.Context, _ *entities.School) error { return nil }
func (m *mockSchoolRepo) Delete(_ context.Context, _ uuid.UUID) error        { return nil }
func (m *mockSchoolRepo) List(_ context.Context, _ sharedrepo.ListFilters) ([]*entities.School, int64, error) {
	return nil, 0, nil
}
func (m *mockSchoolRepo) ExistsByCode(_ context.Context, _ string) (bool, error) {
	return false, nil
}

type mockAcademicUnitRepo struct {
	findByIDFn       func(ctx context.Context, id uuid.UUID) (*entities.AcademicUnit, error)
	findBySchoolIDFn func(ctx context.Context, schoolID uuid.UUID, filters sharedrepo.ListFilters) ([]*entities.AcademicUnit, int64, error)
}

func (m *mockAcademicUnitRepo) FindByID(ctx context.Context, id uuid.UUID) (*entities.AcademicUnit, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockAcademicUnitRepo) FindBySchoolID(ctx context.Context, schoolID uuid.UUID, filters sharedrepo.ListFilters) ([]*entities.AcademicUnit, int64, error) {
	if m.findBySchoolIDFn != nil {
		return m.findBySchoolIDFn(ctx, schoolID, filters)
	}
	return nil, 0, nil
}

type mockLoginAttemptRepo struct {
	createFn           func(ctx context.Context, attempt *model.LoginAttempt) error
	countFailedSinceFn func(ctx context.Context, identifier string, since time.Time) (int64, error)
}

func (m *mockLoginAttemptRepo) Create(ctx context.Context, attempt *model.LoginAttempt) error {
	if m.createFn != nil {
		return m.createFn(ctx, attempt)
	}
	return nil
}
func (m *mockLoginAttemptRepo) CountFailedSince(ctx context.Context, identifier string, since time.Time) (int64, error) {
	if m.countFailedSinceFn != nil {
		return m.countFailedSinceFn(ctx, identifier, since)
	}
	return 0, nil
}

// ─── Helpers ────────────────────────────────────────────────────────────────

var _ repository.UserRoleRepository = (*mockUserRoleRepo)(nil)
var _ repository.RoleRepository = (*mockRoleRepository)(nil)

func newTestTokenService() *TokenService {
	jwtManager := auth.NewJWTManager("test-secret-key-for-unit-tests-only", "test-issuer")
	return NewTokenService(jwtManager, 15*time.Minute, 7*24*time.Hour)
}

func newTestUser() *entities.User {
	hash, _ := auth.HashPassword("correct-password")
	return &entities.User{
		ID:           uuid.MustParse("00000000-0000-0000-0000-000000000099"),
		Email:        "test@edugo.test",
		PasswordHash: hash,
		FirstName:    "Test",
		LastName:     "User",
		IsActive:     true,
	}
}

func newTestRole(name string) *entities.Role {
	return &entities.Role{
		ID:   uuid.New(),
		Name: name,
	}
}

// ─── Tests ──────────────────────────────────────────────────────────────────

func TestLogin_Success_GlobalRole(t *testing.T) {
	user := newTestUser()
	role := newTestRole("super_admin")

	svc := NewAuthService(
		&mockUserRepo{
			findByEmailFn: func(_ context.Context, _ string) (*entities.User, error) {
				return user, nil
			},
		},
		&mockUserRoleRepo{
			findByUserInContextFn: func(_ context.Context, _ uuid.UUID, schoolID *uuid.UUID, _ *uuid.UUID) ([]*entities.UserRole, error) {
				if schoolID == nil {
					return []*entities.UserRole{{RoleID: role.ID, UserID: user.ID}}, nil
				}
				return nil, nil
			},
		},
		&mockRoleRepository{
			findByIDFn: func(_ context.Context, _ uuid.UUID) (*entities.Role, error) {
				return role, nil
			},
		},
		&mockMembershipRepo{},
		&mockSchoolRepo{},
		&mockAcademicUnitRepo{},
		newTestTokenService(),
		&mockLog{},
		&mockAuditLog{},
		&mockLoginAttemptRepo{},
	)

	resp, err := svc.Login(context.Background(), "test@edugo.test", "correct-password", "127.0.0.1", "test-agent")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, "Bearer", resp.TokenType)
	require.NotNil(t, resp.ActiveContext)
	assert.Equal(t, "super_admin", resp.ActiveContext.RoleName)
}

func TestLogin_Success_SchoolRole(t *testing.T) {
	user := newTestUser()
	role := newTestRole("teacher")
	schoolID := uuid.New()

	svc := NewAuthService(
		&mockUserRepo{
			findByEmailFn: func(_ context.Context, _ string) (*entities.User, error) {
				return user, nil
			},
		},
		&mockUserRoleRepo{
			findByUserInContextFn: func(_ context.Context, _ uuid.UUID, sid *uuid.UUID, _ *uuid.UUID) ([]*entities.UserRole, error) {
				if sid == nil {
					return nil, nil // no global role
				}
				return []*entities.UserRole{{RoleID: role.ID, UserID: user.ID, SchoolID: &schoolID}}, nil
			},
		},
		&mockRoleRepository{
			findByIDFn: func(_ context.Context, _ uuid.UUID) (*entities.Role, error) {
				return role, nil
			},
		},
		&mockMembershipRepo{
			findByUserFn: func(_ context.Context, _ uuid.UUID, _ sharedrepo.ListFilters) ([]*entities.Membership, int64, error) {
				return []*entities.Membership{{SchoolID: schoolID, IsActive: true}}, 1, nil
			},
		},
		&mockSchoolRepo{
			findByIDFn: func(_ context.Context, _ uuid.UUID) (*entities.School, error) {
				return &entities.School{ID: schoolID, Name: "Test School"}, nil
			},
		},
		&mockAcademicUnitRepo{},
		newTestTokenService(),
		&mockLog{},
		&mockAuditLog{},
		&mockLoginAttemptRepo{},
	)

	resp, err := svc.Login(context.Background(), "test@edugo.test", "correct-password", "127.0.0.1", "")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "teacher", resp.ActiveContext.RoleName)
	assert.Equal(t, schoolID.String(), resp.ActiveContext.SchoolID)
	assert.Len(t, resp.Schools, 1)
}

func TestLogin_InvalidPassword(t *testing.T) {
	user := newTestUser()

	svc := NewAuthService(
		&mockUserRepo{
			findByEmailFn: func(_ context.Context, _ string) (*entities.User, error) {
				return user, nil
			},
		},
		&mockUserRoleRepo{},
		&mockRoleRepository{},
		&mockMembershipRepo{},
		&mockSchoolRepo{},
		&mockAcademicUnitRepo{},
		newTestTokenService(),
		&mockLog{},
		&mockAuditLog{},
		&mockLoginAttemptRepo{},
	)

	resp, err := svc.Login(context.Background(), "test@edugo.test", "wrong-password", "127.0.0.1", "")
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestLogin_UserNotFound(t *testing.T) {
	svc := NewAuthService(
		&mockUserRepo{
			findByEmailFn: func(_ context.Context, _ string) (*entities.User, error) {
				return nil, sharedrepo.ErrNotFound
			},
		},
		&mockUserRoleRepo{},
		&mockRoleRepository{},
		&mockMembershipRepo{},
		&mockSchoolRepo{},
		&mockAcademicUnitRepo{},
		newTestTokenService(),
		&mockLog{},
		&mockAuditLog{},
		&mockLoginAttemptRepo{},
	)

	resp, err := svc.Login(context.Background(), "nobody@test.com", "password", "127.0.0.1", "")
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestLogin_InactiveUser(t *testing.T) {
	user := newTestUser()
	user.IsActive = false

	svc := NewAuthService(
		&mockUserRepo{
			findByEmailFn: func(_ context.Context, _ string) (*entities.User, error) {
				return user, nil
			},
		},
		&mockUserRoleRepo{},
		&mockRoleRepository{},
		&mockMembershipRepo{},
		&mockSchoolRepo{},
		&mockAcademicUnitRepo{},
		newTestTokenService(),
		&mockLog{},
		&mockAuditLog{},
		&mockLoginAttemptRepo{},
	)

	resp, err := svc.Login(context.Background(), "test@edugo.test", "correct-password", "127.0.0.1", "")
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrUserInactive)
}

func TestLogin_RateLimited(t *testing.T) {
	user := newTestUser()

	svc := NewAuthService(
		&mockUserRepo{
			findByEmailFn: func(_ context.Context, _ string) (*entities.User, error) {
				return user, nil
			},
		},
		&mockUserRoleRepo{},
		&mockRoleRepository{},
		&mockMembershipRepo{},
		&mockSchoolRepo{},
		&mockAcademicUnitRepo{},
		newTestTokenService(),
		&mockLog{},
		&mockAuditLog{},
		&mockLoginAttemptRepo{
			countFailedSinceFn: func(_ context.Context, _ string, _ time.Time) (int64, error) {
				return 10, nil // over threshold of 5
			},
		},
	)

	resp, err := svc.Login(context.Background(), "test@edugo.test", "correct-password", "127.0.0.1", "")
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrTooManyLoginAttempts)
}

func TestLogin_NoRoles(t *testing.T) {
	user := newTestUser()

	svc := NewAuthService(
		&mockUserRepo{
			findByEmailFn: func(_ context.Context, _ string) (*entities.User, error) {
				return user, nil
			},
		},
		&mockUserRoleRepo{
			findByUserInContextFn: func(_ context.Context, _ uuid.UUID, _ *uuid.UUID, _ *uuid.UUID) ([]*entities.UserRole, error) {
				return nil, nil // no roles
			},
		},
		&mockRoleRepository{},
		&mockMembershipRepo{},
		&mockSchoolRepo{},
		&mockAcademicUnitRepo{},
		newTestTokenService(),
		&mockLog{},
		&mockAuditLog{},
		&mockLoginAttemptRepo{},
	)

	resp, err := svc.Login(context.Background(), "test@edugo.test", "correct-password", "127.0.0.1", "")
	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no assigned roles")
}

func TestLogin_AuditLogRecorded(t *testing.T) {
	user := newTestUser()
	role := newTestRole("super_admin")
	var auditAction string

	svc := NewAuthService(
		&mockUserRepo{
			findByEmailFn: func(_ context.Context, _ string) (*entities.User, error) {
				return user, nil
			},
		},
		&mockUserRoleRepo{
			findByUserInContextFn: func(_ context.Context, _ uuid.UUID, schoolID *uuid.UUID, _ *uuid.UUID) ([]*entities.UserRole, error) {
				if schoolID == nil {
					return []*entities.UserRole{{RoleID: role.ID, UserID: user.ID}}, nil
				}
				return nil, nil
			},
		},
		&mockRoleRepository{
			findByIDFn: func(_ context.Context, _ uuid.UUID) (*entities.Role, error) {
				return role, nil
			},
		},
		&mockMembershipRepo{},
		&mockSchoolRepo{},
		&mockAcademicUnitRepo{},
		newTestTokenService(),
		&mockLog{},
		&mockAuditLog{
			logFn: func(_ context.Context, event audit.AuditEvent) error {
				if event.Action == "login" {
					auditAction = event.Action
				}
				return nil
			},
		},
		&mockLoginAttemptRepo{},
	)

	resp, err := svc.Login(context.Background(), "test@edugo.test", "correct-password", "127.0.0.1", "")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "login", auditAction, "audit log should record login action synchronously")
}

func TestLogin_EmailNormalized(t *testing.T) {
	user := newTestUser()
	role := newTestRole("super_admin")
	var receivedEmail string

	svc := NewAuthService(
		&mockUserRepo{
			findByEmailFn: func(_ context.Context, email string) (*entities.User, error) {
				receivedEmail = email
				return user, nil
			},
		},
		&mockUserRoleRepo{
			findByUserInContextFn: func(_ context.Context, _ uuid.UUID, schoolID *uuid.UUID, _ *uuid.UUID) ([]*entities.UserRole, error) {
				if schoolID == nil {
					return []*entities.UserRole{{RoleID: role.ID, UserID: user.ID}}, nil
				}
				return nil, nil
			},
		},
		&mockRoleRepository{
			findByIDFn: func(_ context.Context, _ uuid.UUID) (*entities.Role, error) {
				return role, nil
			},
		},
		&mockMembershipRepo{},
		&mockSchoolRepo{},
		&mockAcademicUnitRepo{},
		newTestTokenService(),
		&mockLog{},
		&mockAuditLog{},
		&mockLoginAttemptRepo{},
	)

	_, err := svc.Login(context.Background(), "  TEST@Edugo.Test  ", "correct-password", "", "")
	require.NoError(t, err)
	assert.Equal(t, "test@edugo.test", receivedEmail)
}

func TestSwitchContext_GlobalRole(t *testing.T) {
	user := newTestUser()
	role := newTestRole("super_admin")
	schoolID := uuid.New()

	svc := NewAuthService(
		&mockUserRepo{
			findByIDFn: func(_ context.Context, _ uuid.UUID) (*entities.User, error) {
				return user, nil
			},
		},
		&mockUserRoleRepo{
			findByUserInContextFn: func(_ context.Context, _ uuid.UUID, sid *uuid.UUID, _ *uuid.UUID) ([]*entities.UserRole, error) {
				if sid == nil {
					return []*entities.UserRole{{RoleID: role.ID, UserID: user.ID}}, nil
				}
				return nil, nil
			},
		},
		&mockRoleRepository{
			findByIDFn: func(_ context.Context, _ uuid.UUID) (*entities.Role, error) {
				return role, nil
			},
		},
		&mockMembershipRepo{},
		&mockSchoolRepo{
			findByIDFn: func(_ context.Context, _ uuid.UUID) (*entities.School, error) {
				return &entities.School{ID: schoolID, Name: "Target School"}, nil
			},
		},
		&mockAcademicUnitRepo{},
		newTestTokenService(),
		&mockLog{},
		&mockAuditLog{},
		&mockLoginAttemptRepo{},
	)

	resp, err := svc.SwitchContext(context.Background(), user.ID.String(), schoolID.String(), "")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "super_admin", resp.Context.Role)
	assert.Equal(t, "Target School", resp.Context.SchoolName)
}

func TestSwitchContext_NoMembershipNoGlobalRole(t *testing.T) {
	user := newTestUser()

	svc := NewAuthService(
		&mockUserRepo{
			findByIDFn: func(_ context.Context, _ uuid.UUID) (*entities.User, error) {
				return user, nil
			},
		},
		&mockUserRoleRepo{
			findByUserInContextFn: func(_ context.Context, _ uuid.UUID, _ *uuid.UUID, _ *uuid.UUID) ([]*entities.UserRole, error) {
				return nil, nil
			},
		},
		&mockRoleRepository{},
		&mockMembershipRepo{},
		&mockSchoolRepo{},
		&mockAcademicUnitRepo{},
		newTestTokenService(),
		&mockLog{},
		&mockAuditLog{},
		&mockLoginAttemptRepo{},
	)

	resp, err := svc.SwitchContext(context.Background(), user.ID.String(), uuid.New().String(), "")
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrNoMembership)
}

// ─── GetSchoolUnits Tests ──────────────────────────────────────────────────

func TestGetSchoolUnits_InvalidSchoolID(t *testing.T) {
	svc := NewAuthService(
		&mockUserRepo{},
		&mockUserRoleRepo{},
		&mockRoleRepository{},
		&mockMembershipRepo{},
		&mockSchoolRepo{},
		&mockAcademicUnitRepo{},
		newTestTokenService(),
		&mockLog{},
		&mockAuditLog{},
		&mockLoginAttemptRepo{},
	)

	resp, err := svc.GetSchoolUnits(context.Background(), "not-a-uuid")
	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid school_id")
}

func TestGetSchoolUnits_Success(t *testing.T) {
	schoolID := uuid.New()
	unit1ID := uuid.New()
	unit2ID := uuid.New()

	svc := NewAuthService(
		&mockUserRepo{},
		&mockUserRoleRepo{},
		&mockRoleRepository{},
		&mockMembershipRepo{},
		&mockSchoolRepo{},
		&mockAcademicUnitRepo{
			findBySchoolIDFn: func(_ context.Context, sid uuid.UUID, _ sharedrepo.ListFilters) ([]*entities.AcademicUnit, int64, error) {
				assert.Equal(t, schoolID, sid)
				return []*entities.AcademicUnit{
					{ID: unit1ID, SchoolID: schoolID, Name: "Primary", Type: "sede"},
					{ID: unit2ID, SchoolID: schoolID, Name: "Secondary", Type: "sede"},
				}, 2, nil
			},
		},
		newTestTokenService(),
		&mockLog{},
		&mockAuditLog{},
		&mockLoginAttemptRepo{},
	)

	resp, err := svc.GetSchoolUnits(context.Background(), schoolID.String())
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, int64(2), resp.Total)
	assert.Len(t, resp.Units, 2)
	assert.Equal(t, "Primary", resp.Units[0].Name)
	assert.Equal(t, "Secondary", resp.Units[1].Name)
}

func TestGetSchoolUnits_RepoError(t *testing.T) {
	schoolID := uuid.New()

	svc := NewAuthService(
		&mockUserRepo{},
		&mockUserRoleRepo{},
		&mockRoleRepository{},
		&mockMembershipRepo{},
		&mockSchoolRepo{},
		&mockAcademicUnitRepo{
			findBySchoolIDFn: func(_ context.Context, _ uuid.UUID, _ sharedrepo.ListFilters) ([]*entities.AcademicUnit, int64, error) {
				return nil, 0, fmt.Errorf("database connection error")
			},
		},
		newTestTokenService(),
		&mockLog{},
		&mockAuditLog{},
		&mockLoginAttemptRepo{},
	)

	resp, err := svc.GetSchoolUnits(context.Background(), schoolID.String())
	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error fetching units")
}

// ─── GetAvailableContexts Tests ────────────────────────────────────────────

func TestGetAvailableContexts_MembershipWithUnit(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000099")
	schoolID := uuid.New()
	unitID := uuid.New()
	roleID := uuid.New()

	svc := NewAuthService(
		&mockUserRepo{},
		&mockUserRoleRepo{
			findByUserFn: func(_ context.Context, _ uuid.UUID) ([]*entities.UserRole, error) {
				return []*entities.UserRole{
					{RoleID: roleID, UserID: userID, SchoolID: &schoolID},
				}, nil
			},
			getUserPermissionsFn: func(_ context.Context, _ uuid.UUID, _ *uuid.UUID, _ *uuid.UUID) ([]string, error) {
				return []string{"materials:read", "assessments:read"}, nil
			},
		},
		&mockRoleRepository{
			findByIDFn: func(_ context.Context, _ uuid.UUID) (*entities.Role, error) {
				return &entities.Role{ID: roleID, Name: "teacher"}, nil
			},
		},
		&mockMembershipRepo{
			findByUserFn: func(_ context.Context, _ uuid.UUID, _ sharedrepo.ListFilters) ([]*entities.Membership, int64, error) {
				return []*entities.Membership{
					{SchoolID: schoolID, AcademicUnitID: &unitID, IsActive: true},
				}, 1, nil
			},
		},
		&mockSchoolRepo{
			findByIDFn: func(_ context.Context, _ uuid.UUID) (*entities.School, error) {
				return &entities.School{ID: schoolID, Name: "Test School"}, nil
			},
		},
		&mockAcademicUnitRepo{
			findByIDFn: func(_ context.Context, _ uuid.UUID) (*entities.AcademicUnit, error) {
				return &entities.AcademicUnit{ID: unitID, SchoolID: schoolID, Name: "Primary"}, nil
			},
		},
		newTestTokenService(),
		&mockLog{},
		&mockAuditLog{},
		&mockLoginAttemptRepo{},
	)

	resp, err := svc.GetAvailableContexts(context.Background(), userID.String(), nil)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp.Available)

	// Find the membership-based context with the unit
	var found *dto.UserContextDTO
	for _, ctx := range resp.Available {
		if ctx.AcademicUnitID == unitID.String() {
			found = ctx
			break
		}
	}
	require.NotNil(t, found, "should have a context with the academic unit")
	assert.Equal(t, "teacher", found.RoleName)
	assert.Equal(t, schoolID.String(), found.SchoolID)
	assert.Equal(t, "Primary", found.AcademicUnitName)
	assert.NotEmpty(t, found.Permissions, "membership-based context should have permissions populated")
}

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/EduGoGroup/edugo-api-iam-platform/internal/application/dto"
	"github.com/EduGoGroup/edugo-infrastructure/postgres/entities"
	sharedErrors "github.com/EduGoGroup/edugo-shared/common/errors"
	sharedrepo "github.com/EduGoGroup/edugo-shared/repository"
	"github.com/google/uuid"
)

func newRoleService(roleRepo *mockRoleRepo, permRepo *mockPermissionRepo, urRepo *mockUserRoleRepo) RoleService {
	return NewRoleService(roleRepo, permRepo, urRepo, &mockLogger{})
}

// ─── GetRoles ────────────────────────────────────────────────────────────────

func TestRoleService_GetRoles(t *testing.T) {
	ctx := context.Background()

	t.Run("lista todos los roles cuando scope está vacío", func(t *testing.T) {
		roles := []*entities.Role{
			{ID: uuid.New(), Name: "admin", DisplayName: "Admin", Scope: "platform", IsActive: true},
			{ID: uuid.New(), Name: "teacher", DisplayName: "Teacher", Scope: "school", IsActive: true},
		}
		roleRepo := &mockRoleRepo{
			findAllFn: func(ctx context.Context, _ sharedrepo.ListFilters) ([]*entities.Role, error) { return roles, nil },
		}

		svc := newRoleService(roleRepo, &mockPermissionRepo{}, &mockUserRoleRepo{})
		resp, err := svc.GetRoles(ctx, "", sharedrepo.ListFilters{})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if len(resp.Roles) != 2 {
			t.Errorf("esperaba 2 roles, obtuvo %d", len(resp.Roles))
		}
	})

	t.Run("filtra roles por scope", func(t *testing.T) {
		roles := []*entities.Role{
			{ID: uuid.New(), Name: "teacher", DisplayName: "Teacher", Scope: "school", IsActive: true},
		}
		var capturedScope string
		roleRepo := &mockRoleRepo{
			findByScopeFn: func(ctx context.Context, scope string, _ sharedrepo.ListFilters) ([]*entities.Role, error) {
				capturedScope = scope
				return roles, nil
			},
		}

		svc := newRoleService(roleRepo, &mockPermissionRepo{}, &mockUserRoleRepo{})
		resp, err := svc.GetRoles(ctx, "school", sharedrepo.ListFilters{})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if capturedScope != "school" {
			t.Errorf("scope incorrecto: %s", capturedScope)
		}
		if len(resp.Roles) != 1 {
			t.Errorf("esperaba 1 rol, obtuvo %d", len(resp.Roles))
		}
	})

	t.Run("propaga error de base de datos en FindAll", func(t *testing.T) {
		roleRepo := &mockRoleRepo{
			findAllFn: func(ctx context.Context, _ sharedrepo.ListFilters) ([]*entities.Role, error) { return nil, errors.New("db error") },
		}
		svc := newRoleService(roleRepo, &mockPermissionRepo{}, &mockUserRoleRepo{})

		_, err := svc.GetRoles(ctx, "", sharedrepo.ListFilters{})
		assertAppError(t, err, sharedErrors.ErrorCodeDatabaseError)
	})

	t.Run("propaga error de base de datos en FindByScope", func(t *testing.T) {
		roleRepo := &mockRoleRepo{
			findByScopeFn: func(ctx context.Context, scope string, _ sharedrepo.ListFilters) ([]*entities.Role, error) { return nil, errors.New("db error") },
		}
		svc := newRoleService(roleRepo, &mockPermissionRepo{}, &mockUserRoleRepo{})

		_, err := svc.GetRoles(ctx, "school", sharedrepo.ListFilters{})
		assertAppError(t, err, sharedErrors.ErrorCodeDatabaseError)
	})
}

// ─── GetRole ─────────────────────────────────────────────────────────────────

func TestRoleService_GetRole(t *testing.T) {
	ctx := context.Background()

	t.Run("retorna role existente", func(t *testing.T) {
		id := uuid.New()
		desc := "administrador"
		role := &entities.Role{ID: id, Name: "admin", DisplayName: "Admin", Description: &desc, Scope: "platform", IsActive: true}
		roleRepo := &mockRoleRepo{
			findByIDFn: func(ctx context.Context, gotID uuid.UUID) (*entities.Role, error) { return role, nil },
		}

		svc := newRoleService(roleRepo, &mockPermissionRepo{}, &mockUserRoleRepo{})
		resp, err := svc.GetRole(ctx, id.String())
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if resp.ID != id.String() {
			t.Errorf("ID incorrecto")
		}
		if resp.Description != desc {
			t.Errorf("descripción incorrecta: %s", resp.Description)
		}
	})

	t.Run("retorna error de validación con UUID inválido", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{}, &mockPermissionRepo{}, &mockUserRoleRepo{})
		_, err := svc.GetRole(ctx, "bad-uuid")
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})

	t.Run("retorna not found cuando role no existe", func(t *testing.T) {
		roleRepo := &mockRoleRepo{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Role, error) { return nil, nil },
		}
		svc := newRoleService(roleRepo, &mockPermissionRepo{}, &mockUserRoleRepo{})
		_, err := svc.GetRole(ctx, uuid.New().String())
		assertAppError(t, err, sharedErrors.ErrorCodeNotFound)
	})

	t.Run("propaga error de base de datos", func(t *testing.T) {
		roleRepo := &mockRoleRepo{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Role, error) { return nil, errors.New("db error") },
		}
		svc := newRoleService(roleRepo, &mockPermissionRepo{}, &mockUserRoleRepo{})
		_, err := svc.GetRole(ctx, uuid.New().String())
		assertAppError(t, err, sharedErrors.ErrorCodeDatabaseError)
	})
}

// ─── GetRolePermissions ───────────────────────────────────────────────────────

func TestRoleService_GetRolePermissions(t *testing.T) {
	ctx := context.Background()

	t.Run("retorna permisos del role", func(t *testing.T) {
		roleID := uuid.New()
		resID := uuid.New()
		perms := []*entities.Permission{
			{ID: uuid.New(), Name: "res:read", DisplayName: "Read", ResourceID: resID, ResourceKey: "res", Action: "read", Scope: "school"},
		}
		permRepo := &mockPermissionRepo{
			findByRoleFn: func(ctx context.Context, gotRoleID uuid.UUID) ([]*entities.Permission, error) {
				if gotRoleID != roleID {
					t.Errorf("roleID incorrecto: %s", gotRoleID)
				}
				return perms, nil
			},
		}
		svc := newRoleService(&mockRoleRepo{}, permRepo, &mockUserRoleRepo{})
		resp, err := svc.GetRolePermissions(ctx, roleID.String())
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if len(resp.Permissions) != 1 {
			t.Errorf("esperaba 1 permiso, obtuvo %d", len(resp.Permissions))
		}
	})

	t.Run("retorna error de validación con UUID inválido", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{}, &mockPermissionRepo{}, &mockUserRoleRepo{})
		_, err := svc.GetRolePermissions(ctx, "invalid")
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})
}

// ─── GetUserRoles ─────────────────────────────────────────────────────────────

func TestRoleService_GetUserRoles(t *testing.T) {
	ctx := context.Background()

	t.Run("retorna roles del usuario con nombre de role", func(t *testing.T) {
		userID := uuid.New()
		roleID := uuid.New()
		now := time.Now()
		userRoles := []*entities.UserRole{
			{ID: uuid.New(), UserID: userID, RoleID: roleID, IsActive: true, GrantedAt: now},
		}
		role := &entities.Role{ID: roleID, Name: "admin", DisplayName: "Admin", Scope: "platform", IsActive: true}

		urRepo := &mockUserRoleRepo{
			findByUserFn: func(ctx context.Context, uid uuid.UUID) ([]*entities.UserRole, error) { return userRoles, nil },
		}
		roleRepo := &mockRoleRepo{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Role, error) { return role, nil },
		}

		svc := newRoleService(roleRepo, &mockPermissionRepo{}, urRepo)
		resp, err := svc.GetUserRoles(ctx, userID.String())
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if len(resp.UserRoles) != 1 {
			t.Errorf("esperaba 1 user role, obtuvo %d", len(resp.UserRoles))
		}
		if resp.UserRoles[0].RoleName != "admin" {
			t.Errorf("nombre de role incorrecto: %s", resp.UserRoles[0].RoleName)
		}
	})

	t.Run("retorna error de validación con UUID inválido", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{}, &mockPermissionRepo{}, &mockUserRoleRepo{})
		_, err := svc.GetUserRoles(ctx, "bad-uuid")
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})

	t.Run("incluye schoolID y academicUnitID cuando están presentes", func(t *testing.T) {
		userID := uuid.New()
		roleID := uuid.New()
		schoolID := uuid.New()
		unitID := uuid.New()
		now := time.Now()
		userRoles := []*entities.UserRole{
			{ID: uuid.New(), UserID: userID, RoleID: roleID, SchoolID: &schoolID, AcademicUnitID: &unitID, IsActive: true, GrantedAt: now},
		}

		urRepo := &mockUserRoleRepo{
			findByUserFn: func(ctx context.Context, uid uuid.UUID) ([]*entities.UserRole, error) { return userRoles, nil },
		}
		svc := newRoleService(&mockRoleRepo{}, &mockPermissionRepo{}, urRepo)
		resp, err := svc.GetUserRoles(ctx, userID.String())
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if resp.UserRoles[0].SchoolID == nil || *resp.UserRoles[0].SchoolID != schoolID.String() {
			t.Errorf("schoolID incorrecto")
		}
		if resp.UserRoles[0].AcademicUnitID == nil || *resp.UserRoles[0].AcademicUnitID != unitID.String() {
			t.Errorf("academicUnitID incorrecto")
		}
	})
}

// ─── GrantRoleToUser ──────────────────────────────────────────────────────────

func TestRoleService_GrantRoleToUser(t *testing.T) {
	ctx := context.Background()

	t.Run("asigna rol al usuario correctamente", func(t *testing.T) {
		userID := uuid.New()
		roleID := uuid.New()
		role := &entities.Role{ID: roleID, Name: "teacher", DisplayName: "Teacher", Scope: "school", IsActive: true}

		roleRepo := &mockRoleRepo{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Role, error) { return role, nil },
		}
		urRepo := &mockUserRoleRepo{
			userHasRoleFn: func(ctx context.Context, uID, rID uuid.UUID, sID, aID *uuid.UUID) (bool, error) { return false, nil },
			grantFn:       func(ctx context.Context, ur *entities.UserRole) error { return nil },
		}

		svc := newRoleService(roleRepo, &mockPermissionRepo{}, urRepo)
		req := &dto.GrantRoleRequest{RoleID: roleID.String()}
		resp, err := svc.GrantRoleToUser(ctx, userID.String(), req, uuid.New().String())
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if resp.UserRole.RoleName != "teacher" {
			t.Errorf("nombre de role incorrecto: %s", resp.UserRole.RoleName)
		}
		if !resp.UserRole.IsActive {
			t.Error("el user role debería estar activo")
		}
	})

	t.Run("retorna already exists cuando el usuario ya tiene el rol", func(t *testing.T) {
		userID := uuid.New()
		roleID := uuid.New()
		role := &entities.Role{ID: roleID, Name: "teacher", DisplayName: "Teacher", Scope: "school", IsActive: true}

		roleRepo := &mockRoleRepo{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Role, error) { return role, nil },
		}
		urRepo := &mockUserRoleRepo{
			userHasRoleFn: func(ctx context.Context, uID, rID uuid.UUID, sID, aID *uuid.UUID) (bool, error) { return true, nil },
		}

		svc := newRoleService(roleRepo, &mockPermissionRepo{}, urRepo)
		req := &dto.GrantRoleRequest{RoleID: roleID.String()}
		_, err := svc.GrantRoleToUser(ctx, userID.String(), req, "")
		assertAppError(t, err, sharedErrors.ErrorCodeAlreadyExists)
	})

	t.Run("retorna error de validación con userID inválido", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{}, &mockPermissionRepo{}, &mockUserRoleRepo{})
		req := &dto.GrantRoleRequest{RoleID: uuid.New().String()}
		_, err := svc.GrantRoleToUser(ctx, "bad-uuid", req, "")
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})

	t.Run("retorna error de validación con roleID inválido", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{}, &mockPermissionRepo{}, &mockUserRoleRepo{})
		req := &dto.GrantRoleRequest{RoleID: "bad-uuid"}
		_, err := svc.GrantRoleToUser(ctx, uuid.New().String(), req, "")
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})

	t.Run("retorna error cuando el role no existe", func(t *testing.T) {
		roleRepo := &mockRoleRepo{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Role, error) { return nil, nil },
		}
		svc := newRoleService(roleRepo, &mockPermissionRepo{}, &mockUserRoleRepo{})
		req := &dto.GrantRoleRequest{RoleID: uuid.New().String()}
		_, err := svc.GrantRoleToUser(ctx, uuid.New().String(), req, "")
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})

	t.Run("asigna con schoolID válido", func(t *testing.T) {
		userID := uuid.New()
		roleID := uuid.New()
		schoolID := uuid.New().String()
		role := &entities.Role{ID: roleID, Name: "teacher", DisplayName: "Teacher", Scope: "school", IsActive: true}

		var capturedSchoolID *uuid.UUID
		roleRepo := &mockRoleRepo{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Role, error) { return role, nil },
		}
		urRepo := &mockUserRoleRepo{
			userHasRoleFn: func(ctx context.Context, uID, rID uuid.UUID, sID, aID *uuid.UUID) (bool, error) {
				capturedSchoolID = sID
				return false, nil
			},
			grantFn: func(ctx context.Context, ur *entities.UserRole) error { return nil },
		}

		svc := newRoleService(roleRepo, &mockPermissionRepo{}, urRepo)
		req := &dto.GrantRoleRequest{RoleID: roleID.String(), SchoolID: &schoolID}
		_, err := svc.GrantRoleToUser(ctx, userID.String(), req, "")
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if capturedSchoolID == nil || capturedSchoolID.String() != schoolID {
			t.Error("schoolID no fue pasado correctamente al repositorio")
		}
	})

	t.Run("retorna error con schoolID inválido", func(t *testing.T) {
		roleID := uuid.New()
		role := &entities.Role{ID: roleID, Name: "teacher", DisplayName: "Teacher", Scope: "school", IsActive: true}
		roleRepo := &mockRoleRepo{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Role, error) { return role, nil },
		}
		svc := newRoleService(roleRepo, &mockPermissionRepo{}, &mockUserRoleRepo{})
		badSchool := "not-valid"
		req := &dto.GrantRoleRequest{RoleID: roleID.String(), SchoolID: &badSchool}
		_, err := svc.GrantRoleToUser(ctx, uuid.New().String(), req, "")
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})

	t.Run("retorna error con expires_at inválido", func(t *testing.T) {
		userID := uuid.New()
		roleID := uuid.New()
		role := &entities.Role{ID: roleID, Name: "teacher", DisplayName: "Teacher", Scope: "school", IsActive: true}
		roleRepo := &mockRoleRepo{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Role, error) { return role, nil },
		}
		urRepo := &mockUserRoleRepo{
			userHasRoleFn: func(ctx context.Context, uID, rID uuid.UUID, sID, aID *uuid.UUID) (bool, error) { return false, nil },
		}
		svc := newRoleService(roleRepo, &mockPermissionRepo{}, urRepo)
		badDate := "not-a-date"
		req := &dto.GrantRoleRequest{RoleID: roleID.String(), ExpiresAt: &badDate}
		_, err := svc.GrantRoleToUser(ctx, userID.String(), req, "")
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})
}

// ─── RevokeRoleFromUser ───────────────────────────────────────────────────────

func TestRoleService_RevokeRoleFromUser(t *testing.T) {
	ctx := context.Background()

	t.Run("revoca rol correctamente", func(t *testing.T) {
		var capturedUserID, capturedRoleID uuid.UUID
		urRepo := &mockUserRoleRepo{
			revokeByUserAndRoleFn: func(ctx context.Context, userID, roleID uuid.UUID, sID, aID *uuid.UUID) error {
				capturedUserID = userID
				capturedRoleID = roleID
				return nil
			},
		}
		svc := newRoleService(&mockRoleRepo{}, &mockPermissionRepo{}, urRepo)
		userID := uuid.New()
		roleID := uuid.New()
		err := svc.RevokeRoleFromUser(ctx, userID.String(), roleID.String())
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if capturedUserID != userID {
			t.Errorf("userID incorrecto: %s", capturedUserID)
		}
		if capturedRoleID != roleID {
			t.Errorf("roleID incorrecto: %s", capturedRoleID)
		}
	})

	t.Run("retorna error de validación con userID inválido", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{}, &mockPermissionRepo{}, &mockUserRoleRepo{})
		err := svc.RevokeRoleFromUser(ctx, "bad-uuid", uuid.New().String())
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})

	t.Run("retorna error de validación con roleID inválido", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{}, &mockPermissionRepo{}, &mockUserRoleRepo{})
		err := svc.RevokeRoleFromUser(ctx, uuid.New().String(), "bad-uuid")
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})

	t.Run("propaga error de base de datos", func(t *testing.T) {
		urRepo := &mockUserRoleRepo{
			revokeByUserAndRoleFn: func(ctx context.Context, userID, roleID uuid.UUID, sID, aID *uuid.UUID) error {
				return errors.New("db error")
			},
		}
		svc := newRoleService(&mockRoleRepo{}, &mockPermissionRepo{}, urRepo)
		err := svc.RevokeRoleFromUser(ctx, uuid.New().String(), uuid.New().String())
		assertAppError(t, err, sharedErrors.ErrorCodeDatabaseError)
	})
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func assertAppError(t *testing.T, err error, code sharedErrors.ErrorCode) {
	t.Helper()
	if err == nil {
		t.Fatal("esperaba error, no obtuvo ninguno")
	}
	appErr, ok := sharedErrors.GetAppError(err)
	if !ok {
		t.Fatalf("esperaba AppError, obtuvo: %T %v", err, err)
	}
	if appErr.Code != code {
		t.Errorf("código de error incorrecto: esperaba %s, obtuvo %s", code, appErr.Code)
	}
}

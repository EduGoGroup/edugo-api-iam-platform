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
	return NewRoleService(roleRepo, permRepo, urRepo, &mockRolePermRepo{}, &mockLogger{})
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
			findAllFn: func(ctx context.Context, _ sharedrepo.ListFilters) ([]*entities.Role, int, error) { return roles, len(roles), nil },
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
			findByScopeFn: func(ctx context.Context, scope string, _ sharedrepo.ListFilters) ([]*entities.Role, int, error) {
				capturedScope = scope
				return roles, len(roles), nil
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
			findAllFn: func(ctx context.Context, _ sharedrepo.ListFilters) ([]*entities.Role, int, error) { return nil, 0, errors.New("db error") },
		}
		svc := newRoleService(roleRepo, &mockPermissionRepo{}, &mockUserRoleRepo{})

		_, err := svc.GetRoles(ctx, "", sharedrepo.ListFilters{})
		assertAppError(t, err, sharedErrors.ErrorCodeDatabaseError)
	})

	t.Run("propaga error de base de datos en FindByScope", func(t *testing.T) {
		roleRepo := &mockRoleRepo{
			findByScopeFn: func(ctx context.Context, scope string, _ sharedrepo.ListFilters) ([]*entities.Role, int, error) { return nil, 0, errors.New("db error") },
		}
		svc := newRoleService(roleRepo, &mockPermissionRepo{}, &mockUserRoleRepo{})

		_, err := svc.GetRoles(ctx, "school", sharedrepo.ListFilters{})
		assertAppError(t, err, sharedErrors.ErrorCodeDatabaseError)
	})

	t.Run("pasa filtros al repositorio en FindAll correctamente", func(t *testing.T) {
		var capturedFilters sharedrepo.ListFilters
		roleRepo := &mockRoleRepo{
			findAllFn: func(ctx context.Context, filters sharedrepo.ListFilters) ([]*entities.Role, int, error) {
				capturedFilters = filters
				return []*entities.Role{}, 0, nil
			},
		}
		svc := newRoleService(roleRepo, &mockPermissionRepo{}, &mockUserRoleRepo{})

		input := sharedrepo.ListFilters{Search: "admin", SearchFields: []string{"name"}}
		_, err := svc.GetRoles(ctx, "", input)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if capturedFilters.Search != input.Search {
			t.Errorf("Search no fue pasado: esperaba %q, obtuvo %q", input.Search, capturedFilters.Search)
		}
		if len(capturedFilters.SearchFields) != len(input.SearchFields) {
			t.Errorf("SearchFields no fue pasado correctamente: %v", capturedFilters.SearchFields)
		}
	})

	t.Run("pasa filtros al repositorio en FindByScope correctamente", func(t *testing.T) {
		var capturedFilters sharedrepo.ListFilters
		roleRepo := &mockRoleRepo{
			findByScopeFn: func(ctx context.Context, scope string, filters sharedrepo.ListFilters) ([]*entities.Role, int, error) {
				capturedFilters = filters
				return []*entities.Role{}, 0, nil
			},
		}
		svc := newRoleService(roleRepo, &mockPermissionRepo{}, &mockUserRoleRepo{})

		input := sharedrepo.ListFilters{Search: "teacher", SearchFields: []string{"display_name"}}
		_, err := svc.GetRoles(ctx, "school", input)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if capturedFilters.Search != input.Search {
			t.Errorf("Search no fue pasado: esperaba %q, obtuvo %q", input.Search, capturedFilters.Search)
		}
		if len(capturedFilters.SearchFields) != len(input.SearchFields) {
			t.Errorf("SearchFields no fue pasado correctamente: %v", capturedFilters.SearchFields)
		}
	})

	t.Run("metadatos de paginación: page y limit se propagan correctamente", func(t *testing.T) {
		roles := []*entities.Role{
			{ID: uuid.New(), Name: "admin", DisplayName: "Admin", Scope: "platform", IsActive: true},
		}
		roleRepo := &mockRoleRepo{
			findAllFn: func(ctx context.Context, _ sharedrepo.ListFilters) ([]*entities.Role, int, error) {
				return roles, 50, nil
			},
		}
		svc := newRoleService(roleRepo, &mockPermissionRepo{}, &mockUserRoleRepo{})

		resp, err := svc.GetRoles(ctx, "", sharedrepo.ListFilters{Page: 3, Limit: 15})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if resp.Total != 50 {
			t.Errorf("Total incorrecto: esperaba 50, obtuvo %d", resp.Total)
		}
		if resp.Page != 3 {
			t.Errorf("Page incorrecto: esperaba 3, obtuvo %d", resp.Page)
		}
		if resp.Limit != 15 {
			t.Errorf("Limit incorrecto: esperaba 15, obtuvo %d", resp.Limit)
		}
	})

	t.Run("metadatos de paginación: sin page ni limit usa defaults (page=1, limit=total)", func(t *testing.T) {
		roles := []*entities.Role{
			{ID: uuid.New(), Name: "admin", DisplayName: "Admin", Scope: "platform", IsActive: true},
			{ID: uuid.New(), Name: "teacher", DisplayName: "Teacher", Scope: "school", IsActive: true},
		}
		roleRepo := &mockRoleRepo{
			findAllFn: func(ctx context.Context, _ sharedrepo.ListFilters) ([]*entities.Role, int, error) {
				return roles, 2, nil
			},
		}
		svc := newRoleService(roleRepo, &mockPermissionRepo{}, &mockUserRoleRepo{})

		resp, err := svc.GetRoles(ctx, "", sharedrepo.ListFilters{})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if resp.Page != 1 {
			t.Errorf("Page default incorrecto: esperaba 1, obtuvo %d", resp.Page)
		}
		if resp.Limit != 2 {
			t.Errorf("Limit default incorrecto: esperaba total(2), obtuvo %d", resp.Limit)
		}
	})

	t.Run("metadatos de paginación: page>0 y limit=0 aplica default de 50", func(t *testing.T) {
		roleRepo := &mockRoleRepo{
			findAllFn: func(ctx context.Context, _ sharedrepo.ListFilters) ([]*entities.Role, int, error) {
				return []*entities.Role{}, 300, nil
			},
		}
		svc := newRoleService(roleRepo, &mockPermissionRepo{}, &mockUserRoleRepo{})

		resp, err := svc.GetRoles(ctx, "", sharedrepo.ListFilters{Page: 1, Limit: 0})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if resp.Limit != 50 {
			t.Errorf("Limit default esperaba 50 cuando page>0 y limit=0, obtuvo %d", resp.Limit)
		}
		if resp.Page != 1 {
			t.Errorf("Page incorrecto: esperaba 1, obtuvo %d", resp.Page)
		}
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
			{ID: uuid.New(), Name: "res:read", DisplayName: "Read", ResourceID: resID, Action: "read", Scope: "school"},
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

// ─── CreateRole ───────────────────────────────────────────────────────────────

func TestRoleService_CreateRole(t *testing.T) {
	ctx := context.Background()

	t.Run("crea role correctamente", func(t *testing.T) {
		var captured *entities.Role
		roleRepo := &mockRoleRepo{
			createFn: func(ctx context.Context, role *entities.Role) error {
				captured = role
				return nil
			},
		}
		svc := newRoleService(roleRepo, &mockPermissionRepo{}, &mockUserRoleRepo{})
		req := &dto.CreateRoleRequest{Name: "editor", DisplayName: "Editor", Description: "edit stuff", Scope: "school"}
		resp, err := svc.CreateRole(ctx, req)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if resp.Name != "editor" {
			t.Errorf("nombre incorrecto: %s", resp.Name)
		}
		if resp.DisplayName != "Editor" {
			t.Errorf("display name incorrecto: %s", resp.DisplayName)
		}
		if !resp.IsActive {
			t.Error("role debería estar activo")
		}
		if captured == nil {
			t.Fatal("no se llamó al repo")
		}
		if captured.Description == nil || *captured.Description != "edit stuff" {
			t.Errorf("descripción incorrecta")
		}
	})

	t.Run("retorna error con scope inválido", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{}, &mockPermissionRepo{}, &mockUserRoleRepo{})
		req := &dto.CreateRoleRequest{Name: "test", DisplayName: "Test", Scope: "invalid"}
		_, err := svc.CreateRole(ctx, req)
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})

	t.Run("acepta scope platform", func(t *testing.T) {
		roleRepo := &mockRoleRepo{
			createFn: func(ctx context.Context, role *entities.Role) error { return nil },
		}
		svc := newRoleService(roleRepo, &mockPermissionRepo{}, &mockUserRoleRepo{})
		req := &dto.CreateRoleRequest{Name: "admin", DisplayName: "Admin", Scope: "platform"}
		_, err := svc.CreateRole(ctx, req)
		if err != nil {
			t.Fatalf("scope platform debería ser válido: %v", err)
		}
	})

	t.Run("propaga error de base de datos", func(t *testing.T) {
		roleRepo := &mockRoleRepo{
			createFn: func(ctx context.Context, role *entities.Role) error { return errors.New("db error") },
		}
		svc := newRoleService(roleRepo, &mockPermissionRepo{}, &mockUserRoleRepo{})
		req := &dto.CreateRoleRequest{Name: "test", DisplayName: "Test", Scope: "school"}
		_, err := svc.CreateRole(ctx, req)
		assertAppError(t, err, sharedErrors.ErrorCodeDatabaseError)
	})
}

// ─── UpdateRole ───────────────────────────────────────────────────────────────

func TestRoleService_UpdateRole(t *testing.T) {
	ctx := context.Background()

	t.Run("actualiza role correctamente", func(t *testing.T) {
		id := uuid.New()
		role := &entities.Role{ID: id, Name: "admin", DisplayName: "Admin", Scope: "platform", IsActive: true}
		roleRepo := &mockRoleRepo{
			findByIDFn: func(ctx context.Context, gotID uuid.UUID) (*entities.Role, error) { return role, nil },
			updateFn:   func(ctx context.Context, r *entities.Role) error { return nil },
		}
		svc := newRoleService(roleRepo, &mockPermissionRepo{}, &mockUserRoleRepo{})

		newName := "super_admin"
		newDisplay := "Super Admin"
		req := &dto.UpdateRoleRequest{Name: &newName, DisplayName: &newDisplay}
		resp, err := svc.UpdateRole(ctx, id.String(), req)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if resp.Name != "super_admin" {
			t.Errorf("nombre no actualizado: %s", resp.Name)
		}
		if resp.DisplayName != "Super Admin" {
			t.Errorf("display name no actualizado: %s", resp.DisplayName)
		}
	})

	t.Run("retorna error con UUID inválido", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{}, &mockPermissionRepo{}, &mockUserRoleRepo{})
		_, err := svc.UpdateRole(ctx, "bad-uuid", &dto.UpdateRoleRequest{})
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})

	t.Run("retorna not found cuando role no existe", func(t *testing.T) {
		roleRepo := &mockRoleRepo{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Role, error) { return nil, nil },
		}
		svc := newRoleService(roleRepo, &mockPermissionRepo{}, &mockUserRoleRepo{})
		_, err := svc.UpdateRole(ctx, uuid.New().String(), &dto.UpdateRoleRequest{})
		assertAppError(t, err, sharedErrors.ErrorCodeNotFound)
	})

	t.Run("retorna error con scope inválido", func(t *testing.T) {
		id := uuid.New()
		role := &entities.Role{ID: id, Name: "admin", DisplayName: "Admin", Scope: "platform", IsActive: true}
		roleRepo := &mockRoleRepo{
			findByIDFn: func(ctx context.Context, gotID uuid.UUID) (*entities.Role, error) { return role, nil },
		}
		svc := newRoleService(roleRepo, &mockPermissionRepo{}, &mockUserRoleRepo{})
		bad := "invalid"
		req := &dto.UpdateRoleRequest{Scope: &bad}
		_, err := svc.UpdateRole(ctx, id.String(), req)
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})

	t.Run("propaga error de base de datos en update", func(t *testing.T) {
		id := uuid.New()
		role := &entities.Role{ID: id, Name: "admin", DisplayName: "Admin", Scope: "platform", IsActive: true}
		roleRepo := &mockRoleRepo{
			findByIDFn: func(ctx context.Context, gotID uuid.UUID) (*entities.Role, error) { return role, nil },
			updateFn:   func(ctx context.Context, r *entities.Role) error { return errors.New("db error") },
		}
		svc := newRoleService(roleRepo, &mockPermissionRepo{}, &mockUserRoleRepo{})
		_, err := svc.UpdateRole(ctx, id.String(), &dto.UpdateRoleRequest{})
		assertAppError(t, err, sharedErrors.ErrorCodeDatabaseError)
	})
}

// ─── DeleteRole ───────────────────────────────────────────────────────────────

func TestRoleService_DeleteRole(t *testing.T) {
	ctx := context.Background()

	t.Run("elimina role correctamente", func(t *testing.T) {
		id := uuid.New()
		role := &entities.Role{ID: id, Name: "admin", DisplayName: "Admin", Scope: "platform", IsActive: true}
		var deletedID uuid.UUID
		roleRepo := &mockRoleRepo{
			findByIDFn:           func(ctx context.Context, gotID uuid.UUID) (*entities.Role, error) { return role, nil },
			hasActiveUserRolesFn: func(ctx context.Context, roleID uuid.UUID) (bool, error) { return false, nil },
			softDeleteFn: func(ctx context.Context, gotID uuid.UUID) error {
				deletedID = gotID
				return nil
			},
		}
		svc := newRoleService(roleRepo, &mockPermissionRepo{}, &mockUserRoleRepo{})
		err := svc.DeleteRole(ctx, id.String())
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if deletedID != id {
			t.Errorf("ID de delete incorrecto: %s", deletedID)
		}
	})

	t.Run("retorna error con UUID inválido", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{}, &mockPermissionRepo{}, &mockUserRoleRepo{})
		err := svc.DeleteRole(ctx, "bad-uuid")
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})

	t.Run("retorna not found cuando role no existe", func(t *testing.T) {
		roleRepo := &mockRoleRepo{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Role, error) { return nil, nil },
		}
		svc := newRoleService(roleRepo, &mockPermissionRepo{}, &mockUserRoleRepo{})
		err := svc.DeleteRole(ctx, uuid.New().String())
		assertAppError(t, err, sharedErrors.ErrorCodeNotFound)
	})

	t.Run("retorna conflict cuando tiene user roles activos", func(t *testing.T) {
		id := uuid.New()
		role := &entities.Role{ID: id, Name: "admin", DisplayName: "Admin", Scope: "platform", IsActive: true}
		roleRepo := &mockRoleRepo{
			findByIDFn:           func(ctx context.Context, gotID uuid.UUID) (*entities.Role, error) { return role, nil },
			hasActiveUserRolesFn: func(ctx context.Context, roleID uuid.UUID) (bool, error) { return true, nil },
		}
		svc := newRoleService(roleRepo, &mockPermissionRepo{}, &mockUserRoleRepo{})
		err := svc.DeleteRole(ctx, id.String())
		assertAppError(t, err, sharedErrors.ErrorCodeConflict)
	})

	t.Run("propaga error de base de datos en SoftDelete", func(t *testing.T) {
		id := uuid.New()
		role := &entities.Role{ID: id, Name: "admin", DisplayName: "Admin", Scope: "platform", IsActive: true}
		roleRepo := &mockRoleRepo{
			findByIDFn:           func(ctx context.Context, gotID uuid.UUID) (*entities.Role, error) { return role, nil },
			hasActiveUserRolesFn: func(ctx context.Context, roleID uuid.UUID) (bool, error) { return false, nil },
			softDeleteFn:         func(ctx context.Context, gotID uuid.UUID) error { return errors.New("db error") },
		}
		svc := newRoleService(roleRepo, &mockPermissionRepo{}, &mockUserRoleRepo{})
		err := svc.DeleteRole(ctx, id.String())
		assertAppError(t, err, sharedErrors.ErrorCodeDatabaseError)
	})
}

// ─── AssignPermission ─────────────────────────────────────────────────────────

func newRoleServiceFull(roleRepo *mockRoleRepo, permRepo *mockPermissionRepo, urRepo *mockUserRoleRepo, rpRepo *mockRolePermRepo) RoleService {
	return NewRoleService(roleRepo, permRepo, urRepo, rpRepo, &mockLogger{})
}

func TestRoleService_AssignPermission(t *testing.T) {
	ctx := context.Background()

	t.Run("asigna permiso al role correctamente", func(t *testing.T) {
		roleID := uuid.New()
		permID := uuid.New()
		resID := uuid.New()
		role := &entities.Role{ID: roleID, Name: "admin", DisplayName: "Admin", Scope: "platform", IsActive: true}
		perm := &entities.Permission{ID: permID, Name: "users:read", DisplayName: "Read Users", ResourceID: resID, Action: "read", Scope: "school"}

		roleRepo := &mockRoleRepo{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Role, error) { return role, nil },
		}
		permRepo := &mockPermissionRepo{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Permission, error) { return perm, nil },
		}
		rpRepo := &mockRolePermRepo{
			existsFn: func(ctx context.Context, rID, pID uuid.UUID) (bool, error) { return false, nil },
			assignFn: func(ctx context.Context, rp *entities.RolePermission) error { return nil },
		}

		svc := newRoleServiceFull(roleRepo, permRepo, &mockUserRoleRepo{}, rpRepo)
		req := &dto.AssignPermissionRequest{PermissionID: permID.String()}
		resp, err := svc.AssignPermission(ctx, roleID.String(), req)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if resp.RoleID != roleID.String() {
			t.Errorf("roleID incorrecto: %s", resp.RoleID)
		}
		if resp.PermissionID != permID.String() {
			t.Errorf("permissionID incorrecto: %s", resp.PermissionID)
		}
	})

	t.Run("retorna already exists cuando ya está asignado", func(t *testing.T) {
		roleID := uuid.New()
		permID := uuid.New()
		resID := uuid.New()
		role := &entities.Role{ID: roleID, Name: "admin", DisplayName: "Admin", Scope: "platform", IsActive: true}
		perm := &entities.Permission{ID: permID, Name: "users:read", DisplayName: "Read Users", ResourceID: resID, Action: "read", Scope: "school"}

		roleRepo := &mockRoleRepo{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Role, error) { return role, nil },
		}
		permRepo := &mockPermissionRepo{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Permission, error) { return perm, nil },
		}
		rpRepo := &mockRolePermRepo{
			existsFn: func(ctx context.Context, rID, pID uuid.UUID) (bool, error) { return true, nil },
		}

		svc := newRoleServiceFull(roleRepo, permRepo, &mockUserRoleRepo{}, rpRepo)
		req := &dto.AssignPermissionRequest{PermissionID: permID.String()}
		_, err := svc.AssignPermission(ctx, roleID.String(), req)
		assertAppError(t, err, sharedErrors.ErrorCodeAlreadyExists)
	})

	t.Run("retorna error con roleID inválido", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{}, &mockPermissionRepo{}, &mockUserRoleRepo{})
		req := &dto.AssignPermissionRequest{PermissionID: uuid.New().String()}
		_, err := svc.AssignPermission(ctx, "bad-uuid", req)
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})

	t.Run("retorna error con permissionID inválido", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{}, &mockPermissionRepo{}, &mockUserRoleRepo{})
		req := &dto.AssignPermissionRequest{PermissionID: "bad-uuid"}
		_, err := svc.AssignPermission(ctx, uuid.New().String(), req)
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})

	t.Run("retorna not found cuando role no existe", func(t *testing.T) {
		roleRepo := &mockRoleRepo{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Role, error) { return nil, nil },
		}
		svc := newRoleService(roleRepo, &mockPermissionRepo{}, &mockUserRoleRepo{})
		req := &dto.AssignPermissionRequest{PermissionID: uuid.New().String()}
		_, err := svc.AssignPermission(ctx, uuid.New().String(), req)
		assertAppError(t, err, sharedErrors.ErrorCodeNotFound)
	})

	t.Run("retorna not found cuando permiso no existe", func(t *testing.T) {
		roleID := uuid.New()
		role := &entities.Role{ID: roleID, Name: "admin", DisplayName: "Admin", Scope: "platform", IsActive: true}
		roleRepo := &mockRoleRepo{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Role, error) { return role, nil },
		}
		permRepo := &mockPermissionRepo{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Permission, error) { return nil, nil },
		}
		svc := newRoleServiceFull(roleRepo, permRepo, &mockUserRoleRepo{}, &mockRolePermRepo{})
		req := &dto.AssignPermissionRequest{PermissionID: uuid.New().String()}
		_, err := svc.AssignPermission(ctx, roleID.String(), req)
		assertAppError(t, err, sharedErrors.ErrorCodeNotFound)
	})
}

// ─── RevokePermission ─────────────────────────────────────────────────────────

func TestRoleService_RevokePermission(t *testing.T) {
	ctx := context.Background()

	t.Run("revoca permiso correctamente", func(t *testing.T) {
		var capturedRoleID, capturedPermID uuid.UUID
		rpRepo := &mockRolePermRepo{
			revokeFn: func(ctx context.Context, rID, pID uuid.UUID) error {
				capturedRoleID = rID
				capturedPermID = pID
				return nil
			},
		}
		svc := newRoleServiceFull(&mockRoleRepo{}, &mockPermissionRepo{}, &mockUserRoleRepo{}, rpRepo)
		roleID := uuid.New()
		permID := uuid.New()
		err := svc.RevokePermission(ctx, roleID.String(), permID.String())
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if capturedRoleID != roleID {
			t.Errorf("roleID incorrecto: %s", capturedRoleID)
		}
		if capturedPermID != permID {
			t.Errorf("permID incorrecto: %s", capturedPermID)
		}
	})

	t.Run("retorna error con roleID inválido", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{}, &mockPermissionRepo{}, &mockUserRoleRepo{})
		err := svc.RevokePermission(ctx, "bad-uuid", uuid.New().String())
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})

	t.Run("retorna error con permissionID inválido", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{}, &mockPermissionRepo{}, &mockUserRoleRepo{})
		err := svc.RevokePermission(ctx, uuid.New().String(), "bad-uuid")
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})

	t.Run("propaga error de base de datos", func(t *testing.T) {
		rpRepo := &mockRolePermRepo{
			revokeFn: func(ctx context.Context, rID, pID uuid.UUID) error { return errors.New("db error") },
		}
		svc := newRoleServiceFull(&mockRoleRepo{}, &mockPermissionRepo{}, &mockUserRoleRepo{}, rpRepo)
		err := svc.RevokePermission(ctx, uuid.New().String(), uuid.New().String())
		assertAppError(t, err, sharedErrors.ErrorCodeDatabaseError)
	})
}

// ─── BulkReplacePermissions ───────────────────────────────────────────────────

func TestRoleService_BulkReplacePermissions(t *testing.T) {
	ctx := context.Background()

	t.Run("reemplaza permisos correctamente", func(t *testing.T) {
		roleID := uuid.New()
		permID1 := uuid.New()
		permID2 := uuid.New()
		resID := uuid.New()
		role := &entities.Role{ID: roleID, Name: "admin", DisplayName: "Admin", Scope: "platform", IsActive: true}
		perm1 := &entities.Permission{ID: permID1, Name: "users:read", DisplayName: "Read", ResourceID: resID, Action: "read", Scope: "school"}
		perm2 := &entities.Permission{ID: permID2, Name: "users:write", DisplayName: "Write", ResourceID: resID, Action: "write", Scope: "school"}

		roleRepo := &mockRoleRepo{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Role, error) { return role, nil },
		}
		permRepo := &mockPermissionRepo{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Permission, error) {
				if id == permID1 {
					return perm1, nil
				}
				return perm2, nil
			},
			findByRoleFn: func(ctx context.Context, rID uuid.UUID) ([]*entities.Permission, error) {
				return []*entities.Permission{perm1, perm2}, nil
			},
		}
		var capturedIDs []uuid.UUID
		rpRepo := &mockRolePermRepo{
			bulkReplaceFn: func(ctx context.Context, rID uuid.UUID, pIDs []uuid.UUID) error {
				capturedIDs = pIDs
				return nil
			},
		}

		svc := newRoleServiceFull(roleRepo, permRepo, &mockUserRoleRepo{}, rpRepo)
		req := &dto.BulkPermissionsRequest{PermissionIDs: []string{permID1.String(), permID2.String()}}
		resp, err := svc.BulkReplacePermissions(ctx, roleID.String(), req)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if len(capturedIDs) != 2 {
			t.Errorf("esperaba 2 IDs, obtuvo %d", len(capturedIDs))
		}
		if len(resp.Permissions) != 2 {
			t.Errorf("esperaba 2 permisos, obtuvo %d", len(resp.Permissions))
		}
	})

	t.Run("retorna error con roleID inválido", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{}, &mockPermissionRepo{}, &mockUserRoleRepo{})
		req := &dto.BulkPermissionsRequest{PermissionIDs: []string{uuid.New().String()}}
		_, err := svc.BulkReplacePermissions(ctx, "bad-uuid", req)
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})

	t.Run("retorna not found cuando role no existe", func(t *testing.T) {
		roleRepo := &mockRoleRepo{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Role, error) { return nil, nil },
		}
		svc := newRoleService(roleRepo, &mockPermissionRepo{}, &mockUserRoleRepo{})
		req := &dto.BulkPermissionsRequest{PermissionIDs: []string{uuid.New().String()}}
		_, err := svc.BulkReplacePermissions(ctx, uuid.New().String(), req)
		assertAppError(t, err, sharedErrors.ErrorCodeNotFound)
	})

	t.Run("retorna error con permissionID inválido en lista", func(t *testing.T) {
		roleID := uuid.New()
		role := &entities.Role{ID: roleID, Name: "admin", DisplayName: "Admin", Scope: "platform", IsActive: true}
		roleRepo := &mockRoleRepo{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Role, error) { return role, nil },
		}
		svc := newRoleService(roleRepo, &mockPermissionRepo{}, &mockUserRoleRepo{})
		req := &dto.BulkPermissionsRequest{PermissionIDs: []string{"bad-uuid"}}
		_, err := svc.BulkReplacePermissions(ctx, roleID.String(), req)
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})

	t.Run("retorna not found cuando un permiso no existe", func(t *testing.T) {
		roleID := uuid.New()
		role := &entities.Role{ID: roleID, Name: "admin", DisplayName: "Admin", Scope: "platform", IsActive: true}
		roleRepo := &mockRoleRepo{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Role, error) { return role, nil },
		}
		permRepo := &mockPermissionRepo{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Permission, error) { return nil, nil },
		}
		svc := newRoleServiceFull(roleRepo, permRepo, &mockUserRoleRepo{}, &mockRolePermRepo{})
		req := &dto.BulkPermissionsRequest{PermissionIDs: []string{uuid.New().String()}}
		_, err := svc.BulkReplacePermissions(ctx, roleID.String(), req)
		assertAppError(t, err, sharedErrors.ErrorCodeNotFound)
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

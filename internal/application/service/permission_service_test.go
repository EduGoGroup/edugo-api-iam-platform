package service

import (
	"context"
	"errors"
	"testing"

	"github.com/EduGoGroup/edugo-infrastructure/postgres/entities"
	sharedErrors "github.com/EduGoGroup/edugo-shared/common/errors"
	sharedrepo "github.com/EduGoGroup/edugo-shared/repository"
	"github.com/google/uuid"
)

func TestPermissionService_ListPermissions(t *testing.T) {
	ctx := context.Background()

	t.Run("retorna lista de permisos correctamente", func(t *testing.T) {
		id1, id2 := uuid.New(), uuid.New()
		resID := uuid.New()
		perms := []*entities.Permission{
			{ID: id1, Name: "resource:read", DisplayName: "Read", ResourceID: resID, ResourceKey: "resource", Action: "read", Scope: "school"},
			{ID: id2, Name: "resource:write", DisplayName: "Write", ResourceID: resID, ResourceKey: "resource", Action: "write", Scope: "school"},
		}

		svc := NewPermissionService(
			&mockPermissionRepo{findAllFn: func(ctx context.Context, filters sharedrepo.ListFilters) ([]*entities.Permission, error) { return perms, nil }},
			&mockLogger{},
		)

		resp, err := svc.ListPermissions(ctx, sharedrepo.ListFilters{})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if len(resp.Permissions) != 2 {
			t.Errorf("esperaba 2 permisos, obtuvo %d", len(resp.Permissions))
		}
		if resp.Permissions[0].ID != id1.String() {
			t.Errorf("ID incorrecto: esperaba %s, obtuvo %s", id1.String(), resp.Permissions[0].ID)
		}
	})

	t.Run("retorna lista vacía sin error", func(t *testing.T) {
		svc := NewPermissionService(
			&mockPermissionRepo{findAllFn: func(ctx context.Context, filters sharedrepo.ListFilters) ([]*entities.Permission, error) { return []*entities.Permission{}, nil }},
			&mockLogger{},
		)

		resp, err := svc.ListPermissions(ctx, sharedrepo.ListFilters{})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if len(resp.Permissions) != 0 {
			t.Errorf("esperaba 0 permisos, obtuvo %d", len(resp.Permissions))
		}
	})

	t.Run("propaga error de base de datos", func(t *testing.T) {
		svc := NewPermissionService(
			&mockPermissionRepo{findAllFn: func(ctx context.Context, filters sharedrepo.ListFilters) ([]*entities.Permission, error) {
				return nil, errors.New("db error")
			}},
			&mockLogger{},
		)

		_, err := svc.ListPermissions(ctx, sharedrepo.ListFilters{})
		if err == nil {
			t.Fatal("esperaba error, no obtuvo ninguno")
		}
		appErr, ok := sharedErrors.GetAppError(err)
		if !ok {
			t.Fatal("esperaba AppError")
		}
		if appErr.Code != sharedErrors.ErrorCodeDatabaseError {
			t.Errorf("código de error incorrecto: %s", appErr.Code)
		}
	})

	t.Run("pasa filtros al repositorio correctamente", func(t *testing.T) {
		var capturedFilters sharedrepo.ListFilters
		svc := NewPermissionService(
			&mockPermissionRepo{findAllFn: func(ctx context.Context, filters sharedrepo.ListFilters) ([]*entities.Permission, error) {
				capturedFilters = filters
				return []*entities.Permission{}, nil
			}},
			&mockLogger{},
		)

		input := sharedrepo.ListFilters{Search: "read", SearchFields: []string{"name", "display_name"}}
		_, err := svc.ListPermissions(ctx, input)
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
}

func TestPermissionService_GetPermission(t *testing.T) {
	ctx := context.Background()

	t.Run("retorna permiso existente correctamente", func(t *testing.T) {
		id := uuid.New()
		resID := uuid.New()
		perm := &entities.Permission{ID: id, Name: "res:read", DisplayName: "Read", ResourceID: resID, ResourceKey: "res", Action: "read", Scope: "school"}

		svc := NewPermissionService(
			&mockPermissionRepo{findByIDFn: func(ctx context.Context, gotID uuid.UUID) (*entities.Permission, error) {
				if gotID != id {
					t.Errorf("ID incorrecto pasado al repo: %s", gotID)
				}
				return perm, nil
			}},
			&mockLogger{},
		)

		resp, err := svc.GetPermission(ctx, id.String())
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if resp.ID != id.String() {
			t.Errorf("ID incorrecto: esperaba %s, obtuvo %s", id.String(), resp.ID)
		}
		if resp.Action != "read" {
			t.Errorf("acción incorrecta: %s", resp.Action)
		}
	})

	t.Run("retorna error de validación con UUID inválido", func(t *testing.T) {
		svc := NewPermissionService(&mockPermissionRepo{}, &mockLogger{})

		_, err := svc.GetPermission(ctx, "not-a-uuid")
		if err == nil {
			t.Fatal("esperaba error de validación")
		}
		appErr, ok := sharedErrors.GetAppError(err)
		if !ok {
			t.Fatal("esperaba AppError")
		}
		if appErr.Code != sharedErrors.ErrorCodeValidation {
			t.Errorf("código incorrecto: %s", appErr.Code)
		}
	})

	t.Run("retorna not found cuando el permiso no existe", func(t *testing.T) {
		svc := NewPermissionService(
			&mockPermissionRepo{findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Permission, error) {
				return nil, nil
			}},
			&mockLogger{},
		)

		_, err := svc.GetPermission(ctx, uuid.New().String())
		if err == nil {
			t.Fatal("esperaba error not found")
		}
		appErr, ok := sharedErrors.GetAppError(err)
		if !ok {
			t.Fatal("esperaba AppError")
		}
		if appErr.Code != sharedErrors.ErrorCodeNotFound {
			t.Errorf("código incorrecto: %s", appErr.Code)
		}
	})

	t.Run("propaga error de base de datos", func(t *testing.T) {
		svc := NewPermissionService(
			&mockPermissionRepo{findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Permission, error) {
				return nil, errors.New("db error")
			}},
			&mockLogger{},
		)

		_, err := svc.GetPermission(ctx, uuid.New().String())
		if err == nil {
			t.Fatal("esperaba error")
		}
		appErr, ok := sharedErrors.GetAppError(err)
		if !ok {
			t.Fatal("esperaba AppError")
		}
		if appErr.Code != sharedErrors.ErrorCodeDatabaseError {
			t.Errorf("código incorrecto: %s", appErr.Code)
		}
	})

	t.Run("mapea campos de Description correctamente", func(t *testing.T) {
		id := uuid.New()
		desc := "permiso de lectura"
		resID := uuid.New()
		perm := &entities.Permission{ID: id, Name: "res:read", DisplayName: "Read", Description: &desc, ResourceID: resID, ResourceKey: "res", Action: "read", Scope: "school"}

		svc := NewPermissionService(
			&mockPermissionRepo{findByIDFn: func(ctx context.Context, _ uuid.UUID) (*entities.Permission, error) { return perm, nil }},
			&mockLogger{},
		)

		resp, err := svc.GetPermission(ctx, id.String())
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if resp.Description != desc {
			t.Errorf("descripción incorrecta: %s", resp.Description)
		}
	})
}

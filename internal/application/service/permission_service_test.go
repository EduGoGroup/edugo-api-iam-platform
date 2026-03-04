package service

import (
	"context"
	"errors"
	"testing"

	"github.com/EduGoGroup/edugo-api-iam-platform/internal/application/dto"
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
			{ID: id1, Name: "resource:read", DisplayName: "Read", ResourceID: resID, Action: "read", Scope: "school"},
			{ID: id2, Name: "resource:write", DisplayName: "Write", ResourceID: resID, Action: "write", Scope: "school"},
		}

		svc := NewPermissionService(
			&mockPermissionRepo{findAllFn: func(ctx context.Context, filters sharedrepo.ListFilters) ([]*entities.Permission, int, error) { return perms, len(perms), nil }},
			&mockResourceRepo{},
			&mockLogger{},
			&mockAuditLogger{},
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
			&mockPermissionRepo{findAllFn: func(ctx context.Context, filters sharedrepo.ListFilters) ([]*entities.Permission, int, error) { return []*entities.Permission{}, 0, nil }},
			&mockResourceRepo{},
			&mockLogger{},
			&mockAuditLogger{},
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
			&mockPermissionRepo{findAllFn: func(ctx context.Context, filters sharedrepo.ListFilters) ([]*entities.Permission, int, error) {
				return nil, 0, errors.New("db error")
			}},
			&mockResourceRepo{},
			&mockLogger{},
			&mockAuditLogger{},
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
			&mockPermissionRepo{findAllFn: func(ctx context.Context, filters sharedrepo.ListFilters) ([]*entities.Permission, int, error) {
				capturedFilters = filters
				return []*entities.Permission{}, 0, nil
			}},
			&mockResourceRepo{},
			&mockLogger{},
			&mockAuditLogger{},
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

	t.Run("metadatos de paginación: page y limit se propagan correctamente", func(t *testing.T) {
		resID := uuid.New()
		perms := []*entities.Permission{
			{ID: uuid.New(), Name: "resource:read", DisplayName: "Read", ResourceID: resID, Action: "read", Scope: "school"},
		}
		svc := NewPermissionService(
			&mockPermissionRepo{findAllFn: func(ctx context.Context, filters sharedrepo.ListFilters) ([]*entities.Permission, int, error) {
				return perms, 100, nil
			}},
			&mockResourceRepo{},
			&mockLogger{},
			&mockAuditLogger{},
		)

		resp, err := svc.ListPermissions(ctx, sharedrepo.ListFilters{Page: 2, Limit: 10})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if resp.Total != 100 {
			t.Errorf("Total incorrecto: esperaba 100, obtuvo %d", resp.Total)
		}
		if resp.Page != 2 {
			t.Errorf("Page incorrecto: esperaba 2, obtuvo %d", resp.Page)
		}
		if resp.Limit != 10 {
			t.Errorf("Limit incorrecto: esperaba 10, obtuvo %d", resp.Limit)
		}
	})

	t.Run("metadatos de paginación: sin page ni limit usa defaults (page=1, limit=total)", func(t *testing.T) {
		resID := uuid.New()
		perms := []*entities.Permission{
			{ID: uuid.New(), Name: "resource:read", DisplayName: "Read", ResourceID: resID, Action: "read", Scope: "school"},
			{ID: uuid.New(), Name: "resource:write", DisplayName: "Write", ResourceID: resID, Action: "write", Scope: "school"},
		}
		svc := NewPermissionService(
			&mockPermissionRepo{findAllFn: func(ctx context.Context, filters sharedrepo.ListFilters) ([]*entities.Permission, int, error) {
				return perms, 2, nil
			}},
			&mockResourceRepo{},
			&mockLogger{},
			&mockAuditLogger{},
		)

		resp, err := svc.ListPermissions(ctx, sharedrepo.ListFilters{})
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
		svc := NewPermissionService(
			&mockPermissionRepo{findAllFn: func(ctx context.Context, filters sharedrepo.ListFilters) ([]*entities.Permission, int, error) {
				return []*entities.Permission{}, 200, nil
			}},
			&mockResourceRepo{},
			&mockLogger{},
			&mockAuditLogger{},
		)

		resp, err := svc.ListPermissions(ctx, sharedrepo.ListFilters{Page: 1, Limit: 0})
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

func TestPermissionService_GetPermission(t *testing.T) {
	ctx := context.Background()

	t.Run("retorna permiso existente correctamente", func(t *testing.T) {
		id := uuid.New()
		resID := uuid.New()
		perm := &entities.Permission{ID: id, Name: "res:read", DisplayName: "Read", ResourceID: resID, Action: "read", Scope: "school"}

		svc := NewPermissionService(
			&mockPermissionRepo{findByIDFn: func(ctx context.Context, gotID uuid.UUID) (*entities.Permission, error) {
				if gotID != id {
					t.Errorf("ID incorrecto pasado al repo: %s", gotID)
				}
				return perm, nil
			}},
			&mockResourceRepo{},
			&mockLogger{},
			&mockAuditLogger{},
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
		svc := NewPermissionService(&mockPermissionRepo{}, &mockResourceRepo{}, &mockLogger{}, &mockAuditLogger{})

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
			&mockResourceRepo{},
			&mockLogger{},
			&mockAuditLogger{},
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
			&mockResourceRepo{},
			&mockLogger{},
			&mockAuditLogger{},
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
		perm := &entities.Permission{ID: id, Name: "res:read", DisplayName: "Read", Description: &desc, ResourceID: resID, Action: "read", Scope: "school"}

		svc := NewPermissionService(
			&mockPermissionRepo{findByIDFn: func(ctx context.Context, _ uuid.UUID) (*entities.Permission, error) { return perm, nil }},
			&mockResourceRepo{},
			&mockLogger{},
			&mockAuditLogger{},
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

// ─── CreatePermission ─────────────────────────────────────────────────────────

func TestPermissionService_CreatePermission(t *testing.T) {
	ctx := context.Background()

	t.Run("crea permiso correctamente", func(t *testing.T) {
		resID := uuid.New()
		resource := &entities.Resource{ID: resID, Key: "users", DisplayName: "Users", Scope: "platform", IsActive: true}

		var captured *entities.Permission
		permRepo := &mockPermissionRepo{
			createFn: func(ctx context.Context, perm *entities.Permission) error {
				captured = perm
				return nil
			},
		}
		resRepo := &mockResourceRepo{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Resource, error) { return resource, nil },
		}

		svc := NewPermissionService(permRepo, resRepo, &mockLogger{}, &mockAuditLogger{})
		req := &dto.CreatePermissionRequest{
			Name:        "users:read",
			DisplayName: "Read Users",
			Description: "allows reading users",
			ResourceID:  resID.String(),
			Action:      "read",
			Scope:       "school",
		}
		resp, err := svc.CreatePermission(ctx, req)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if resp.Name != "users:read" {
			t.Errorf("nombre incorrecto: %s", resp.Name)
		}
		if resp.Action != "read" {
			t.Errorf("acción incorrecta: %s", resp.Action)
		}
		if resp.ResourceKey != "users" {
			t.Errorf("resource key incorrecto: %s", resp.ResourceKey)
		}
		if captured == nil {
			t.Fatal("no se llamó al repo")
		}
		if captured.Description == nil || *captured.Description != "allows reading users" {
			t.Errorf("descripción incorrecta")
		}
	})

	t.Run("retorna error con nombre inválido (formato incorrecto)", func(t *testing.T) {
		svc := NewPermissionService(&mockPermissionRepo{}, &mockResourceRepo{}, &mockLogger{}, &mockAuditLogger{})
		req := &dto.CreatePermissionRequest{Name: "INVALID", DisplayName: "Test", ResourceID: uuid.New().String(), Action: "read", Scope: "school"}
		_, err := svc.CreatePermission(ctx, req)
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})

	t.Run("acepta nombre con puntos en resource (admin.users:read)", func(t *testing.T) {
		resID := uuid.New()
		resource := &entities.Resource{ID: resID, Key: "admin.users", DisplayName: "Admin Users", Scope: "platform", IsActive: true}
		permRepo := &mockPermissionRepo{
			createFn: func(ctx context.Context, perm *entities.Permission) error { return nil },
		}
		resRepo := &mockResourceRepo{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Resource, error) { return resource, nil },
		}

		svc := NewPermissionService(permRepo, resRepo, &mockLogger{}, &mockAuditLogger{})
		req := &dto.CreatePermissionRequest{Name: "admin.users:read", DisplayName: "Read Admin Users", ResourceID: resID.String(), Action: "read", Scope: "school"}
		resp, err := svc.CreatePermission(ctx, req)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if resp.Name != "admin.users:read" {
			t.Errorf("nombre incorrecto: %s", resp.Name)
		}
	})

	t.Run("retorna error con resource_id inválido", func(t *testing.T) {
		svc := NewPermissionService(&mockPermissionRepo{}, &mockResourceRepo{}, &mockLogger{}, &mockAuditLogger{})
		req := &dto.CreatePermissionRequest{Name: "users:read", DisplayName: "Read", ResourceID: "bad-uuid", Action: "read", Scope: "school"}
		_, err := svc.CreatePermission(ctx, req)
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})

	t.Run("retorna not found cuando resource no existe", func(t *testing.T) {
		resRepo := &mockResourceRepo{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Resource, error) { return nil, nil },
		}
		svc := NewPermissionService(&mockPermissionRepo{}, resRepo, &mockLogger{}, &mockAuditLogger{})
		req := &dto.CreatePermissionRequest{Name: "users:read", DisplayName: "Read", ResourceID: uuid.New().String(), Action: "read", Scope: "school"}
		_, err := svc.CreatePermission(ctx, req)
		assertAppError(t, err, sharedErrors.ErrorCodeNotFound)
	})

	t.Run("retorna error cuando name no es consistente con resource key y action", func(t *testing.T) {
		resID := uuid.New()
		resource := &entities.Resource{ID: resID, Key: "users", DisplayName: "Users", Scope: "platform", IsActive: true}
		resRepo := &mockResourceRepo{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Resource, error) { return resource, nil },
		}
		svc := NewPermissionService(&mockPermissionRepo{}, resRepo, &mockLogger{}, &mockAuditLogger{})
		// name says "roles:read" but resource key is "users"
		req := &dto.CreatePermissionRequest{Name: "roles:read", DisplayName: "Read", ResourceID: resID.String(), Action: "read", Scope: "school"}
		_, err := svc.CreatePermission(ctx, req)
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})

	t.Run("propaga error de base de datos", func(t *testing.T) {
		resID := uuid.New()
		resource := &entities.Resource{ID: resID, Key: "users", DisplayName: "Users", Scope: "platform", IsActive: true}
		permRepo := &mockPermissionRepo{
			createFn: func(ctx context.Context, perm *entities.Permission) error { return errors.New("db error") },
		}
		resRepo := &mockResourceRepo{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Resource, error) { return resource, nil },
		}
		svc := NewPermissionService(permRepo, resRepo, &mockLogger{}, &mockAuditLogger{})
		req := &dto.CreatePermissionRequest{Name: "users:read", DisplayName: "Read", ResourceID: resID.String(), Action: "read", Scope: "school"}
		_, err := svc.CreatePermission(ctx, req)
		assertAppError(t, err, sharedErrors.ErrorCodeDatabaseError)
	})
}

// ─── UpdatePermission ─────────────────────────────────────────────────────────

func TestPermissionService_UpdatePermission(t *testing.T) {
	ctx := context.Background()

	t.Run("actualiza permiso correctamente", func(t *testing.T) {
		id := uuid.New()
		resID := uuid.New()
		perm := &entities.Permission{ID: id, Name: "users:read", DisplayName: "Read", ResourceID: resID, Action: "read", Scope: "school", IsActive: true}

		permRepo := &mockPermissionRepo{
			findByIDFn: func(ctx context.Context, gotID uuid.UUID) (*entities.Permission, error) { return perm, nil },
			updateFn:   func(ctx context.Context, p *entities.Permission) error { return nil },
		}
		svc := NewPermissionService(permRepo, &mockResourceRepo{}, &mockLogger{}, &mockAuditLogger{})

		newDisplay := "Read All Users"
		newDesc := "updated description"
		req := &dto.UpdatePermissionRequest{DisplayName: &newDisplay, Description: &newDesc}
		resp, err := svc.UpdatePermission(ctx, id.String(), req)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if resp.DisplayName != "Read All Users" {
			t.Errorf("display name no actualizado: %s", resp.DisplayName)
		}
		if resp.Description != "updated description" {
			t.Errorf("descripción no actualizada: %s", resp.Description)
		}
	})

	t.Run("retorna error con UUID inválido", func(t *testing.T) {
		svc := NewPermissionService(&mockPermissionRepo{}, &mockResourceRepo{}, &mockLogger{}, &mockAuditLogger{})
		_, err := svc.UpdatePermission(ctx, "bad-uuid", &dto.UpdatePermissionRequest{})
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})

	t.Run("retorna not found cuando permiso no existe", func(t *testing.T) {
		permRepo := &mockPermissionRepo{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Permission, error) { return nil, nil },
		}
		svc := NewPermissionService(permRepo, &mockResourceRepo{}, &mockLogger{}, &mockAuditLogger{})
		_, err := svc.UpdatePermission(ctx, uuid.New().String(), &dto.UpdatePermissionRequest{})
		assertAppError(t, err, sharedErrors.ErrorCodeNotFound)
	})

	t.Run("propaga error de base de datos en update", func(t *testing.T) {
		id := uuid.New()
		resID := uuid.New()
		perm := &entities.Permission{ID: id, Name: "users:read", DisplayName: "Read", ResourceID: resID, Action: "read", Scope: "school", IsActive: true}

		permRepo := &mockPermissionRepo{
			findByIDFn: func(ctx context.Context, gotID uuid.UUID) (*entities.Permission, error) { return perm, nil },
			updateFn:   func(ctx context.Context, p *entities.Permission) error { return errors.New("db error") },
		}
		svc := NewPermissionService(permRepo, &mockResourceRepo{}, &mockLogger{}, &mockAuditLogger{})
		_, err := svc.UpdatePermission(ctx, id.String(), &dto.UpdatePermissionRequest{})
		assertAppError(t, err, sharedErrors.ErrorCodeDatabaseError)
	})
}

// ─── DeletePermission ─────────────────────────────────────────────────────────

func TestPermissionService_DeletePermission(t *testing.T) {
	ctx := context.Background()

	t.Run("elimina permiso correctamente", func(t *testing.T) {
		id := uuid.New()
		resID := uuid.New()
		perm := &entities.Permission{ID: id, Name: "users:read", DisplayName: "Read", ResourceID: resID, Action: "read", Scope: "school", IsActive: true}

		var deletedID uuid.UUID
		permRepo := &mockPermissionRepo{
			findByIDFn: func(ctx context.Context, gotID uuid.UUID) (*entities.Permission, error) { return perm, nil },
			hasActiveRolePermissionsFn: func(ctx context.Context, pID uuid.UUID) (bool, error) { return false, nil },
			softDeleteFn: func(ctx context.Context, gotID uuid.UUID) error {
				deletedID = gotID
				return nil
			},
		}
		svc := NewPermissionService(permRepo, &mockResourceRepo{}, &mockLogger{}, &mockAuditLogger{})
		err := svc.DeletePermission(ctx, id.String())
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if deletedID != id {
			t.Errorf("ID de delete incorrecto: %s", deletedID)
		}
	})

	t.Run("retorna error con UUID inválido", func(t *testing.T) {
		svc := NewPermissionService(&mockPermissionRepo{}, &mockResourceRepo{}, &mockLogger{}, &mockAuditLogger{})
		err := svc.DeletePermission(ctx, "bad-uuid")
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})

	t.Run("retorna not found cuando permiso no existe", func(t *testing.T) {
		permRepo := &mockPermissionRepo{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Permission, error) { return nil, nil },
		}
		svc := NewPermissionService(permRepo, &mockResourceRepo{}, &mockLogger{}, &mockAuditLogger{})
		err := svc.DeletePermission(ctx, uuid.New().String())
		assertAppError(t, err, sharedErrors.ErrorCodeNotFound)
	})

	t.Run("retorna conflict cuando tiene role permissions activos", func(t *testing.T) {
		id := uuid.New()
		resID := uuid.New()
		perm := &entities.Permission{ID: id, Name: "users:read", DisplayName: "Read", ResourceID: resID, Action: "read", Scope: "school", IsActive: true}

		permRepo := &mockPermissionRepo{
			findByIDFn:                 func(ctx context.Context, gotID uuid.UUID) (*entities.Permission, error) { return perm, nil },
			hasActiveRolePermissionsFn: func(ctx context.Context, pID uuid.UUID) (bool, error) { return true, nil },
		}
		svc := NewPermissionService(permRepo, &mockResourceRepo{}, &mockLogger{}, &mockAuditLogger{})
		err := svc.DeletePermission(ctx, id.String())
		assertAppError(t, err, sharedErrors.ErrorCodeConflict)
	})

	t.Run("propaga error de base de datos en SoftDelete", func(t *testing.T) {
		id := uuid.New()
		resID := uuid.New()
		perm := &entities.Permission{ID: id, Name: "users:read", DisplayName: "Read", ResourceID: resID, Action: "read", Scope: "school", IsActive: true}

		permRepo := &mockPermissionRepo{
			findByIDFn:                 func(ctx context.Context, gotID uuid.UUID) (*entities.Permission, error) { return perm, nil },
			hasActiveRolePermissionsFn: func(ctx context.Context, pID uuid.UUID) (bool, error) { return false, nil },
			softDeleteFn:               func(ctx context.Context, gotID uuid.UUID) error { return errors.New("db error") },
		}
		svc := NewPermissionService(permRepo, &mockResourceRepo{}, &mockLogger{}, &mockAuditLogger{})
		err := svc.DeletePermission(ctx, id.String())
		assertAppError(t, err, sharedErrors.ErrorCodeDatabaseError)
	})
}

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

func newResourceService(repo *mockResourceRepo) ResourceService {
	return NewResourceService(repo, &mockLogger{})
}

// ─── ListResources ────────────────────────────────────────────────────────────

func TestResourceService_ListResources(t *testing.T) {
	ctx := context.Background()

	t.Run("retorna lista de recursos con total correcto", func(t *testing.T) {
		resources := []*entities.Resource{
			{ID: uuid.New(), Key: "dashboard", DisplayName: "Dashboard", Scope: "platform", IsActive: true, IsMenuVisible: true},
			{ID: uuid.New(), Key: "users", DisplayName: "Users", Scope: "platform", IsActive: true, IsMenuVisible: true},
		}
		repo := &mockResourceRepo{
			findAllFn: func(ctx context.Context, filters sharedrepo.ListFilters) ([]*entities.Resource, error) { return resources, nil },
		}

		svc := newResourceService(repo)
		resp, err := svc.ListResources(ctx, sharedrepo.ListFilters{})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if len(resp.Resources) != 2 {
			t.Errorf("esperaba 2 recursos, obtuvo %d", len(resp.Resources))
		}
		if resp.Total != 2 {
			t.Errorf("total incorrecto: %d", resp.Total)
		}
	})

	t.Run("retorna lista vacía correctamente", func(t *testing.T) {
		repo := &mockResourceRepo{
			findAllFn: func(ctx context.Context, filters sharedrepo.ListFilters) ([]*entities.Resource, error) { return []*entities.Resource{}, nil },
		}
		svc := newResourceService(repo)
		resp, err := svc.ListResources(ctx, sharedrepo.ListFilters{})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if resp.Total != 0 {
			t.Errorf("total esperado 0, obtuvo %d", resp.Total)
		}
	})

	t.Run("propaga error de base de datos", func(t *testing.T) {
		repo := &mockResourceRepo{
			findAllFn: func(ctx context.Context, filters sharedrepo.ListFilters) ([]*entities.Resource, error) { return nil, errors.New("db fail") },
		}
		svc := newResourceService(repo)
		_, err := svc.ListResources(ctx, sharedrepo.ListFilters{})
		assertAppError(t, err, sharedErrors.ErrorCodeDatabaseError)
	})
}

// ─── GetResource ─────────────────────────────────────────────────────────────

func TestResourceService_GetResource(t *testing.T) {
	ctx := context.Background()

	t.Run("retorna recurso existente", func(t *testing.T) {
		id := uuid.New()
		parentID := uuid.New()
		icon := "home"
		desc := "pagina principal"
		resource := &entities.Resource{
			ID: id, Key: "dashboard", DisplayName: "Dashboard",
			Description: &desc, Icon: &icon, ParentID: &parentID,
			Scope: "platform", IsActive: true, IsMenuVisible: true,
		}
		repo := &mockResourceRepo{
			findByIDFn: func(ctx context.Context, gotID uuid.UUID) (*entities.Resource, error) {
				if gotID != id {
					t.Errorf("ID incorrecto: %s", gotID)
				}
				return resource, nil
			},
		}

		svc := newResourceService(repo)
		resp, err := svc.GetResource(ctx, id.String())
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if resp.ID != id.String() {
			t.Errorf("ID incorrecto: %s", resp.ID)
		}
		if resp.Icon != icon {
			t.Errorf("icon incorrecto: %s", resp.Icon)
		}
		if resp.Description != desc {
			t.Errorf("descripción incorrecta: %s", resp.Description)
		}
		if resp.ParentID == nil || *resp.ParentID != parentID.String() {
			t.Errorf("parentID incorrecto")
		}
	})

	t.Run("retorna error de validación con UUID inválido", func(t *testing.T) {
		svc := newResourceService(&mockResourceRepo{})
		_, err := svc.GetResource(ctx, "bad-uuid")
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})

	t.Run("retorna not found cuando el recurso no existe", func(t *testing.T) {
		repo := &mockResourceRepo{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Resource, error) { return nil, nil },
		}
		svc := newResourceService(repo)
		_, err := svc.GetResource(ctx, uuid.New().String())
		assertAppError(t, err, sharedErrors.ErrorCodeNotFound)
	})

	t.Run("propaga error de base de datos", func(t *testing.T) {
		repo := &mockResourceRepo{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Resource, error) { return nil, errors.New("db fail") },
		}
		svc := newResourceService(repo)
		_, err := svc.GetResource(ctx, uuid.New().String())
		assertAppError(t, err, sharedErrors.ErrorCodeDatabaseError)
	})
}

// ─── CreateResource ───────────────────────────────────────────────────────────

func TestResourceService_CreateResource(t *testing.T) {
	ctx := context.Background()

	t.Run("crea recurso sin parentID", func(t *testing.T) {
		var created *entities.Resource
		repo := &mockResourceRepo{
			createFn: func(ctx context.Context, r *entities.Resource) error {
				created = r
				return nil
			},
		}
		svc := newResourceService(repo)

		req := dto.CreateResourceRequest{
			Key: "dashboard", DisplayName: "Dashboard",
			Scope: "platform", IsMenuVisible: true, SortOrder: 1,
		}
		resp, err := svc.CreateResource(ctx, req)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if resp.Key != "dashboard" {
			t.Errorf("key incorrecta: %s", resp.Key)
		}
		if !resp.IsActive {
			t.Error("el recurso debería estar activo")
		}
		if created == nil {
			t.Error("el recurso no fue pasado al repositorio")
		}
	})

	t.Run("crea recurso con parentID válido", func(t *testing.T) {
		parentID := uuid.New().String()
		repo := &mockResourceRepo{
			createFn: func(ctx context.Context, r *entities.Resource) error { return nil },
		}
		svc := newResourceService(repo)

		req := dto.CreateResourceRequest{
			Key: "sub-menu", DisplayName: "Sub Menu",
			Scope: "school", ParentID: &parentID,
		}
		resp, err := svc.CreateResource(ctx, req)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if resp.ParentID == nil || *resp.ParentID != parentID {
			t.Errorf("parentID incorrecto: %v", resp.ParentID)
		}
	})

	t.Run("retorna error con parentID inválido", func(t *testing.T) {
		svc := newResourceService(&mockResourceRepo{})
		bad := "not-valid"
		req := dto.CreateResourceRequest{
			Key: "test", DisplayName: "Test", Scope: "platform", ParentID: &bad,
		}
		_, err := svc.CreateResource(ctx, req)
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})

	t.Run("crea recurso con description e icon", func(t *testing.T) {
		repo := &mockResourceRepo{
			createFn: func(ctx context.Context, r *entities.Resource) error { return nil },
		}
		svc := newResourceService(repo)
		req := dto.CreateResourceRequest{
			Key: "settings", DisplayName: "Settings",
			Description: "configuración", Icon: "gear", Scope: "platform",
		}
		resp, err := svc.CreateResource(ctx, req)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if resp.Description != "configuración" {
			t.Errorf("descripción incorrecta: %s", resp.Description)
		}
		if resp.Icon != "gear" {
			t.Errorf("icon incorrecto: %s", resp.Icon)
		}
	})

	t.Run("propaga error de base de datos", func(t *testing.T) {
		repo := &mockResourceRepo{
			createFn: func(ctx context.Context, r *entities.Resource) error { return errors.New("db fail") },
		}
		svc := newResourceService(repo)
		req := dto.CreateResourceRequest{Key: "test", DisplayName: "Test", Scope: "platform"}
		_, err := svc.CreateResource(ctx, req)
		assertAppError(t, err, sharedErrors.ErrorCodeDatabaseError)
	})
}

// ─── UpdateResource ───────────────────────────────────────────────────────────

func TestResourceService_UpdateResource(t *testing.T) {
	ctx := context.Background()

	t.Run("actualiza campos provistos", func(t *testing.T) {
		id := uuid.New()
		existing := &entities.Resource{
			ID: id, Key: "dashboard", DisplayName: "Old Name",
			Scope: "platform", IsActive: true,
		}
		repo := &mockResourceRepo{
			findByIDFn: func(ctx context.Context, gotID uuid.UUID) (*entities.Resource, error) { return existing, nil },
			updateFn:   func(ctx context.Context, r *entities.Resource) error { return nil },
		}
		svc := newResourceService(repo)

		newName := "New Name"
		req := dto.UpdateResourceRequest{DisplayName: &newName}
		resp, err := svc.UpdateResource(ctx, id.String(), req)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if resp.DisplayName != newName {
			t.Errorf("nombre incorrecto: %s", resp.DisplayName)
		}
	})

	t.Run("limpia parentID cuando se envía string vacío", func(t *testing.T) {
		id := uuid.New()
		parentID := uuid.New()
		existing := &entities.Resource{
			ID: id, Key: "sub", DisplayName: "Sub", ParentID: &parentID,
			Scope: "platform", IsActive: true,
		}
		repo := &mockResourceRepo{
			findByIDFn: func(ctx context.Context, gotID uuid.UUID) (*entities.Resource, error) { return existing, nil },
			updateFn:   func(ctx context.Context, r *entities.Resource) error { return nil },
		}
		svc := newResourceService(repo)

		emptyStr := ""
		req := dto.UpdateResourceRequest{ParentID: &emptyStr}
		resp, err := svc.UpdateResource(ctx, id.String(), req)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if resp.ParentID != nil {
			t.Errorf("parentID debería ser nil, obtuvo: %v", resp.ParentID)
		}
	})

	t.Run("retorna error de validación con ID inválido", func(t *testing.T) {
		svc := newResourceService(&mockResourceRepo{})
		_, err := svc.UpdateResource(ctx, "bad-uuid", dto.UpdateResourceRequest{})
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})

	t.Run("retorna not found cuando el recurso no existe", func(t *testing.T) {
		repo := &mockResourceRepo{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Resource, error) { return nil, nil },
		}
		svc := newResourceService(repo)
		_, err := svc.UpdateResource(ctx, uuid.New().String(), dto.UpdateResourceRequest{})
		assertAppError(t, err, sharedErrors.ErrorCodeNotFound)
	})

	t.Run("retorna error con parentID inválido en update", func(t *testing.T) {
		id := uuid.New()
		existing := &entities.Resource{ID: id, Key: "k", DisplayName: "K", Scope: "platform", IsActive: true}
		repo := &mockResourceRepo{
			findByIDFn: func(ctx context.Context, gotID uuid.UUID) (*entities.Resource, error) { return existing, nil },
		}
		svc := newResourceService(repo)
		bad := "not-uuid"
		req := dto.UpdateResourceRequest{ParentID: &bad}
		_, err := svc.UpdateResource(ctx, id.String(), req)
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})

	t.Run("propaga error de base de datos en update", func(t *testing.T) {
		id := uuid.New()
		existing := &entities.Resource{ID: id, Key: "k", DisplayName: "K", Scope: "platform", IsActive: true}
		repo := &mockResourceRepo{
			findByIDFn: func(ctx context.Context, gotID uuid.UUID) (*entities.Resource, error) { return existing, nil },
			updateFn:   func(ctx context.Context, r *entities.Resource) error { return errors.New("db fail") },
		}
		svc := newResourceService(repo)
		newName := "Updated"
		req := dto.UpdateResourceRequest{DisplayName: &newName}
		_, err := svc.UpdateResource(ctx, id.String(), req)
		assertAppError(t, err, sharedErrors.ErrorCodeDatabaseError)
	})
}

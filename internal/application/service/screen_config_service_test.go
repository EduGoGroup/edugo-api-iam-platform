package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	domainrepo "github.com/EduGoGroup/edugo-api-iam-platform/internal/domain/repository"
	"github.com/EduGoGroup/edugo-infrastructure/postgres/entities"
	sharedErrors "github.com/EduGoGroup/edugo-shared/common/errors"
	"github.com/google/uuid"
)

func newScreenConfigService(
	tplRepo *mockScreenTemplateRepo,
	instRepo *mockScreenInstanceRepo,
	rsRepo *mockResourceScreenRepo,
) ScreenConfigService {
	return NewScreenConfigService(tplRepo, instRepo, rsRepo, &mockLogger{})
}

func sampleDefinition() json.RawMessage {
	return json.RawMessage(`{"type":"list","columns":[]}`)
}

// ─── CreateTemplate ───────────────────────────────────────────────────────────

func TestScreenConfigService_CreateTemplate(t *testing.T) {
	ctx := context.Background()

	t.Run("crea template correctamente", func(t *testing.T) {
		var saved *entities.ScreenTemplate
		tplRepo := &mockScreenTemplateRepo{
			createFn: func(ctx context.Context, t *entities.ScreenTemplate) error {
				saved = t
				return nil
			},
		}

		svc := newScreenConfigService(tplRepo, &mockScreenInstanceRepo{}, &mockResourceScreenRepo{})
		req := &CreateTemplateRequest{
			Pattern:    "list-view",
			Name:       "List View",
			Definition: sampleDefinition(),
		}
		resp, err := svc.CreateTemplate(ctx, req)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if resp.Pattern != "list-view" {
			t.Errorf("pattern incorrecto: %s", resp.Pattern)
		}
		if resp.Version != 1 {
			t.Errorf("versión inicial debería ser 1, obtuvo %d", resp.Version)
		}
		if !resp.IsActive {
			t.Error("el template debería estar activo")
		}
		if saved == nil {
			t.Error("el template no fue pasado al repositorio")
		}
	})

	t.Run("crea template con descripción", func(t *testing.T) {
		tplRepo := &mockScreenTemplateRepo{
			createFn: func(ctx context.Context, t *entities.ScreenTemplate) error { return nil },
		}
		svc := newScreenConfigService(tplRepo, &mockScreenInstanceRepo{}, &mockResourceScreenRepo{})
		req := &CreateTemplateRequest{
			Pattern:     "detail-view",
			Name:        "Detail",
			Description: "vista detallada",
			Definition:  sampleDefinition(),
		}
		resp, err := svc.CreateTemplate(ctx, req)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if resp.Description != "vista detallada" {
			t.Errorf("descripción incorrecta: %s", resp.Description)
		}
	})

	t.Run("propaga error de base de datos", func(t *testing.T) {
		tplRepo := &mockScreenTemplateRepo{
			createFn: func(ctx context.Context, t *entities.ScreenTemplate) error { return errors.New("db fail") },
		}
		svc := newScreenConfigService(tplRepo, &mockScreenInstanceRepo{}, &mockResourceScreenRepo{})
		req := &CreateTemplateRequest{Pattern: "p", Name: "n", Definition: sampleDefinition()}
		_, err := svc.CreateTemplate(ctx, req)
		assertAppError(t, err, sharedErrors.ErrorCodeDatabaseError)
	})
}

// ─── GetTemplate ─────────────────────────────────────────────────────────────

func TestScreenConfigService_GetTemplate(t *testing.T) {
	ctx := context.Background()

	t.Run("retorna template existente", func(t *testing.T) {
		id := uuid.New()
		tpl := &entities.ScreenTemplate{ID: id, Pattern: "list", Name: "List", Version: 2, IsActive: true, Definition: sampleDefinition()}
		tplRepo := &mockScreenTemplateRepo{
			getByIDFn: func(ctx context.Context, gotID uuid.UUID) (*entities.ScreenTemplate, error) { return tpl, nil },
		}

		svc := newScreenConfigService(tplRepo, &mockScreenInstanceRepo{}, &mockResourceScreenRepo{})
		resp, err := svc.GetTemplate(ctx, id.String())
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if resp.ID != id.String() {
			t.Errorf("ID incorrecto: %s", resp.ID)
		}
		if resp.Version != 2 {
			t.Errorf("versión incorrecta: %d", resp.Version)
		}
	})

	t.Run("retorna error de validación con UUID inválido", func(t *testing.T) {
		svc := newScreenConfigService(&mockScreenTemplateRepo{}, &mockScreenInstanceRepo{}, &mockResourceScreenRepo{})
		_, err := svc.GetTemplate(ctx, "bad-uuid")
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})
}

// ─── ListTemplates ────────────────────────────────────────────────────────────

func TestScreenConfigService_ListTemplates(t *testing.T) {
	ctx := context.Background()

	t.Run("retorna lista paginada", func(t *testing.T) {
		templates := []*entities.ScreenTemplate{
			{ID: uuid.New(), Pattern: "list", Name: "List", Version: 1, IsActive: true, Definition: sampleDefinition()},
			{ID: uuid.New(), Pattern: "detail", Name: "Detail", Version: 1, IsActive: true, Definition: sampleDefinition()},
		}
		var capturedFilter domainrepo.ScreenTemplateFilter
		tplRepo := &mockScreenTemplateRepo{
			listFn: func(ctx context.Context, f domainrepo.ScreenTemplateFilter) ([]*entities.ScreenTemplate, int, error) {
				capturedFilter = f
				return templates, 10, nil
			},
		}

		svc := newScreenConfigService(tplRepo, &mockScreenInstanceRepo{}, &mockResourceScreenRepo{})
		dtos, total, err := svc.ListTemplates(ctx, TemplateFilter{Page: 2, PerPage: 5, Pattern: "list"})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if total != 10 {
			t.Errorf("total incorrecto: %d", total)
		}
		if len(dtos) != 2 {
			t.Errorf("esperaba 2 DTOs, obtuvo %d", len(dtos))
		}
		if capturedFilter.Offset != 5 {
			t.Errorf("offset incorrecto: %d (esperaba 5 para page=2, per_page=5)", capturedFilter.Offset)
		}
		if capturedFilter.Limit != 5 {
			t.Errorf("limit incorrecto: %d", capturedFilter.Limit)
		}
		if capturedFilter.Pattern != "list" {
			t.Errorf("pattern incorrecto: %s", capturedFilter.Pattern)
		}
	})

	t.Run("aplica valores por defecto cuando page y per_page son 0", func(t *testing.T) {
		var capturedFilter domainrepo.ScreenTemplateFilter
		tplRepo := &mockScreenTemplateRepo{
			listFn: func(ctx context.Context, f domainrepo.ScreenTemplateFilter) ([]*entities.ScreenTemplate, int, error) {
				capturedFilter = f
				return nil, 0, nil
			},
		}
		svc := newScreenConfigService(tplRepo, &mockScreenInstanceRepo{}, &mockResourceScreenRepo{})
		_, _, err := svc.ListTemplates(ctx, TemplateFilter{})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if capturedFilter.Limit != 20 {
			t.Errorf("limit por defecto incorrecto: %d", capturedFilter.Limit)
		}
		if capturedFilter.Offset != 0 {
			t.Errorf("offset por defecto incorrecto: %d", capturedFilter.Offset)
		}
	})
}

// ─── UpdateTemplate ───────────────────────────────────────────────────────────

func TestScreenConfigService_UpdateTemplate(t *testing.T) {
	ctx := context.Background()

	t.Run("actualiza campos y versión cuando definition cambia", func(t *testing.T) {
		id := uuid.New()
		tpl := &entities.ScreenTemplate{ID: id, Pattern: "list", Name: "Old", Version: 1, IsActive: true, Definition: sampleDefinition()}
		tplRepo := &mockScreenTemplateRepo{
			getByIDFn: func(ctx context.Context, gotID uuid.UUID) (*entities.ScreenTemplate, error) { return tpl, nil },
			updateFn:  func(ctx context.Context, t *entities.ScreenTemplate) error { return nil },
		}

		svc := newScreenConfigService(tplRepo, &mockScreenInstanceRepo{}, &mockResourceScreenRepo{})
		newName := "New Name"
		newDef := json.RawMessage(`{"type":"detail"}`)
		req := &UpdateTemplateRequest{Name: &newName, Definition: &newDef}
		resp, err := svc.UpdateTemplate(ctx, id.String(), req)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if resp.Name != newName {
			t.Errorf("nombre incorrecto: %s", resp.Name)
		}
		if resp.Version != 2 {
			t.Errorf("versión debería incrementar a 2, obtuvo %d", resp.Version)
		}
	})

	t.Run("no incrementa versión sin cambio de definition", func(t *testing.T) {
		id := uuid.New()
		tpl := &entities.ScreenTemplate{ID: id, Pattern: "list", Name: "Old", Version: 3, IsActive: true, Definition: sampleDefinition()}
		tplRepo := &mockScreenTemplateRepo{
			getByIDFn: func(ctx context.Context, gotID uuid.UUID) (*entities.ScreenTemplate, error) { return tpl, nil },
			updateFn:  func(ctx context.Context, t *entities.ScreenTemplate) error { return nil },
		}

		svc := newScreenConfigService(tplRepo, &mockScreenInstanceRepo{}, &mockResourceScreenRepo{})
		newPattern := "list-v2"
		req := &UpdateTemplateRequest{Pattern: &newPattern}
		resp, err := svc.UpdateTemplate(ctx, id.String(), req)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if resp.Version != 3 {
			t.Errorf("versión no debería cambiar: %d", resp.Version)
		}
	})

	t.Run("retorna error de validación con UUID inválido", func(t *testing.T) {
		svc := newScreenConfigService(&mockScreenTemplateRepo{}, &mockScreenInstanceRepo{}, &mockResourceScreenRepo{})
		_, err := svc.UpdateTemplate(ctx, "bad", &UpdateTemplateRequest{})
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})
}

// ─── DeleteTemplate ───────────────────────────────────────────────────────────

func TestScreenConfigService_DeleteTemplate(t *testing.T) {
	ctx := context.Background()

	t.Run("elimina template existente", func(t *testing.T) {
		id := uuid.New()
		tpl := &entities.ScreenTemplate{ID: id, Pattern: "p", Name: "n", Version: 1, IsActive: true, Definition: sampleDefinition()}
		var deletedID uuid.UUID
		tplRepo := &mockScreenTemplateRepo{
			getByIDFn: func(ctx context.Context, gotID uuid.UUID) (*entities.ScreenTemplate, error) { return tpl, nil },
			deleteFn:  func(ctx context.Context, gotID uuid.UUID) error { deletedID = gotID; return nil },
		}

		svc := newScreenConfigService(tplRepo, &mockScreenInstanceRepo{}, &mockResourceScreenRepo{})
		err := svc.DeleteTemplate(ctx, id.String())
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if deletedID != id {
			t.Errorf("ID incorrecto pasado al delete: %s", deletedID)
		}
	})

	t.Run("retorna error de validación con UUID inválido", func(t *testing.T) {
		svc := newScreenConfigService(&mockScreenTemplateRepo{}, &mockScreenInstanceRepo{}, &mockResourceScreenRepo{})
		err := svc.DeleteTemplate(ctx, "bad-uuid")
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})
}

// ─── CreateInstance ───────────────────────────────────────────────────────────

func TestScreenConfigService_CreateInstance(t *testing.T) {
	ctx := context.Background()

	t.Run("crea instancia correctamente", func(t *testing.T) {
		templateID := uuid.New()
		tpl := &entities.ScreenTemplate{ID: templateID, Pattern: "list", Name: "List", Version: 1, IsActive: true, Definition: sampleDefinition()}
		tplRepo := &mockScreenTemplateRepo{
			getByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.ScreenTemplate, error) { return tpl, nil },
		}
		instRepo := &mockScreenInstanceRepo{
			createFn: func(ctx context.Context, inst *entities.ScreenInstance) error { return nil },
		}

		svc := newScreenConfigService(tplRepo, instRepo, &mockResourceScreenRepo{})
		req := &CreateInstanceRequest{
			ScreenKey:  "dashboard-list",
			TemplateID: templateID.String(),
			Name:       "Dashboard List",
		}
		resp, err := svc.CreateInstance(ctx, req)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if resp.ScreenKey != "dashboard-list" {
			t.Errorf("screen_key incorrecto: %s", resp.ScreenKey)
		}
		if resp.TemplateID != templateID.String() {
			t.Errorf("template_id incorrecto: %s", resp.TemplateID)
		}
	})

	t.Run("usa '{}' como slot_data por defecto cuando es nil", func(t *testing.T) {
		templateID := uuid.New()
		tpl := &entities.ScreenTemplate{ID: templateID, Pattern: "list", Name: "List", Version: 1, IsActive: true, Definition: sampleDefinition()}
		var savedInst *entities.ScreenInstance
		tplRepo := &mockScreenTemplateRepo{
			getByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.ScreenTemplate, error) { return tpl, nil },
		}
		instRepo := &mockScreenInstanceRepo{
			createFn: func(ctx context.Context, inst *entities.ScreenInstance) error {
				savedInst = inst
				return nil
			},
		}

		svc := newScreenConfigService(tplRepo, instRepo, &mockResourceScreenRepo{})
		req := &CreateInstanceRequest{
			ScreenKey:  "test-key",
			TemplateID: templateID.String(),
			Name:       "Test",
		}
		_, err := svc.CreateInstance(ctx, req)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if string(savedInst.SlotData) != "{}" {
			t.Errorf("slot_data por defecto incorrecto: %s", string(savedInst.SlotData))
		}
	})

	t.Run("retorna error de validación con template_id inválido", func(t *testing.T) {
		svc := newScreenConfigService(&mockScreenTemplateRepo{}, &mockScreenInstanceRepo{}, &mockResourceScreenRepo{})
		req := &CreateInstanceRequest{ScreenKey: "k", TemplateID: "bad-uuid", Name: "n"}
		_, err := svc.CreateInstance(ctx, req)
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})

	t.Run("retorna error cuando template no existe", func(t *testing.T) {
		tplRepo := &mockScreenTemplateRepo{
			getByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.ScreenTemplate, error) {
				return nil, errors.New("not found")
			},
		}
		svc := newScreenConfigService(tplRepo, &mockScreenInstanceRepo{}, &mockResourceScreenRepo{})
		req := &CreateInstanceRequest{ScreenKey: "k", TemplateID: uuid.New().String(), Name: "n"}
		_, err := svc.CreateInstance(ctx, req)
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})
}

// ─── GetInstance ──────────────────────────────────────────────────────────────

func TestScreenConfigService_GetInstance(t *testing.T) {
	ctx := context.Background()

	t.Run("retorna instancia existente", func(t *testing.T) {
		id := uuid.New()
		templateID := uuid.New()
		inst := &entities.ScreenInstance{
			ID: id, ScreenKey: "test-screen", TemplateID: templateID,
			Name: "Test", SlotData: json.RawMessage(`{}`), Scope: "system", IsActive: true,
		}
		instRepo := &mockScreenInstanceRepo{
			getByIDFn: func(ctx context.Context, gotID uuid.UUID) (*entities.ScreenInstance, error) { return inst, nil },
		}

		svc := newScreenConfigService(&mockScreenTemplateRepo{}, instRepo, &mockResourceScreenRepo{})
		resp, err := svc.GetInstance(ctx, id.String())
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if resp.ScreenKey != "test-screen" {
			t.Errorf("screen_key incorrecto: %s", resp.ScreenKey)
		}
	})

	t.Run("retorna error de validación con UUID inválido", func(t *testing.T) {
		svc := newScreenConfigService(&mockScreenTemplateRepo{}, &mockScreenInstanceRepo{}, &mockResourceScreenRepo{})
		_, err := svc.GetInstance(ctx, "bad-uuid")
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})
}

// ─── ResolveScreenByKey ───────────────────────────────────────────────────────

func TestScreenConfigService_ResolveScreenByKey(t *testing.T) {
	ctx := context.Background()

	t.Run("combina instancia y template correctamente", func(t *testing.T) {
		templateID := uuid.New()
		instanceID := uuid.New()
		handlerKey := "my-handler"
		now := time.Now()
		inst := &entities.ScreenInstance{
			ID: instanceID, ScreenKey: "my-screen", TemplateID: templateID,
			Name: "My Screen", SlotData: json.RawMessage(`{"col":1}`),
			Scope: "system", IsActive: true, UpdatedAt: now, HandlerKey: &handlerKey,
		}
		tpl := &entities.ScreenTemplate{
			ID: templateID, Pattern: "list", Name: "List",
			Version: 3, Definition: sampleDefinition(), IsActive: true,
		}
		instRepo := &mockScreenInstanceRepo{
			getByScreenKeyFn: func(ctx context.Context, key string) (*entities.ScreenInstance, error) { return inst, nil },
		}
		tplRepo := &mockScreenTemplateRepo{
			getByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.ScreenTemplate, error) { return tpl, nil },
		}

		svc := newScreenConfigService(tplRepo, instRepo, &mockResourceScreenRepo{})
		resp, err := svc.ResolveScreenByKey(ctx, "my-screen")
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if resp.ScreenKey != "my-screen" {
			t.Errorf("screen_key incorrecto: %s", resp.ScreenKey)
		}
		if resp.Pattern != "list" {
			t.Errorf("pattern incorrecto: %s", resp.Pattern)
		}
		if resp.Version != 3 {
			t.Errorf("versión incorrecta: %d", resp.Version)
		}
		if resp.HandlerKey == nil || *resp.HandlerKey != handlerKey {
			t.Errorf("handler_key incorrecto")
		}
	})
}

// ─── LinkScreenToResource ─────────────────────────────────────────────────────

func TestScreenConfigService_LinkScreenToResource(t *testing.T) {
	ctx := context.Background()

	t.Run("vincula pantalla a recurso correctamente", func(t *testing.T) {
		resourceID := uuid.New()
		rsRepo := &mockResourceScreenRepo{
			createFn: func(ctx context.Context, rs *entities.ResourceScreen) error { return nil },
		}

		svc := newScreenConfigService(&mockScreenTemplateRepo{}, &mockScreenInstanceRepo{}, rsRepo)
		req := &LinkScreenRequest{
			ResourceID:  resourceID.String(),
			ResourceKey: "dashboard",
			ScreenKey:   "dashboard-main",
			ScreenType:  "main",
			IsDefault:   true,
		}
		resp, err := svc.LinkScreenToResource(ctx, req)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if resp.ResourceKey != "dashboard" {
			t.Errorf("resource_key incorrecto: %s", resp.ResourceKey)
		}
		if resp.ScreenKey != "dashboard-main" {
			t.Errorf("screen_key incorrecto: %s", resp.ScreenKey)
		}
		if !resp.IsDefault {
			t.Error("is_default debería ser true")
		}
	})

	t.Run("retorna error con resource_id inválido", func(t *testing.T) {
		svc := newScreenConfigService(&mockScreenTemplateRepo{}, &mockScreenInstanceRepo{}, &mockResourceScreenRepo{})
		req := &LinkScreenRequest{ResourceID: "bad-uuid", ResourceKey: "k", ScreenKey: "sk", ScreenType: "main"}
		_, err := svc.LinkScreenToResource(ctx, req)
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})
}

// ─── GetScreensForResource ────────────────────────────────────────────────────

func TestScreenConfigService_GetScreensForResource(t *testing.T) {
	ctx := context.Background()

	t.Run("retorna pantallas del recurso", func(t *testing.T) {
		resourceID := uuid.New()
		screens := []*entities.ResourceScreen{
			{ID: uuid.New(), ResourceID: resourceID, ResourceKey: "dashboard", ScreenKey: "dash-list", ScreenType: "list", IsDefault: true},
			{ID: uuid.New(), ResourceID: resourceID, ResourceKey: "dashboard", ScreenKey: "dash-detail", ScreenType: "detail", IsDefault: false},
		}
		rsRepo := &mockResourceScreenRepo{
			getByResourceIDFn: func(ctx context.Context, resID uuid.UUID) ([]*entities.ResourceScreen, error) {
				if resID != resourceID {
					t.Errorf("resourceID incorrecto: %s", resID)
				}
				return screens, nil
			},
		}

		svc := newScreenConfigService(&mockScreenTemplateRepo{}, &mockScreenInstanceRepo{}, rsRepo)
		dtos, err := svc.GetScreensForResource(ctx, resourceID.String())
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if len(dtos) != 2 {
			t.Errorf("esperaba 2 pantallas, obtuvo %d", len(dtos))
		}
	})

	t.Run("retorna error con resource_id inválido", func(t *testing.T) {
		svc := newScreenConfigService(&mockScreenTemplateRepo{}, &mockScreenInstanceRepo{}, &mockResourceScreenRepo{})
		_, err := svc.GetScreensForResource(ctx, "bad-uuid")
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})
}

// ─── UnlinkScreen ─────────────────────────────────────────────────────────────

func TestScreenConfigService_UnlinkScreen(t *testing.T) {
	ctx := context.Background()

	t.Run("desvincula pantalla correctamente", func(t *testing.T) {
		id := uuid.New()
		var deletedID uuid.UUID
		rsRepo := &mockResourceScreenRepo{
			deleteFn: func(ctx context.Context, gotID uuid.UUID) error {
				deletedID = gotID
				return nil
			},
		}

		svc := newScreenConfigService(&mockScreenTemplateRepo{}, &mockScreenInstanceRepo{}, rsRepo)
		err := svc.UnlinkScreen(ctx, id.String())
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if deletedID != id {
			t.Errorf("ID incorrecto pasado al delete: %s", deletedID)
		}
	})

	t.Run("retorna error con UUID inválido", func(t *testing.T) {
		svc := newScreenConfigService(&mockScreenTemplateRepo{}, &mockScreenInstanceRepo{}, &mockResourceScreenRepo{})
		err := svc.UnlinkScreen(ctx, "bad-uuid")
		assertAppError(t, err, sharedErrors.ErrorCodeValidation)
	})
}

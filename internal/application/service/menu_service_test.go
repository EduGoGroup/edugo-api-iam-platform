package service

import (
	"context"
	"errors"
	"testing"

	"github.com/EduGoGroup/edugo-infrastructure/postgres/entities"
	sharedErrors "github.com/EduGoGroup/edugo-shared/common/errors"
	"github.com/google/uuid"
)

func newMenuService(resourceRepo *mockResourceRepo, screenRepo *mockResourceScreenRepo) MenuService {
	return NewMenuService(resourceRepo, screenRepo, &mockLogger{})
}

// ─── extractResourceKeys (función pura interna) ───────────────────────────────

func TestExtractResourceKeys(t *testing.T) {
	t.Run("extrae claves únicas de permisos", func(t *testing.T) {
		perms := []string{"dashboard:read", "dashboard:write", "users:read", "roles:view"}
		keys := extractResourceKeys(perms)
		if len(keys) != 3 {
			t.Errorf("esperaba 3 claves únicas, obtuvo %d: %v", len(keys), keys)
		}
	})

	t.Run("retorna slice vacío con permisos vacíos", func(t *testing.T) {
		keys := extractResourceKeys([]string{})
		if len(keys) != 0 {
			t.Errorf("esperaba 0 claves, obtuvo %d", len(keys))
		}
	})

	t.Run("ignora permisos sin separador ':'", func(t *testing.T) {
		perms := []string{"no-separator", "valid:read"}
		keys := extractResourceKeys(perms)
		if len(keys) != 1 {
			t.Errorf("esperaba 1 clave, obtuvo %d: %v", len(keys), keys)
		}
		if keys[0] != "valid" {
			t.Errorf("clave incorrecta: %s", keys[0])
		}
	})

	t.Run("no duplica claves", func(t *testing.T) {
		perms := []string{"res:read", "res:write", "res:delete"}
		keys := extractResourceKeys(perms)
		if len(keys) != 1 {
			t.Errorf("esperaba 1 clave única, obtuvo %d", len(keys))
		}
	})
}

// ─── GetMenuForUser ───────────────────────────────────────────────────────────

func TestMenuService_GetMenuForUser(t *testing.T) {
	ctx := context.Background()

	t.Run("retorna menú vacío cuando no hay permisos", func(t *testing.T) {
		svc := newMenuService(&mockResourceRepo{}, nil)
		resp, err := svc.GetMenuForUser(ctx, []string{})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if len(resp.Items) != 0 {
			t.Errorf("esperaba 0 items, obtuvo %d", len(resp.Items))
		}
	})

	t.Run("retorna items filtrados por permisos del usuario", func(t *testing.T) {
		dashID := uuid.New()
		usersID := uuid.New()
		icon := "dashboard-icon"
		resources := []*entities.Resource{
			{ID: dashID, Key: "dashboard", DisplayName: "Dashboard", Icon: &icon, Scope: "platform", IsMenuVisible: true, SortOrder: 1},
			{ID: usersID, Key: "users", DisplayName: "Users", Scope: "platform", IsMenuVisible: true, SortOrder: 2},
		}
		resourceRepo := &mockResourceRepo{
			findMenuVisibleFn: func(ctx context.Context) ([]*entities.Resource, error) { return resources, nil },
		}
		screenRepo := &mockResourceScreenRepo{
			getByResourceKeyFn: func(ctx context.Context, key string) ([]*entities.ResourceScreen, error) {
				return nil, nil
			},
		}

		svc := newMenuService(resourceRepo, screenRepo)
		resp, err := svc.GetMenuForUser(ctx, []string{"dashboard:read", "dashboard:write"})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if len(resp.Items) != 1 {
			t.Errorf("esperaba 1 item (solo dashboard), obtuvo %d", len(resp.Items))
		}
		if resp.Items[0].Key != "dashboard" {
			t.Errorf("key incorrecta: %s", resp.Items[0].Key)
		}
		if resp.Items[0].Icon != icon {
			t.Errorf("icon incorrecto: %s", resp.Items[0].Icon)
		}
	})

	t.Run("incluye permisos agrupados por recurso", func(t *testing.T) {
		dashID := uuid.New()
		resources := []*entities.Resource{
			{ID: dashID, Key: "dashboard", DisplayName: "Dashboard", Scope: "platform", IsMenuVisible: true},
		}
		resourceRepo := &mockResourceRepo{
			findMenuVisibleFn: func(ctx context.Context) ([]*entities.Resource, error) { return resources, nil },
		}
		screenRepo := &mockResourceScreenRepo{
			getByResourceKeyFn: func(ctx context.Context, key string) ([]*entities.ResourceScreen, error) { return nil, nil },
		}

		svc := newMenuService(resourceRepo, screenRepo)
		perms := []string{"dashboard:read", "dashboard:write"}
		resp, err := svc.GetMenuForUser(ctx, perms)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if len(resp.Items[0].Permissions) != 2 {
			t.Errorf("esperaba 2 permisos en el item, obtuvo %d", len(resp.Items[0].Permissions))
		}
	})

	t.Run("el recurso padre es visible cuando el hijo lo es", func(t *testing.T) {
		parentID := uuid.New()
		childID := uuid.New()
		resources := []*entities.Resource{
			{ID: parentID, Key: "admin", DisplayName: "Admin", Scope: "platform", IsMenuVisible: true},
			{ID: childID, Key: "admin.users", DisplayName: "Admin Users", ParentID: &parentID, Scope: "platform", IsMenuVisible: true},
		}
		resourceRepo := &mockResourceRepo{
			findMenuVisibleFn: func(ctx context.Context) ([]*entities.Resource, error) { return resources, nil },
		}
		screenRepo := &mockResourceScreenRepo{
			getByResourceKeyFn: func(ctx context.Context, key string) ([]*entities.ResourceScreen, error) { return nil, nil },
		}

		svc := newMenuService(resourceRepo, screenRepo)
		// El usuario solo tiene permiso en el hijo
		resp, err := svc.GetMenuForUser(ctx, []string{"admin.users:read"})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if len(resp.Items) != 1 {
			t.Errorf("esperaba 1 item raíz (admin), obtuvo %d", len(resp.Items))
		}
		if resp.Items[0].Key != "admin" {
			t.Errorf("key incorrecta: %s", resp.Items[0].Key)
		}
		if len(resp.Items[0].Children) != 1 {
			t.Errorf("esperaba 1 hijo, obtuvo %d", len(resp.Items[0].Children))
		}
		if resp.Items[0].Children[0].Key != "admin.users" {
			t.Errorf("key hijo incorrecta: %s", resp.Items[0].Children[0].Key)
		}
	})

	t.Run("incluye pantallas mapeadas", func(t *testing.T) {
		dashID := uuid.New()
		resources := []*entities.Resource{
			{ID: dashID, Key: "dashboard", DisplayName: "Dashboard", Scope: "platform", IsMenuVisible: true},
		}
		resourceRepo := &mockResourceRepo{
			findMenuVisibleFn: func(ctx context.Context) ([]*entities.Resource, error) { return resources, nil },
		}
		screenRepo := &mockResourceScreenRepo{
			getByResourceKeyFn: func(ctx context.Context, key string) ([]*entities.ResourceScreen, error) {
				if key == "dashboard" {
					return []*entities.ResourceScreen{
						{ScreenKey: "dashboard-main", ScreenType: "main"},
					}, nil
				}
				return nil, nil
			},
		}

		svc := newMenuService(resourceRepo, screenRepo)
		resp, err := svc.GetMenuForUser(ctx, []string{"dashboard:read"})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if resp.Items[0].Screens == nil {
			t.Fatal("esperaba screens, obtuvo nil")
		}
		if resp.Items[0].Screens["main"] != "dashboard-main" {
			t.Errorf("screen incorrecta: %v", resp.Items[0].Screens)
		}
	})

	t.Run("propaga error de base de datos en FindMenuVisible", func(t *testing.T) {
		resourceRepo := &mockResourceRepo{
			findMenuVisibleFn: func(ctx context.Context) ([]*entities.Resource, error) { return nil, errors.New("db fail") },
		}
		svc := newMenuService(resourceRepo, nil)
		_, err := svc.GetMenuForUser(ctx, []string{"resource:read"})
		assertAppError(t, err, sharedErrors.ErrorCodeDatabaseError)
	})
}

// ─── GetFullMenu ──────────────────────────────────────────────────────────────

func TestMenuService_GetFullMenu(t *testing.T) {
	ctx := context.Background()

	t.Run("retorna todos los recursos visibles del menú", func(t *testing.T) {
		resources := []*entities.Resource{
			{ID: uuid.New(), Key: "dashboard", DisplayName: "Dashboard", Scope: "platform", IsMenuVisible: true},
			{ID: uuid.New(), Key: "users", DisplayName: "Users", Scope: "platform", IsMenuVisible: true},
		}
		resourceRepo := &mockResourceRepo{
			findMenuVisibleFn: func(ctx context.Context) ([]*entities.Resource, error) { return resources, nil },
		}
		screenRepo := &mockResourceScreenRepo{
			getByResourceKeyFn: func(ctx context.Context, key string) ([]*entities.ResourceScreen, error) { return nil, nil },
		}

		svc := newMenuService(resourceRepo, screenRepo)
		resp, err := svc.GetFullMenu(ctx)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if len(resp.Items) != 2 {
			t.Errorf("esperaba 2 items, obtuvo %d", len(resp.Items))
		}
	})

	t.Run("retorna menú vacío cuando no hay recursos", func(t *testing.T) {
		resourceRepo := &mockResourceRepo{
			findMenuVisibleFn: func(ctx context.Context) ([]*entities.Resource, error) { return []*entities.Resource{}, nil },
		}
		svc := newMenuService(resourceRepo, nil)
		resp, err := svc.GetFullMenu(ctx)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if len(resp.Items) != 0 {
			t.Errorf("esperaba 0 items, obtuvo %d", len(resp.Items))
		}
	})

	t.Run("propaga error de base de datos", func(t *testing.T) {
		resourceRepo := &mockResourceRepo{
			findMenuVisibleFn: func(ctx context.Context) ([]*entities.Resource, error) { return nil, errors.New("db fail") },
		}
		svc := newMenuService(resourceRepo, nil)
		_, err := svc.GetFullMenu(ctx)
		assertAppError(t, err, sharedErrors.ErrorCodeDatabaseError)
	})

	t.Run("construye árbol jerárquico correctamente", func(t *testing.T) {
		parentID := uuid.New()
		child1ID := uuid.New()
		child2ID := uuid.New()
		resources := []*entities.Resource{
			{ID: parentID, Key: "admin", DisplayName: "Admin", Scope: "platform", IsMenuVisible: true},
			{ID: child1ID, Key: "admin.roles", DisplayName: "Roles", ParentID: &parentID, Scope: "platform", IsMenuVisible: true},
			{ID: child2ID, Key: "admin.perms", DisplayName: "Permisos", ParentID: &parentID, Scope: "platform", IsMenuVisible: true},
		}
		resourceRepo := &mockResourceRepo{
			findMenuVisibleFn: func(ctx context.Context) ([]*entities.Resource, error) { return resources, nil },
		}
		screenRepo := &mockResourceScreenRepo{
			getByResourceKeyFn: func(ctx context.Context, key string) ([]*entities.ResourceScreen, error) { return nil, nil },
		}

		svc := newMenuService(resourceRepo, screenRepo)
		resp, err := svc.GetFullMenu(ctx)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if len(resp.Items) != 1 {
			t.Errorf("esperaba 1 item raíz, obtuvo %d", len(resp.Items))
		}
		if len(resp.Items[0].Children) != 2 {
			t.Errorf("esperaba 2 hijos, obtuvo %d", len(resp.Items[0].Children))
		}
	})
}

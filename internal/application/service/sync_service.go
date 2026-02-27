package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/EduGoGroup/edugo-api-iam-platform/internal/application/dto"
	authDto "github.com/EduGoGroup/edugo-api-iam-platform/internal/auth/dto"
	authService "github.com/EduGoGroup/edugo-api-iam-platform/internal/auth/service"
	"github.com/EduGoGroup/edugo-api-iam-platform/internal/domain/repository"
	"github.com/EduGoGroup/edugo-shared/auth"
	"github.com/EduGoGroup/edugo-shared/logger"
	"golang.org/x/sync/errgroup"
)

// SyncService defines the sync service interface
type SyncService interface {
	GetFullBundle(ctx context.Context, userID string, activeContext *auth.UserContext, buckets []string) (*dto.SyncBundleResponse, error)
	GetDeltaSync(ctx context.Context, userID string, activeContext *auth.UserContext, clientHashes map[string]string) (*dto.DeltaSyncResponse, error)
}

type syncService struct {
	menuService         MenuService
	screenConfigService ScreenConfigService
	authService         authService.AuthService
	screenInstanceRepo  repository.ScreenInstanceRepository
	logger              logger.Logger
}

// NewSyncService creates a new sync service
func NewSyncService(
	menuService MenuService,
	screenConfigService ScreenConfigService,
	authSvc authService.AuthService,
	screenInstanceRepo repository.ScreenInstanceRepository,
	logger logger.Logger,
) SyncService {
	return &syncService{
		menuService:         menuService,
		screenConfigService: screenConfigService,
		authService:         authSvc,
		screenInstanceRepo:  screenInstanceRepo,
		logger:              logger,
	}
}

// GetFullBundle builds the sync bundle for a user.
// If buckets is non-empty, only the specified buckets are loaded (e.g. ["menu","permissions","available_contexts","screens"]).
// If buckets is empty, all buckets are loaded (backward compatible).
func (s *syncService) GetFullBundle(ctx context.Context, userID string, activeContext *auth.UserContext, buckets []string) (*dto.SyncBundleResponse, error) {
	var (
		mu      sync.Mutex
		bundle  dto.SyncBundleResponse
		hashes  = make(map[string]string)
		screens = make(map[string]*dto.ScreenBundle)
	)

	bundle.Hashes = hashes
	bundle.Screens = screens

	// Build a set of requested buckets for fast lookup
	bucketSet := make(map[string]bool, len(buckets))
	for _, b := range buckets {
		bucketSet[b] = true
	}
	loadAll := len(buckets) == 0

	g, gCtx := errgroup.WithContext(ctx)

	// 1. Menu
	if loadAll || bucketSet["menu"] {
		g.Go(func() error {
			menu, err := s.menuService.GetMenuForUser(gCtx, activeContext.Permissions)
			if err != nil {
				s.logger.Warn("sync: error fetching menu", "user_id", userID, "error", err)
				mu.Lock()
				bundle.Menu = []dto.MenuItemDTO{}
				hashes["menu"] = hashJSON([]dto.MenuItemDTO{})
				mu.Unlock()
				return nil
			}
			mu.Lock()
			bundle.Menu = menu.Items
			hashes["menu"] = hashJSON(menu.Items)
			mu.Unlock()
			return nil
		})
	}

	// 2. Permissions
	if loadAll || bucketSet["permissions"] {
		g.Go(func() error {
			perms := activeContext.Permissions
			if perms == nil {
				perms = []string{}
			}
			mu.Lock()
			bundle.Permissions = perms
			hashes["permissions"] = hashPermissions(perms)
			mu.Unlock()
			return nil
		})
	}

	// 3. Available contexts
	if loadAll || bucketSet["available_contexts"] {
		g.Go(func() error {
			resp, err := s.authService.GetAvailableContexts(gCtx, userID, activeContext)
			if err != nil {
				s.logger.Warn("sync: error fetching available contexts", "user_id", userID, "error", err)
				mu.Lock()
				bundle.AvailableContexts = []*authDto.UserContextDTO{}
				hashes["available_contexts"] = hashJSON([]*authDto.UserContextDTO{})
				mu.Unlock()
				return nil
			}
			sort.Slice(resp.Available, func(i, j int) bool {
				if resp.Available[i].SchoolID != resp.Available[j].SchoolID {
					return resp.Available[i].SchoolID < resp.Available[j].SchoolID
				}
				return resp.Available[i].RoleID < resp.Available[j].RoleID
			})
			mu.Lock()
			bundle.AvailableContexts = resp.Available
			hashes["available_contexts"] = hashJSON(resp.Available)
			mu.Unlock()
			return nil
		})
	}

	// 4. Screens
	if loadAll || bucketSet["screens"] {
		g.Go(func() error {
			instances, _, err := s.screenInstanceRepo.List(gCtx, repository.ScreenInstanceFilter{
				Offset: 0,
				Limit:  1000,
			})
			if err != nil {
				s.logger.Warn("sync: error listing screen instances", "user_id", userID, "error", err)
				return nil
			}

			for _, inst := range instances {
				resolved, err := s.screenConfigService.ResolveScreenByKey(gCtx, inst.ScreenKey)
				if err != nil {
					s.logger.Warn("sync: error resolving screen", "key", inst.ScreenKey, "error", err)
					continue
				}

				screenBundle := &dto.ScreenBundle{
					ScreenKey:  resolved.ScreenKey,
					ScreenName: resolved.ScreenName,
					Pattern:    resolved.Pattern,
					Version:    resolved.Version,
					Template:   resolved.Template,
					SlotData:   resolved.SlotData,
					HandlerKey: resolved.HandlerKey,
				}

				hashKey := "screen:" + inst.ScreenKey
				hashVal := hashScreen(resolved.Version, resolved.UpdatedAt.UTC().Format(time.RFC3339Nano))

				mu.Lock()
				screens[inst.ScreenKey] = screenBundle
				hashes[hashKey] = hashVal
				mu.Unlock()
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("error building sync bundle: %w", err)
	}

	if bundle.Menu == nil {
		bundle.Menu = []dto.MenuItemDTO{}
	}
	if bundle.Permissions == nil {
		bundle.Permissions = []string{}
	}
	if bundle.AvailableContexts == nil {
		bundle.AvailableContexts = []*authDto.UserContextDTO{}
	}

	return &bundle, nil
}

// GetDeltaSync compares client hashes and returns only changed buckets
func (s *syncService) GetDeltaSync(ctx context.Context, userID string, activeContext *auth.UserContext, clientHashes map[string]string) (*dto.DeltaSyncResponse, error) {
	fullBundle, err := s.GetFullBundle(ctx, userID, activeContext, nil)
	if err != nil {
		return nil, err
	}

	changed := make(map[string]*dto.BucketData)
	var unchanged []string

	for key, serverHash := range fullBundle.Hashes {
		clientHash, exists := clientHashes[key]
		if exists && clientHash == serverHash {
			unchanged = append(unchanged, key)
			continue
		}

		data, err := s.extractBucketData(fullBundle, key)
		if err != nil {
			s.logger.Warn("sync: error extracting bucket data", "key", key, "error", err)
			continue
		}

		changed[key] = &dto.BucketData{
			Data: data,
			Hash: serverHash,
		}
	}

	if unchanged == nil {
		unchanged = []string{}
	}
	sort.Strings(unchanged)

	return &dto.DeltaSyncResponse{
		Changed:   changed,
		Unchanged: unchanged,
	}, nil
}

// extractBucketData extracts JSON data for a specific bucket key from the full bundle
func (s *syncService) extractBucketData(bundle *dto.SyncBundleResponse, key string) (json.RawMessage, error) {
	switch {
	case key == "menu":
		return json.Marshal(bundle.Menu)
	case key == "permissions":
		return json.Marshal(bundle.Permissions)
	case key == "available_contexts":
		return json.Marshal(bundle.AvailableContexts)
	case strings.HasPrefix(key, "screen:"):
		screenKey := strings.TrimPrefix(key, "screen:")
		screen, ok := bundle.Screens[screenKey]
		if !ok {
			return json.Marshal(nil)
		}
		return json.Marshal(screen)
	default:
		return json.Marshal(nil)
	}
}

// Hash helpers

func hashJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func hashPermissions(permissions []string) string {
	sorted := make([]string, len(permissions))
	copy(sorted, permissions)
	sort.Strings(sorted)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(strings.Join(sorted, ","))))
}

func hashScreen(version int, updatedAt string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%d:%s", version, updatedAt))))
}

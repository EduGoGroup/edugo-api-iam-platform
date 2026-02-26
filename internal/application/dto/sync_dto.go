package dto

import (
	"encoding/json"

	authDto "github.com/EduGoGroup/edugo-api-iam-platform/internal/auth/dto"
)

// SyncBundleResponse represents the full sync bundle with all user data
type SyncBundleResponse struct {
	Menu              []MenuItemDTO            `json:"menu"`
	Permissions       []string                 `json:"permissions"`
	Screens           map[string]*ScreenBundle `json:"screens"`
	AvailableContexts []*authDto.UserContextDTO `json:"available_contexts"`
	Hashes            map[string]string        `json:"hashes"`
}

// ScreenBundle represents a resolved screen definition within the sync bundle
type ScreenBundle struct {
	ScreenKey  string          `json:"screen_key"`
	ScreenName string          `json:"screen_name"`
	Pattern    string          `json:"pattern"`
	Version    int             `json:"version"`
	Template   json.RawMessage `json:"template"`
	SlotData   json.RawMessage `json:"slot_data"`
	HandlerKey *string         `json:"handler_key,omitempty"`
}

// DeltaSyncRequest represents the client's current hashes for delta comparison
type DeltaSyncRequest struct {
	Hashes map[string]string `json:"hashes" binding:"required"`
}

// DeltaSyncResponse represents the delta sync result with only changed buckets
type DeltaSyncResponse struct {
	Changed   map[string]*BucketData `json:"changed"`
	Unchanged []string               `json:"unchanged"`
}

// BucketData represents a single changed bucket with its data and new hash
type BucketData struct {
	Data json.RawMessage `json:"data"`
	Hash string          `json:"hash"`
}

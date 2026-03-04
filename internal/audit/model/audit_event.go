package model

import "time"

// AuditEvent maps to audit.events table for query purposes
type AuditEvent struct {
	ID             string                 `gorm:"column:id;primaryKey" json:"id"`
	ActorID        string                 `gorm:"column:actor_id" json:"actor_id"`
	ActorEmail     string                 `gorm:"column:actor_email" json:"actor_email"`
	ActorRole      string                 `gorm:"column:actor_role" json:"actor_role"`
	ActorIP        *string                `gorm:"column:actor_ip" json:"actor_ip,omitempty"`
	ActorUserAgent *string                `gorm:"column:actor_user_agent" json:"actor_user_agent,omitempty"`
	SchoolID       *string                `gorm:"column:school_id" json:"school_id,omitempty"`
	UnitID         *string                `gorm:"column:unit_id" json:"unit_id,omitempty"`
	ServiceName    string                 `gorm:"column:service_name" json:"service_name"`
	Action         string                 `gorm:"column:action" json:"action"`
	ResourceType   string                 `gorm:"column:resource_type" json:"resource_type"`
	ResourceID     *string                `gorm:"column:resource_id" json:"resource_id,omitempty"`
	PermissionUsed *string                `gorm:"column:permission_used" json:"permission_used,omitempty"`
	RequestMethod  *string                `gorm:"column:request_method" json:"request_method,omitempty"`
	RequestPath    *string                `gorm:"column:request_path" json:"request_path,omitempty"`
	RequestID      *string                `gorm:"column:request_id" json:"request_id,omitempty"`
	StatusCode     *int                   `gorm:"column:status_code" json:"status_code,omitempty"`
	Changes        map[string]interface{} `gorm:"column:changes;serializer:json" json:"changes,omitempty"`
	Metadata       map[string]interface{} `gorm:"column:metadata;serializer:json" json:"metadata,omitempty"`
	ErrorMessage   *string                `gorm:"column:error_message" json:"error_message,omitempty"`
	CreatedAt      time.Time              `gorm:"column:created_at" json:"created_at"`
	Severity       string                 `gorm:"column:severity" json:"severity"`
	Category       string                 `gorm:"column:category" json:"category"`
}

func (AuditEvent) TableName() string {
	return "audit.events"
}

// AuditFilters for querying audit events
type AuditFilters struct {
	Action       string
	ResourceType string
	Severity     string
	Category     string
	ActorID      string
	ServiceName  string
	From         *time.Time
	To           *time.Time
}

// AuditSummary represents aggregated audit data
type AuditSummary struct {
	TotalEvents    int64              `json:"total_events"`
	ByAction       map[string]int64   `json:"by_action"`
	BySeverity     map[string]int64   `json:"by_severity"`
	ByCategory     map[string]int64   `json:"by_category"`
	ByResourceType map[string]int64   `json:"by_resource_type"`
}

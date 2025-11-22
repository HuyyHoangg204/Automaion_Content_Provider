package models

import (
	"time"
)

// ProcessLog represents a log entry for process tracking (Chrome launch, Gem creation, etc.)
type ProcessLog struct {
	// Primary key
	ID string `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`

	// Entity identification
	EntityType string `json:"entity_type" gorm:"type:varchar(50);not null;index" example:"topic"` // "topic", "script", "automation"
	EntityID   string `json:"entity_id" gorm:"type:uuid;not null;index" example:"550e8400-e29b-41d4-a716-446655440000"`

	// User and machine info
	UserID    string `json:"user_id" gorm:"type:uuid;not null;index" example:"550e8400-e29b-41d4-a716-446655440001"`
	MachineID string `json:"machine_id,omitempty" gorm:"type:varchar(255);index" example:"PC-001"` // Machine ID from automation backend

	// Log details
	Stage   string `json:"stage" gorm:"type:varchar(50);not null;index" example:"chrome_launched"` // "chrome_launching", "chrome_launched", "gem_creating", "gem_created", "completed", "failed"
	Status  string `json:"status" gorm:"type:varchar(20);not null;index" example:"success"`        // "info", "success", "warning", "error"
	Message string `json:"message" gorm:"type:text;not null" example:"Chrome launched successfully"`

	// Additional metadata
	Metadata JSON `json:"metadata,omitempty" gorm:"type:jsonb"` // {machine_id, tunnel_url, gem_id, error_details, etc.}

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
}

// TableName specifies the table name for the ProcessLog model
func (ProcessLog) TableName() string {
	return "process_logs"
}

// ProcessLogRequest represents the request to create a process log (from automation backend)
type ProcessLogRequest struct {
	EntityType string                 `json:"entity_type" binding:"required" example:"topic"`
	EntityID   string                 `json:"entity_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID     string                 `json:"user_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440001"`
	MachineID  string                 `json:"machine_id,omitempty" example:"PC-001"`
	Stage      string                 `json:"stage" binding:"required" example:"chrome_launched"`
	Status     string                 `json:"status" binding:"required" example:"success"`
	Message    string                 `json:"message" binding:"required" example:"Chrome launched successfully"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ProcessLogResponse represents the response for process log operations
type ProcessLogResponse struct {
	ID         string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	EntityType string `json:"entity_type" example:"topic"`
	EntityID   string `json:"entity_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	UserID     string `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	MachineID  string `json:"machine_id,omitempty" example:"PC-001"`
	Stage      string `json:"stage" example:"chrome_launched"`
	Status     string `json:"status" example:"success"`
	Message    string `json:"message" example:"Chrome launched successfully"`
	Metadata   JSON   `json:"metadata,omitempty"`
	CreatedAt  string `json:"created_at" example:"2025-01-21T10:30:00Z"`
}



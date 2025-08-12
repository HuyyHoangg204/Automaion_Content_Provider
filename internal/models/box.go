package models

import (
	"time"
)

// Box represents a machine/computer that belongs to a user
type Box struct {
	ID        string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID    string    `json:"user_id" gorm:"not null;index;type:uuid"`
	MachineID string    `json:"machine_id" gorm:"type:varchar(255);not null;unique;index"`
	Name      string    `json:"name" gorm:"type:varchar(255);not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	User User  `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	Apps []App `json:"apps,omitempty" gorm:"foreignKey:BoxID;references:ID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for the Box model
func (Box) TableName() string {
	return "boxes"
}

// CreateBoxRequest represents the request to create a new box
type CreateBoxRequest struct {
	MachineID string `json:"machine_id" binding:"required" example:"PC-001"`
	Name      string `json:"name" binding:"required" example:"My Computer"`
}

// UpdateBoxRequest represents the request to update a box
type UpdateBoxRequest struct {
	Name string `json:"name" binding:"required" example:"Updated Computer Name"`
}

// BoxResponse represents the response for box operations
type BoxResponse struct {
	ID        string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID    string `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	MachineID string `json:"machine_id" example:"PC-001"`
	Name      string `json:"name" example:"My Computer"`
	CreatedAt string `json:"created_at" example:"2025-01-09T10:30:00Z"`
	UpdatedAt string `json:"updated_at" example:"2025-01-09T10:30:00Z"`
}

// HidemiumProfile represents a profile from Hidemium API
type HidemiumProfile struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	CreatedAt string                 `json:"created_at"`
	UpdatedAt string                 `json:"updated_at"`
	IsActive  bool                   `json:"is_active"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// HidemiumResponse represents the response from Hidemium API
type HidemiumResponse struct {
	Success  bool              `json:"success"`
	Data     interface{}       `json:"data"` // Use interface{} to handle different data formats
	Message  string            `json:"message,omitempty"`
	Profiles []HidemiumProfile `json:"profiles,omitempty"` // Alternative field name
	Result   []HidemiumProfile `json:"result,omitempty"`   // Another possible field name
}

// SyncBoxProfilesResponse represents the response for syncing profiles from a box
type SyncBoxProfilesResponse struct {
	BoxID           string `json:"box_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	MachineID       string `json:"machine_id" example:"pc-91542"`
	TunnelURL       string `json:"tunnel_url" example:"http://pc-91542.agent-controller.onegreen.cloud/frps"`
	ProfilesSynced  int    `json:"profiles_synced" example:"10"`
	ProfilesCreated int    `json:"profiles_created" example:"5"`
	ProfilesUpdated int    `json:"profiles_updated" example:"3"`
	ProfilesDeleted int    `json:"profiles_deleted" example:"2"`
	Message         string `json:"message" example:"Sync completed successfully"`
}

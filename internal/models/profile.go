package models

import (
	"time"
)

// Profile represents a browser profile
type Profile struct {
	ID        string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	AppID     string    `json:"app_id" gorm:"not null;index;type:uuid"`
	Name      string    `json:"name" gorm:"type:varchar(255);not null"`
	Data      JSON      `json:"data" gorm:"type:jsonb"` // Store complex profile data from anti-detect browsers
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// Relationships
	App   App    `json:"app,omitempty" gorm:"foreignKey:AppID;references:ID;constraint:OnDelete:CASCADE"`
	Flows []Flow `json:"flows,omitempty" gorm:"foreignKey:ProfileID;references:ID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for the Profile model
func (Profile) TableName() string {
	return "profiles"
}

// CreateProfileRequest represents the request to create a new profile
type CreateProfileRequest struct {
	AppID string `json:"app_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name  string `json:"name" binding:"required" example:"My Profile"`
	Data  JSON   `json:"data" binding:"required"`
}

// UpdateProfileRequest represents the request to update a profile
type UpdateProfileRequest struct {
	Name string `json:"name" binding:"required" example:"Updated Profile Name"`
	Data JSON   `json:"data,omitempty"`
}

// ProfileResponse represents the response for profile operations
type ProfileResponse struct {
	ID        string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	AppID     string `json:"app_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Name      string `json:"name" example:"My Profile"`
	Data      JSON   `json:"data,omitempty"`
	CreatedAt string `json:"created_at" example:"2025-01-09T10:00:00Z"`
	UpdatedAt string `json:"updated_at" example:"2025-01-09T10:00:00Z"`
}

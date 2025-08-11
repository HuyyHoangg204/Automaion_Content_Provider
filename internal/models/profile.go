package models

import (
	"time"
)

// Profile represents a profile that belongs to an app
type Profile struct {
	ID        string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	AppID     string    `json:"app_id" gorm:"not null;index;type:uuid"`
	Name      string    `json:"name" gorm:"type:varchar(255);not null"`
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
	Name  string `json:"name" binding:"required" example:"Facebook Profile 1"`
}

// UpdateProfileRequest represents the request to update a profile
type UpdateProfileRequest struct {
	Name string `json:"name" binding:"required" example:"Updated Profile Name"`
}

// ProfileResponse represents the response for profile operations
type ProfileResponse struct {
	ID        string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	AppID     string `json:"app_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Name      string `json:"name" example:"Facebook Profile 1"`
	CreatedAt string `json:"created_at" example:"2025-01-09T10:30:00Z"`
	UpdatedAt string `json:"updated_at" example:"2025-01-09T10:30:00Z"`
}

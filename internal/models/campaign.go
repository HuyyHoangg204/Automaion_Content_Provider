package models

import (
	"time"
)

// Campaign represents a campaign that belongs to a user
type Campaign struct {
	ID         string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID     string    `json:"user_id" gorm:"not null;index;type:uuid"`
	Name       string    `json:"name" gorm:"type:varchar(255);not null"`
	ScriptName string    `json:"script_name" gorm:"type:varchar(255);not null"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	// Relationships
	User  User   `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	Flows []Flow `json:"flows,omitempty" gorm:"foreignKey:CampaignID;references:ID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for the Campaign model
func (Campaign) TableName() string {
	return "campaigns"
}

// CreateCampaignRequest represents the request to create a new campaign
type CreateCampaignRequest struct {
	Name       string `json:"name" binding:"required" example:"Auto Post Campaign"`
	ScriptName string `json:"script_name" binding:"required" example:"auto_post.js"`
}

// UpdateCampaignRequest represents the request to update a campaign
type UpdateCampaignRequest struct {
	Name       string `json:"name" binding:"required" example:"Updated Campaign Name"`
	ScriptName string `json:"script_name" binding:"required" example:"updated_script.js"`
}

// CampaignResponse represents the response for campaign operations
type CampaignResponse struct {
	ID         string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID     string `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Name       string `json:"name" example:"Auto Post Campaign"`
	ScriptName string `json:"script_name" example:"auto_post.js"`
	CreatedAt  string `json:"created_at" example:"2025-01-09T10:30:00Z"`
	UpdatedAt  string `json:"updated_at" example:"2025-01-09T10:30:00Z"`
}

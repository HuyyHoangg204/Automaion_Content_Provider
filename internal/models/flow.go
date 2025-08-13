package models

import (
	"time"
)

// Flow represents the execution flow of a campaign on a specific profile
type Flow struct {
	ID              string     `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	GroupCampaignID string     `json:"group_campaign_id" gorm:"not null;index;type:uuid"`
	ProfileID       string     `json:"profile_id" gorm:"not null;index;type:uuid"`
	Status          string     `json:"status" gorm:"type:varchar(255);not null"`
	StartedAt       *time.Time `json:"started_at"`
	FinishedAt      *time.Time `json:"finished_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`

	// Relationships
	GroupCampaign GroupCampaign `json:"group_campaign,omitempty" gorm:"foreignKey:GroupCampaignID;references:ID;constraint:OnDelete:CASCADE"`
	Profile       Profile       `json:"profile,omitempty" gorm:"foreignKey:ProfileID;references:ID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for the Flow model
func (Flow) TableName() string {
	return "flows"
}

// CreateFlowRequest represents the request to create a new flow
type CreateFlowRequest struct {
	GroupCampaignID string `json:"group_campaign_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	ProfileID       string `json:"profile_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440001"`
	Status          string `json:"status" binding:"required" example:"Started"`
}

// UpdateFlowRequest represents the request to update a flow
type UpdateFlowRequest struct {
	Status string `json:"status" binding:"required" example:"Completed"`
}

// FlowResponse represents the response for flow operations
type FlowResponse struct {
	ID         string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ProfileID  string `json:"profile_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	Status     string `json:"status" example:"Started"`
	StartedAt  string `json:"started_at,omitempty" example:"2025-01-09T10:00:00Z"`
	FinishedAt string `json:"finished_at,omitempty" example:"2025-01-09T10:30:00Z"`
	CreatedAt  string `json:"created_at" example:"2025-01-09T10:30:00Z"`
	UpdatedAt  string `json:"updated_at" example:"2025-01-09T10:30:00Z"`
}

package models

import (
	"time"
)

// StatusCampaign represents the completion status of a campaign
type StatusCampaign struct {
	ID         uint       `json:"id" gorm:"primaryKey"`
	CampaignID uint       `json:"campaign_id" gorm:"not null;index"`
	ProfileID  uint       `json:"profile_id" gorm:"not null;index"`
	Status     string     `json:"status" gorm:"type:varchar(255);not null"`
	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`

	// Relationships
	Campaign Campaign `json:"campaign,omitempty" gorm:"foreignKey:CampaignID;references:ID;constraint:OnDelete:CASCADE"`
	Profile  Profile  `json:"profile,omitempty" gorm:"foreignKey:ProfileID;references:ID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for the StatusCampaign model
func (StatusCampaign) TableName() string {
	return "status_campaigns"
}

// CreateStatusCampaignRequest represents the request to create a new status campaign
type CreateStatusCampaignRequest struct {
	CampaignID uint   `json:"campaign_id" binding:"required" example:"1"`
	ProfileID  uint   `json:"profile_id" binding:"required" example:"1"`
	Status     string `json:"status" binding:"required" example:"Started"`
}

// UpdateStatusCampaignRequest represents the request to update a status campaign
type UpdateStatusCampaignRequest struct {
	Status string `json:"status" binding:"required" example:"Completed"`
}

// StatusCampaignResponse represents the response for status campaign operations
type StatusCampaignResponse struct {
	ID         uint   `json:"id" example:"1"`
	CampaignID uint   `json:"campaign_id" example:"1"`
	ProfileID  uint   `json:"profile_id" example:"1"`
	Status     string `json:"status" example:"Started"`
	StartedAt  string `json:"started_at,omitempty" example:"2025-01-09T10:00:00Z"`
	FinishedAt string `json:"finished_at,omitempty" example:"2025-01-09T10:30:00Z"`
	CreatedAt  string `json:"created_at" example:"2025-01-09T10:30:00Z"`
	UpdatedAt  string `json:"updated_at" example:"2025-01-09T10:30:00Z"`
}

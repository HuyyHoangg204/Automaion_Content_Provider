package models

import (
	"time"
)

// CampaignLog represents a log entry for a campaign
type CampaignLog struct {
	ID         string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	CampaignID string    `json:"campaign_id" gorm:"not null;index;type:uuid"`
	UserID     string    `json:"user_id" gorm:"not null;index;type:uuid"`
	ExecutedAt time.Time `json:"executed_at" gorm:"not null"`
	Status     string    `json:"status" gorm:"type:varchar(50);not null"` // e.g., "started", "completed", "failed"
	Message    string    `json:"message" gorm:"type:text"`
	Details    JSON      `json:"details" gorm:"type:jsonb"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	Campaign Campaign `json:"-" gorm:"foreignKey:CampaignID;references:ID;constraint:OnDelete:CASCADE"`
	User     User     `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for the CampaignLog model
func (CampaignLog) TableName() string {
	return "campaign_logs"
}

// CreateCampaignLogRequest represents the request to create a new campaign log
type CreateCampaignLogRequest struct {
	CampaignID string `json:"campaign_id" binding:"required"`
	ExecutedAt string `json:"executed_at" binding:"required"`
	Status     string `json:"status" binding:"required"`
	Message    string `json:"message"`
	Details    JSON   `json:"details"`
}

// CampaignLogResponse represents the response for campaign log operations
type CampaignLogResponse struct {
	ID         string    `json:"id"`
	CampaignID string    `json:"campaign_id"`
	UserID     string    `json:"user_id"`
	ExecutedAt time.Time `json:"executed_at"`
	Status     string    `json:"status"`
	Message    string    `json:"message"`
	Details    JSON      `json:"details"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

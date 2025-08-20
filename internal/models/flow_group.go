package models

import (
	"time"
)

// FlowGroup represents a campaign execution history/group
type FlowGroup struct {
	ID         string     `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	CampaignID string     `json:"campaign_id" gorm:"not null;index;type:uuid"`
	Name       string     `json:"name" gorm:"type:varchar(255);not null"`
	Status     string     `json:"status" gorm:"type:varchar(255);not null;default:'pending'"`
	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`

	// Relationships
	Campaign Campaign `json:"campaign,omitempty" gorm:"foreignKey:CampaignID;references:ID;constraint:OnDelete:CASCADE"`
	Flows    []Flow   `json:"flows,omitempty" gorm:"foreignKey:FlowGroupID;references:ID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for the FlowGroup model
func (FlowGroup) TableName() string {
	return "flow_groups"
}

// FlowGroupResponse represents the response for group campaign operations
type FlowGroupResponse struct {
	ID         string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	CampaignID string `json:"campaign_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Name       string `json:"name" example:"Campaign Run #1"`
	Status     string `json:"status" example:"running"`
	StartedAt  string `json:"started_at,omitempty" example:"2025-01-09T10:00:00Z"`
	FinishedAt string `json:"finished_at,omitempty" example:"2025-01-09T10:30:00Z"`
	CreatedAt  string `json:"created_at" example:"2025-01-09T10:30:00Z"`
	UpdatedAt  string `json:"updated_at" example:"2025-01-09T10:30:00Z"`
}

// FlowGroupStats represents statistics for a group campaign
type FlowGroupStats struct {
	ID          string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name        string  `json:"name" example:"Campaign Run #1"`
	Status      string  `json:"status" example:"completed"`
	SuccessRate float64 `json:"success_rate" example:"80.0"`
	Duration    string  `json:"duration,omitempty" example:"30m"`
	StartedAt   string  `json:"started_at" example:"2025-01-09T10:00:00Z"`
	FinishedAt  string  `json:"finished_at,omitempty" example:"2025-01-09T10:30:00Z"`
}

package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FlowGroup represents a group of flows created during campaign schedule execution
type FlowGroup struct {
	ID            string    `json:"id" gorm:"primaryKey;type:varchar(255)"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Name          string    `json:"name" gorm:"type:varchar(255);not null"`
	Description   string    `json:"description" gorm:"type:text"`
	Status        string    `json:"status" gorm:"type:varchar(50);default:'pending';index"` // pending, running, completed, failed
	ExecutionData JSON      `json:"execution_data" gorm:"type:jsonb"`                       // Store execution metadata
	CampaignID    string    `json:"campaign_id" gorm:"type:varchar(255);index"`
	ScriptName    string    `json:"script_name" gorm:"type:varchar(255);index"`
	UserID        string    `json:"user_id" gorm:"index"`

	// Relationships
	Campaign Campaign `json:"-" gorm:"foreignKey:CampaignID;references:ID;constraint:OnDelete:CASCADE"`
	Flows    []Flow   `json:"flows,omitempty" gorm:"foreignKey:FlowGroupID;constraint:OnDelete:CASCADE"`
	User     User     `json:"-" gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

// BeforeCreate is a GORM hook that runs before creating a new record
func (fg *FlowGroup) BeforeCreate(tx *gorm.DB) error {
	if fg.ID == "" {
		fg.ID = uuid.New().String()
	}
	return nil
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

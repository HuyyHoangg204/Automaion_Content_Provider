package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Flow represents an automation flow
type Flow struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(255)"`
	CreatedAt   time.Time `json:"created_at" gorm:"index"`
	UpdatedAt   time.Time `json:"updated_at"`
	Name        string    `json:"name" gorm:"type:varchar(255);not null"`
	Description string    `json:"description" gorm:"type:text"`
	ScriptName  string    `json:"script_name" gorm:"type:varchar(255);index"`
	FlowGroupID string    `json:"flow_group_id" gorm:"column:flow_group_id;type:varchar(255);index"`
	ProfileID   string    `json:"profile_id" gorm:"column:profile_id;index"`
	Status      string    `json:"status" gorm:"type:varchar(50);default:'pending';index"`
	Result      JSON      `json:"result" gorm:"type:jsonb"`
	UserID      string    `json:"user_id" gorm:"index"`

	// Relationships
	FlowGroup FlowGroup `json:"-" gorm:"foreignKey:FlowGroupID;references:ID;constraint:OnDelete:CASCADE"`
	Profile   Profile   `json:"-" gorm:"foreignKey:ProfileID;references:ID;constraint:OnDelete:CASCADE"`
	User      User      `json:"-" gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

// BeforeCreate is a GORM hook that runs before creating a new record
func (f *Flow) BeforeCreate(tx *gorm.DB) error {
	if f.ID == "" {
		f.ID = uuid.New().String()
	}
	return nil
}

// TableName specifies the table name for the Flow model
func (Flow) TableName() string {
	return "flows"
}

// CreateFlowRequest represents the request to create a new flow
type CreateFlowRequest struct {
	FlowGroupID string `json:"flow_group_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	ProfileID   string `json:"profile_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440001"`
	Status      string `json:"status" binding:"required,oneof=Started Running Completed Failed Stopped" example:"Started"`
}

// UpdateFlowRequest represents the request to update a flow
type UpdateFlowRequest struct {
	Status string `json:"status" binding:"required,oneof=Started Running Completed Failed Stopped" example:"Completed"`
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

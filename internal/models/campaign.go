package models

import (
	"time"
)

// Campaign represents a campaign that belongs to a user
type Campaign struct {
	ID              string `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID          string `json:"user_id" gorm:"not null;index;type:uuid"`
	Name            string `json:"name" gorm:"type:varchar(255);not null"`
	Description     string `json:"description" gorm:"type:text"`
	ScriptName      string `json:"script_name" gorm:"type:varchar(255);not null"`
	ScriptVariables JSON   `json:"script_variables" gorm:"type:jsonb;default:'{}'"`

	// Campaign details
	ConcurrentPhones int `json:"concurrent_phones" gorm:"type:int;default:50"`

	// Scheduling
	Schedule JSON `json:"schedule" gorm:"type:jsonb"`

	IsActive bool   `json:"is_active" gorm:"default:true"`
	Status   string `json:"status" gorm:"type:varchar(50);default:'idle';index"` // idle, running, failed, completed

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	User       User          `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	FlowGroups []FlowGroup   `json:"flow_groups,omitempty" gorm:"foreignKey:CampaignID;references:ID;constraint:OnDelete:CASCADE"`
	Profiles   []Profile     `json:"profiles,omitempty" gorm:"many2many:campaign_profiles;"`
	Logs       []CampaignLog `json:"logs,omitempty" gorm:"foreignKey:CampaignID;references:ID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for the Campaign model
func (Campaign) TableName() string {
	return "campaigns"
}

// CreateCampaignRequest represents the request to create a new campaign
type CreateCampaignRequest struct {
	Name             string   `json:"name" binding:"required" example:"Tăng view campaign"`
	Description      string   `json:"description" example:"This is a campaign to increase views"`
	ScriptName       string   `json:"script_name" binding:"required" example:"increase_views.js"`
	ScriptVariables  JSON     `json:"script_variables"`
	ConcurrentPhones int      `json:"concurrent_phones" example:"10"`
	Schedule         JSON     `json:"schedule" binding:"required"`
	IsActive         *bool    `json:"is_active" example:"true"`
	ProfileIDs       []string `json:"profile_ids" binding:"required"`
}

// UpdateCampaignRequest represents the request to update a campaign
type UpdateCampaignRequest struct {
	Name             string   `json:"name" binding:"required" example:"Updated Campaign Name"`
	Description      string   `json:"description" example:"This is an updated campaign"`
	ScriptName       string   `json:"script_name" binding:"required" example:"updated_script.js"`
	ScriptVariables  JSON     `json:"script_variables"`
	ConcurrentPhones int      `json:"concurrent_phones" example:"20"`
	Schedule         JSON     `json:"schedule" binding:"required"`
	IsActive         *bool    `json:"is_active" example:"false"`
	ProfileIDs       []string `json:"profile_ids" binding:"required"`
}

// CampaignResponse represents the response for campaign operations
type CampaignResponse struct {
	ID               string                   `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID           string                   `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Name             string                   `json:"name" example:"Tăng view campaign"`
	Description      string                   `json:"description" example:"This is a campaign to increase views"`
	ScriptName       string                   `json:"script_name" example:"increase_views.js"`
	ScriptVariables  JSON                     `json:"script_variables"`
	ConcurrentPhones int                      `json:"concurrent_phones" example:"10"`
	Schedule         JSON                     `json:"schedule"`
	IsActive         bool                     `json:"is_active" example:"true"`
	Status           string                   `json:"status" example:"idle"`
	Profiles         []ProfileWithBoxResponse `json:"profiles,omitempty"`
	Logs             []CampaignLog            `json:"logs,omitempty"`
	CreatedAt        string                   `json:"created_at" example:"2025-01-09T10:30:00Z"`
	UpdatedAt        string                   `json:"updated_at" example:"2025-01-09T10:30:00Z"`
}

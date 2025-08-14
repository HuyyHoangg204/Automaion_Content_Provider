package models

import (
	"time"
)

// Campaign represents a campaign that belongs to a user
type Campaign struct {
	ID         string `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID     string `json:"user_id" gorm:"not null;index;type:uuid"`
	Name       string `json:"name" gorm:"type:varchar(255);not null"`
	ScriptName string `json:"script_name" gorm:"type:varchar(255);not null"`

	// Campaign type and target
	CampaignType string `json:"campaign_type" gorm:"type:varchar(50);index;default:'video_views'"` // video_views, likes, comments, reads, shares, etc.
	TargetURL    string `json:"target_url" gorm:"type:text"`                                       // URL mục tiêu (video, post, article, etc.)

	// Campaign details
	TargetCount  int `json:"target_count" gorm:"default:0"`  // Số lượng mục tiêu (views, likes, comments, etc.)
	CurrentCount int `json:"current_count" gorm:"default:0"` // Số lượng hiện tại đã đạt được

	// Scheduling
	Frequency string     `json:"frequency" gorm:"type:varchar(20);default:'once'"` // once, daily, weekly, monthly, custom
	StartDate *time.Time `json:"start_date" gorm:"index"`                          // Ngày bắt đầu
	EndDate   *time.Time `json:"end_date" gorm:"index"`                            // Ngày kết thúc

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	User           User            `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	GroupCampaigns []GroupCampaign `json:"group_campaigns,omitempty" gorm:"foreignKey:CampaignID;references:ID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for the Campaign model
func (Campaign) TableName() string {
	return "campaigns"
}

// CreateCampaignRequest represents the request to create a new campaign
type CreateCampaignRequest struct {
	Name         string     `json:"name" binding:"required" example:"Tăng view campaign"`
	ScriptName   string     `json:"script_name" binding:"required" example:"increase_views.js"`
	CampaignType string     `json:"campaign_type" binding:"required" example:"video_views"`
	TargetURL    string     `json:"target_url" binding:"required" example:"https://youtube.com/watch?v=..."`
	TargetCount  int        `json:"target_count" binding:"required,min=1" example:"1000"`
	Frequency    string     `json:"frequency" example:"once"`
	StartDate    *time.Time `json:"start_date" example:"2025-08-14T00:00:00Z"`
	EndDate      *time.Time `json:"end_date" example:"2025-08-14T23:59:59Z"`
}

// UpdateCampaignRequest represents the request to update a campaign
type UpdateCampaignRequest struct {
	Name         string     `json:"name" binding:"required" example:"Updated Campaign Name"`
	ScriptName   string     `json:"script_name" binding:"required" example:"updated_script.js"`
	CampaignType string     `json:"campaign_type" binding:"required" example:"video_views"`
	TargetURL    string     `json:"target_url" binding:"required" example:"https://youtube.com/watch?v=..."`
	TargetCount  int        `json:"target_count" binding:"required,min=1" example:"1000"`
	Frequency    string     `json:"frequency" example:"daily"`
	StartDate    *time.Time `json:"start_date" example:"2025-08-14T00:00:00Z"`
	EndDate      *time.Time `json:"end_date" example:"2025-08-14T23:59:59Z"`
}

// CampaignResponse represents the response for campaign operations
type CampaignResponse struct {
	ID           string     `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID       string     `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Name         string     `json:"name" example:"Tăng view campaign"`
	ScriptName   string     `json:"script_name" example:"increase_views.js"`
	CampaignType string     `json:"campaign_type" example:"video_views"`
	TargetURL    string     `json:"target_url" example:"https://youtube.com/watch?v=..."`
	TargetCount  int        `json:"target_count" example:"1000"`
	CurrentCount int        `json:"current_count" example:"150"`
	Frequency    string     `json:"frequency" example:"once"`
	StartDate    *time.Time `json:"start_date" example:"2025-08-14T00:00:00Z"`
	EndDate      *time.Time `json:"end_date" example:"2025-08-14T23:59:59Z"`
	CreatedAt    string     `json:"created_at" example:"2025-01-09T10:30:00Z"`
	UpdatedAt    string     `json:"updated_at" example:"2025-01-09T10:30:00Z"`
}

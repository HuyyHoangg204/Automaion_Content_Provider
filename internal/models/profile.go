package models

import (
	"time"
)

// Profile represents a browser profile
type Profile struct {
	ID        string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	AppID     string    `json:"app_id" gorm:"not null;index;type:uuid"`
	Name      string    `json:"name" gorm:"type:varchar(255);not null"`
	Data      JSON      `json:"data" gorm:"type:jsonb"` // Store complex profile data from anti-detect browsers
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// Relationships
	App       App        `json:"app,omitempty" gorm:"foreignKey:AppID;references:ID;constraint:OnDelete:CASCADE"`
	Flows     []Flow     `json:"flows,omitempty" gorm:"foreignKey:ProfileID;references:ID;constraint:OnDelete:CASCADE"`
	Campaigns []Campaign `json:"campaigns,omitempty" gorm:"many2many:campaign_profiles;references:ID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for the Profile model
func (Profile) TableName() string {
	return "profiles"
}

// ProfileResponse represents the response for profile operations
type ProfileResponse struct {
	ID        string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	AppID     string `json:"app_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Name      string `json:"name" example:"My Profile"`
	Data      JSON   `json:"data,omitempty"`
	CreatedAt string `json:"created_at" example:"2025-01-09T10:00:00Z"`
	UpdatedAt string `json:"updated_at" example:"2025-01-09T10:00:00Z"`
}

// PaginatedProfileResponse represents a paginated response for profile operations
type PaginatedProfileResponse struct {
	Profiles    []*ProfileResponse `json:"profiles"`
	Total       int                `json:"total" example:"150"`
	Page        int                `json:"page" example:"1"`
	PageSize    int                `json:"page_size" example:"20"`
	TotalPages  int                `json:"total_pages" example:"8"`
	HasNext     bool               `json:"has_next" example:"true"`
	HasPrevious bool               `json:"has_previous" example:"false"`
}

// ProfileWithBoxResponse represents the response for profile operations in campaigns
type ProfileWithBoxResponse struct {
	ID        string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	AppID     string `json:"app_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Name      string `json:"name" example:"My Profile"`
	BoxName   string `json:"box_name" example:"My Computer"`
	CreatedAt string `json:"created_at" example:"2025-01-09T10:00:00Z"`
	UpdatedAt string `json:"updated_at" example:"2025-01-09T10:00:00Z"`
}

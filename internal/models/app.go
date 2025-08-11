package models

import (
	"time"
)

// App represents an application that the system supports (Hidemium, Genlogin, etc.)
type App struct {
	ID        string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	BoxID     string    `json:"box_id" gorm:"not null;index;type:uuid"`
	Name      string    `json:"name" gorm:"type:varchar(255);not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	Box      Box       `json:"box,omitempty" gorm:"foreignKey:BoxID;references:ID;constraint:OnDelete:CASCADE"`
	Profiles []Profile `json:"profiles,omitempty" gorm:"foreignKey:AppID;references:ID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for the App model
func (App) TableName() string {
	return "apps"
}

// CreateAppRequest represents the request to create a new app
type CreateAppRequest struct {
	BoxID string `json:"box_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name  string `json:"name" binding:"required" example:"Hidemium"`
}

// UpdateAppRequest represents the request to update an app
type UpdateAppRequest struct {
	Name string `json:"name" binding:"required" example:"Updated App Name"`
}

// AppResponse represents the response for app operations
type AppResponse struct {
	ID        string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	BoxID     string `json:"box_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Name      string `json:"name" example:"Hidemium"`
	CreatedAt string `json:"created_at" example:"2025-01-09T10:30:00Z"`
	UpdatedAt string `json:"updated_at" example:"2025-01-09T10:30:00Z"`
}

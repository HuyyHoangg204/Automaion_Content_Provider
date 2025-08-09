package models

import (
	"time"
)

// App represents an application that the system supports (Hidemium, Genlogin, etc.)
type App struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	BoxID     uint      `json:"box_id" gorm:"not null;index"`
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
	BoxID uint   `json:"box_id" binding:"required" example:"1"`
	Name  string `json:"name" binding:"required" example:"Hidemium"`
}

// UpdateAppRequest represents the request to update an app
type UpdateAppRequest struct {
	Name string `json:"name" binding:"required" example:"Updated App Name"`
}

// AppResponse represents the response for app operations
type AppResponse struct {
	ID        uint   `json:"id" example:"1"`
	BoxID     uint   `json:"box_id" example:"1"`
	Name      string `json:"name" example:"Hidemium"`
	CreatedAt string `json:"created_at" example:"2025-01-09T10:30:00Z"`
	UpdatedAt string `json:"updated_at" example:"2025-01-09T10:30:00Z"`
}

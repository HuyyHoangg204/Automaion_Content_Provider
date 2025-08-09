package models

import (
	"time"
)

// Box represents a machine/computer that belongs to a user
type Box struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"not null;index"`
	MachineID string    `json:"machine_id" gorm:"type:varchar(255);not null;unique;index"`
	Name      string    `json:"name" gorm:"type:varchar(255);not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	User User  `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	Apps []App `json:"apps,omitempty" gorm:"foreignKey:BoxID;references:ID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for the Box model
func (Box) TableName() string {
	return "boxes"
}

// CreateBoxRequest represents the request to create a new box
type CreateBoxRequest struct {
	MachineID string `json:"machine_id" binding:"required" example:"PC-001"`
	Name      string `json:"name" binding:"required" example:"My Computer"`
}

// UpdateBoxRequest represents the request to update a box
type UpdateBoxRequest struct {
	Name string `json:"name" binding:"required" example:"Updated Computer Name"`
}

// BoxResponse represents the response for box operations
type BoxResponse struct {
	ID        uint   `json:"id" example:"1"`
	UserID    uint   `json:"user_id" example:"1"`
	MachineID string `json:"machine_id" example:"PC-001"`
	Name      string `json:"name" example:"My Computer"`
	CreatedAt string `json:"created_at" example:"2025-01-09T10:30:00Z"`
	UpdatedAt string `json:"updated_at" example:"2025-01-09T10:30:00Z"`
}

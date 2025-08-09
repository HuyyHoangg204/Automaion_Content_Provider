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
	User User `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for the Box model
func (Box) TableName() string {
	return "boxes"
}

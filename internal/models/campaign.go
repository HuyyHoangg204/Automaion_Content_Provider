package models

import (
	"time"
)

// Campaign represents a campaign that belongs to a user
type Campaign struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	UserID     uint      `json:"user_id" gorm:"not null;index"`
	Name       string    `json:"name" gorm:"type:varchar(255);not null"`
	ScriptName string    `json:"script_name" gorm:"type:varchar(255);not null"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	// Relationships
	User User `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for the Campaign model
func (Campaign) TableName() string {
	return "campaigns"
}

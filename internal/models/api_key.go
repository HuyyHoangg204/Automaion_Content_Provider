package models

import (
	"time"
)

// APIKey represents an API key for a user
type APIKey struct {
	ID         uint       `json:"id" gorm:"primaryKey"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	Key        string     `json:"key" gorm:"type:varchar(255);not null;unique;index"`
	UserID     string     `json:"user_id" gorm:"not null;index"`
	IsActive   bool       `json:"is_active" gorm:"default:true;index"`
	LastUsedAt *time.Time `json:"last_used_at"`

	// Relationships
	User User `json:"-" gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for the APIKey model
func (APIKey) TableName() string {
	return "api_keys"
}

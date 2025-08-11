package models

import (
	"time"
)

// RefreshToken represents a refresh token for user authentication
type RefreshToken struct {
	ID        string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Token     string    `json:"token" gorm:"type:varchar(500);not null;unique;index"`
	UserID    string    `json:"user_id" gorm:"not null;index;type:uuid"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null;index"`
	IsRevoked bool      `json:"is_revoked" gorm:"default:false;index"`
	UserAgent string    `json:"user_agent" gorm:"type:varchar(500)"`
	IPAddress string    `json:"ip_address" gorm:"type:varchar(45)"`
	// Relationships
	User User `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for the RefreshToken model
func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

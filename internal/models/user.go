package models

import (
	"time"
)

// User represents a user in the system
type User struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	Username     string     `json:"username" gorm:"type:varchar(255);not null;unique;index"`
	PasswordHash string     `json:"-" gorm:"type:varchar(255);not null"`
	FirstName    string     `json:"first_name" gorm:"type:varchar(255)"`
	LastName     string     `json:"last_name" gorm:"type:varchar(255)"`
	IsActive     bool       `json:"is_active" gorm:"default:true;index"`
	IsAdmin      bool       `json:"is_admin" gorm:"default:false;index"`
	TokenVersion uint       `json:"token_version" gorm:"default:0"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	// Relationships
	RefreshTokens []RefreshToken `json:"refresh_tokens,omitempty" gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for the User model
func (User) TableName() string {
	return "users"
}

package models

import (
	"time"
)

// GeminiAccount represents a Gemini account setup on a specific machine
// Note: Nhiều machines có thể share cùng một account (cùng email/password)
// Unique constraint: (Email + MachineID) - đảm bảo 1 machine không có 2 accounts cùng email
type GeminiAccount struct {
	ID               string     `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	MachineID        string     `json:"machine_id" gorm:"type:varchar(255);not null;uniqueIndex:idx_gemini_accounts_email_machine"` // MachineID từ Box
	AppID            *string    `json:"app_id,omitempty" gorm:"type:uuid"`                                                          // Optional: link với App
	Email            string     `json:"email" gorm:"type:varchar(255);not null;uniqueIndex:idx_gemini_accounts_email_machine"`      // Unique với MachineID
	Password         string     `json:"-" gorm:"type:varchar(255)"`                                                                 // Encrypted, không trả về trong JSON
	IsActive         bool       `json:"is_active" gorm:"default:true;index"`
	IsLocked         bool       `json:"is_locked" gorm:"default:false;index"`
	LockedAt         *time.Time `json:"locked_at,omitempty"`
	LockedReason     string     `json:"locked_reason,omitempty" gorm:"type:text"`
	GeminiAccessible bool       `json:"gemini_accessible" gorm:"default:false"`
	TopicsCount      int        `json:"topics_count" gorm:"default:0;index"` // Số topics đã tạo trên account này
	LastUsedAt       *time.Time `json:"last_used_at,omitempty"`
	LastCheckedAt    *time.Time `json:"last_checked_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`

	// Relationships
	App *App `json:"app,omitempty" gorm:"foreignKey:AppID;references:ID;constraint:OnDelete:SET NULL"`
}

// TableName specifies the table name for the GeminiAccount model
func (GeminiAccount) TableName() string {
	return "gemini_accounts"
}

// SetupGeminiAccountRequest represents request to setup Gemini account
type SetupGeminiAccountRequest struct {
	MachineID string `json:"machine_id" binding:"required" example:"PC-001"`
	Email     string `json:"email" binding:"required" example:"user@gmail.com"`
	Password  string `json:"password" binding:"required" example:"password123"`
	DebugPort *int   `json:"debug_port,omitempty" example:"9222"`
}

// LockGeminiAccountRequest represents request to lock a Gemini account
type LockGeminiAccountRequest struct {
	Reason string `json:"reason,omitempty" example:"Account was locked by Google"`
}

// GeminiAccountResponse represents the response for Gemini account operations
type GeminiAccountResponse struct {
	ID               string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	MachineID        string  `json:"machine_id" example:"PC-001"`
	AppID            *string `json:"app_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
	Email            string  `json:"email" example:"user@gmail.com"`
	IsActive         bool    `json:"is_active" example:"true"`
	IsLocked         bool    `json:"is_locked" example:"false"`
	LockedAt         *string `json:"locked_at,omitempty" example:"2025-01-21T10:30:00Z"`
	LockedReason     string  `json:"locked_reason,omitempty" example:""`
	GeminiAccessible bool    `json:"gemini_accessible" example:"true"`
	TopicsCount      int     `json:"topics_count" example:"5"`
	LastUsedAt       *string `json:"last_used_at,omitempty" example:"2025-01-21T10:30:00Z"`
	LastCheckedAt    *string `json:"last_checked_at,omitempty" example:"2025-01-21T10:00:00Z"`
	CreatedAt        string  `json:"created_at" example:"2025-01-21T10:00:00Z"`
	UpdatedAt        string  `json:"updated_at" example:"2025-01-21T10:00:00Z"`
}

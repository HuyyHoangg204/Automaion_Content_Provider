package models

import (
	"time"
)

// Topic represents a logical topic/chủ đề that user creates.
// In the new model, a Topic is only a container for scripts/projects and knowledge,
// it no longer directly corresponds to a single Gemini Gem.
type Topic struct {
	// Primary key
	ID            string `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserProfileID string `json:"user_profile_id" gorm:"not null;index;type:uuid"`

	// Basic info
	Name string `json:"name" gorm:"type:varchar(255);not null" example:"Lịch sử"`

	// Content
	Description string `json:"description" gorm:"type:text" example:"Chủ đề về lịch sử Việt Nam"`

	// Status & Tracking
	IsActive     bool       `json:"is_active" gorm:"default:true;index"`
	LastSyncedAt *time.Time `json:"last_synced_at,omitempty" example:"2025-01-21T10:30:00Z"`
	SyncStatus   string     `json:"sync_status" gorm:"type:varchar(50);default:'pending';index" example:"synced"` // "pending", "synced", "failed"
	SyncError    string     `json:"sync_error,omitempty" gorm:"type:text" example:"API returned status 404"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	UserProfile UserProfile `json:"user_profile,omitempty" gorm:"foreignKey:UserProfileID;references:ID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for the Topic model
func (Topic) TableName() string {
	return "topics"
}

// CreateTopicRequest represents the request to create a new topic
type CreateTopicRequest struct {
	Name        string `json:"name" binding:"required" example:"Lịch sử"`
	Description string `json:"description" example:"Chủ đề về lịch sử Việt Nam"`
}

// UpdateTopicRequest represents the request to update a topic
type UpdateTopicRequest struct {
	Name        string `json:"name,omitempty" example:"Lịch sử Việt Nam"`
	Description string `json:"description,omitempty" example:"Updated description"`
	IsActive    *bool  `json:"is_active,omitempty" example:"true"`
}

// TopicResponse represents the response for topic operations
type TopicResponse struct {
	ID            string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserProfileID string  `json:"user_profile_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Name          string  `json:"name" example:"Lịch sử"`
	Description   string  `json:"description" example:"Chủ đề về lịch sử Việt Nam"`
	IsActive      bool    `json:"is_active" example:"true"`
	LastSyncedAt  *string `json:"last_synced_at,omitempty" example:"2025-01-21T10:30:00Z"`
	SyncStatus    string  `json:"sync_status" example:"synced"`
	SyncError     string  `json:"sync_error,omitempty" example:""`
	CreatedAt     string  `json:"created_at" example:"2025-01-21T10:00:00Z"`
	UpdatedAt     string  `json:"updated_at" example:"2025-01-21T10:00:00Z"`
}

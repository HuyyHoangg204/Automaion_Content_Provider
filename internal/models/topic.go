package models

import (
	"time"
)

// Topic represents a topic/chủ đề that user creates, which corresponds to a Gemini Gem
type Topic struct {
	// Primary key
	ID            string `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserProfileID string `json:"user_profile_id" gorm:"not null;index;type:uuid"`

	// Basic info
	Name string `json:"name" gorm:"type:varchar(255);not null" example:"Lịch sử"`

	// Gemini Gem info
	GeminiGemID      *string `json:"gemini_gem_id,omitempty" gorm:"type:varchar(255);index" example:"gemini-gem-123"`
	GeminiGemName    string  `json:"gemini_gem_name" gorm:"type:varchar(255)" example:"My History Gem"`
	GeminiAccountID  *string `json:"gemini_account_id,omitempty" gorm:"type:uuid;index"` // Gemini account được dùng để tạo topic này

	// Content
	Description      string `json:"description" gorm:"type:text" example:"Chủ đề về lịch sử Việt Nam"`
	Instructions     string `json:"instructions" gorm:"type:text" example:"You are a history expert..."`
	KnowledgeFiles   JSON   `json:"knowledge_files" gorm:"type:jsonb"` // Array of file paths/IDs
	NotebooklmPrompt string `json:"notebooklm_prompt,omitempty" gorm:"type:text" example:"Tóm tắt chi tiết..."`
	SendPromptText   string `json:"send_prompt_text,omitempty" gorm:"type:text" example:"Prompt text to send..."`

	// Status & Tracking
	IsActive     bool       `json:"is_active" gorm:"default:true;index"`
	LastSyncedAt *time.Time `json:"last_synced_at,omitempty" example:"2025-01-21T10:30:00Z"`
	SyncStatus   string     `json:"sync_status" gorm:"type:varchar(50);default:'pending';index" example:"synced"` // "pending", "synced", "failed"
	SyncError    string     `json:"sync_error,omitempty" gorm:"type:text" example:"API returned status 404"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	UserProfile     UserProfile     `json:"user_profile,omitempty" gorm:"foreignKey:UserProfileID;references:ID;constraint:OnDelete:CASCADE"`
	GeminiAccount   *GeminiAccount  `json:"gemini_account,omitempty" gorm:"foreignKey:GeminiAccountID;references:ID;constraint:OnDelete:SET NULL"`
}

// TableName specifies the table name for the Topic model
func (Topic) TableName() string {
	return "topics"
}

// CreateTopicRequest represents the request to create a new topic
type CreateTopicRequest struct {
	Name             string   `json:"name" binding:"required" example:"Lịch sử"`
	Description      string   `json:"description" example:"Chủ đề về lịch sử Việt Nam"`
	Instructions     string   `json:"instructions" example:"You are a history expert..."`
	KnowledgeFiles   []string `json:"knowledge_files,omitempty" example:"[\"file1.pdf\", \"file2.txt\"]"`
	NotebooklmPrompt string   `json:"notebooklm_prompt,omitempty" example:"Tóm tắt chi tiết..."`
	SendPromptText   string   `json:"send_prompt_text,omitempty" example:"Prompt text to send..."`
}

// UpdateTopicRequest represents the request to update a topic
type UpdateTopicRequest struct {
	Name             string   `json:"name,omitempty" example:"Lịch sử Việt Nam"`
	Description      string   `json:"description,omitempty" example:"Updated description"`
	Instructions     string   `json:"instructions,omitempty" example:"Updated instructions"`
	KnowledgeFiles   []string `json:"knowledge_files,omitempty" example:"[\"file1.pdf\", \"file2.txt\"]"`
	NotebooklmPrompt string   `json:"notebooklm_prompt,omitempty" example:"Tóm tắt chi tiết..."`
	SendPromptText   string   `json:"send_prompt_text,omitempty" example:"Prompt text to send..."`
	IsActive         *bool    `json:"is_active,omitempty" example:"true"`
}

// UpdateTopicPromptsRequest represents the request to update topic prompts only
type UpdateTopicPromptsRequest struct {
	NotebooklmPrompt string `json:"notebooklm_prompt" example:"Tóm tắt chi tiết..."`
	SendPromptText   string `json:"send_prompt_text" example:"Prompt text to send..."`
}

// TopicPromptsResponse represents the response for topic prompts
type TopicPromptsResponse struct {
	ID               string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	NotebooklmPrompt string `json:"notebooklm_prompt" example:"Tóm tắt chi tiết..."`
	SendPromptText   string `json:"send_prompt_text" example:"Prompt text to send..."`
}

// TopicResponse represents the response for topic operations
type TopicResponse struct {
	ID               string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserProfileID    string  `json:"user_profile_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Name             string  `json:"name" example:"Lịch sử"`
	GeminiGemID      *string `json:"gemini_gem_id,omitempty" example:"gemini-gem-123"`
	GeminiGemName    string  `json:"gemini_gem_name" example:"My History Gem"`
	Description      string  `json:"description" example:"Chủ đề về lịch sử Việt Nam"`
	Instructions     string  `json:"instructions" example:"You are a history expert..."`
	KnowledgeFiles   JSON    `json:"knowledge_files"`
	NotebooklmPrompt string  `json:"notebooklm_prompt,omitempty" example:"Tóm tắt chi tiết..."`
	SendPromptText   string  `json:"send_prompt_text,omitempty" example:"Prompt text to send..."`
	IsActive         bool    `json:"is_active" example:"true"`
	LastSyncedAt     *string `json:"last_synced_at,omitempty" example:"2025-01-21T10:30:00Z"`
	SyncStatus       string  `json:"sync_status" example:"synced"`
	SyncError        string  `json:"sync_error,omitempty" example:""`
	CreatedAt        string  `json:"created_at" example:"2025-01-21T10:00:00Z"`
	UpdatedAt        string  `json:"updated_at" example:"2025-01-21T10:00:00Z"`
}

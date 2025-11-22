package models

import (
	"time"
)

// UserProfile represents a user's automation profile (1-1 with User)
type UserProfile struct {
	// Primary key
	ID     string `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID string `json:"user_id" gorm:"not null;unique;index;type:uuid"`

	// Basic info
	Name string `json:"name" gorm:"type:varchar(255);not null"`

	// Paths & Directories
	// Note: UserDataDir không lưu trong DB vì path khác nhau trên mỗi máy automation
	// Automation backend tự resolve path: path.join(os.homedir(), 'AppData', 'Local', 'Automation_Profiles')
	ProfileDirName string `json:"profile_dir_name" gorm:"type:varchar(255);not null"`

	// ⭐ QUAN TRỌNG: Track sync status trên từng machine
	MachineSyncStatus JSON `json:"machine_sync_status" gorm:"type:jsonb"`
	// Format: { "app-id": { "platform_profile_id": "...", "last_synced_at": "...", "sync_version": 1, "is_synced": true } }

	// Distribution & Load balancing
	DeployedMachines JSON    `json:"deployed_machines" gorm:"type:jsonb"` // Array of AppIDs
	CurrentMachineID *string `json:"current_machine_id" gorm:"type:uuid"` // Machine đang chạy
	CurrentAppID     *string `json:"current_app_id" gorm:"type:uuid"`     // AppID đang chạy

	// Run tracking
	LastRunStartedAt *time.Time `json:"last_run_started_at"`
	LastRunEndedAt   *time.Time `json:"last_run_ended_at"`

	// Version tracking
	ProfileVersion int        `json:"profile_version" gorm:"default:1"`
	LastModifiedAt *time.Time `json:"last_modified_at"`

	// Configuration
	Config   JSON `json:"config" gorm:"type:jsonb"`   // Automation config, scripts
	Settings JSON `json:"settings" gorm:"type:jsonb"` // User preferences

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	User   User    `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	Topics []Topic `json:"topics,omitempty" gorm:"foreignKey:UserProfileID;references:ID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for the UserProfile model
func (UserProfile) TableName() string {
	return "user_profiles"
}

// CreateUserProfileRequest represents the request to create a new user profile
type CreateUserProfileRequest struct {
	Name           string `json:"name" binding:"required" example:"My Profile"`
	ProfileDirName string `json:"profile_dir_name" binding:"required" example:"Profile_username"`
	Config         JSON   `json:"config,omitempty"`   // Optional automation config
	Settings       JSON   `json:"settings,omitempty"` // Optional user settings
}

// UpdateUserProfileRequest represents the request to update a user profile
type UpdateUserProfileRequest struct {
	Name     string `json:"name,omitempty" example:"Updated Profile Name"`
	Config   JSON   `json:"config,omitempty"`   // Update automation config
	Settings JSON   `json:"settings,omitempty"` // Update user settings
}

// UserProfileResponse represents the response for user profile operations
type UserProfileResponse struct {
	ID               string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID           string  `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Name             string  `json:"name" example:"My Profile"`
	ProfileDirName   string  `json:"profile_dir_name" example:"Profile_username"`
	DeployedMachines JSON    `json:"deployed_machines"` // Array of AppIDs
	CurrentAppID     *string `json:"current_app_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440002"`
	LastRunStartedAt *string `json:"last_run_started_at,omitempty" example:"2025-01-21T10:30:00Z"`
	LastRunEndedAt   *string `json:"last_run_ended_at,omitempty" example:"2025-01-21T10:35:00Z"`
	ProfileVersion   int     `json:"profile_version" example:"1"`
	LastModifiedAt   *string `json:"last_modified_at,omitempty" example:"2025-01-21T10:00:00Z"`
	Config           JSON    `json:"config,omitempty"`
	Settings         JSON    `json:"settings,omitempty"`
	CreatedAt        string  `json:"created_at" example:"2025-01-21T10:00:00Z"`
	UpdatedAt        string  `json:"updated_at" example:"2025-01-21T10:00:00Z"`
}

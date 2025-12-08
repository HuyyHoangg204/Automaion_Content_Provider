package models

import (
	"time"
)

// TopicUser represents the many-to-many relationship between topics and users
// This allows admin to assign topics to users for access
type TopicUser struct {
	ID             string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TopicID        string    `json:"topic_id" gorm:"not null;index;type:uuid"`
	UserID         string    `json:"user_id" gorm:"not null;index;type:uuid"`
	AssignedBy     *string   `json:"assigned_by,omitempty" gorm:"type:uuid"` // Admin who assigned (nullable)
	AssignedAt     time.Time `json:"assigned_at" gorm:"default:now()"`
	PermissionType string    `json:"permission_type" gorm:"type:varchar(20);default:'read';index"` // 'read', 'write', 'full'

	// Relationships
	Topic Topic `json:"topic,omitempty" gorm:"foreignKey:TopicID;references:ID;constraint:OnDelete:CASCADE"`
	User  User  `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for the TopicUser model
func (TopicUser) TableName() string {
	return "topic_users"
}

// AssignTopicRequest represents the request to assign a topic to a user (Admin only)
type AssignTopicRequest struct {
	UserID         string `json:"user_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	PermissionType string `json:"permission_type,omitempty" example:"read"` // 'read', 'write', 'full' (default: 'read')
}

// TopicUserResponse represents a user assigned to a topic
type TopicUserResponse struct {
	ID             string    `json:"id"`
	TopicID        string    `json:"topic_id"`
	UserID         string    `json:"user_id"`
	Username       string    `json:"username"`
	FirstName      string    `json:"first_name"`
	LastName       string    `json:"last_name"`
	AssignedBy     *string   `json:"assigned_by,omitempty"`
	AssignedByUser *string   `json:"assigned_by_user,omitempty"` // Username of admin who assigned
	AssignedAt     time.Time `json:"assigned_at"`
	PermissionType string    `json:"permission_type"`
}

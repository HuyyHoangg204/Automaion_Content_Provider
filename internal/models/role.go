package models

import (
	"time"
)

// Role represents a role in the system (e.g., "topic_creator")
type Role struct {
	ID          string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Name        string    `json:"name" gorm:"type:varchar(100);not null;unique;index" example:"topic_creator"`
	Description string    `json:"description" gorm:"type:text" example:"Can create topics"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	// Relationships
	Users []User `json:"users,omitempty" gorm:"many2many:user_roles;"`
}

// TableName specifies the table name for the Role model
func (Role) TableName() string {
	return "roles"
}

// AssignRoleRequest represents the request to assign a role to a user
type AssignRoleRequest struct {
	RoleID string `json:"role_id" binding:"required" example:"b45c2631-fe68-4040-ad19-d8949ebad22a"`
}

// RemoveRoleRequest represents the request to remove a role from a user
type RemoveRoleRequest struct {
	RoleID string `json:"role_id" binding:"required" example:"b45c2631-fe68-4040-ad19-d8949ebad22a"`
}

// UserRoleResponse represents a user with their roles
type UserRoleResponse struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"` // List of role names
}

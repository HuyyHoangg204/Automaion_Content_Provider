package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// LoginRequest represents the login request payload
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegisterRequest represents the registration request payload
type RegisterRequest struct {
	Username  string `json:"username" binding:"required,min=3,max=50"`
	Password  string `json:"password" binding:"required,min=6"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	RoleID    string `json:"role_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"` // Optional: Role ID to assign to user. If not provided, defaults to "topic_user"
}

// AuthResponse represents the authentication response
type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	User         User   `json:"user"`
}

// RefreshTokenRequest represents the refresh token request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// LogoutRequest represents the logout request
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// JWTClaims represents the JWT claims
type JWTClaims struct {
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
	TokenVersion uint   `json:"token_version"`
	jwt.RegisteredClaims
}

// TokenInfo represents token information
type TokenInfo struct {
	UserID       string    `json:"user_id"`
	Username     string    `json:"username"`
	TokenVersion uint      `json:"token_version"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// SetUserActiveRequest represents a request to set user active status
type SetUserActiveRequest struct {
	IsActive bool `json:"is_active"`
}

// ChangePasswordRequest represents a request to change user's own password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

// ResetPasswordRequest represents a request to reset user's password (admin only)
type ResetPasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// AssignBoxRequest represents a request to assign a box to a user (admin only)
type AssignBoxRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

// UserWithRolesResponse represents the response for user profile with roles
type UserWithRolesResponse struct {
	ID          string     `json:"id"`
	Username    string     `json:"username"`
	FirstName   string     `json:"first_name"`
	LastName    string     `json:"last_name"`
	IsActive    bool       `json:"is_active"`
	IsAdmin     bool       `json:"is_admin"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	Roles       []string   `json:"roles"` // List of role names
}

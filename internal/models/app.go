package models

import (
	"time"
)

// App represents an application that the system supports (Hidemium, Genlogin, etc.)
type App struct {
	ID        string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	BoxID     string    `json:"box_id" gorm:"not null;index;type:uuid"`
	Name      string    `json:"name" gorm:"type:varchar(255);not null"`
	TunnelURL *string   `json:"tunnel_url,omitempty" gorm:"type:text"` // Optional tunnel URL
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	Box      Box       `json:"box,omitempty" gorm:"foreignKey:BoxID;references:ID;constraint:OnDelete:CASCADE"`
	Profiles []Profile `json:"profiles,omitempty" gorm:"foreignKey:AppID;references:ID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for the App model
func (App) TableName() string {
	return "apps"
}

// CreateAppRequest represents the request to create a new app
type CreateAppRequest struct {
	BoxID     string  `json:"box_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name      string  `json:"name" binding:"required" example:"Hidemium"`
	TunnelURL *string `json:"tunnel_url,omitempty" example:"http://machineid-platform-userid.agent-controller.onegreen.cloud/"`
}

// UpdateAppRequest represents the request to update an app
type UpdateAppRequest struct {
	Name      string  `json:"name" binding:"required" example:"Updated App Name"`
	TunnelURL *string `json:"tunnel_url,omitempty" example:"http://machineid-platform-userid.agent-controller.onegreen.cloud/"`
}

// RegisterAppResponse represents the response for register-app API
// @Description Response containing subdomain and FRP configuration for app registration
type RegisterAppResponse struct {
	// @Description Subdomain configuration for the requested platform
	// @Example {"hidemium": "machineid-hidemium-userid"}
	SubDomain map[string]string `json:"subDomain"`

	// @Description FRP domain
	// @Example "frp.onegreen.cloud"
	FrpServerDomain string `json:"frpServerDomain"`

	// @Description FRP server port
	// @Example 8080
	FrpServerPort int `json:"frpServerPort"`

	// @Description FRP token
	// @Example "HelloWorld"
	FrpToken string `json:"frpToken"`

	// @Description FRP protocol
	// @Example "tcp"
	FrpProtocol string `json:"frpProtocol"`

	// @Description FRP custom domain host
	// @Example "agent-controller.onegreen.cloud"
	FrpCustomDomain string `json:"frpCustomDomain"`
}

// CheckTunnelResponse represents the response for tunnel check API
type CheckTunnelResponse struct {
	// @Description Whether the tunnel URL is accessible
	// @Example true
	IsAccessible bool `json:"is_accessible"`

	// @Description Response time in milliseconds
	// @Example 150
	ResponseTime int64 `json:"response_time_ms"`

	// @Description Status message
	// @Example "Tunnel is accessible and ready for Hidemium"
	Message string `json:"message"`

	// @Description HTTP status code if available
	// @Example 200
	StatusCode *int `json:"status_code,omitempty"`

	// @Description Error message if tunnel is not accessible
	// @Example "Connection timeout"
	Error *string `json:"error,omitempty"`
}

// SyncAppProfilesResponse represents the response for syncing app profiles
type SyncAppProfilesResponse struct {
	ProfilesCreated int    `json:"profiles_created"`
	ProfilesUpdated int    `json:"profiles_updated"`
	ProfilesDeleted int    `json:"profiles_deleted"`
	ProfilesSynced  int    `json:"profiles_synced"`
	Message         string `json:"message"`
}

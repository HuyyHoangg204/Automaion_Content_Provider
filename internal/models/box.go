package models

import (
	"time"
)

// Box represents a machine/computer that belongs to a user
type Box struct {
	ID        string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID    string    `json:"user_id" gorm:"not null;index;type:uuid"`
	MachineID string    `json:"machine_id" gorm:"type:varchar(255);not null;unique;index"`
	Name      string    `json:"name" gorm:"type:varchar(255);not null"`
	IsOnline  bool      `json:"is_online" gorm:"default:false;index"` // Online/offline status
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// System metrics (updated via heartbeat for load balancing)
	CPUUsage        *float64 `json:"cpu_usage,omitempty" gorm:"type:decimal(5,2)"`      // CPU usage percentage
	MemoryFreeGB    *float64 `json:"memory_free_gb,omitempty" gorm:"type:decimal(5,2)"` // Free memory in GB
	RunningProfiles int      `json:"running_profiles" gorm:"default:0;index"`           // Number of profiles currently running

	// Relationships
	User User  `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	Apps []App `json:"apps,omitempty" gorm:"foreignKey:BoxID;references:ID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for the Box model
func (Box) TableName() string {
	return "boxes"
}

// BoxAlreadyExistsError represents an error when a box with the same machine ID already exists
type BoxAlreadyExistsError struct {
	BoxID     string `json:"box_id"`
	MachineID string `json:"machine_id"`
	Message   string `json:"message"`
}

func (e *BoxAlreadyExistsError) Error() string {
	return e.Message
}

// CreateBoxRequest represents the request to create a new box
type CreateBoxRequest struct {
	MachineID string `json:"machine_id" binding:"required" example:"PC-001"`
	Name      string `json:"name" binding:"required" example:"My Computer"`
}

// UpdateBoxRequest represents the request to update a box
type UpdateBoxRequest struct {
	Name string `json:"name" binding:"required" example:"Updated Computer Name" description:"Required: New name for the box"`
}

// BoxResponse represents the response for box operations
type BoxResponse struct {
	ID        string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID    string `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	MachineID string `json:"machine_id" example:"PC-001"`
	Name      string `json:"name" example:"My Computer"`
	CreatedAt string `json:"created_at" example:"2025-01-09T10:30:00Z"`
	UpdatedAt string `json:"updated_at" example:"2025-01-09T10:30:00Z"`
}

// SyncBoxProfilesResponse represents the response for syncing box profiles
type SyncBoxProfilesResponse struct {
	ProfilesCreated int    `json:"profiles_created"`
	ProfilesUpdated int    `json:"profiles_updated"`
	ProfilesDeleted int    `json:"profiles_deleted"`
	ProfilesSynced  int    `json:"profiles_synced"`
	Message         string `json:"message"`
}

// RegisterMachineRequest represents the request to register a machine
type RegisterMachineRequest struct {
	MachineID string `json:"machine_id" binding:"required" example:"abc123def456..."`
	Name      string `json:"name" binding:"required" example:"My Computer"`
}

// RegisterMachineResponse represents the response for machine registration
type RegisterMachineResponse struct {
	BoxID   string  `json:"box_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID  *string `json:"user_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
	Message string  `json:"message" example:"Machine registered successfully"`
}

// UpdateTunnelURLRequest represents the request to update tunnel URL
type UpdateTunnelURLRequest struct {
	TunnelURL string `json:"tunnel_url" binding:"required" example:"http://machineid-automation-userid.agent-controller.onegreen.cloud/"`
}

// UpdateTunnelURLResponse represents the response for tunnel URL update
type UpdateTunnelURLResponse struct {
	Success bool   `json:"success" example:"true"`
	AppID   string `json:"app_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Message string `json:"message" example:"Tunnel URL updated successfully"`
}

// HeartbeatRequest represents the request for machine heartbeat
type HeartbeatRequest struct {
	TunnelURL       string      `json:"tunnel_url" example:"http://machineid-automation-userid.agent-controller.onegreen.cloud/"`
	TunnelConnected bool        `json:"tunnel_connected" example:"true"`
	APIRunning      bool        `json:"api_running" example:"true"`
	APIPort         int         `json:"api_port" example:"3000"`
	SystemInfo      *SystemInfo `json:"system_info,omitempty"` // System metrics for load balancing
}

// SystemInfo represents system metrics sent in heartbeat
type SystemInfo struct {
	CPU      *CPUInfo      `json:"cpu,omitempty"`
	Memory   *MemoryInfo   `json:"memory,omitempty"`
	Profiles *ProfilesInfo `json:"profiles,omitempty"`
}

// CPUInfo represents CPU metrics
type CPUInfo struct {
	Usage float64 `json:"usage" example:"45.2"` // CPU usage percentage
}

// MemoryInfo represents memory metrics
type MemoryInfo struct {
	FreeGB float64 `json:"freeGB" example:"8.5"` // Free memory in GB
}

// ProfilesInfo represents profile metrics
type ProfilesInfo struct {
	Running int `json:"running" example:"3"` // Number of profiles currently running
}

// HeartbeatResponse represents the response for heartbeat
type HeartbeatResponse struct {
	Success  bool   `json:"success" example:"true"`
	LastSeen string `json:"last_seen" example:"2025-01-21T10:30:00Z"`
	Message  string `json:"message" example:"Heartbeat received"`
}

// BoxWithStatusResponse represents the response for box with online/offline status
type BoxWithStatusResponse struct {
	ID          string           `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID      string           `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	MachineID   string           `json:"machine_id" example:"PC-001"`
	Name        string           `json:"name" example:"My Computer"`
	IsOnline    bool             `json:"is_online" example:"true"`
	LastSeen    string           `json:"last_seen" example:"2025-01-21T10:30:00Z"`
	StatusCheck *StatusCheckInfo `json:"status_check,omitempty"`
	CreatedAt   string           `json:"created_at" example:"2025-01-09T10:30:00Z"`
	UpdatedAt   string           `json:"updated_at" example:"2025-01-09T10:30:00Z"`
}

// StatusCheckInfo contains information about the health check
type StatusCheckInfo struct {
	IsAccessible bool    `json:"is_accessible" example:"true"`
	ResponseTime int64   `json:"response_time_ms" example:"150"`
	Message      string  `json:"message" example:"Tunnel is accessible"`
	StatusCode   *int    `json:"status_code,omitempty" example:"200"`
	Error        *string `json:"error,omitempty"`
}

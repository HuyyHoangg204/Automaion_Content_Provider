package models

import (
	"time"
)

// Profile represents a profile that belongs to an app
type Profile struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	AppID     uint      `json:"app_id" gorm:"not null;index"`
	Name      string    `json:"name" gorm:"type:varchar(255);not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	App             App              `json:"app,omitempty" gorm:"foreignKey:AppID;references:ID;constraint:OnDelete:CASCADE"`
	StatusCampaigns []StatusCampaign `json:"status_campaigns,omitempty" gorm:"foreignKey:ProfileID;references:ID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for the Profile model
func (Profile) TableName() string {
	return "profiles"
}

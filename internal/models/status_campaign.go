package models

import (
	"time"
)

// StatusCampaign represents the completion status of a campaign
type StatusCampaign struct {
	ID         uint       `json:"id" gorm:"primaryKey"`
	CampaignID uint       `json:"campaign_id" gorm:"not null;index"`
	ProfileID  uint       `json:"profile_id" gorm:"not null;index"`
	Status     string     `json:"status" gorm:"type:varchar(255);not null"`
	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`

	// Relationships
	Campaign Campaign `json:"campaign,omitempty" gorm:"foreignKey:CampaignID;references:ID;constraint:OnDelete:CASCADE"`
	Profile  Profile  `json:"profile,omitempty" gorm:"foreignKey:ProfileID;references:ID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for the StatusCampaign model
func (StatusCampaign) TableName() string {
	return "status_campaigns"
}

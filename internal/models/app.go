package models

import (
	"time"
)

// App represents an application that the system supports (Hidemium, Genlogin, etc.)
type App struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	BoxID     uint      `json:"box_id" gorm:"not null;index"`
	Name      string    `json:"name" gorm:"type:varchar(255);not null"`
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

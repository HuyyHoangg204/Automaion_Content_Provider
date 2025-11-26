package models

import (
	"time"
)

// File represents an uploaded file
type File struct {
	// Primary key
	ID string `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`

	// File info
	UserID       string `json:"user_id" gorm:"not null;index;type:uuid"`
	FileName     string `json:"file_name" gorm:"type:varchar(255);not null"`
	OriginalName string `json:"original_name" gorm:"type:varchar(255);not null"`
	MimeType     string `json:"mime_type" gorm:"type:varchar(100)"`
	FileSize     int64  `json:"file_size" gorm:"type:bigint"` // Size in bytes
	FilePath     string `json:"file_path" gorm:"type:varchar(500);not null"` // Path on server storage

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	User User `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for the File model
func (File) TableName() string {
	return "files"
}

// FileUploadRequest represents the request to upload a file
type FileUploadRequest struct {
	Category string `json:"category,omitempty" form:"category" example:"knowledge"`
}

// FileResponse represents the response for file operations
type FileResponse struct {
	ID           string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID       string `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	FileName     string `json:"file_name" example:"abc123.pdf"`
	OriginalName string `json:"original_name" example:"document.pdf"`
	MimeType     string `json:"mime_type" example:"application/pdf"`
	FileSize     int64  `json:"file_size" example:1024`
	DownloadURL  string `json:"download_url" example:"/api/v1/files/550e8400-e29b-41d4-a716-446655440000/download"`
	CreatedAt    string `json:"created_at" example:"2025-01-21T10:00:00Z"`
	UpdatedAt    string `json:"updated_at" example:"2025-01-21T10:00:00Z"`
}



package repository

import (
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"gorm.io/gorm"
)

type FileRepository struct {
	db *gorm.DB
}

func NewFileRepository(db *gorm.DB) *FileRepository {
	return &FileRepository{db: db}
}

// Create creates a new file record
func (r *FileRepository) Create(file *models.File) error {
	return r.db.Create(file).Error
}

// GetByID retrieves a file by ID
func (r *FileRepository) GetByID(id string) (*models.File, error) {
	var file models.File
	err := r.db.Preload("User").First(&file, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

// GetByUserID retrieves all files for a user
func (r *FileRepository) GetByUserID(userID string) ([]*models.File, error) {
	var files []*models.File
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&files).Error
	return files, err
}

// Delete deletes a file record
func (r *FileRepository) Delete(id string) error {
	return r.db.Delete(&models.File{}, "id = ?", id).Error
}



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

// GetByProjectIDAndTempPromptID retrieves files for a specific prompt (before saving script)
func (r *FileRepository) GetByProjectIDAndTempPromptID(userID, projectID, tempPromptID string) ([]*models.File, error) {
	var files []*models.File
	err := r.db.Where("user_id = ? AND project_id = ? AND temp_prompt_id = ?", userID, projectID, tempPromptID).
		Order("created_at DESC").
		Find(&files).Error
	return files, err
}

// UpdateProjectIDAndTempPromptID updates project_id and temp_prompt_id for a file
func (r *FileRepository) UpdateProjectIDAndTempPromptID(fileID, projectID, tempPromptID string) error {
	return r.db.Model(&models.File{}).
		Where("id = ?", fileID).
		Updates(map[string]interface{}{
			"project_id":     projectID,
			"temp_prompt_id": tempPromptID,
		}).Error
}

// ClearProjectIDAndTempPromptID clears project_id and temp_prompt_id for files (after saving script)
func (r *FileRepository) ClearProjectIDAndTempPromptID(userID, projectID, tempPromptID string) error {
	return r.db.Model(&models.File{}).
		Where("user_id = ? AND project_id = ? AND temp_prompt_id = ?", userID, projectID, tempPromptID).
		Updates(map[string]interface{}{
			"project_id":     nil,
			"temp_prompt_id": nil,
		}).Error
}

// Delete deletes a file record
func (r *FileRepository) Delete(id string) error {
	return r.db.Delete(&models.File{}, "id = ?", id).Error
}

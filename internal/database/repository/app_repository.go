package repository

import (
	"fmt"

	"github.com/onegreenvn/green-provider-services-backend/internal/models"

	"gorm.io/gorm"
)

type AppRepository struct {
	db *gorm.DB
}

func NewAppRepository(db *gorm.DB) *AppRepository {
	return &AppRepository{db: db}
}

// Create creates a new app
func (r *AppRepository) Create(app *models.App) error {
	return r.db.Create(app).Error
}

// GetByID retrieves an app by ID
func (r *AppRepository) GetByID(id string) (*models.App, error) {
	var app models.App
	err := r.db.First(&app, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

// GetByUserID retrieves all apps for a specific user (through boxes)
func (r *AppRepository) GetByUserID(userID string) ([]*models.App, error) {
	var apps []*models.App
	err := r.db.Joins("JOIN boxes ON apps.box_id = boxes.id").
		Where("boxes.user_id = ?", userID).
		Find(&apps).Error
	return apps, err
}

// GetByUserIDAndBoxID retrieves all apps for a specific user and box
func (r *AppRepository) GetByUserIDAndBoxID(userID, boxID string) ([]*models.App, error) {
	var apps []*models.App
	err := r.db.Joins("JOIN boxes ON apps.box_id = boxes.id").
		Where("boxes.user_id = ? AND apps.box_id = ?", userID, boxID).
		Find(&apps).Error
	return apps, err
}

// GetByBoxID retrieves all apps for a specific box
func (r *AppRepository) GetByBoxID(boxID string) ([]*models.App, error) {
	var apps []*models.App
	err := r.db.Where("box_id = ?", boxID).Find(&apps).Error
	return apps, err
}

// GetByUserIDAndID retrieves an app by user ID and app ID
func (r *AppRepository) GetByUserIDAndID(userID, appID string) (*models.App, error) {
	var app models.App
	err := r.db.Joins("JOIN boxes ON apps.box_id = boxes.id").
		Where("boxes.user_id = ? AND apps.id = ?", userID, appID).
		First(&app).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

// Update updates an app
func (r *AppRepository) Update(app *models.App) error {
	return r.db.Save(app).Error
}

// DeleteByUserIDAndID deletes an app by user ID and app ID
func (r *AppRepository) DeleteByUserIDAndID(userID, appID string) error {
	result := r.db.Unscoped().Where("id = ? AND box_id IN (SELECT id FROM boxes WHERE user_id = ?)", appID, userID).
		Delete(&models.App{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete app: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("app not found or access denied")
	}
	return nil
}

// CheckNameExistsInBox checks if an app name already exists in a specific box
func (r *AppRepository) CheckNameExistsInBox(boxID, name string) (bool, error) {
	var count int64
	err := r.db.Model(&models.App{}).Where("box_id = ? AND name = ?", boxID, name).Count(&count).Error
	return count > 0, err
}

// GetByNameAndBoxID retrieves an app by name and box ID
func (r *AppRepository) GetByNameAndBoxID(boxID, name string) (*models.App, error) {
	var app models.App
	err := r.db.Where("box_id = ? AND name = ?", boxID, name).First(&app).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

// GetAll retrieves all apps (admin only)
func (r *AppRepository) GetAll() ([]*models.App, error) {
	var apps []*models.App
	err := r.db.Find(&apps).Error
	return apps, err
}

package repository

import (
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
	err := r.db.Preload("Box").Preload("Profiles").First(&app, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

// GetByBoxID retrieves all apps for a specific box
func (r *AppRepository) GetByBoxID(boxID string) ([]*models.App, error) {
	var apps []*models.App
	err := r.db.Where("box_id = ?", boxID).Preload("Profiles").Find(&apps).Error
	return apps, err
}

// GetByUserID retrieves all apps for a specific user (through boxes)
func (r *AppRepository) GetByUserID(userID string) ([]*models.App, error) {
	var apps []*models.App
	err := r.db.Joins("JOIN boxes ON apps.box_id = boxes.id").
		Where("boxes.user_id = ?", userID).
		Preload("Box").
		Preload("Profiles").
		Find(&apps).Error
	return apps, err
}

// GetByUserIDAndID retrieves an app by user ID and app ID
func (r *AppRepository) GetByUserIDAndID(userID, appID string) (*models.App, error) {
	var app models.App
	err := r.db.Joins("JOIN boxes ON apps.box_id = boxes.id").
		Where("boxes.user_id = ? AND apps.id = ?", userID, appID).
		Preload("Box").
		Preload("Profiles").
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

// Delete deletes an app
func (r *AppRepository) Delete(id string) error {
	return r.db.Delete(&models.App{}, "id = ?", id).Error
}

// DeleteByUserIDAndID deletes an app by user ID and app ID
func (r *AppRepository) DeleteByUserIDAndID(userID, appID string) error {
	return r.db.Joins("JOIN boxes ON apps.box_id = boxes.id").
		Where("boxes.user_id = ? AND apps.id = ?", userID, appID).
		Delete(&models.App{}).Error
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
	err := r.db.Preload("Box").Preload("Profiles").Find(&apps).Error
	return apps, err
}

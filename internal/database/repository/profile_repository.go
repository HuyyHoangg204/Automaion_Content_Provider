package repository

import (
	"green-anti-detect-browser-backend-v1/internal/models"

	"gorm.io/gorm"
)

type ProfileRepository struct {
	db *gorm.DB
}

func NewProfileRepository(db *gorm.DB) *ProfileRepository {
	return &ProfileRepository{db: db}
}

// Create creates a new profile
func (r *ProfileRepository) Create(profile *models.Profile) error {
	return r.db.Create(profile).Error
}

// GetByID retrieves a profile by ID
func (r *ProfileRepository) GetByID(id string) (*models.Profile, error) {
	var profile models.Profile
	err := r.db.Preload("App").Preload("Flows").First(&profile, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

// GetByAppID retrieves all profiles for a specific app
func (r *ProfileRepository) GetByAppID(appID string) ([]*models.Profile, error) {
	var profiles []*models.Profile
	err := r.db.Where("app_id = ?", appID).Preload("Flows").Find(&profiles).Error
	return profiles, err
}

// GetByUserID retrieves all profiles for a specific user (through apps and boxes)
func (r *ProfileRepository) GetByUserID(userID string) ([]*models.Profile, error) {
	var profiles []*models.Profile
	err := r.db.Joins("JOIN apps ON profiles.app_id = apps.id").
		Joins("JOIN boxes ON apps.box_id = boxes.id").
		Where("boxes.user_id = ?", userID).
		Preload("App").
		Preload("Flows").
		Find(&profiles).Error
	return profiles, err
}

// GetByUserIDAndID retrieves a profile by user ID and profile ID
func (r *ProfileRepository) GetByUserIDAndID(userID, profileID string) (*models.Profile, error) {
	var profile models.Profile
	err := r.db.Joins("JOIN apps ON profiles.app_id = apps.id").
		Joins("JOIN boxes ON apps.box_id = boxes.id").
		Where("boxes.user_id = ? AND profiles.id = ?", userID, profileID).
		Preload("App").
		Preload("Flows").
		First(&profile).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

// Update updates a profile
func (r *ProfileRepository) Update(profile *models.Profile) error {
	return r.db.Save(profile).Error
}

// Delete deletes a profile
func (r *ProfileRepository) Delete(id string) error {
	return r.db.Delete(&models.Profile{}, "id = ?", id).Error
}

// DeleteByUserIDAndID deletes a profile by user ID and profile ID
func (r *ProfileRepository) DeleteByUserIDAndID(userID, profileID string) error {
	return r.db.Joins("JOIN apps ON profiles.app_id = apps.id").
		Joins("JOIN boxes ON apps.box_id = boxes.id").
		Where("boxes.user_id = ? AND profiles.id = ?", userID, profileID).
		Delete(&models.Profile{}).Error
}

// CheckNameExistsInApp checks if a profile name already exists in a specific app
func (r *ProfileRepository) CheckNameExistsInApp(appID, name string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Profile{}).Where("app_id = ? AND name = ?", appID, name).Count(&count).Error
	return count > 0, err
}

// GetAll retrieves all profiles (admin only)
func (r *ProfileRepository) GetAll() ([]*models.Profile, error) {
	var profiles []*models.Profile
	err := r.db.Preload("App").Preload("Flows").Find(&profiles).Error
	return profiles, err
}

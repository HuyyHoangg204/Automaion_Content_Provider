package repository

import (
	"green-anti-detect-browser-backend-v1/internal/models"
	"green-anti-detect-browser-backend-v1/internal/utils"

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

// GetByBoxID retrieves all profiles for a specific box
func (r *ProfileRepository) GetByBoxID(boxID string) ([]*models.Profile, error) {
	var profiles []*models.Profile
	err := r.db.Joins("JOIN apps ON profiles.app_id = apps.id").
		Where("apps.box_id = ?", boxID).
		Preload("App").
		Preload("Flows").
		Find(&profiles).Error
	return profiles, err
}

// GetByBoxIDPaginated retrieves paginated profiles for a specific box
func (r *ProfileRepository) GetByBoxIDPaginated(boxID string, page, pageSize int) ([]*models.Profile, int, error) {
	var profiles []*models.Profile
	var total int64

	// Count total records
	err := r.db.Joins("JOIN apps ON profiles.app_id = apps.id").
		Where("apps.box_id = ?", boxID).
		Model(&models.Profile{}).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// Calculate offset
	offset := utils.CalculateOffset(page, pageSize)

	// Get paginated results
	err = r.db.Joins("JOIN apps ON profiles.app_id = apps.id").
		Where("apps.box_id = ?", boxID).
		Preload("App").
		Preload("Flows").
		Offset(offset).
		Limit(pageSize).
		Order("profiles.created_at DESC").
		Find(&profiles).Error

	return profiles, int(total), err
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

// GetByUserIDPaginated retrieves paginated profiles for a specific user
func (r *ProfileRepository) GetByUserIDPaginated(userID string, page, pageSize int) ([]*models.Profile, int, error) {
	var profiles []*models.Profile
	var total int64

	// Count total records
	err := r.db.Joins("JOIN apps ON profiles.app_id = apps.id").
		Joins("JOIN boxes ON apps.box_id = boxes.id").
		Where("boxes.user_id = ?", userID).
		Model(&models.Profile{}).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// Calculate offset
	offset := utils.CalculateOffset(page, pageSize)

	// Get paginated results
	err = r.db.Joins("JOIN apps ON profiles.app_id = apps.id").
		Joins("JOIN boxes ON apps.box_id = boxes.id").
		Where("boxes.user_id = ?", userID).
		Preload("App").
		Preload("Flows").
		Offset(offset).
		Limit(pageSize).
		Order("profiles.created_at DESC").
		Find(&profiles).Error

	return profiles, int(total), err
}

// GetByAppIDPaginated retrieves paginated profiles for a specific app
func (r *ProfileRepository) GetByAppIDPaginated(appID string, page, pageSize int) ([]*models.Profile, int, error) {
	var profiles []*models.Profile
	var total int64

	// Count total records
	err := r.db.Model(&models.Profile{}).Where("app_id = ?", appID).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// Calculate offset
	offset := utils.CalculateOffset(page, pageSize)

	// Get paginated results
	err = r.db.Where("app_id = ?", appID).
		Preload("Flows").
		Offset(offset).
		Limit(pageSize).
		Order("created_at DESC").
		Find(&profiles).Error

	return profiles, int(total), err
}

package repository

import (
	"fmt"

	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/onegreenvn/green-provider-services-backend/internal/utils"

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

// CreateMany creates multiple profiles in a transaction
func (r *ProfileRepository) CreateMany(profiles []*models.Profile) error {
	if len(profiles) == 0 {
		return nil
	}
	return r.db.Create(&profiles).Error
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

// GetByIDs retrieves profiles by a list of IDs
func (r *ProfileRepository) GetByIDs(ids []string) ([]*models.Profile, error) {
	var profiles []*models.Profile
	err := r.db.Where("id IN ?", ids).Find(&profiles).Error
	return profiles, err
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

// UpdateMany updates multiple profiles in a transaction
func (r *ProfileRepository) UpdateMany(profiles []*models.Profile) error {
	if len(profiles) == 0 {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, profile := range profiles {
			if err := tx.Save(profile).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// DeleteMany deletes multiple profiles by their IDs
func (r *ProfileRepository) DeleteMany(profiles []*models.Profile) error {
	if len(profiles) == 0 {
		return nil
	}
	profileIDs := make([]string, len(profiles))
	for i, p := range profiles {
		profileIDs[i] = p.ID
	}
	return r.db.Unscoped().Where("id IN ?", profileIDs).Delete(&models.Profile{}).Error
}

// DeleteByUserIDAndID deletes a profile by user ID and profile ID
func (r *ProfileRepository) DeleteByUserIDAndID(userID, profileID string) error {
	// The subquery finds the profile ID to delete by joining through apps and boxes to check ownership.
	subQuery := r.db.Model(&models.Profile{}).
		Joins("JOIN apps ON profiles.app_id = apps.id").
		Joins("JOIN boxes ON apps.box_id = boxes.id").
		Where("boxes.user_id = ?", userID).
		Where("profiles.id = ?", profileID).
		Select("profiles.id")

	// We delete from profiles where the ID is in the result of the subquery.
	// This ensures we only delete the profile if it belongs to the user.
	// Note: Flows will be automatically deleted due to CASCADE constraint
	result := r.db.Unscoped().Where("id IN (?)", subQuery).Delete(&models.Profile{})

	if result.Error != nil {
		return fmt.Errorf("failed to delete profile: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("profile not found or access denied")
	}

	return nil
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

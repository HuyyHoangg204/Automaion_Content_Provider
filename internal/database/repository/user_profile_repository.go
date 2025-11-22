package repository

import (
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"gorm.io/gorm"
)

type UserProfileRepository struct {
	db *gorm.DB
}

func NewUserProfileRepository(db *gorm.DB) *UserProfileRepository {
	return &UserProfileRepository{db: db}
}

// Create creates a new user profile
func (r *UserProfileRepository) Create(profile *models.UserProfile) error {
	return r.db.Create(profile).Error
}

// GetByID retrieves a user profile by ID
func (r *UserProfileRepository) GetByID(id string) (*models.UserProfile, error) {
	var profile models.UserProfile
	err := r.db.Preload("User").First(&profile, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

// GetByUserID retrieves a user profile by user ID (1-1 relationship)
func (r *UserProfileRepository) GetByUserID(userID string) (*models.UserProfile, error) {
	var profile models.UserProfile
	err := r.db.Preload("User").Where("user_id = ?", userID).First(&profile).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

// Update updates a user profile
func (r *UserProfileRepository) Update(profile *models.UserProfile) error {
	return r.db.Save(profile).Error
}

// Delete deletes a user profile
func (r *UserProfileRepository) Delete(id string) error {
	return r.db.Delete(&models.UserProfile{}, "id = ?", id).Error
}

// GetAll retrieves all user profiles (admin only)
func (r *UserProfileRepository) GetAll() ([]*models.UserProfile, error) {
	var profiles []*models.UserProfile
	err := r.db.Preload("User").Find(&profiles).Error
	return profiles, err
}

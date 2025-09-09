package repository

import (
	"errors"
	"time"

	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"gorm.io/gorm"
)

// APIKeyRepository handles database operations for APIKey entities
type APIKeyRepository struct {
	db *gorm.DB
}

// NewAPIKeyRepository creates a new APIKeyRepository instance
func NewAPIKeyRepository(db *gorm.DB) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

// GetByKey retrieves an API key by its key value
func (r *APIKeyRepository) GetByKey(key string) (*models.APIKey, error) {
	var apiKey models.APIKey
	if err := r.db.Where("key = ?", key).First(&apiKey).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil, nil when not found
		}
		return nil, err
	}
	return &apiKey, nil
}

// GetByUserID retrieves an API key by user ID
func (r *APIKeyRepository) GetByUserID(userID string) (*models.APIKey, error) {
	var apiKey models.APIKey
	if err := r.db.Where("user_id = ?", userID).First(&apiKey).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil, nil when not found
		}
		return nil, err
	}
	return &apiKey, nil
}

// Create adds a new API key
func (r *APIKeyRepository) Create(apiKey *models.APIKey) (*models.APIKey, error) {
	if err := r.db.Create(apiKey).Error; err != nil {
		return nil, err
	}
	return apiKey, nil
}

// Update modifies an existing API key
func (r *APIKeyRepository) Update(id uint, updates map[string]interface{}) (*models.APIKey, error) {
	// First check if the API key exists
	var apiKey models.APIKey
	if err := r.db.Where("id = ?", id).First(&apiKey).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // API key not found
		}
		return nil, err
	}

	// Update the API key
	if err := r.db.Model(&models.APIKey{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}

	// Retrieve the updated API key
	return r.GetByID(id)
}

// GetByID retrieves an API key by its ID
func (r *APIKeyRepository) GetByID(id uint) (*models.APIKey, error) {
	var apiKey models.APIKey
	if err := r.db.Where("id = ?", id).First(&apiKey).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil, nil when not found
		}
		return nil, err
	}
	return &apiKey, nil
}

// UpdateLastUsed updates the last used timestamp for an API key
func (r *APIKeyRepository) UpdateLastUsed(id uint) error {
	now := time.Now()
	return r.db.Model(&models.APIKey{}).Where("id = ?", id).Update("last_used_at", now).Error
}

// Delete removes an API key by its ID
func (r *APIKeyRepository) Delete(id uint) (bool, error) {
	result := r.db.Unscoped().Delete(&models.APIKey{}, "id = ?", id)
	if result.Error != nil {
		return false, result.Error
	}
	// If no rows were affected, the API key was not found
	return result.RowsAffected > 0, nil
}

// DeleteByUserID removes an API key by user ID
func (r *APIKeyRepository) DeleteByUserID(userID string) (bool, error) {
	result := r.db.Unscoped().Delete(&models.APIKey{}, "user_id = ?", userID)
	if result.Error != nil {
		return false, result.Error
	}
	// If no rows were affected, the API key was not found
	return result.RowsAffected > 0, nil
}

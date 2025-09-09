package api_key

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"gorm.io/gorm"
)

// Service handles API key operations
type Service struct {
	db         *gorm.DB
	apiKeyRepo *repository.APIKeyRepository
	userRepo   *repository.UserRepository
}

// NewService creates a new API key service
func NewService(db *gorm.DB) *Service {
	return &Service{
		db:         db,
		apiKeyRepo: repository.NewAPIKeyRepository(db),
		userRepo:   repository.NewUserRepository(db),
	}
}

// GenerateAPIKey generates a new API key for a user
func (s *Service) GenerateAPIKey(userID string) (*models.APIKey, error) {
	// Check if user exists and is active
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}
	if !user.IsActive {
		return nil, fmt.Errorf("user is not active")
	}

	// Check if user already has an API key and delete it
	existingAPIKey, err := s.apiKeyRepo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing API key: %w", err)
	}

	if existingAPIKey != nil {
		// Delete the existing API key
		_, err = s.apiKeyRepo.Delete(existingAPIKey.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to delete existing API key: %w", err)
		}
	}

	// Generate a random API key
	key, err := s.generateRandomKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	// Create the API key
	apiKey := &models.APIKey{
		Key:      key,
		UserID:   userID,
		IsActive: true,
	}

	createdAPIKey, err := s.apiKeyRepo.Create(apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	return createdAPIKey, nil
}

// ValidateAPIKey validates an API key and returns the associated user
func (s *Service) ValidateAPIKey(key string) (*models.User, error) {
	// Get the API key
	apiKey, err := s.apiKeyRepo.GetByKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}
	if apiKey == nil {
		return nil, fmt.Errorf("invalid API key")
	}

	// Check if API key is active
	if !apiKey.IsActive {
		return nil, fmt.Errorf("API key is disabled")
	}

	// Get the user
	user, err := s.userRepo.GetByID(apiKey.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Check if user is active
	if !user.IsActive {
		return nil, fmt.Errorf("user is not active")
	}

	// Update last used timestamp
	err = s.apiKeyRepo.UpdateLastUsed(apiKey.ID)
	if err != nil {
		// Log the error but don't fail the request
		fmt.Printf("Failed to update API key last used timestamp: %v\n", err)
	}

	return user, nil
}

// GetAPIKeyByUserID gets the API key for a user
func (s *Service) GetAPIKeyByUserID(userID string) (*models.APIKey, error) {
	return s.apiKeyRepo.GetByUserID(userID)
}

// UpdateAPIKeyStatus updates the active status of an API key
func (s *Service) UpdateAPIKeyStatus(userID string, isActive bool) (*models.APIKey, error) {
	// Get the API key
	apiKey, err := s.apiKeyRepo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}
	if apiKey == nil {
		return nil, fmt.Errorf("API key not found")
	}

	// Update the status
	updates := map[string]interface{}{
		"is_active": isActive,
	}

	updatedAPIKey, err := s.apiKeyRepo.Update(apiKey.ID, updates)
	if err != nil {
		return nil, fmt.Errorf("failed to update API key: %w", err)
	}

	return updatedAPIKey, nil
}

// DeleteAPIKey deletes an API key for a user
func (s *Service) DeleteAPIKey(userID string) error {
	// Check if API key exists
	apiKey, err := s.apiKeyRepo.GetByUserID(userID)
	if err != nil {
		return fmt.Errorf("failed to get API key: %w", err)
	}
	if apiKey == nil {
		return fmt.Errorf("API key not found")
	}

	// Delete the API key
	deleted, err := s.apiKeyRepo.Delete(apiKey.ID)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}
	if !deleted {
		return fmt.Errorf("failed to delete API key")
	}

	return nil
}

// generateRandomKey generates a random 32-byte hex string
func (s *Service) generateRandomKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

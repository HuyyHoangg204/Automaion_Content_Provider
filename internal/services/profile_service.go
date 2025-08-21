package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/onegreenvn/green-provider-services-backend/internal/utils"
)

type ProfileService struct {
	profileRepo     *repository.ProfileRepository
	appRepo         *repository.AppRepository
	userRepo        *repository.UserRepository
	boxRepo         *repository.BoxRepository
	platformWrapper *PlatformWrapperService
}

func NewProfileService(profileRepo *repository.ProfileRepository, appRepo *repository.AppRepository, userRepo *repository.UserRepository, boxRepo *repository.BoxRepository) *ProfileService {
	return &ProfileService{
		profileRepo:     profileRepo,
		appRepo:         appRepo,
		userRepo:        userRepo,
		boxRepo:         boxRepo,
		platformWrapper: NewPlatformWrapperService(),
	}
}

// CreateProfile creates a new profile for a user
func (s *ProfileService) CreateProfile(userID string, req *models.CreateProfileRequest) (*models.ProfileResponse, error) {
	// Verify user exists
	_, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Verify app exists and belongs to user
	_, err = s.appRepo.GetByUserIDAndID(userID, req.AppID)
	if err != nil {
		return nil, errors.New("app not found or access denied")
	}

	// Validate data field is not empty
	if len(req.Data) == 0 {
		return nil, errors.New("profile data is required")
	}

	// Check if profile name already exists in this app
	exists, err := s.profileRepo.CheckNameExistsInApp(req.AppID, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check profile name: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("profile with name '%s' already exists in this app", req.Name)
	}

	// Create profile
	profile := &models.Profile{
		AppID: req.AppID,
		Name:  req.Name,
		Data:  req.Data,
	}

	if err := s.profileRepo.Create(profile); err != nil {
		return nil, fmt.Errorf("failed to create profile: %w", err)
	}

	return s.toResponse(profile), nil
}

// GetProfilesByUser retrieves all profiles for a specific user
func (s *ProfileService) GetProfilesByUser(userID string) ([]*models.ProfileResponse, error) {
	profiles, err := s.profileRepo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get profiles: %w", err)
	}

	responses := make([]*models.ProfileResponse, len(profiles))
	for i, profile := range profiles {
		responses[i] = s.toResponse(profile)
	}

	return responses, nil
}

// GetProfilesByBox retrieves all profiles for a specific box (user must own the box)
func (s *ProfileService) GetProfilesByBox(userID, boxID string) ([]*models.ProfileResponse, error) {
	// Verify box belongs to user
	_, err := s.boxRepo.GetByUserIDAndID(userID, boxID)
	if err != nil {
		return nil, errors.New("box not found or access denied")
	}

	profiles, err := s.profileRepo.GetByBoxID(boxID)
	if err != nil {
		return nil, fmt.Errorf("failed to get profiles: %w", err)
	}

	responses := make([]*models.ProfileResponse, len(profiles))
	for i, profile := range profiles {
		responses[i] = s.toResponse(profile)
	}

	return responses, nil
}

// GetProfilesByBoxPaginated retrieves paginated profiles for a specific box
func (s *ProfileService) GetProfilesByBoxPaginated(userID, boxID string, page, pageSize int) ([]*models.ProfileResponse, int, error) {
	// Verify box belongs to user
	_, err := s.boxRepo.GetByUserIDAndID(userID, boxID)
	if err != nil {
		return nil, 0, errors.New("box not found or access denied")
	}

	// Validate and normalize pagination parameters
	page, pageSize = utils.ValidateAndNormalizePagination(page, pageSize)

	profiles, total, err := s.profileRepo.GetByBoxIDPaginated(boxID, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get profiles: %w", err)
	}

	responses := make([]*models.ProfileResponse, len(profiles))
	for i, profile := range profiles {
		responses[i] = s.toResponse(profile)
	}

	return responses, total, nil
}

// GetProfilesByAppPaginated retrieves paginated profiles for a specific app
func (s *ProfileService) GetProfilesByAppPaginated(userID, appID string, page, pageSize int) ([]*models.ProfileResponse, int, error) {
	// Verify app belongs to user
	_, err := s.appRepo.GetByUserIDAndID(userID, appID)
	if err != nil {
		return nil, 0, errors.New("app not found or access denied")
	}

	// Validate and normalize pagination parameters
	page, pageSize = utils.ValidateAndNormalizePagination(page, pageSize)

	profiles, total, err := s.profileRepo.GetByAppIDPaginated(appID, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get profiles: %w", err)
	}

	responses := make([]*models.ProfileResponse, len(profiles))
	for i, profile := range profiles {
		responses[i] = s.toResponse(profile)
	}

	return responses, total, nil
}

// GetProfileByID retrieves a profile by ID (user must own it)
func (s *ProfileService) GetProfileByID(userID, profileID string) (*models.ProfileResponse, error) {
	profile, err := s.profileRepo.GetByUserIDAndID(userID, profileID)
	if err != nil {
		return nil, errors.New("profile not found")
	}

	return s.toResponse(profile), nil
}

// UpdateProfile updates a profile (user must own it)
func (s *ProfileService) UpdateProfile(userID, profileID string, req *models.UpdateProfileRequest) (*models.ProfileResponse, error) {
	profile, err := s.profileRepo.GetByUserIDAndID(userID, profileID)
	if err != nil {
		return nil, errors.New("profile not found")
	}

	// Check if new name already exists in this app (if name is being changed)
	if req.Name != profile.Name {
		exists, err := s.profileRepo.CheckNameExistsInApp(profile.AppID, req.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to check profile name: %w", err)
		}
		if exists {
			return nil, fmt.Errorf("profile with name '%s' already exists in this app", req.Name)
		}
	}

	// Update fields
	profile.Name = req.Name
	profile.Data = req.Data

	if err := s.profileRepo.Update(profile); err != nil {
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}

	return s.toResponse(profile), nil
}

// Now supports multiple platforms through platform system
func (s *ProfileService) DeleteProfile(userID, profileID string) error {
	// Check if profile exists and belongs to user
	profile, err := s.profileRepo.GetByUserIDAndID(userID, profileID)
	if err != nil {
		return errors.New("profile not found")
	}

	// Determine platform from app name
	platformType := s.determinePlatformFromAppName(profile.App.Name)
	if platformType == "" {
		return fmt.Errorf("unsupported platform: %s", profile.App.Name)
	}

	// You may need to adjust this based on your data model
	machineID := s.getMachineIDFromProfile(profile)
	if machineID == "" {
		return fmt.Errorf("machine_id not found for profile")
	}

	fmt.Printf("Starting profile deletion on %s for profile ID: %s, Name: %s, MachineID: %s\n", platformType, profileID, profile.Name, machineID)

	// Use platform wrapper to delete profile
	if err := s.platformWrapper.DeleteProfileOnPlatform(context.Background(), platformType, profile, machineID); err != nil {
		return fmt.Errorf("failed to delete profile on %s: %w", platformType, err)
	}

	fmt.Printf("Profile successfully deleted on %s platform\n", platformType)
	fmt.Printf("Note: Local database will be updated when user syncs the box\n")
	return nil
}

// determinePlatformFromAppName determines platform type from app name
func (s *ProfileService) determinePlatformFromAppName(appName string) string {
	switch appName {
	case "Hidemium":
		return "hidemium"
	case "Genlogin":
		return "genlogin"
	default:
		return ""
	}
}

// getMachineIDFromProfile extracts machine_id from profile data
func (s *ProfileService) getMachineIDFromProfile(profile *models.Profile) string {
	if profile.Data == nil {
		return ""
	}

	// Try to get machine_id from profile data
	if machineID, exists := profile.Data["machine_id"]; exists {
		if machineIDStr, ok := machineID.(string); ok {
			return machineIDStr
		}
	}

	// Try alternative field names
	if machineID, exists := profile.Data["machineId"]; exists {
		if machineIDStr, ok := machineID.(string); ok {
			return machineIDStr
		}
	}

	if machineID, exists := profile.Data["box_machine_id"]; exists {
		if machineIDStr, ok := machineID.(string); ok {
			return machineIDStr
		}
	}

	// If profile has a box relationship, try to get machine_id from there
	// This would require additional database query
	// For now, return empty string and let caller handle it
	return ""
}

// GetAllProfiles retrieves all profiles (admin only)
func (s *ProfileService) GetAllProfiles() ([]*models.ProfileResponse, error) {
	profiles, err := s.profileRepo.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get all profiles: %w", err)
	}

	responses := make([]*models.ProfileResponse, len(profiles))
	for i, profile := range profiles {
		responses[i] = s.toResponse(profile)
	}

	return responses, nil
}

// toResponse converts Profile model to response DTO
func (s *ProfileService) toResponse(profile *models.Profile) *models.ProfileResponse {
	return &models.ProfileResponse{
		ID:        profile.ID,
		AppID:     profile.AppID,
		Name:      profile.Name,
		Data:      profile.Data,
		CreatedAt: profile.CreatedAt.Format(time.RFC3339),
		UpdatedAt: profile.UpdatedAt.Format(time.RFC3339),
	}
}

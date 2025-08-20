package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/onegreenvn/green-provider-services-backend/internal/config"
	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/onegreenvn/green-provider-services-backend/internal/utils"
)

type ProfileService struct {
	profileRepo *repository.ProfileRepository
	appRepo     *repository.AppRepository
	userRepo    *repository.UserRepository
	boxRepo     *repository.BoxRepository
}

func NewProfileService(profileRepo *repository.ProfileRepository, appRepo *repository.AppRepository, userRepo *repository.UserRepository, boxRepo *repository.BoxRepository) *ProfileService {
	return &ProfileService{
		profileRepo: profileRepo,
		appRepo:     appRepo,
		userRepo:    userRepo,
		boxRepo:     boxRepo,
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

// DeleteProfile deletes a profile on the appropriate platform
// Currently only supports Hidemium
func (s *ProfileService) DeleteProfile(userID, profileID string) error {
	// Check if profile exists and belongs to user
	profile, err := s.profileRepo.GetByUserIDAndID(userID, profileID)
	if err != nil {
		return errors.New("profile not found")
	}

	// Check platform from app name
	switch profile.App.Name {
	case "Hidemium":
		fmt.Printf("Starting profile deletion on Hidemium for profile ID: %s, Name: %s\n", profileID, profile.Name)

		// Get the actual Hidemium profile ID from profile.Data.uuid
		hidemiumProfileID := s.getHidemiumProfileID(profile)
		fmt.Printf("Using Hidemium profile ID: %s\n", hidemiumProfileID)

		// Delete profile on Hidemium platform
		if err := s.deleteProfileOnHidemium(profile, hidemiumProfileID); err != nil {
			return fmt.Errorf("failed to delete profile on Hidemium: %w", err)
		}

		fmt.Printf("Profile successfully deleted on Hidemium platform\n")
		fmt.Printf("Note: Local database will be updated when user syncs the box\n")
		return nil

	default:
		return fmt.Errorf("unsupported platform: %s. Currently only supports Hidemium", profile.App.Name)
	}
}

// deleteProfileOnHidemium deletes a profile on Hidemium platform
func (s *ProfileService) deleteProfileOnHidemium(profile *models.Profile, hidemiumProfileID string) error {

	// Get app to find box
	app, err := s.appRepo.GetByID(profile.AppID)
	if err != nil {
		return fmt.Errorf("failed to get app: %w", err)
	}

	// Get box to find machine ID
	box, err := s.boxRepo.GetByID(app.BoxID)
	if err != nil {
		return fmt.Errorf("failed to get box: %w", err)
	}

	fmt.Printf("Deleting profile '%s' on Hidemium for box '%s' (MachineID: %s)\n",
		profile.Name, box.Name, box.MachineID)

	// Get Hidemium config
	hidemiumConfig := config.GetHidemiumConfig()

	// Construct tunnel URL using config
	baseURL := hidemiumConfig.BaseURL
	baseURL = strings.Replace(baseURL, "{machine_id}", box.MachineID, 1)

	// Get delete_profile route from config
	deleteProfileRoute, exists := hidemiumConfig.Routes["delete_profile"]
	if !exists {
		return fmt.Errorf("delete_profile route not found in Hidemium config")
	}

	// Construct full API URL
	apiURL := baseURL + deleteProfileRoute

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Prepare request body for profile deletion
	requestBody := map[string]interface{}{
		"uuid_browser": []string{hidemiumProfileID},
	}

	// Convert to JSON
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create DELETE request (according to Hidemium API docs)
	req, err := http.NewRequest("DELETE", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Make HTTP request to Hidemium API
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to Hidemium API: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("hidemium API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response to check if deletion was successful
	var hidemiumResponse map[string]interface{}
	if err := json.Unmarshal(body, &hidemiumResponse); err != nil {
		return fmt.Errorf("failed to parse Hidemium response: %w", err)
	}

	// Check if deletion was successful based on response
	if responseType, exists := hidemiumResponse["type"]; !exists || responseType != "success" {
		title := "Unknown error"
		if responseTitle, exists := hidemiumResponse["title"]; exists {
			title = fmt.Sprintf("%v", responseTitle)
		}
		return fmt.Errorf("hidemium API deletion failed: %s", title)
	}

	fmt.Printf("Profile successfully deleted on Hidemium platform\n")
	return nil
}

// getHidemiumProfileID extracts the Hidemium profile ID from profile.Data.id
func (s *ProfileService) getHidemiumProfileID(profile *models.Profile) string {
	profileData := profile.Data

	// Look for "uuid" field in the data (this is the Hidemium profile ID)
	if hidemiumID, exists := profileData["uuid"]; exists {
		if hidemiumIDStr, ok := hidemiumID.(string); ok && hidemiumIDStr != "" {
			return hidemiumIDStr
		}
	}
	return profile.ID
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

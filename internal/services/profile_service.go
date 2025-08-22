package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

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

func NewProfileService(ctx context.Context, profileRepo *repository.ProfileRepository, appRepo *repository.AppRepository, userRepo *repository.UserRepository, boxRepo *repository.BoxRepository) *ProfileService {
	return &ProfileService{
		profileRepo: profileRepo,
		appRepo:     appRepo,
		userRepo:    userRepo,
		boxRepo:     boxRepo,
	}
}

// CreateProfile creates a new profile for a user
func (s *ProfileService) CreateProfile(ctx context.Context, userID string, req *models.CreateProfileRequest) (*models.ProfileResponse, error) {
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
	if req.Data == nil || len(req.Data) == 0 {
		return nil, errors.New("profile data is required")
	}

	// Validate that name exists in data
	if req.Data["name"] == nil || req.Data["name"] == "" {
		return nil, fmt.Errorf("name field is required in data")
	}

	// Check if profile name already exists in this app
	profileName := req.Data["name"].(string)
	exists, err := s.profileRepo.CheckNameExistsInApp(req.AppID, profileName)
	if err != nil {
		return nil, fmt.Errorf("failed to check profile name: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("profile with name '%s' already exists in this app", profileName)
	}

	// Get app to determine platform type
	app, err := s.appRepo.GetByID(req.AppID)
	if err != nil {
		return nil, fmt.Errorf("failed to get app: %w", err)
	}

	// Determine platform from app name
	platformType := s.determinePlatformFromAppName(app.Name)
	if platformType == "" {
		return nil, fmt.Errorf("unsupported platform: %s", app.Name)
	}

	// Get box to get machine_id
	box, err := s.boxRepo.GetByID(app.BoxID)
	if err != nil {
		return nil, fmt.Errorf("failed to get box: %w", err)
	}

	// Add machine_id to profile data for platform operations
	if req.Data == nil {
		req.Data = make(map[string]interface{})
	}
	req.Data["machine_id"] = box.MachineID

	// Create profile on platform using box-proxy approach
	platformProfile, err := s.createProfileOnPlatform(app, platformType, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create profile on %s platform: %w", platformType, err)
	}

	// Create profile in local database
	profile := &models.Profile{
		AppID: req.AppID,
		Name:  req.Data["name"].(string),
		Data:  req.Data,
	}

	if err := s.profileRepo.Create(profile); err != nil {
		return nil, fmt.Errorf("failed to create profile in database: %w", err)
	}

	// Update profile with platform UUID if available
	if platformProfile != nil && platformProfile["uuid"] != "" {
		if profile.Data == nil {
			profile.Data = make(map[string]interface{})
		}
		profile.Data["uuid"] = platformProfile["uuid"]

		// Update the profile with UUID
		if err := s.profileRepo.Update(profile); err != nil {
			fmt.Printf("Warning: Failed to update profile with UUID: %v\n", err)
		}
	}

	return s.toResponse(profile), nil
}

// GetProfilesByUser retrieves all profiles for a specific user
func (s *ProfileService) GetProfilesByUser(ctx context.Context, userID string) ([]*models.ProfileResponse, error) {
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
func (s *ProfileService) GetProfilesByBox(ctx context.Context, userID, boxID string) ([]*models.ProfileResponse, error) {
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
func (s *ProfileService) GetProfilesByBoxPaginated(ctx context.Context, userID, boxID string, page, pageSize int) ([]*models.ProfileResponse, int, error) {
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
func (s *ProfileService) GetProfilesByAppPaginated(ctx context.Context, userID, appID string, page, pageSize int) ([]*models.ProfileResponse, int, error) {
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
func (s *ProfileService) GetProfileByID(ctx context.Context, userID, profileID string) (*models.ProfileResponse, error) {
	profile, err := s.profileRepo.GetByUserIDAndID(userID, profileID)
	if err != nil {
		return nil, errors.New("profile not found")
	}

	return s.toResponse(profile), nil
}

// UpdateProfile updates a profile (user must own it)
func (s *ProfileService) UpdateProfile(ctx context.Context, userID, profileID string, req *models.UpdateProfileRequest) (*models.ProfileResponse, error) {
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
func (s *ProfileService) DeleteProfile(ctx context.Context, userID, profileID string) error {
	// Check if profile exists and belongs to user
	profile, err := s.profileRepo.GetByUserIDAndID(userID, profileID)
	if err != nil {
		return errors.New("profile not found")
	}

	// Get app to determine platform type
	app, err := s.appRepo.GetByID(profile.AppID)
	if err != nil {
		return fmt.Errorf("failed to get app: %w", err)
	}

	// Determine platform from app name
	platformType := s.determinePlatformFromAppName(app.Name)
	if platformType == "" {
		return fmt.Errorf("unsupported platform: %s", app.Name)
	}

	// Extract machine_id from profile data
	machineID := s.getMachineIDFromProfile(profile)
	if machineID == "" {
		return fmt.Errorf("machine_id not found for profile")
	}

	fmt.Printf("Starting profile deletion on %s for profile ID: %s, Name: %s, MachineID: %s\n", platformType, profileID, profile.Name, machineID)

	// Delete profile on platform using box-proxy approach
	if err := s.deleteProfileOnPlatform(app, platformType, profile, machineID); err != nil {
		return fmt.Errorf("failed to delete profile on %s: %w", platformType, err)
	}

	fmt.Printf("Profile successfully deleted on %s platform\n", platformType)

	// Delete profile from local database
	if err := s.profileRepo.Delete(profile.ID); err != nil {
		return fmt.Errorf("failed to delete profile from local database: %w", err)
	}

	fmt.Printf("Profile successfully deleted from local database\n")
	return nil
}

// GetDefaultConfigsFromPlatform gets default configuration options from a specific platform
// Now uses HTTP requests to get actual configs from platform
func (s *ProfileService) GetDefaultConfigsFromPlatform(ctx context.Context, userID, platformType, boxID string, page, limit int) (map[string]interface{}, error) {
	// Verify box belongs to user
	_, err := s.boxRepo.GetByUserIDAndID(userID, boxID)
	if err != nil {
		return nil, fmt.Errorf("box not found or access denied")
	}

	// Get an app from this box to get appID and tunnel URL
	apps, err := s.appRepo.GetByBoxID(boxID)
	if err != nil {
		return nil, fmt.Errorf("failed to get apps for box: %w", err)
	}
	if len(apps) == 0 {
		return nil, fmt.Errorf("no apps found for box")
	}

	// Use the first app's ID and tunnel URL
	app := apps[0]
	if app.TunnelURL == nil || *app.TunnelURL == "" {
		return nil, fmt.Errorf("tunnel URL not configured for app %s", app.ID)
	}

	// Build the URL to get default configs from platform
	var configURL string
	switch platformType {
	case "hidemium":
		configURL = fmt.Sprintf("%s/v2/default-config?page=%d&limit=%d", *app.TunnelURL, page, limit)
	case "genlogin":
		configURL = fmt.Sprintf("%s/configs?page=%d&limit=%d", *app.TunnelURL, page, limit)
	default:
		return nil, fmt.Errorf("unsupported platform type: %s", platformType)
	}

	// Make HTTP request to get configs
	client := &http.Client{Timeout: 30 * time.Second}

	var req *http.Request
	var err2 error

	// Both platforms use GET method
	req, err2 = http.NewRequest("GET", configURL, nil)
	if err2 != nil {
		return nil, fmt.Errorf("failed to create request: %w", err2)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Green-Controller/1.0")

	// Make the request
	resp, err2 := client.Do(req)
	if err2 != nil {
		return nil, fmt.Errorf("failed to make request to platform: %w", err2)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("platform returned status %d", resp.StatusCode)
	}

	// Read response body
	body, err2 := io.ReadAll(resp.Body)
	if err2 != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err2)
	}

	// Parse response
	var platformResponse map[string]interface{}
	if err2 := json.Unmarshal(body, &platformResponse); err2 != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err2)
	}

	// Return the platform response with additional metadata
	result := map[string]interface{}{
		"platform": platformType,
		"box_id":   boxID,
		"page":     page,
		"limit":    limit,
		"data":     platformResponse,
	}

	return result, nil
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

// createProfileOnPlatform creates a profile on the platform using HTTP requests
func (s *ProfileService) createProfileOnPlatform(app *models.App, platformType string, req *models.CreateProfileRequest) (map[string]interface{}, error) {
	// Check if tunnel URL is available
	if app.TunnelURL == nil || *app.TunnelURL == "" {
		return nil, fmt.Errorf("tunnel URL not configured for app %s", app.ID)
	}

	// Build the URL to create profile on platform
	var profileURL string
	switch platformType {
	case "hidemium":
		profileURL = fmt.Sprintf("%s/create-profile-customize", *app.TunnelURL)
	case "genlogin":
		profileURL = fmt.Sprintf("%s/profiles/create", *app.TunnelURL)
	default:
		return nil, fmt.Errorf("unsupported platform type: %s", platformType)
	}

	// Prepare request body for profile creation
	requestBody := s.buildCreateProfileRequestBody(req)

	// Convert to JSON
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create POST request to platform API
	httpReq, err := http.NewRequest("POST", profileURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "Green-Controller/1.0")

	// Make HTTP request to platform API
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to platform API: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("platform API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var platformResponse map[string]interface{}
	if err := json.Unmarshal(body, &platformResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract profile information from response
	profileData, err := s.extractCreatedProfileFromResponse(platformResponse, req)
	if err != nil {
		return nil, fmt.Errorf("failed to extract created profile: %w", err)
	}

	fmt.Printf("Profile successfully created on %s platform\n", platformType)
	return profileData, nil
}

// buildCreateProfileRequestBody builds the request body for platform profile creation
func (s *ProfileService) buildCreateProfileRequestBody(req *models.CreateProfileRequest) map[string]interface{} {
	// Start with the original data
	requestBody := make(map[string]interface{})
	for key, value := range req.Data {
		requestBody[key] = value
	}

	// Add platform-specific fields if needed
	// This can be customized based on platform requirements
	return requestBody
}

// extractCreatedProfileFromResponse extracts profile information from platform response
func (s *ProfileService) extractCreatedProfileFromResponse(platformResponse map[string]interface{}, req *models.CreateProfileRequest) (map[string]interface{}, error) {
	// Extract profile information based on platform response structure
	// This is a simplified version - can be enhanced based on actual platform responses

	profileData := make(map[string]interface{})

	// Try to extract common fields
	if data, exists := platformResponse["data"]; exists {
		if profileDataMap, ok := data.(map[string]interface{}); ok {
			profileData = profileDataMap
		}
	}

	// If no data field, use the entire response
	if len(profileData) == 0 {
		profileData = platformResponse
	}

	// Ensure we have at least the name field
	if profileData["name"] == nil {
		profileData["name"] = req.Data["name"]
	}

	return profileData, nil
}

// deleteProfileOnPlatform deletes a profile on the platform using HTTP requests
func (s *ProfileService) deleteProfileOnPlatform(app *models.App, platformType string, profile *models.Profile, machineID string) error {
	// Check if tunnel URL is available
	if app.TunnelURL == nil || *app.TunnelURL == "" {
		return fmt.Errorf("tunnel URL not configured for app %s", app.ID)
	}

	// Extract platform profile ID from profile data
	platformProfileID := s.getPlatformProfileID(profile)
	if platformProfileID == "" {
		return fmt.Errorf("platform profile ID not found")
	}

	// Build the URL to delete profile on platform
	var profileURL string
	switch platformType {
	case "hidemium":
		// Hidemium uses POST with form data
		profileURL = fmt.Sprintf("%s/v1/browser/destroy", *app.TunnelURL)
	case "genlogin":
		// GenLogin uses DELETE method
		profileURL = fmt.Sprintf("%s/profiles/%s", *app.TunnelURL, platformProfileID)
	default:
		return fmt.Errorf("unsupported platform type: %s", platformType)
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	var httpReq *http.Request
	var err error

	if platformType == "hidemium" {
		// Hidemium uses POST with form data
		formData := url.Values{}
		formData.Set("uuid_browser", platformProfileID)
		formData.Set("machine_id", machineID)

		httpReq, err = http.NewRequest("POST", profileURL, strings.NewReader(formData.Encode()))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		// GenLogin uses DELETE method
		httpReq, err = http.NewRequest("DELETE", profileURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
	}

	httpReq.Header.Set("User-Agent", "Green-Controller/1.0")

	// Make HTTP request to platform API
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to connect to platform API: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("platform API returned status %d: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("Profile successfully deleted on %s platform\n", platformType)
	return nil
}

// getPlatformProfileID extracts platform profile ID from profile data
func (s *ProfileService) getPlatformProfileID(profile *models.Profile) string {
	if profile.Data == nil {
		return ""
	}

	// Try to get UUID from profile data
	if uuid, exists := profile.Data["uuid"]; exists {
		if uuidStr, ok := uuid.(string); ok {
			return uuidStr
		}
	}

	// Try alternative field names
	if uuid, exists := profile.Data["id"]; exists {
		if uuidStr, ok := uuid.(string); ok {
			return uuidStr
		}
	}

	return ""
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

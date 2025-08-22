package hidemium

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/onegreenvn/green-provider-services-backend/internal/config"
	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
)

// ProfileService implements profile operations for Hidemium platform
type ProfileService struct {
	appRepo repository.AppRepository
}

// NewProfileService creates a new Hidemium profile service
func NewProfileService(ctx context.Context, appRepo repository.AppRepository) *ProfileService {
	return &ProfileService{
		appRepo: appRepo,
	}
}

// getBaseURLFromAppID retrieves the base URL (TunnelURL) for a specific app
func (s *ProfileService) getBaseURLFromAppID(appID string) (string, error) {
	app, err := s.appRepo.GetByID(appID)
	if err != nil {
		return "", fmt.Errorf("failed to get app: %w", err)
	}

	if app.TunnelURL == nil || *app.TunnelURL == "" {
		return "", fmt.Errorf("tunnel URL not found for app %s", appID)
	}

	return *app.TunnelURL, nil
}

// CreateProfile creates a new profile on Hidemium platform using customize method
func (s *ProfileService) CreateProfile(appID string, profileData *models.CreateProfileRequest) (*models.ProfileResponse, error) {
	// Validate profile data
	if err := s.ValidateProfileData(profileData); err != nil {
		return nil, err
	}

	// Get machine_id from profile data
	machineID := s.getMachineIDFromProfileData(profileData)
	if machineID == "" {
		return nil, fmt.Errorf("machine_id is required for Hidemium profile creation")
	}

	// Get profile name from data for logging
	var profileName string
	if profileData.Data != nil {
		if name, exists := profileData.Data["name"]; exists {
			if nameStr, ok := name.(string); ok {
				profileName = nameStr
			}
		}
	}

	fmt.Printf("Creating profile '%s' on Hidemium (MachineID: %s)\n", profileName, machineID)

	// Get base URL from app
	app, err := s.appRepo.GetByID(appID)
	if err != nil {
		return nil, fmt.Errorf("failed to get app: %w", err)
	}

	if app.TunnelURL == nil || *app.TunnelURL == "" {
		return nil, fmt.Errorf("tunnel URL not found for app %s", appID)
	}

	baseURL := *app.TunnelURL

	// Get Hidemium config
	hidemiumConfig := config.GetHidemiumConfig()

	// Get create_profile_customize route from config
	createProfileRoute, exists := hidemiumConfig.Routes["create_profile_customize"]
	if !exists {
		return nil, fmt.Errorf("create_profile_customize route not found in Hidemium config")
	}

	// Construct full API URL
	apiURL := baseURL + createProfileRoute

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Prepare request body for profile creation
	requestBody := s.buildCreateProfileRequestBody(profileData)

	// Convert to JSON
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create POST request to Hidemium API
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Make HTTP request to Hidemium API
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Hidemium API: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Hidemium API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var hidemiumResponse map[string]interface{}
	if err := json.Unmarshal(body, &hidemiumResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract profile information from response
	profileResponse, err := s.extractCreatedProfileFromResponse(hidemiumResponse, profileData)
	if err != nil {
		return nil, fmt.Errorf("failed to extract created profile: %w", err)
	}

	fmt.Printf("Profile successfully created on Hidemium platform\n")
	return profileResponse, nil
}

// UpdateProfile updates an existing profile on Hidemium platform
func (s *ProfileService) UpdateProfile(profileID string, profileData *models.UpdateProfileRequest) (*models.ProfileResponse, error) {
	// TODO: Implement Hidemium-specific profile update logic
	return nil, fmt.Errorf("profile update on Hidemium not implemented yet")
}

// DeleteProfile deletes a profile on Hidemium platform
func (s *ProfileService) DeleteProfile(appID string, profile *models.Profile, machineID string) error {
	// Get the actual Hidemium profile ID from profile.Data.uuid
	hidemiumProfileID := s.getHidemiumProfileID(profile)
	if hidemiumProfileID == "" {
		return fmt.Errorf("invalid Hidemium profile ID")
	}

	if machineID == "" {
		return fmt.Errorf("machine_id is required for Hidemium profile deletion")
	}

	fmt.Printf("Deleting profile '%s' on Hidemium (ProfileID: %s, MachineID: %s)\n", profile.Name, hidemiumProfileID, machineID)

	// Get base URL from app
	baseURL, err := s.getBaseURLFromAppID(appID)
	if err != nil {
		return fmt.Errorf("failed to get base URL: %w", err)
	}

	// Get Hidemium config
	hidemiumConfig := config.GetHidemiumConfig()

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

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Hidemium API returned status %d: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("Profile successfully deleted on Hidemium platform\n")
	return nil
}

// GetProfile retrieves profile information from Hidemium platform
func (s *ProfileService) GetProfile(profileID string) (*models.ProfileResponse, error) {
	// TODO: Implement Hidemium-specific profile retrieval logic
	return nil, fmt.Errorf("profile retrieval on Hidemium not implemented yet")
}

// ListProfiles lists all profiles from Hidemium platform
func (s *ProfileService) ListProfiles(filters map[string]interface{}) ([]*models.ProfileResponse, error) {
	// TODO: Implement Hidemium-specific profile listing logic
	return nil, fmt.Errorf("profile listing on Hidemium not implemented yet")
}

// SyncProfilesFromPlatform syncs profiles from Hidemium platform
func (s *ProfileService) SyncProfilesFromPlatform(appID string, machineID string) ([]models.HidemiumProfile, error) {
	// Get base URL from app
	baseURL, err := s.getBaseURLFromAppID(appID)
	if err != nil {
		return nil, fmt.Errorf("failed to get base URL: %w", err)
	}

	// Use tunnel_url from app instead of config
	if baseURL == "" {
		return nil, fmt.Errorf("tunnel_url not found for app %s", appID)
	}

	// Get list_profiles route from config
	hidemiumConfig := config.GetHidemiumConfig()
	listProfilesRoute, exists := hidemiumConfig.Routes["list_profiles"]
	if !exists {
		return nil, fmt.Errorf("list_profiles route not found in Hidemium config")
	}

	// Construct full API URL
	apiURL := baseURL + listProfilesRoute

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create POST request to get profiles
	req, err := http.NewRequest("POST", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Make HTTP request to Hidemium API
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Hidemium API: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Hidemium API returned status %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse response
	var rawResponse map[string]interface{}
	if err := json.Unmarshal(body, &rawResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract profiles from response
	hidemiumProfiles := s.extractProfilesFromResponse(rawResponse, body)
	return hidemiumProfiles, nil
}

// GetPlatformName returns the platform name
func (s *ProfileService) GetPlatformName() string {
	return "hidemium"
}

// GetPlatformVersion returns the platform version
func (s *ProfileService) GetPlatformVersion() string {
	return "4.0"
}

// ValidateProfileData validates profile data for Hidemium platform
func (s *ProfileService) ValidateProfileData(profileData *models.CreateProfileRequest) error {
	// Check if name exists in data
	if profileData.Data == nil {
		return fmt.Errorf("profile data is required")
	}

	if name, exists := profileData.Data["name"]; !exists || name == "" {
		return fmt.Errorf("profile name is required in data")
	}

	if len(profileData.Data) == 0 {
		return fmt.Errorf("profile data is required")
	}
	return nil
}

// Helper methods

// getMachineIDFromProfileData extracts machine_id from profile data
func (s *ProfileService) getMachineIDFromProfileData(profileData *models.CreateProfileRequest) string {
	if profileData.Data == nil {
		return ""
	}

	// Try to get machine_id from profile data
	if machineID, exists := profileData.Data["machine_id"]; exists {
		if machineIDStr, ok := machineID.(string); ok {
			return machineIDStr
		}
	}

	return ""
}

// buildCreateProfileRequestBody builds the request body for Hidemium profile creation
func (s *ProfileService) buildCreateProfileRequestBody(profileData *models.CreateProfileRequest) map[string]interface{} {
	// Get name from data
	var profileName string
	if profileData.Data != nil {
		if name, exists := profileData.Data["name"]; exists {
			if nameStr, ok := name.(string); ok {
				profileName = nameStr
			}
		}
	}

	// Start with default values based on Hidemium API documentation
	requestBody := map[string]interface{}{
		"os":                    "win",    // Default to Windows
		"osVersion":             "10",     // Default to Windows 10
		"browser":               "chrome", // Default to Chrome
		"version":               "136",    // Default Chrome version
		"userAgent":             "",
		"canvas":                "noise", // Default canvas fingerprint
		"webGLImage":            "false", // Default webGL image
		"audioContext":          "false", // Default audio context
		"webGLMetadata":         "false", // Default webGL metadata
		"webGLVendor":           "",
		"webGLMetadataRenderer": "",
		"clientRectsEnable":     "false",                // Default client rects
		"noiseFont":             "false",                // Default noise font
		"language":              "en-US",                // Default language
		"deviceMemory":          4,                      // Default device memory
		"hardwareConcurrency":   32,                     // Default hardware concurrency
		"resolution":            "1280x800",             // Default resolution
		"StartURL":              "https://hidemium.io/", // Default start URL
		"name":                  profileName,            // Profile name from data
		"checkname":             true,                   // Check for duplicate names
	}

	// Override with values from profileData.Data if provided
	if profileData.Data != nil {
		for key, value := range profileData.Data {
			// Skip internal fields
			if key == "machine_id" || key == "uuid" {
				continue
			}
			requestBody[key] = value
		}
	}

	return requestBody
}

// extractCreatedProfileFromResponse extracts profile information from Hidemium API response
func (s *ProfileService) extractCreatedProfileFromResponse(hidemiumResponse map[string]interface{}, profileData *models.CreateProfileRequest) (*models.ProfileResponse, error) {
	// Get profile name from data
	var profileName string
	if profileData.Data != nil {
		if name, exists := profileData.Data["name"]; exists {
			if nameStr, ok := name.(string); ok {
				profileName = nameStr
			}
		}
	}

	// Check if response indicates success
	if responseType, exists := hidemiumResponse["type"]; exists {
		if typeStr, ok := responseType.(string); ok && typeStr == "success" {
			// Extract content from response
			if content, exists := hidemiumResponse["content"]; exists {
				if contentMap, ok := content.(map[string]interface{}); ok {
					// Create profile response with extracted data
					profileResponse := &models.ProfileResponse{
						ID:        s.getStringFromMap(contentMap, "uuid"),
						Name:      profileName,
						AppID:     profileData.AppID,
						Data:      contentMap,
						CreatedAt: time.Now().Format(time.RFC3339),
						UpdatedAt: time.Now().Format(time.RFC3339),
					}
					return profileResponse, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("failed to extract profile from Hidemium response")
}

// getHidemiumProfileID extracts Hidemium profile ID from profile data
func (s *ProfileService) getHidemiumProfileID(profile *models.Profile) string {
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

// extractProfilesFromResponse extracts profiles from Hidemium API response
func (s *ProfileService) extractProfilesFromResponse(rawResponse map[string]interface{}, body []byte) []models.HidemiumProfile {
	var hidemiumProfiles []models.HidemiumProfile

	// Try to extract profiles from different possible response structures
	// First try: data.content structure (Hidemium API format)
	if data, exists := rawResponse["data"]; exists {
		if dataMap, ok := data.(map[string]interface{}); ok {
			if content, exists := dataMap["content"]; exists {
				if contentArray, ok := content.([]interface{}); ok {
					for _, item := range contentArray {
						if profileMap, ok := item.(map[string]interface{}); ok {
							profile := models.HidemiumProfile{
								ID:        s.getStringFromMap(profileMap, "uuid"), // Use 'uuid' instead of 'id'
								Name:      s.getStringFromMap(profileMap, "name"),
								CreatedAt: s.getStringFromMap(profileMap, "created_at"),
								UpdatedAt: s.getStringFromMap(profileMap, "updated_at"),
								IsActive:  s.getBoolFromMap(profileMap, "is_active"),
								Data:      profileMap,
							}
							hidemiumProfiles = append(hidemiumProfiles, profile)
						}
					}
				}
			}
		}

		// Fallback: try data as direct array
		if len(hidemiumProfiles) == 0 {
			if dataArray, ok := data.([]interface{}); ok {
				for _, item := range dataArray {
					if profileMap, ok := item.(map[string]interface{}); ok {
						profile := models.HidemiumProfile{
							ID:        s.getStringFromMap(profileMap, "uuid"),
							Name:      s.getStringFromMap(profileMap, "name"),
							CreatedAt: s.getStringFromMap(profileMap, "created_at"),
							UpdatedAt: s.getStringFromMap(profileMap, "updated_at"),
							IsActive:  s.getBoolFromMap(profileMap, "is_active"),
							Data:      profileMap,
						}
						hidemiumProfiles = append(hidemiumProfiles, profile)
					}
				}
			}
		}
	}

	// If no profiles found in data field, try other fields
	if len(hidemiumProfiles) == 0 {
		// Try profiles field
		if profilesData, exists := rawResponse["profiles"]; exists {
			if profilesArray, ok := profilesData.([]interface{}); ok {
				for _, item := range profilesArray {
					if profileMap, ok := item.(map[string]interface{}); ok {
						profile := models.HidemiumProfile{
							ID:        s.getStringFromMap(profileMap, "id"),
							Name:      s.getStringFromMap(profileMap, "name"),
							CreatedAt: s.getStringFromMap(profileMap, "created_at"),
							UpdatedAt: s.getStringFromMap(profileMap, "updated_at"),
							IsActive:  s.getBoolFromMap(profileMap, "is_active"),
							Data:      profileMap,
						}
						hidemiumProfiles = append(hidemiumProfiles, profile)
					}
				}
			}
		}
	}

	// If still no profiles, try direct array response
	if len(hidemiumProfiles) == 0 {
		var directProfiles []map[string]interface{}
		if err := json.Unmarshal(body, &directProfiles); err == nil {
			for _, profileMap := range directProfiles {
				profile := models.HidemiumProfile{
					ID:        s.getStringFromMap(profileMap, "id"),
					Name:      s.getStringFromMap(profileMap, "name"),
					CreatedAt: s.getStringFromMap(profileMap, "created_at"),
					UpdatedAt: s.getStringFromMap(profileMap, "updated_at"),
					IsActive:  s.getBoolFromMap(profileMap, "is_active"),
					Data:      profileMap,
				}
				hidemiumProfiles = append(hidemiumProfiles, profile)
			}
		}
	}

	return hidemiumProfiles
}

// getStringFromMap safely extracts string value from map
func (s *ProfileService) getStringFromMap(m map[string]interface{}, key string) string {
	if val, exists := m[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// getBoolFromMap safely extracts bool value from map
func (s *ProfileService) getBoolFromMap(m map[string]interface{}, key string) bool {
	if val, exists := m[key]; exists {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

// GetDefaultConfigs retrieves default configurations from Hidemium platform
func (s *ProfileService) GetDefaultConfigs(appID string, machineID string, page, limit int) (map[string]interface{}, error) {
	// Get Hidemium config
	hidemiumConfig := config.GetHidemiumConfig()

	// Get list_config_default route from config
	listConfigRoute, exists := hidemiumConfig.Routes["list_config_default"]
	if !exists {
		return nil, fmt.Errorf("list_config_default route not found in Hidemium config")
	}

	if machineID == "" {
		return nil, fmt.Errorf("machine_id is required for tunnel")
	}

	// Get base URL from app
	baseURL, err := s.getBaseURLFromAppID(appID)
	if err != nil {
		return nil, fmt.Errorf("failed to get base URL: %w", err)
	}

	// Use tunnel_url from app
	if baseURL == "" {
		return nil, fmt.Errorf("tunnel_url not found for app %s", appID)
	}

	// Replace pagination parameters with actual values
	routeWithParams := strings.Replace(listConfigRoute, "{page}", fmt.Sprintf("%d", page), 1)
	routeWithParams = strings.Replace(routeWithParams, "{limit}", fmt.Sprintf("%d", limit), 1)

	// Construct full API URL
	apiURL := baseURL + routeWithParams

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create GET request to Hidemium API
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Make HTTP request to Hidemium API
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Hidemium API: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Hidemium API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var hidemiumResponse map[string]interface{}
	if err := json.Unmarshal(body, &hidemiumResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return hidemiumResponse, nil
}

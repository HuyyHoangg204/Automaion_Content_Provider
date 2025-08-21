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
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
)

// ProfileService implements profile operations for Hidemium platform
type ProfileService struct{}

// NewProfileService creates a new Hidemium profile service
func NewProfileService() *ProfileService {
	return &ProfileService{}
}

// CreateProfile creates a new profile on Hidemium platform
func (s *ProfileService) CreateProfile(ctx context.Context, profileData *models.CreateProfileRequest) (*models.ProfileResponse, error) {
	// Validate profile data
	if err := s.ValidateProfileData(profileData); err != nil {
		return nil, err
	}

	// TODO: Implement Hidemium-specific profile creation logic
	// This would involve calling Hidemium API to create profile
	return nil, fmt.Errorf("profile creation on Hidemium not implemented yet")
}

// UpdateProfile updates an existing profile on Hidemium platform
func (s *ProfileService) UpdateProfile(ctx context.Context, profileID string, profileData *models.UpdateProfileRequest) (*models.ProfileResponse, error) {
	// TODO: Implement Hidemium-specific profile update logic
	return nil, fmt.Errorf("profile update on Hidemium not implemented yet")
}

// DeleteProfile deletes a profile on Hidemium platform
func (s *ProfileService) DeleteProfile(ctx context.Context, profile *models.Profile, machineID string) error {
	// Get the actual Hidemium profile ID from profile.Data.uuid
	hidemiumProfileID := s.getHidemiumProfileID(profile)
	if hidemiumProfileID == "" {
		return fmt.Errorf("invalid Hidemium profile ID")
	}

	if machineID == "" {
		return fmt.Errorf("machine_id is required for Hidemium profile deletion")
	}

	fmt.Printf("Deleting profile '%s' on Hidemium (ProfileID: %s, MachineID: %s)\n", profile.Name, hidemiumProfileID, machineID)

	// Get Hidemium config
	hidemiumConfig := config.GetHidemiumConfig()

	// Construct tunnel URL using machine ID
	baseURL := hidemiumConfig.BaseURL
	baseURL = strings.Replace(baseURL, "{machine_id}", machineID, 1)

	// Get delete_profile route from config
	deleteProfileRoute, exists := hidemiumConfig.Routes["delete_profile"]
	if !exists {
		return fmt.Errorf("delete_profile route not found in Hidemium config")
	}

	// Construct full API URL
	apiURL := baseURL + deleteProfileRoute

	fmt.Printf("Calling Hidemium API: %s\n", apiURL)

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
func (s *ProfileService) GetProfile(ctx context.Context, profileID string) (*models.ProfileResponse, error) {
	// TODO: Implement Hidemium-specific profile retrieval logic
	return nil, fmt.Errorf("profile retrieval on Hidemium not implemented yet")
}

// ListProfiles lists all profiles from Hidemium platform
func (s *ProfileService) ListProfiles(ctx context.Context, filters map[string]interface{}) ([]*models.ProfileResponse, error) {
	// TODO: Implement Hidemium-specific profile listing logic
	return nil, fmt.Errorf("profile listing on Hidemium not implemented yet")
}

// SyncProfilesFromPlatform syncs profiles from Hidemium platform
func (s *ProfileService) SyncProfilesFromPlatform(ctx context.Context, boxID string, machineID string) ([]models.HidemiumProfile, error) {
	// Get Hidemium config
	hidemiumConfig := config.GetHidemiumConfig()

	// Construct tunnel URL using machine ID
	baseURL := hidemiumConfig.BaseURL
	baseURL = strings.Replace(baseURL, "{machine_id}", machineID, 1)

	// Get list_profiles route from config
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
	if profileData.Name == "" {
		return fmt.Errorf("profile name is required")
	}
	if len(profileData.Data) == 0 {
		return fmt.Errorf("profile data is required")
	}
	return nil
}

// Helper methods

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

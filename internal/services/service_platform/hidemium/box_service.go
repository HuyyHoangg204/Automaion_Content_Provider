package hidemium

import (
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

// BoxService implements box operations for Hidemium platform
type BoxService struct{}

// NewBoxService creates a new Hidemium box service
func NewBoxService() *BoxService {
	return &BoxService{}
}

// CreateBox creates a new box on Hidemium platform
func (s *BoxService) CreateBox(ctx context.Context, boxData *models.CreateBoxRequest) (*models.BoxResponse, error) {
	// Validate box data
	if err := s.ValidateBoxData(boxData); err != nil {
		return nil, err
	}

	// TODO: Implement Hidemium-specific box creation logic
	// This would involve calling Hidemium API to create box
	return nil, fmt.Errorf("box creation on Hidemium not implemented yet")
}

// UpdateBox updates an existing box on Hidemium platform
func (s *BoxService) UpdateBox(ctx context.Context, boxID string, boxData *models.UpdateBoxRequest) (*models.BoxResponse, error) {
	// TODO: Implement Hidemium-specific box update logic
	return nil, fmt.Errorf("box update on Hidemium not implemented yet")
}

// DeleteBox deletes a box on Hidemium platform
func (s *BoxService) DeleteBox(ctx context.Context, boxID string) error {
	// TODO: Implement Hidemium-specific box deletion logic
	return fmt.Errorf("box deletion on Hidemium not implemented yet")
}

// GetBox retrieves box information from Hidemium platform
func (s *BoxService) GetBox(ctx context.Context, boxID string) (*models.BoxResponse, error) {
	// TODO: Implement Hidemium-specific box retrieval logic
	return nil, fmt.Errorf("box retrieval on Hidemium not implemented yet")
}

// ListBoxes lists all boxes from Hidemium platform
func (s *BoxService) ListBoxes(ctx context.Context, filters map[string]interface{}) ([]*models.BoxResponse, error) {
	// TODO: Implement Hidemium-specific box listing logic
	return nil, fmt.Errorf("box listing on Hidemium not implemented yet")
}

// SyncBoxProfilesFromPlatform syncs profiles from Hidemium platform for a specific box
func (s *BoxService) SyncBoxProfilesFromPlatform(ctx context.Context, boxID string, machineID string) ([]models.HidemiumProfile, error) {
	fmt.Printf("Starting profile sync from Hidemium for box ID: %s (MachineID: %s)\n", boxID, machineID)

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
	var rawResponse map[string]interface{}
	if err := json.Unmarshal(body, &rawResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract profiles from response
	hidemiumProfiles := s.extractProfilesFromResponse(rawResponse, body)

	fmt.Printf("Successfully fetched %d profiles from Hidemium for box %s\n", len(hidemiumProfiles), boxID)

	// Return the fetched profiles - the main service will handle database operations
	return hidemiumProfiles, nil
}

// GetPlatformName returns the platform name
func (s *BoxService) GetPlatformName() string {
	return "hidemium"
}

// GetPlatformVersion returns the platform version
func (s *BoxService) GetPlatformVersion() string {
	return "4.0"
}

// ValidateBoxData validates box data for Hidemium platform
func (s *BoxService) ValidateBoxData(boxData *models.CreateBoxRequest) error {
	if boxData.MachineID == "" {
		return fmt.Errorf("machine ID is required")
	}
	if boxData.Name == "" {
		return fmt.Errorf("box name is required")
	}
	return nil
}

// Helper methods

// extractProfilesFromResponse extracts profiles from Hidemium API response
// FIXED: Now properly handles data.content structure from Hidemium API
func (s *BoxService) extractProfilesFromResponse(rawResponse map[string]interface{}, body []byte) []models.HidemiumProfile {
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
func (s *BoxService) getStringFromMap(m map[string]interface{}, key string) string {
	if val, exists := m[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// getBoolFromMap safely extracts bool value from map
func (s *BoxService) getBoolFromMap(m map[string]interface{}, key string) bool {
	if val, exists := m[key]; exists {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

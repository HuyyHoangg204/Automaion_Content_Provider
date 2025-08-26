package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/onegreenvn/green-provider-services-backend/internal/models"
)

// PlatformType represents supported platform types
type PlatformType string

const (
	PlatformHidemium PlatformType = "hidemium"
	PlatformGenLogin PlatformType = "genlogin"
)

// AppHelper handles app-related operations
type AppHelper struct{}

// NewAppHelper creates a new AppHelper instance
func NewAppHelper() *AppHelper {
	return &AppHelper{}
}

// GetPlatformType determines the platform type from app name
func GetPlatformType(appName string) PlatformType {
	appNameLower := strings.ToLower(appName)

	switch {
	case strings.Contains(appNameLower, "hidemium"):
		return PlatformHidemium
	case strings.Contains(appNameLower, "genlogin"):
		return PlatformGenLogin
	default:
		return ""
	}
}

// BuildProfilesURL builds the profiles endpoint URL for the platform
func BuildProfilesURL(baseURL string, platformType PlatformType) string {
	baseURL = strings.TrimSuffix(baseURL, "/")

	switch platformType {
	case PlatformHidemium:
		return fmt.Sprintf("%s/v1/browser/list", baseURL)
	case PlatformGenLogin:
		return fmt.Sprintf("%s/profiles/list", baseURL)
	default:
		return fmt.Sprintf("%s/profiles", baseURL)
	}
}

// FetchProfilesFromPlatform fetches profiles from the platform using the app's tunnel URL
func (h *AppHelper) FetchProfilesFromPlatform(app *models.App, platformType string) ([]map[string]interface{}, error) {
	if app.TunnelURL == nil || *app.TunnelURL == "" {
		return nil, fmt.Errorf("no tunnel URL configured for app %s", app.Name)
	}

	// Build the profiles endpoint URL
	profilesURL := BuildProfilesURL(*app.TunnelURL, PlatformType(platformType))

	// Make HTTP request to fetch profiles
	client := &http.Client{Timeout: 30 * time.Second}

	var req *http.Request
	var err error

	// Hidemium uses POST, GenLogin uses GET
	if platformType == "hidemium" {
		req, err = http.NewRequest("POST", profilesURL, nil)
	} else {
		req, err = http.NewRequest("GET", profilesURL, nil)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Green-Controller/1.0")

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request to platform: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("platform returned status %d", resp.StatusCode)
	}

	// Parse response based on platform
	var platformProfiles []map[string]interface{}
	switch platformType {
	case "hidemium":
		platformProfiles, err = ParseHidemiumProfilesResponse(resp.Body)
	case "genlogin":
		platformProfiles, err = ParseGenLoginProfilesResponse(resp.Body)
	default:
		return nil, fmt.Errorf("unsupported platform for parsing: %s", platformType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse %s response: %w", platformType, err)
	}

	return platformProfiles, nil
}

// ParseHidemiumProfilesResponse parses Hidemium profiles response
func ParseHidemiumProfilesResponse(body io.Reader) ([]map[string]interface{}, error) {
	var response map[string]interface{}
	if err := json.NewDecoder(body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode hidemium response: %w", err)
	}

	// Extract profiles from response - Hidemium uses data.content structure
	var profiles []map[string]interface{}

	// Check if data.content exists (Hidemium structure)
	if data, ok := response["data"].(map[string]interface{}); ok {
		if content, ok := data["content"].([]interface{}); ok {
			for _, item := range content {
				if profile, ok := item.(map[string]interface{}); ok {
					profiles = append(profiles, profile)
				}
			}
			return profiles, nil
		}
	}

	// Fallback: try other possible field names
	possibleFields := []string{"data", "profiles", "result", "list", "browsers"}

	for _, field := range possibleFields {
		if value, exists := response[field]; exists {
			switch v := value.(type) {
			case []interface{}:
				for _, item := range v {
					if profile, ok := item.(map[string]interface{}); ok {
						profiles = append(profiles, profile)
					}
				}
				return profiles, nil
			case map[string]interface{}:
				// Check if this map contains arrays (like data.content)
				for _, nestedValue := range v {
					if nestedArray, ok := nestedValue.([]interface{}); ok {
						for _, item := range nestedArray {
							if profile, ok := item.(map[string]interface{}); ok {
								profiles = append(profiles, profile)
							}
						}
						return profiles, nil
					}
				}
			}
		}
	}

	return nil, nil
}

// ParseGenLoginProfilesResponse parses GenLogin profiles response
func ParseGenLoginProfilesResponse(body io.Reader) ([]map[string]interface{}, error) {
	var response map[string]interface{}
	if err := json.NewDecoder(body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode genlogin response: %w", err)
	}

	// Extract profiles from response
	var profiles []map[string]interface{}

	// Try different possible field names
	possibleFields := []string{"data", "profiles", "result", "list"}

	for _, field := range possibleFields {
		if data, exists := response[field]; exists {
			if profilesData, ok := data.([]interface{}); ok {
				for _, profileData := range profilesData {
					if profile, ok := profileData.(map[string]interface{}); ok {
						profiles = append(profiles, profile)
					}
				}
				if len(profiles) > 0 {
					break
				}
			}
		}
	}

	return profiles, nil
}

// ExtractUUID reads uuid from stored profile data
func ExtractUUID(profile *models.Profile) string {
	if profile.Data == nil {
		return ""
	}
	if uuid, exists := profile.Data["uuid"]; exists {
		if uuidStr, ok := uuid.(string); ok {
			return uuidStr
		}
	}
	if uuid, exists := profile.Data["id"]; exists {
		if uuidStr, ok := uuid.(string); ok {
			return uuidStr
		}
	}
	return ""
}

// ExtractUUIDFromPlatformProfile reads uuid from raw platform payload
func ExtractUUIDFromPlatformProfile(platformProfile map[string]interface{}) string {
	fields := []string{"uuid", "id", "browser_id", "profile_id", "browser_uuid", "profile_uuid"}
	for _, f := range fields {
		if v, ok := platformProfile[f]; ok {
			if s, ok2 := v.(string); ok2 && s != "" {
				return s
			}
		}
	}
	return ""
}

// ProcessPlatformProfile processes a single platform profile
func (h *AppHelper) ProcessPlatformProfile(appID string, platformProfile map[string]interface{}, existingProfilesMap map[string]*models.Profile, result *models.SyncBoxProfilesResponse) error {
	uuid := ExtractUUIDFromPlatformProfile(platformProfile)
	if uuid == "" {
		return fmt.Errorf("platform profile has no UUID")
	}

	if existingProfile, exists := existingProfilesMap[uuid]; exists {
		if err := h.UpdateExistingProfile(existingProfile, platformProfile); err != nil {
			return fmt.Errorf("failed to update profile: %w", err)
		}
		result.ProfilesUpdated++
		delete(existingProfilesMap, uuid)
	} else {
		if err := h.CreateNewProfile(appID, platformProfile); err != nil {
			return fmt.Errorf("failed to create profile: %w", err)
		}
		result.ProfilesCreated++
	}
	return nil
}

// CreateNewProfile creates a new profile
func (h *AppHelper) CreateNewProfile(appID string, platformProfile map[string]interface{}) error {
	_ = appID           // Avoid unused parameter warning
	_ = platformProfile // Avoid unused parameter warning
	return nil          // This should be handled by the caller with proper repository
}

// UpdateExistingProfile updates an existing profile
func (h *AppHelper) UpdateExistingProfile(existingProfile *models.Profile, platformProfile map[string]interface{}) error {
	if existingProfile.Data == nil {
		existingProfile.Data = make(map[string]interface{})
	}
	for key, value := range platformProfile {
		existingProfile.Data[key] = value
	}
	if name, exists := platformProfile["name"]; exists {
		if nameStr, ok := name.(string); ok && nameStr != existingProfile.Name {
			existingProfile.Name = nameStr
		}
	}
	return nil // This should be handled by the caller with proper repository
}

// MarkDeletedProfiles clears associations and deletes missing profiles
func (h *AppHelper) MarkDeletedProfiles(existingProfilesMap map[string]*models.Profile, result *models.SyncBoxProfilesResponse) {
	for range existingProfilesMap {
		// Note: Repository operations should be handled by the caller
		result.ProfilesDeleted++
	}
}

// AppResponseConverter converts App models to response DTOs
type AppResponseConverter struct{}

// NewAppResponseConverter creates a new AppResponseConverter instance
func NewAppResponseConverter() *AppResponseConverter {
	return &AppResponseConverter{}
}

// ToAppResponse converts App model to response DTO
func (c *AppResponseConverter) ToAppResponse(app *models.App) *models.AppResponse {
	return &models.AppResponse{
		ID:        app.ID,
		BoxID:     app.BoxID,
		Name:      app.Name,
		TunnelURL: app.TunnelURL,
		CreatedAt: app.CreatedAt.Format(time.RFC3339),
		UpdatedAt: app.UpdatedAt.Format(time.RFC3339),
	}
}

// ToAppResponseList converts a list of App models to response DTOs
func (c *AppResponseConverter) ToAppResponseList(apps []*models.App) []*models.AppResponse {
	responses := make([]*models.AppResponse, len(apps))
	for i, app := range apps {
		responses[i] = c.ToAppResponse(app)
	}
	return responses
}

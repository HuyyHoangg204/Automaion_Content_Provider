package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
)

type BoxService struct {
	boxRepo     *repository.BoxRepository
	appRepo     *repository.AppRepository
	profileRepo *repository.ProfileRepository
	userRepo    *repository.UserRepository
}

// NewBoxService creates a new box service
func NewBoxService(boxRepo *repository.BoxRepository, appRepo *repository.AppRepository, profileRepo *repository.ProfileRepository, userRepo *repository.UserRepository) *BoxService {
	return &BoxService{
		boxRepo:     boxRepo,
		appRepo:     appRepo,
		profileRepo: profileRepo,
		userRepo:    userRepo,
	}
}

// CreateBox creates a new box for a user
func (s *BoxService) CreateBox(userID string, req *models.CreateBoxRequest) (*models.BoxResponse, error) {
	// Check if machine ID already exists
	existingBox, err := s.boxRepo.GetByMachineID(req.MachineID)
	if err == nil {
		// Box exists, return error with box details
		return nil, &models.BoxAlreadyExistsError{
			BoxID:     existingBox.ID,
			MachineID: existingBox.MachineID,
			Message:   "machine ID already exists",
		}
	}

	// Verify user exists
	_, err = s.userRepo.GetByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Create box
	box := &models.Box{
		UserID:    userID,
		MachineID: req.MachineID,
		Name:      req.Name,
	}

	if err := s.boxRepo.Create(box); err != nil {
		return nil, fmt.Errorf("failed to create box: %w", err)
	}

	return s.toResponse(box), nil
}

// GetBoxesByUserPaginated retrieves paginated boxes for a specific user
func (s *BoxService) GetBoxesByUserPaginated(userID string, page, pageSize int) ([]*models.BoxResponse, int, error) {
	boxes, total, err := s.boxRepo.GetByUserID(userID, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get boxes: %w", err)
	}

	responses := make([]*models.BoxResponse, len(boxes))
	for i, box := range boxes {
		responses[i] = s.toResponse(box)
	}

	return responses, total, nil
}

// GetBoxByID retrieves a box by ID (user must own it)
func (s *BoxService) GetBoxByID(userID, boxID string) (*models.BoxResponse, error) {
	box, err := s.boxRepo.GetByUserIDAndID(userID, boxID)
	if err != nil {
		return nil, errors.New("box not found")
	}

	return s.toResponse(box), nil
}

// UpdateBox updates a box (user must own it)
func (s *BoxService) UpdateBox(userID, boxID string, req *models.UpdateBoxRequest) (*models.BoxResponse, error) {
	// Get box by ID (no ownership check)
	box, err := s.boxRepo.GetByID(boxID)
	if err != nil {
		return nil, errors.New("box not found")
	}

	// If updating user_id (transferring ownership)
	if req.UserID != "" {
		// Verify that the new user exists
		_, err := s.userRepo.GetByID(req.UserID)
		if err != nil {
			return nil, errors.New("new user not found")
		}

		// Update user_id
		box.UserID = req.UserID
	}

	// Update name
	box.Name = req.Name

	if err := s.boxRepo.Update(box); err != nil {
		return nil, fmt.Errorf("failed to update box: %w", err)
	}

	// Get the updated box to verify changes
	updatedBox, err := s.boxRepo.GetByID(boxID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated box: %w", err)
	}

	return s.toResponse(updatedBox), nil
}

// DeleteBox deletes a box (user must own it)
func (s *BoxService) DeleteBox(userID, boxID string) error {
	// Check if box exists and belongs to user
	_, err := s.boxRepo.GetByUserIDAndID(userID, boxID)
	if err != nil {
		return errors.New("box not found")
	}

	if err := s.boxRepo.DeleteByUserIDAndID(userID, boxID); err != nil {
		return fmt.Errorf("failed to delete box: %w", err)
	}

	return nil
}

// GetAllBoxes retrieves all boxes (admin only)
func (s *BoxService) GetAllBoxes() ([]*models.BoxResponse, error) {
	boxes, err := s.boxRepo.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get all boxes: %w", err)
	}

	responses := make([]*models.BoxResponse, len(boxes))
	for i, box := range boxes {
		responses[i] = s.toResponse(box)
	}

	return responses, nil
}

// GetBoxByMachineID retrieves a box by machine ID
func (s *BoxService) GetBoxByMachineID(machineID string) (*models.BoxResponse, error) {
	box, err := s.boxRepo.GetByMachineID(machineID)
	if err != nil {
		return nil, errors.New("box not found")
	}

	return s.toResponse(box), nil
}

// SyncSingleBoxProfiles syncs all profiles from a single box's platform instance
// Now uses box-proxy endpoint for platform operations
func (s *BoxService) SyncSingleBoxProfiles(userID, boxID string) (*models.SyncBoxProfilesResponse, error) {
	// Get box by ID and verify ownership
	box, err := s.boxRepo.GetByUserIDAndID(userID, boxID)
	if err != nil {
		return nil, errors.New("box not found")
	}

	// Get all apps for this box
	apps, err := s.appRepo.GetByBoxID(boxID)
	if err != nil {
		return nil, fmt.Errorf("failed to get apps for box: %w", err)
	}

	if len(apps) == 0 {
		return nil, errors.New("no apps found for this box")
	}

	// Sync all apps in this box
	var totalProfilesCreated, totalProfilesUpdated, totalProfilesDeleted int
	var totalProfilesSynced int
	var unsupportedPlatforms []string

	for _, app := range apps {
		// Determine platform from app name
		platformType := s.determinePlatformFromAppName(app.Name)
		if platformType == "" {
			fmt.Printf("Warning: Unsupported platform %s for app %s in box %s\n", app.Name, app.ID, box.ID)
			unsupportedPlatforms = append(unsupportedPlatforms, app.Name)
			continue
		}

		fmt.Printf("Starting sync for box %s (MachineID: %s) on platform %s for app %s\n", box.ID, box.MachineID, platformType, app.Name)

		// Use box-proxy approach to get profiles from platform
		platformProfiles, err := s.getProfilesFromPlatformViaProxy(app, platformType)
		if err != nil {
			fmt.Printf("Warning: Failed to get profiles from %s for app %s: %v\n", platformType, app.Name, err)
			continue
		}

		fmt.Printf("Successfully got %d profiles from %s for app %s in box %s\n", len(platformProfiles), platformType, app.Name, box.ID)

		// Process synced profiles and update local database
		appSyncResult, err := s.processSyncedProfiles(app.ID, platformProfiles)
		if err != nil {
			fmt.Printf("Warning: Failed to process synced profiles for app %s: %v\n", app.Name, err)
			continue
		}

		// Accumulate results
		totalProfilesCreated += appSyncResult.ProfilesCreated
		totalProfilesUpdated += appSyncResult.ProfilesUpdated
		totalProfilesDeleted += appSyncResult.ProfilesDeleted
		totalProfilesSynced += len(platformProfiles)
	}

	// Create combined sync result
	syncResult := &models.SyncBoxProfilesResponse{
		ProfilesCreated: totalProfilesCreated,
		ProfilesUpdated: totalProfilesUpdated,
		ProfilesDeleted: totalProfilesDeleted,
		ProfilesSynced:  totalProfilesSynced,
	}

	// Update response with sync results
	syncResult.BoxID = box.ID
	syncResult.MachineID = box.MachineID

	// Create detailed message including unsupported platforms
	if len(unsupportedPlatforms) > 0 {
		syncResult.Message = fmt.Sprintf("Profiles synced successfully from supported platforms. Unsupported platforms: %s", strings.Join(unsupportedPlatforms, ", "))
	} else {
		syncResult.Message = "Profiles synced successfully from all platforms"
	}

	return syncResult, nil
}

// SyncAllUserBoxes syncs profiles from all boxes owned by a user
// Now uses box-proxy endpoint for platform operations
func (s *BoxService) SyncAllUserBoxes(userID string) (*models.SyncBoxProfilesResponse, error) {
	// Get all boxes for user (with pagination, using large limit to get all)
	boxes, _, err := s.boxRepo.GetByUserID(userID, 1, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to get user boxes: %w", err)
	}

	if len(boxes) == 0 {
		return nil, errors.New("no boxes found for user")
	}

	// Counters for overall response
	totalProfiles := 0
	totalProfilesCreated := 0
	totalProfilesUpdated := 0
	totalProfilesDeleted := 0
	boxesSynced := 0

	// Sync each box
	for _, box := range boxes {
		// Use the single box sync method
		syncResponse, err := s.SyncSingleBoxProfiles(userID, box.ID)
		if err != nil {
			fmt.Printf("Warning: Failed to sync box %s: %v\n", box.ID, err)
			continue
		}

		// Box synced successfully
		boxesSynced++
		totalProfiles += syncResponse.ProfilesSynced
		totalProfilesCreated += syncResponse.ProfilesCreated
		totalProfilesUpdated += syncResponse.ProfilesUpdated
		totalProfilesDeleted += syncResponse.ProfilesDeleted
	}

	// Note: Profile syncing is now handled through box-proxy endpoint
	// This method aggregates results from individual box syncs
	return &models.SyncBoxProfilesResponse{
		ProfilesCreated: totalProfilesCreated,
		ProfilesUpdated: totalProfilesUpdated,
		ProfilesDeleted: totalProfilesDeleted,
		ProfilesSynced:  totalProfiles,
		Message:         fmt.Sprintf("Sync completed: %d/%d boxes synced, %d profiles processed", boxesSynced, len(boxes), totalProfiles),
	}, nil
}

// toResponse converts Box model to response DTO
func (s *BoxService) toResponse(box *models.Box) *models.BoxResponse {
	return &models.BoxResponse{
		ID:        box.ID,
		UserID:    box.UserID,
		MachineID: box.MachineID,
		Name:      box.Name,
		CreatedAt: box.CreatedAt.Format(time.RFC3339),
		UpdatedAt: box.UpdatedAt.Format(time.RFC3339),
	}
}

// GetBoxRepo returns the box repository
func (s *BoxService) GetBoxRepo() *repository.BoxRepository {
	return s.boxRepo
}

// determinePlatformFromAppName determines platform type from app name
func (s *BoxService) determinePlatformFromAppName(appName string) string {
	appNameLower := strings.ToLower(appName)

	switch {
	case strings.Contains(appNameLower, "hidemium"):
		return "hidemium"
	case strings.Contains(appNameLower, "genlogin"):
		return "genlogin"
	default:
		return ""
	}
}

// getProfilesFromPlatformViaProxy gets profiles from platform using HTTP request to the platform
func (s *BoxService) getProfilesFromPlatformViaProxy(app *models.App, platformType string) ([]map[string]interface{}, error) {
	// Check if tunnel URL is available
	if app.TunnelURL == nil || *app.TunnelURL == "" {
		return nil, fmt.Errorf("tunnel URL not configured for app %s", app.ID)
	}

	// Build the URL to get profiles from platform
	var profileURL string
	switch platformType {
	case "hidemium":
		profileURL = fmt.Sprintf("%s/v1/browser/list", *app.TunnelURL)
	case "genlogin":
		profileURL = fmt.Sprintf("%s/profiles/list", *app.TunnelURL)
	default:
		return nil, fmt.Errorf("unsupported platform type: %s", platformType)
	}

	// Make HTTP request to get profiles
	client := &http.Client{Timeout: 30 * time.Second}

	var req *http.Request
	var err error

	// Hidemium uses POST, GenLogin uses GET
	if platformType == "hidemium" {
		req, err = http.NewRequest("POST", profileURL, nil)
	} else {
		req, err = http.NewRequest("GET", profileURL, nil)
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

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse response based on platform
	var profiles []map[string]interface{}

	switch platformType {
	case "hidemium":
		profiles, err = s.parseHidemiumProfilesResponse(body)
	case "genlogin":
		profiles, err = s.parseGenLoginProfilesResponse(body)
	default:
		return nil, fmt.Errorf("unsupported platform type for parsing: %s", platformType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse profiles response: %w", err)
	}

	return profiles, nil
}

// parseHidemiumProfilesResponse parses Hidemium profiles response
func (s *BoxService) parseHidemiumProfilesResponse(body []byte) ([]map[string]interface{}, error) {
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Hidemium response: %w", err)
	}

	// Extract profiles from response
	var profiles []map[string]interface{}

	// Try different possible field names for profiles
	possibleFields := []string{"data", "profiles", "result", "browsers", "list"}

	for _, field := range possibleFields {
		if data, exists := response[field]; exists {
			// Handle nested structure: data.content for Hidemium
			if field == "data" {
				if dataMap, ok := data.(map[string]interface{}); ok {
					// Try to find profiles in data.content
					if content, exists := dataMap["content"]; exists {
						if profilesData, ok := content.([]interface{}); ok {
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
			}

			// Handle direct array of profiles
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

	if len(profiles) == 0 {
		fmt.Printf("No profiles found in response\n")
	}

	return profiles, nil
}

// parseGenLoginProfilesResponse parses GenLogin profiles response
func (s *BoxService) parseGenLoginProfilesResponse(body []byte) ([]map[string]interface{}, error) {
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal GenLogin response: %w", err)
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

	if len(profiles) == 0 {
		fmt.Printf("No profiles found in response\n")
	}

	return profiles, nil
}

// processSyncedProfiles processes profiles synced from platform and updates local database
func (s *BoxService) processSyncedProfiles(appID string, platformProfiles []map[string]interface{}) (*models.SyncBoxProfilesResponse, error) {
	result := &models.SyncBoxProfilesResponse{
		ProfilesCreated: 0,
		ProfilesUpdated: 0,
		ProfilesDeleted: 0,
		ProfilesSynced:  len(platformProfiles),
	}

	// Get existing profiles for this app
	existingProfiles, err := s.profileRepo.GetByAppID(appID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing profiles: %w", err)
	}

	// Create a map of existing profiles by UUID for quick lookup
	existingProfilesMap := make(map[string]*models.Profile)
	for _, profile := range existingProfiles {
		if uuid := s.extractUUID(profile); uuid != "" {
			existingProfilesMap[uuid] = profile
		}
	}

	// Process each platform profile
	for _, platformProfile := range platformProfiles {
		if err := s.processPlatformProfile(appID, platformProfile, existingProfilesMap, result); err != nil {
			fmt.Printf("Warning: Failed to process profile: %v\n", err)
			continue
		}
	}

	// Mark profiles as deleted if they exist locally but not on platform
	s.markDeletedProfiles(existingProfilesMap, result)

	return result, nil
}

// processPlatformProfile processes a single platform profile
func (s *BoxService) processPlatformProfile(appID string, platformProfile map[string]interface{}, existingProfilesMap map[string]*models.Profile, result *models.SyncBoxProfilesResponse) error {
	// Extract UUID from platform profile
	uuid := s.extractUUIDFromPlatformProfile(platformProfile)
	if uuid == "" {
		return fmt.Errorf("platform profile has no UUID")
	}

	// Check if profile already exists
	existingProfile, exists := existingProfilesMap[uuid]

	if exists {
		// Update existing profile
		if err := s.updateExistingProfile(existingProfile, platformProfile); err != nil {
			return fmt.Errorf("failed to update profile: %w", err)
		}
		result.ProfilesUpdated++
		// Remove from map to avoid marking as deleted
		delete(existingProfilesMap, uuid)
	} else {
		// Create new profile
		if err := s.createNewProfile(appID, platformProfile); err != nil {
			return fmt.Errorf("failed to create profile: %w", err)
		}
		result.ProfilesCreated++
	}

	return nil
}

// createNewProfile creates a new profile from platform data
func (s *BoxService) createNewProfile(appID string, platformProfile map[string]interface{}) error {
	// Extract profile name
	profileName, ok := platformProfile["name"].(string)
	if !ok {
		profileName = "Unknown Profile"
	}

	// Create profile in database
	profile := &models.Profile{
		AppID: appID,
		Name:  profileName,
		Data:  platformProfile,
	}

	return s.profileRepo.Create(profile)
}

// updateExistingProfile updates an existing profile with platform data
func (s *BoxService) updateExistingProfile(existingProfile *models.Profile, platformProfile map[string]interface{}) error {
	// Update profile data with latest platform information
	if existingProfile.Data == nil {
		existingProfile.Data = make(map[string]interface{})
	}

	// Merge platform data
	for key, value := range platformProfile {
		existingProfile.Data[key] = value
	}

	// Update name if changed
	if name, exists := platformProfile["name"]; exists {
		if nameStr, ok := name.(string); ok && nameStr != existingProfile.Name {
			existingProfile.Name = nameStr
		}
	}

	return s.profileRepo.Update(existingProfile)
}

// markDeletedProfiles marks profiles as deleted if they no longer exist on platform
func (s *BoxService) markDeletedProfiles(existingProfilesMap map[string]*models.Profile, result *models.SyncBoxProfilesResponse) {
	for _, profile := range existingProfilesMap {
		// Clear campaign associations before deleting profile
		if err := s.profileRepo.ClearCampaignAssociations(profile.ID); err != nil {
			fmt.Printf("Warning: Failed to clear campaign associations for profile %s: %v\n", profile.Name, err)
			continue
		}

		// Profile exists in local DB but not on platform - delete it
		if err := s.profileRepo.Delete(profile.ID); err != nil {
			fmt.Printf("Warning: Failed to delete profile %s: %v\n", profile.Name, err)
			continue
		}
		result.ProfilesDeleted++
	}
}

// extractUUID extracts UUID from profile data
func (s *BoxService) extractUUID(profile *models.Profile) string {
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

// extractUUIDFromPlatformProfile extracts UUID from platform profile data
func (s *BoxService) extractUUIDFromPlatformProfile(platformProfile map[string]interface{}) string {
	// Try different possible field names for UUID
	possibleUUIDFields := []string{"uuid", "id", "browser_id", "profile_id", "browser_uuid", "profile_uuid"}

	for _, field := range possibleUUIDFields {
		if uuid, exists := platformProfile[field]; exists {
			if uuidStr, ok := uuid.(string); ok && uuidStr != "" {
				return uuidStr
			}
		}
	}

	return ""
}

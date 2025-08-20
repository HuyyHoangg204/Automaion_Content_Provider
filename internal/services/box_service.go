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

type BoxService struct {
	boxRepo     *repository.BoxRepository
	userRepo    *repository.UserRepository
	appRepo     *repository.AppRepository
	profileRepo *repository.ProfileRepository
}

func NewBoxService(boxRepo *repository.BoxRepository, userRepo *repository.UserRepository, appRepo *repository.AppRepository, profileRepo *repository.ProfileRepository) *BoxService {
	return &BoxService{
		boxRepo:     boxRepo,
		userRepo:    userRepo,
		appRepo:     appRepo,
		profileRepo: profileRepo,
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
	// Validate and normalize pagination parameters
	page, pageSize = utils.ValidateAndNormalizePagination(page, pageSize)

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

	fmt.Printf("Before update - Box ID: %s, Current UserID: %s, Request UserID: %s, Request Name: %s\n",
		box.ID, box.UserID, req.UserID, req.Name)

	// If updating user_id (transferring ownership)
	if req.UserID != "" {
		// Verify that the new user exists
		_, err := s.userRepo.GetByID(req.UserID)
		if err != nil {
			return nil, errors.New("new user not found")
		}

		// Update user_id
		box.UserID = req.UserID
		fmt.Printf("Updated UserID to: %s\n", box.UserID)
	}

	// Update name
	box.Name = req.Name
	fmt.Printf("Updated Name to: %s\n", box.Name)

	if err := s.boxRepo.Update(box); err != nil {
		fmt.Printf("Database update error: %v\n", err)
		return nil, fmt.Errorf("failed to update box: %w", err)
	}

	// Get the updated box to verify changes
	updatedBox, err := s.boxRepo.GetByID(boxID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated box: %w", err)
	}

	fmt.Printf("After update - Box ID: %s, UserID: %s, Name: %s\n",
		updatedBox.ID, updatedBox.UserID, updatedBox.Name)

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

// SyncBoxProfilesFromHidemium syncs all profiles from a box's Hidemium instance
func (s *BoxService) SyncBoxProfilesFromHidemium(userID, boxID string) (*models.SyncBoxProfilesResponse, error) {
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

	// For now, we'll sync to the first app. In the future, you might want to sync to specific apps
	app := apps[0]

	// Get Hidemium config
	hidemiumConfig := config.GetHidemiumConfig()

	// Construct tunnel URL using config
	baseURL := hidemiumConfig.BaseURL
	baseURL = strings.Replace(baseURL, "{machine_id}", box.MachineID, 1)

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

	// Fetch all profiles
	var allHidemiumProfiles []models.HidemiumProfile
	page := 1
	limit := 100

	fmt.Printf("Starting sync for box %s (MachineID: %s)\n", boxID, box.MachineID)
	fmt.Printf("Using API URL: %s\n", apiURL)

	for {
		// Prepare request body
		requestBody := map[string]interface{}{
			"orderName":     0,
			"orderLastOpen": 0,
			"page":          page,
			"limit":         limit,
			"search":        "",
			"status":        "",
			"date_range":    []string{"", ""},
			"folder_id":     []string{},
		}

		// Convert to JSON
		jsonBody, err := json.Marshal(requestBody)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}

		// Create POST request
		req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBody))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		// Make HTTP request to Hidemium API
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to Hidemium API on page %d: %w", page, err)
		}
		defer resp.Body.Close()

		// Read response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body on page %d: %w", page, err)
		}

		// Check HTTP status code
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("hidemium API returned status %d on page %d: %s", resp.StatusCode, page, string(body))
		}

		// Parse response for this page
		pageProfiles, hasMore, err := s.parseHidemiumResponse(body)
		if err != nil {
			return nil, fmt.Errorf("failed to parse response on page %d: %w", page, err)
		}

		// Add profiles from this page to the total
		allHidemiumProfiles = append(allHidemiumProfiles, pageProfiles...)

		// If no more pages or no profiles returned, break
		if !hasMore || len(pageProfiles) == 0 {
			break
		}

		page++

		// Safety check to prevent infinite loop
		if page > 100 {
			fmt.Printf("Safety break: reached max pages (%d)\n", page)
			break
		}
	}
	fmt.Printf("Synced profiles successfully from Hidemium\n")
	fmt.Printf("Total profiles fetched: %d\n", len(allHidemiumProfiles))

	// Get existing profiles for this app
	existingProfiles, err := s.profileRepo.GetByAppID(app.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing profiles: %w", err)
	}

	// Create maps for efficient lookup
	existingProfilesMap := make(map[string]*models.Profile)
	for _, profile := range existingProfiles {
		existingProfilesMap[profile.Name] = profile
	}

	hidemiumProfilesMap := make(map[string]models.HidemiumProfile)
	for _, profile := range allHidemiumProfiles {
		hidemiumProfilesMap[profile.Name] = profile
	}

	// Counters for response
	profilesCreated := 0
	profilesUpdated := 0
	profilesDeleted := 0

	// Create or update profiles from Hidemium
	for _, hidemiumProfile := range allHidemiumProfiles {
		if existingProfile, exists := existingProfilesMap[hidemiumProfile.Name]; exists {
			// Profile exists, update it
			existingProfile.Data = models.JSON(hidemiumProfile.Data)
			if err := s.profileRepo.Update(existingProfile); err != nil {
				return nil, fmt.Errorf("failed to update profile '%s': %w", hidemiumProfile.Name, err)
			}
			profilesUpdated++
		} else {
			// Profile doesn't exist, create it
			newProfile := &models.Profile{
				AppID: app.ID,
				Name:  hidemiumProfile.Name,
				Data:  models.JSON(hidemiumProfile.Data),
			}
			if err := s.profileRepo.Create(newProfile); err != nil {
				return nil, fmt.Errorf("failed to create profile '%s': %w", hidemiumProfile.Name, err)
			}
			profilesCreated++
		}
	}

	// Delete profiles that no longer exist in Hidemium
	for name, existingProfile := range existingProfilesMap {
		if _, exists := hidemiumProfilesMap[name]; !exists {
			if err := s.profileRepo.Delete(existingProfile.ID); err != nil {
				return nil, fmt.Errorf("failed to delete profile '%s': %w", name, err)
			}
			profilesDeleted++
		}
	}

	// Create response
	response := &models.SyncBoxProfilesResponse{
		BoxID:           box.ID,
		MachineID:       box.MachineID,
		TunnelURL:       apiURL,
		ProfilesSynced:  len(allHidemiumProfiles),
		ProfilesCreated: profilesCreated,
		ProfilesUpdated: profilesUpdated,
		ProfilesDeleted: profilesDeleted,
		Message:         fmt.Sprintf("Sync completed: %d created, %d updated, %d deleted", profilesCreated, profilesUpdated, profilesDeleted),
	}

	return response, nil
}

// parseHidemiumResponse parses the Hidemium API response and returns profiles and pagination info
func (s *BoxService) parseHidemiumResponse(body []byte) ([]models.HidemiumProfile, bool, error) {
	// First, try to parse as generic JSON to understand the structure
	var rawResponse map[string]interface{}
	if err := json.Unmarshal(body, &rawResponse); err != nil {
		return nil, false, fmt.Errorf("failed to parse response as JSON: %w", err)
	}

	// Extract profiles from different possible response formats
	var hidemiumProfiles []models.HidemiumProfile
	hasMore := false

	// Try different possible field names for profiles
	if data, exists := rawResponse["data"]; exists {
		if dataMap, ok := data.(map[string]interface{}); ok {
			// Check if data has a "content" field (Hidemium format)
			if content, exists := dataMap["content"]; exists {
				if profilesData, ok := content.([]interface{}); ok {
					fmt.Printf("Found %d profiles in content\n", len(profilesData))
					// Convert []interface{} to []HidemiumProfile
					for i, item := range profilesData {
						if profileMap, ok := item.(map[string]interface{}); ok {
							profile := models.HidemiumProfile{
								ID:        getStringFromMap(profileMap, "uuid"),
								Name:      getStringFromMap(profileMap, "name"),
								CreatedAt: getStringFromMap(profileMap, "created_at"),
								UpdatedAt: getStringFromMap(profileMap, "created_at"),
								IsActive:  getBoolFromMap(profileMap, "can_be_running"),
								Data:      profileMap,
							}
							hidemiumProfiles = append(hidemiumProfiles, profile)
							if i < 2 { // Log first 2 profiles for debugging
								fmt.Printf("Profile %d: %s (ID: %s)\n", i+1, profile.Name, profile.ID)
							}
						}
					}
				}
			} else if profilesData, ok := data.([]interface{}); ok {
				fmt.Printf("Found %d profiles in data array\n", len(profilesData))
				// Direct array in data field
				for i, item := range profilesData {
					if profileMap, ok := item.(map[string]interface{}); ok {
						profile := models.HidemiumProfile{
							ID:        getStringFromMap(profileMap, "id"),
							Name:      getStringFromMap(profileMap, "name"),
							CreatedAt: getStringFromMap(profileMap, "created_at"),
							UpdatedAt: getStringFromMap(profileMap, "updated_at"),
							IsActive:  getBoolFromMap(profileMap, "is_active"),
							Data:      profileMap,
						}
						hidemiumProfiles = append(hidemiumProfiles, profile)
						if i < 2 { // Log first 2 profiles for debugging
							fmt.Printf("Profile %d: %s (ID: %s)\n", i+1, profile.Name, profile.ID)
						}
					}
				}
			}
		}
	} else if profilesData, exists := rawResponse["profiles"]; exists {
		// Handle case where profiles are directly in "profiles" field
		if profilesArray, ok := profilesData.([]interface{}); ok {
			fmt.Printf("Found %d profiles in profiles field\n", len(profilesArray))
			for i, item := range profilesArray {
				if profileMap, ok := item.(map[string]interface{}); ok {
					profile := models.HidemiumProfile{
						ID:        getStringFromMap(profileMap, "id"),
						Name:      getStringFromMap(profileMap, "name"),
						CreatedAt: getStringFromMap(profileMap, "created_at"),
						UpdatedAt: getStringFromMap(profileMap, "updated_at"),
						IsActive:  getBoolFromMap(profileMap, "is_active"),
						Data:      profileMap,
					}
					hidemiumProfiles = append(hidemiumProfiles, profile)
					if i < 2 { // Log first 2 profiles for debugging
						fmt.Printf("Profile %d: %s (ID: %s)\n", i+1, profile.Name, profile.ID)
					}
				}
			}
		}
	} else if resultData, exists := rawResponse["result"]; exists {
		// Handle case where profiles are in "result" field
		if resultArray, ok := resultData.([]interface{}); ok {
			fmt.Printf("Found %d profiles in result field\n", len(resultArray))
			for i, item := range resultArray {
				if profileMap, ok := item.(map[string]interface{}); ok {
					profile := models.HidemiumProfile{
						ID:        getStringFromMap(profileMap, "id"),
						Name:      getStringFromMap(profileMap, "name"),
						CreatedAt: getStringFromMap(profileMap, "created_at"),
						UpdatedAt: getStringFromMap(profileMap, "updated_at"),
						IsActive:  getBoolFromMap(profileMap, "is_active"),
						Data:      profileMap,
					}
					hidemiumProfiles = append(hidemiumProfiles, profile)
					if i < 2 { // Log first 2 profiles for debugging
						fmt.Printf("Profile %d: %s (ID: %s)\n", i+1, profile.Name, profile.ID)
					}
				}
			}
		}
	}

	// Check if we found any profiles
	if len(hidemiumProfiles) == 0 {
		// If no profiles found, check if the response itself is an array
		var directProfiles []map[string]interface{}
		if err := json.Unmarshal(body, &directProfiles); err == nil {
			fmt.Printf("Found %d profiles in direct array\n", len(directProfiles))
			// Response is directly an array of profiles
			for i, profileMap := range directProfiles {
				profile := models.HidemiumProfile{
					ID:        getStringFromMap(profileMap, "id"),
					Name:      getStringFromMap(profileMap, "name"),
					CreatedAt: getStringFromMap(profileMap, "created_at"),
					UpdatedAt: getStringFromMap(profileMap, "updated_at"),
					IsActive:  getBoolFromMap(profileMap, "is_active"),
					Data:      profileMap,
				}
				hidemiumProfiles = append(hidemiumProfiles, profile)
				if i < 2 { // Log first 2 profiles for debugging
					fmt.Printf("Profile %d: %s (ID: %s)\n", i+1, profile.Name, profile.ID)
				}
			}
		}
	}

	// Check pagination info - Hidemium uses meta and links structure
	if meta, exists := rawResponse["meta"]; exists {
		if metaMap, ok := meta.(map[string]interface{}); ok {
			// Check current_page vs last_page
			if currentPage, exists := metaMap["current_page"]; exists {
				if lastPage, exists := metaMap["last_page"]; exists {
					if currentPageFloat, ok := currentPage.(float64); ok {
						if lastPageFloat, ok := lastPage.(float64); ok {
							hasMore = currentPageFloat < lastPageFloat
							fmt.Printf("Pagination: page %.0f/%.0f, hasMore=%v\n", currentPageFloat, lastPageFloat, hasMore)
						}
					}
				}
			}

			// Log total for reference
			if total, exists := metaMap["total"]; exists {
				fmt.Printf("Total profiles: %v\n", total)
			}
		}
	}

	// Also check links.next for additional pagination info
	if links, exists := rawResponse["links"]; exists {
		if linksMap, ok := links.(map[string]interface{}); ok {
			if next, exists := linksMap["next"]; exists {
				if next != nil {
					// If next is not null, there are more pages
					if !hasMore {
						hasMore = true
						fmt.Printf("Setting hasMore=true based on links.next\n")
					}
				}
			}
		}
	}

	fmt.Printf("Final: %d profiles, hasMore=%v\n", len(hidemiumProfiles), hasMore)
	return hidemiumProfiles, hasMore, nil
}

// SyncAllUserBoxes syncs all boxes for a specific user
func (s *BoxService) SyncAllUserBoxes(userID string) (*models.SyncAllUserBoxesResponse, error) {
	// Get all boxes for the user (for sync operation, we need all boxes)
	boxes, err := s.boxRepo.GetAllByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user boxes: %w", err)
	}

	if len(boxes) == 0 {
		return &models.SyncAllUserBoxesResponse{
			UserID:          userID,
			TotalBoxes:      0,
			BoxesSynced:     0,
			TotalProfiles:   0,
			ProfilesCreated: 0,
			ProfilesUpdated: 0,
			ProfilesDeleted: 0,
			Message:         "No boxes found for user",
		}, nil
	}

	// Counters for overall response
	totalProfiles := 0
	totalProfilesCreated := 0
	totalProfilesUpdated := 0
	totalProfilesDeleted := 0
	boxesSynced := 0
	boxResults := make([]models.BoxSyncResult, 0)

	// Sync each box
	for _, box := range boxes {
		// Get apps for this box
		apps, err := s.appRepo.GetByBoxID(box.ID)
		if err != nil {
			// Log error but continue with other boxes
			boxResults = append(boxResults, models.BoxSyncResult{
				BoxID:     box.ID,
				MachineID: box.MachineID,
				Name:      box.Name,
				Success:   false,
				Error:     fmt.Sprintf("Failed to get apps: %v", err),
			})
			continue
		}

		if len(apps) == 0 {
			boxResults = append(boxResults, models.BoxSyncResult{
				BoxID:     box.ID,
				MachineID: box.MachineID,
				Name:      box.Name,
				Success:   false,
				Error:     "No apps found for this box",
			})
			continue
		}

		// Sync the first app (assuming one app per box for now)
		syncResponse, err := s.SyncBoxProfilesFromHidemium(userID, box.ID)

		if err != nil {
			boxResults = append(boxResults, models.BoxSyncResult{
				BoxID:     box.ID,
				MachineID: box.MachineID,
				Name:      box.Name,
				Success:   false,
				Error:     err.Error(),
			})
			continue
		}

		// Box synced successfully
		boxesSynced++
		totalProfiles += syncResponse.ProfilesSynced
		totalProfilesCreated += syncResponse.ProfilesCreated
		totalProfilesUpdated += syncResponse.ProfilesUpdated
		totalProfilesDeleted += syncResponse.ProfilesDeleted

		boxResults = append(boxResults, models.BoxSyncResult{
			BoxID:           box.ID,
			MachineID:       box.MachineID,
			Name:            box.Name,
			Success:         true,
			ProfilesSynced:  syncResponse.ProfilesSynced,
			ProfilesCreated: syncResponse.ProfilesCreated,
			ProfilesUpdated: syncResponse.ProfilesUpdated,
			ProfilesDeleted: syncResponse.ProfilesDeleted,
		})
	}

	// Create overall response
	response := &models.SyncAllUserBoxesResponse{
		UserID:          userID,
		TotalBoxes:      len(boxes),
		BoxesSynced:     boxesSynced,
		TotalProfiles:   totalProfiles,
		ProfilesCreated: totalProfilesCreated,
		ProfilesUpdated: totalProfilesUpdated,
		ProfilesDeleted: totalProfilesDeleted,
		BoxResults:      boxResults,
		Message:         fmt.Sprintf("Sync completed: %d/%d boxes synced, %d profiles processed", boxesSynced, len(boxes), totalProfiles),
	}

	return response, nil
}

// Helper functions to safely extract values from map[string]interface{}
func getStringFromMap(m map[string]interface{}, key string) string {
	if val, exists := m[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getBoolFromMap(m map[string]interface{}, key string) bool {
	if val, exists := m[key]; exists {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

// Helper function to get keys of a map
func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
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

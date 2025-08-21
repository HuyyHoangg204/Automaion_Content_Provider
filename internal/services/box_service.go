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

type BoxService struct {
	boxRepo         *repository.BoxRepository
	userRepo        *repository.UserRepository
	appRepo         *repository.AppRepository
	profileRepo     *repository.ProfileRepository
	platformWrapper *PlatformWrapperService
}

func NewBoxService(boxRepo *repository.BoxRepository, userRepo *repository.UserRepository, appRepo *repository.AppRepository, profileRepo *repository.ProfileRepository) *BoxService {
	return &BoxService{
		boxRepo:         boxRepo,
		userRepo:        userRepo,
		appRepo:         appRepo,
		profileRepo:     profileRepo,
		platformWrapper: NewPlatformWrapperService(),
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

// SyncBoxProfilesFromPlatform syncs all profiles from a box's platform instance
// Supports multiple platforms through platform system
func (s *BoxService) SyncBoxProfilesFromPlatform(userID, boxID string) (*models.SyncBoxProfilesResponse, error) {
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

	// Determine platform from app name
	platformType := s.determinePlatformFromAppName(app.Name)
	if platformType == "" {
		return nil, fmt.Errorf("unsupported platform: %s", app.Name)
	}

	fmt.Printf("Starting sync for box %s (MachineID: %s) on platform %s\n", boxID, box.MachineID, platformType)

	// Use platform wrapper to sync profiles from platform
	platformProfiles, err := s.platformWrapper.SyncProfilesFromPlatform(context.Background(), platformType, boxID, box.MachineID)
	if err != nil {
		return nil, fmt.Errorf("failed to sync profiles from %s: %w", platformType, err)
	}

	fmt.Printf("Successfully synced %d profiles from %s for box %s\n", len(platformProfiles), platformType, boxID)

	// Process synced profiles and update local database
	syncResult, err := s.processSyncedProfiles(app.ID, platformProfiles)
	if err != nil {
		return nil, fmt.Errorf("failed to process synced profiles: %w", err)
	}

	// Update response with sync results
	syncResult.BoxID = boxID
	syncResult.MachineID = box.MachineID
	syncResult.Message = fmt.Sprintf("Profiles synced successfully from %s", platformType)

	return syncResult, nil
}

// processSyncedProfiles processes profiles synced from platform and updates local database
func (s *BoxService) processSyncedProfiles(appID string, platformProfiles []models.HidemiumProfile) (*models.SyncBoxProfilesResponse, error) {
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
			fmt.Printf("Warning: Failed to process profile %s: %v\n", platformProfile.ID, err)
			continue
		}
	}

	// Mark profiles as deleted if they exist locally but not on platform
	s.markDeletedProfiles(existingProfilesMap, result)

	return result, nil
}

// processPlatformProfile processes a single platform profile
func (s *BoxService) processPlatformProfile(appID string, platformProfile models.HidemiumProfile, existingProfilesMap map[string]*models.Profile, result *models.SyncBoxProfilesResponse) error {
	// Extract UUID from platform profile
	uuid := platformProfile.ID
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
func (s *BoxService) createNewProfile(appID string, platformProfile models.HidemiumProfile) error {
	// Prepare profile data with platform information
	profileData := map[string]interface{}{
		"uuid":       platformProfile.ID,
		"name":       platformProfile.Name,
		"created_at": platformProfile.CreatedAt,
		"updated_at": platformProfile.UpdatedAt,
		"is_active":  platformProfile.IsActive,
		// Add machine_id from box for future profile operations
		"machine_id": s.getMachineIDFromApp(appID),
	}

	// Merge with platform profile data
	for key, value := range platformProfile.Data {
		profileData[key] = value
	}

	profile := &models.Profile{
		AppID: appID,
		Name:  platformProfile.Name,
		Data:  profileData,
	}

	return s.profileRepo.Create(profile)
}

// updateExistingProfile updates an existing profile with platform data
func (s *BoxService) updateExistingProfile(existingProfile *models.Profile, platformProfile models.HidemiumProfile) error {
	// Update profile data with latest platform information
	if existingProfile.Data == nil {
		existingProfile.Data = make(map[string]interface{})
	}

	// Update platform-specific fields
	existingProfile.Data["uuid"] = platformProfile.ID
	existingProfile.Data["name"] = platformProfile.Name
	existingProfile.Data["updated_at"] = platformProfile.UpdatedAt
	existingProfile.Data["is_active"] = platformProfile.IsActive

	// Merge with platform profile data
	for key, value := range platformProfile.Data {
		existingProfile.Data[key] = value
	}

	// Update name if changed
	if existingProfile.Name != platformProfile.Name {
		existingProfile.Name = platformProfile.Name
	}

	return s.profileRepo.Update(existingProfile)
}

// markDeletedProfiles marks profiles as deleted if they no longer exist on platform
func (s *BoxService) markDeletedProfiles(existingProfilesMap map[string]*models.Profile, result *models.SyncBoxProfilesResponse) {
	for uuid, profile := range existingProfilesMap {
		// Profile exists in local DB but not on platform - delete it
		fmt.Printf("Deleting profile %s (%s) as it no longer exists on platform\n", profile.Name, uuid)

		if err := s.profileRepo.Delete(profile.ID); err != nil {
			fmt.Printf("Warning: Failed to delete profile %s: %v\n", profile.Name, err)
			continue
		}

		fmt.Printf("Successfully deleted profile %s from database\n", profile.Name)
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

// getMachineIDFromApp gets machine_id from app's box
func (s *BoxService) getMachineIDFromApp(appID string) string {
	// Get app to find box
	app, err := s.appRepo.GetByID(appID)
	if err != nil {
		fmt.Printf("Warning: Failed to get app %s: %v\n", appID, err)
		return ""
	}

	// Get box to find machine_id
	box, err := s.boxRepo.GetByID(app.BoxID)
	if err != nil {
		fmt.Printf("Warning: Failed to get box %s: %v\n", app.BoxID, err)
		return ""
	}

	return box.MachineID
}

// determinePlatformFromAppName determines platform type from app name
func (s *BoxService) determinePlatformFromAppName(appName string) string {
	switch appName {
	case "Hidemium":
		return "hidemium"
	case "Genlogin":
		return "genlogin"
	default:
		return ""
	}
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

		// Sync all apps in this box
		var syncResponse *models.SyncBoxProfilesResponse
		var syncErr error

		for _, app := range apps {
			platformType := s.determinePlatformFromAppName(app.Name)

			if platformType == "" {
				fmt.Printf("Warning: Unsupported platform %s for app %s in box %s\n", app.Name, app.ID, box.ID)
				continue
			}

			// Fetch profiles from platform
			platformProfiles, err := s.platformWrapper.SyncProfilesFromPlatform(context.Background(), platformType, box.ID, box.MachineID)
			if err != nil {
				syncErr = err
				break
			}

			// Process the fetched profiles into local database
			appSyncResponse, err := s.processSyncedProfiles(app.ID, platformProfiles)
			if err != nil {
				syncErr = err
				break
			}

			// Use the response from the first successful sync (or accumulate if needed)
			if syncResponse == nil {
				syncResponse = appSyncResponse
				syncResponse.BoxID = box.ID
				syncResponse.MachineID = box.MachineID
			} else {
				// Accumulate sync results from multiple apps
				syncResponse.ProfilesSynced += appSyncResponse.ProfilesSynced
				syncResponse.ProfilesCreated += appSyncResponse.ProfilesCreated
				syncResponse.ProfilesUpdated += appSyncResponse.ProfilesUpdated
				syncResponse.ProfilesDeleted += appSyncResponse.ProfilesDeleted
			}
		}

		// Check if sync failed
		if syncErr != nil {
			boxResults = append(boxResults, models.BoxSyncResult{
				BoxID:     box.ID,
				MachineID: box.MachineID,
				Name:      box.Name,
				Success:   false,
				Error:     syncErr.Error(),
			})
			continue
		}

		// Check if no apps were successfully synced
		if syncResponse == nil {
			boxResults = append(boxResults, models.BoxSyncResult{
				BoxID:     box.ID,
				MachineID: box.MachineID,
				Name:      box.Name,
				Success:   false,
				Error:     "No supported platforms found in apps",
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

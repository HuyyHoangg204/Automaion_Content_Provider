package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/onegreenvn/green-provider-services-backend/internal/config"
	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/onegreenvn/green-provider-services-backend/internal/utils"
	"gorm.io/gorm"
)

// AppAlreadyExistsError represents an error when trying to create an app that already exists
type AppAlreadyExistsError struct {
	Message       string `json:"message"`
	ExistingAppID string `json:"existing_app_id"`
}

func (e *AppAlreadyExistsError) Error() string {
	return e.Message
}

type AppService struct {
	appRepo     *repository.AppRepository
	profileRepo *repository.ProfileRepository
	boxRepo     *repository.BoxRepository
	userRepo    *repository.UserRepository
}

func NewAppService(appRepo *repository.AppRepository, profileRepo *repository.ProfileRepository, boxRepo *repository.BoxRepository, userRepo *repository.UserRepository) *AppService {
	return &AppService{
		appRepo:     appRepo,
		profileRepo: profileRepo,
		boxRepo:     boxRepo,
		userRepo:    userRepo,
	}
}

// CreateApp creates a new app for a user
func (s *AppService) CreateApp(userID string, req *models.CreateAppRequest) (*models.AppResponse, error) {
	// Verify user exists
	_, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Verify box exists and belongs to user
	_, err = s.boxRepo.GetByUserIDAndID(userID, req.BoxID)
	if err != nil {
		return nil, errors.New("box not found or access denied")
	}

	// Check if app name already exists in this box
	existingApp, err := s.appRepo.GetByNameAndBoxID(req.BoxID, req.Name)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check app name: %w", err)
	}
	if existingApp != nil {
		return nil, &AppAlreadyExistsError{
			Message:       fmt.Sprintf("app with name '%s' already exists in this box", req.Name),
			ExistingAppID: existingApp.ID,
		}
	}

	// Create app
	app := &models.App{
		BoxID:     req.BoxID,
		Name:      req.Name,
		TunnelURL: req.TunnelURL,
	}

	if err := s.appRepo.Create(app); err != nil {
		return nil, fmt.Errorf("failed to create app: %w", err)
	}

	converter := utils.NewAppResponseConverter()
	return converter.ToAppResponse(app), nil
}

// GetAppsByUser retrieves all apps for a specific user
func (s *AppService) GetAppsByUser(userID string) ([]*models.AppResponse, error) {
	apps, err := s.appRepo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get apps: %w", err)
	}

	responses := make([]*models.AppResponse, len(apps))
	converter := utils.NewAppResponseConverter()
	for i, app := range apps {
		responses[i] = converter.ToAppResponse(app)
	}

	return responses, nil
}

// GetAppsByBox retrieves all apps for a specific box (user must own the box)
func (s *AppService) GetAppsByBox(userID, boxID string) ([]*models.AppResponse, error) {
	// Verify box belongs to user
	_, err := s.boxRepo.GetByUserIDAndID(userID, boxID)
	if err != nil {
		return nil, errors.New("box not found or access denied")
	}

	apps, err := s.appRepo.GetByBoxID(boxID)
	if err != nil {
		return nil, fmt.Errorf("failed to get apps: %w", err)
	}

	responses := make([]*models.AppResponse, len(apps))
	converter := utils.NewAppResponseConverter()
	for i, app := range apps {
		responses[i] = converter.ToAppResponse(app)
	}

	return responses, nil
}

// GetAppByID retrieves an app by ID (user must own it)
func (s *AppService) GetAppByID(userID, appID string) (*models.AppResponse, error) {
	app, err := s.appRepo.GetByUserIDAndID(userID, appID)
	if err != nil {
		return nil, errors.New("app not found")
	}

	converter := utils.NewAppResponseConverter()
	return converter.ToAppResponse(app), nil
}

// GetAppByUserIDAndID gets an app by ID and verifies user ownership
func (s *AppService) GetAppByUserIDAndID(userID, appID string) (*models.App, error) {
	// Get app by ID
	app, err := s.appRepo.GetByID(appID)
	if err != nil {
		return nil, err
	}

	// Get box that contains this app
	box, err := s.boxRepo.GetByID(app.BoxID)
	if err != nil {
		return nil, err
	}

	// Check if user owns the box
	if box.UserID != userID {
		return nil, errors.New("access denied: app does not belong to user")
	}

	return app, nil
}

// UpdateApp updates an app (user must own it)
func (s *AppService) UpdateApp(userID, appID string, req *models.UpdateAppRequest) (*models.AppResponse, error) {
	app, err := s.appRepo.GetByUserIDAndID(userID, appID)
	if err != nil {
		return nil, errors.New("app not found")
	}

	// Check if new name already exists in this box (if name is being changed)
	if req.Name != app.Name {
		existingApp, err := s.appRepo.GetByNameAndBoxID(app.BoxID, req.Name)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("failed to check app name: %w", err)
		}
		if existingApp != nil {
			return nil, &AppAlreadyExistsError{
				Message:       fmt.Sprintf("app with name '%s' already exists in this box", req.Name),
				ExistingAppID: existingApp.ID,
			}
		}
	}

	// Update fields
	app.Name = req.Name
	if req.TunnelURL != nil {
		app.TunnelURL = req.TunnelURL
	}

	if err := s.appRepo.Update(app); err != nil {
		return nil, fmt.Errorf("failed to update app: %w", err)
	}

	converter := utils.NewAppResponseConverter()
	return converter.ToAppResponse(app), nil
}

// DeleteApp deletes an app (user must own it)
func (s *AppService) DeleteApp(userID, appID string) error {
	// Check if app exists and belongs to user
	_, err := s.appRepo.GetByUserIDAndID(userID, appID)
	if err != nil {
		return errors.New("app not found")
	}

	if err := s.appRepo.DeleteByUserIDAndID(userID, appID); err != nil {
		return fmt.Errorf("failed to delete app: %w", err)
	}

	return nil
}

// GetAllApps retrieves all apps (admin only)
func (s *AppService) GetAllApps() ([]*models.AppResponse, error) {
	apps, err := s.appRepo.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get all apps: %w", err)
	}

	responses := make([]*models.AppResponse, len(apps))
	converter := utils.NewAppResponseConverter()
	for i, app := range apps {
		responses[i] = converter.ToAppResponse(app)
	}

	return responses, nil
}

// GetRegisterAppDomains generates subdomain and FRP configuration for app registration
func (s *AppService) GetRegisterAppDomains(userID, boxID, platformNames string) (*models.RegisterAppResponse, error) {
	// Get user to verify it exists
	_, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Verify box belongs to user and get machine_id
	box, err := s.boxRepo.GetByUserIDAndID(userID, boxID)
	if err != nil {
		return nil, fmt.Errorf("box not found or access denied: %w", err)
	}

	machineID := box.MachineID
	if machineID == "" {
		return nil, fmt.Errorf("box has no machine_id")
	}

	// Create response
	response := &models.RegisterAppResponse{}

	// Create dynamic subdomain map for all requested platforms
	response.SubDomain = make(map[string]string)

	// Split platform names by comma and create subdomain for each
	platformList := strings.Split(platformNames, ",")
	for _, platform := range platformList {
		platform = strings.TrimSpace(platform) // Remove whitespace
		if platform != "" {
			response.SubDomain[platform] = fmt.Sprintf("%s-%s-%s", machineID, platform, userID)
		}
	}

	// Set FRP configuration from environment variables
	frpConfig := config.GetFrpConfig()
	response.FrpDomain = frpConfig.Domain
	response.FrpServerPort = frpConfig.Port
	response.FrpToken = frpConfig.Token
	response.FrpProtocol = frpConfig.Protocol
	return response, nil
}

// CheckTunnelURL checks if a tunnel URL is accessible and ready for Hidemium
func (s *AppService) CheckTunnelURL(tunnelURL string) (*models.CheckTunnelResponse, error) {
	if tunnelURL == "" {
		return nil, errors.New("tunnel URL is empty")
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Record start time for response time measurement
	startTime := time.Now()

	// Test tunnel by calling /user-settings/token endpoint
	testURL := fmt.Sprintf("%s/user-settings/token", strings.TrimSuffix(tunnelURL, "/"))

	resp, err := client.Get(testURL)
	responseTime := time.Since(startTime).Milliseconds()

	if err != nil {
		errorMsg := err.Error()
		return &models.CheckTunnelResponse{
			IsAccessible: false,
			ResponseTime: responseTime,
			Message:      "Tunnel is not accessible",
			Error:        &errorMsg,
		}, nil
	}
	defer resp.Body.Close()

	// Check if response is successful
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &models.CheckTunnelResponse{
			IsAccessible: false,
			ResponseTime: responseTime,
			Message:      fmt.Sprintf("Tunnel returned status code: %d", resp.StatusCode),
			StatusCode:   &resp.StatusCode,
		}, nil
	}

	// Try to parse response to check if it contains token data
	var responseData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		return &models.CheckTunnelResponse{
			IsAccessible: false,
			ResponseTime: responseTime,
			Message:      "Tunnel accessible but response is not valid JSON",
			StatusCode:   &resp.StatusCode,
		}, nil
	}

	// Check if response contains token data
	if _, hasToken := responseData["token"]; hasToken {
		return &models.CheckTunnelResponse{
			IsAccessible: true,
			ResponseTime: responseTime,
			Message:      "Tunnel is accessible and /user-settings/token endpoint is working",
			StatusCode:   &resp.StatusCode,
		}, nil
	}

	// Check for other possible token fields
	tokenFields := []string{"access_token", "api_token", "auth_token", "key"}
	for _, field := range tokenFields {
		if _, hasField := responseData[field]; hasField {
			return &models.CheckTunnelResponse{
				IsAccessible: true,
				ResponseTime: responseTime,
				Message:      fmt.Sprintf("Tunnel is accessible and contains %s", field),
				StatusCode:   &resp.StatusCode,
			}, nil
		}
	}

	return &models.CheckTunnelResponse{
		IsAccessible: false,
		ResponseTime: responseTime,
		Message:      "Tunnel accessible but /user-settings/token endpoint does not return token data",
		StatusCode:   &resp.StatusCode,
	}, nil
}

// SyncAppProfiles syncs profiles for a specific app
func (s *AppService) SyncAppProfiles(app *models.App) (*models.SyncBoxProfilesResponse, error) {
	// Determine platform type from app name
	platformType := string(utils.GetPlatformType(app.Name))
	if platformType == "" {
		return nil, fmt.Errorf("unsupported platform for app %s", app.Name)
	}

	// Fetch profiles from platform
	appHelper := utils.NewAppHelper()
	platformProfiles, err := appHelper.FetchProfilesFromPlatform(app, platformType)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch profiles for app %s: %w", app.Name, err)
	}

	// Process synced profiles
	syncResult, err := s.ProcessSyncedProfiles(app.ID, platformProfiles)
	if err != nil {
		return nil, fmt.Errorf("failed to process profiles for app %s: %w", app.Name, err)
	}

	return syncResult, nil
}

// SyncAllAppsInBox syncs profiles from all apps in a box
func (s *AppService) SyncAllAppsInBox(boxID string) (*models.SyncBoxProfilesResponse, error) {
	// Get all apps for this box
	apps, err := s.appRepo.GetByBoxID(boxID)
	if err != nil {
		return nil, fmt.Errorf("failed to get apps for box %s: %w", boxID, err)
	}

	if len(apps) == 0 {
		return nil, errors.New("no apps found for this box")
	}

	// Track results
	var totalProfilesCreated, totalProfilesUpdated, totalProfilesDeleted int
	var totalProfilesSynced int
	var unsupportedPlatforms []string

	// Sync each app
	for _, app := range apps {
		// Determine platform from app name
		platformType := string(utils.GetPlatformType(app.Name))
		if platformType == "" {
			unsupportedPlatforms = append(unsupportedPlatforms, app.Name)
			continue
		}

		// Fetch profiles from platform
		appHelper := utils.NewAppHelper()
		platformProfiles, err := appHelper.FetchProfilesFromPlatform(app, platformType)
		if err != nil {
			fmt.Printf("Warning: Failed to get profiles from %s for app %s: %v\n", platformType, app.Name, err)
			continue
		}

		// Process synced profiles
		appSyncResult, err := s.ProcessSyncedProfiles(app.ID, platformProfiles)
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

	// Build response
	syncResult := &models.SyncBoxProfilesResponse{
		ProfilesCreated: totalProfilesCreated,
		ProfilesUpdated: totalProfilesUpdated,
		ProfilesDeleted: totalProfilesDeleted,
		ProfilesSynced:  totalProfilesSynced,
	}

	// Set message based on results
	if len(unsupportedPlatforms) > 0 {
		syncResult.Message = fmt.Sprintf("Profiles synced successfully from supported platforms. Unsupported platforms: %s", strings.Join(unsupportedPlatforms, ", "))
	} else {
		syncResult.Message = "Profiles synced successfully from all platforms"
	}

	return syncResult, nil
}

// ProcessSyncedProfiles processes profiles and updates local DB
func (s *AppService) ProcessSyncedProfiles(appID string, platformProfiles []map[string]interface{}) (*models.SyncBoxProfilesResponse, error) {
	result := &models.SyncBoxProfilesResponse{
		ProfilesSynced: len(platformProfiles),
	}

	// Get existing profiles for this app
	existingProfiles, err := s.profileRepo.GetByAppID(appID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing profiles: %w", err)
	}

	existingProfilesMap := make(map[string]*models.Profile)
	for _, profile := range existingProfiles {
		if uuid := utils.ExtractUUID(profile); uuid != "" {
			existingProfilesMap[uuid] = profile
		}
	}

	for _, platformProfile := range platformProfiles {
		if err := s.processPlatformProfile(appID, platformProfile, existingProfilesMap, result); err != nil {
			fmt.Printf("Warning: Failed to process profile: %v\n", err)
			continue
		}
	}

	s.MarkDeletedProfiles(existingProfilesMap, result)

	// Set message based on results
	if result.ProfilesCreated > 0 || result.ProfilesUpdated > 0 || result.ProfilesDeleted > 0 {
		result.Message = fmt.Sprintf("Sync completed: %d created, %d updated, %d deleted",
			result.ProfilesCreated, result.ProfilesUpdated, result.ProfilesDeleted)
	} else {
		result.Message = "No changes detected during sync"
	}
	return result, nil
}

func (s *AppService) processPlatformProfile(appID string, platformProfile map[string]interface{}, existingProfilesMap map[string]*models.Profile, result *models.SyncBoxProfilesResponse) error {
	appHelper := utils.NewAppHelper()
	return appHelper.ProcessPlatformProfile(appID, platformProfile, existingProfilesMap, result)
}

// CreateNewProfile creates a new profile using provided repo
func (s *AppService) CreateNewProfile(appID string, platformProfile map[string]interface{}) error {
	profileName, ok := platformProfile["name"].(string)
	if !ok {
		profileName = "Unknown Profile"
	}
	profile := &models.Profile{AppID: appID, Name: profileName, Data: platformProfile}
	return s.profileRepo.Create(profile)
}

// UpdateExistingProfile updates an existing profile using provided repo
func (s *AppService) UpdateExistingProfile(existingProfile *models.Profile, platformProfile map[string]interface{}) error {
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
	return s.profileRepo.Update(existingProfile)
}

// MarkDeletedProfiles clears associations and deletes missing profiles
func (s *AppService) MarkDeletedProfiles(existingProfilesMap map[string]*models.Profile, result *models.SyncBoxProfilesResponse) {
	for _, profile := range existingProfilesMap {
		if err := s.profileRepo.ClearCampaignAssociations(profile.ID); err != nil {
			fmt.Printf("Warning: Failed to clear campaign associations for profile %s: %v\n", profile.Name, err)
			continue
		}
		if err := s.profileRepo.Delete(profile.ID); err != nil {
			fmt.Printf("Warning: Failed to delete profile %s: %v\n", profile.Name, err)
			continue
		}
		result.ProfilesDeleted++
	}
}

// SyncAllAppsByUser syncs profiles from all apps owned by a user
func (s *AppService) SyncAllAppsByUser(userID string) (*models.SyncBoxProfilesResponse, error) {
	// Get all boxes for user
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
		// Sync all apps in this box
		syncResponse, err := s.SyncAllAppsInBox(box.ID)
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

	return &models.SyncBoxProfilesResponse{
		ProfilesCreated: totalProfilesCreated,
		ProfilesUpdated: totalProfilesUpdated,
		ProfilesDeleted: totalProfilesDeleted,
		ProfilesSynced:  totalProfiles,
		Message:         fmt.Sprintf("Sync completed: %d/%d boxes synced, %d profiles processed", boxesSynced, len(boxes), totalProfiles),
	}, nil
}

// SyncAllProfilesInBox syncs all profiles from all apps in a specific box
func (s *AppService) SyncAllProfilesInBox(userID, boxID string) (*models.SyncBoxProfilesResponse, error) {
	// Get box by ID and verify ownership
	box, err := s.boxRepo.GetByUserIDAndID(userID, boxID)
	if err != nil {
		return nil, errors.New("box not found")
	}

	// Sync all apps in this box
	syncResult, err := s.SyncAllAppsInBox(boxID)
	if err != nil {
		return nil, fmt.Errorf("failed to sync box %s: %w", boxID, err)
	}

	// Set box information
	syncResult.BoxID = boxID
	syncResult.MachineID = box.MachineID

	return syncResult, nil
}

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

	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/sirupsen/logrus"
)

type UserProfileService struct {
	userProfileRepo   *repository.UserProfileRepository
	appRepo           *repository.AppRepository
	geminiAccountRepo *repository.GeminiAccountRepository
	boxRepo           *repository.BoxRepository
}

func NewUserProfileService(userProfileRepo *repository.UserProfileRepository, appRepo *repository.AppRepository, geminiAccountRepo *repository.GeminiAccountRepository, boxRepo *repository.BoxRepository) *UserProfileService {
	return &UserProfileService{
		userProfileRepo:   userProfileRepo,
		appRepo:           appRepo,
		geminiAccountRepo: geminiAccountRepo,
		boxRepo:           boxRepo,
	}
}

// CreateUserProfileAndDeploy creates a UserProfile and deploys it to all machines
func (s *UserProfileService) CreateUserProfileAndDeploy(userID string, req *models.CreateUserProfileRequest) (*models.UserProfile, error) {
	// Check if user already has a profile (1-1 relationship)
	existingProfile, err := s.userProfileRepo.GetByUserID(userID)
	if err == nil && existingProfile != nil {
		return nil, errors.New("user already has a profile")
	}

	// Create UserProfile
	// Note: Không lưu UserDataDir vì path khác nhau trên mỗi máy automation
	// Automation backend tự resolve path: path.join(os.homedir(), 'AppData', 'Local', 'Automation_Profiles')
	userProfile := &models.UserProfile{
		UserID:            userID,
		Name:              req.Name,
		ProfileDirName:    req.ProfileDirName,
		Config:            req.Config,
		Settings:          req.Settings,
		ProfileVersion:    1,
		MachineSyncStatus: models.JSON(make(map[string]interface{})),
		DeployedMachines:  models.JSON(make(map[string]interface{})),
	}

	if err := s.userProfileRepo.Create(userProfile); err != nil {
		return nil, fmt.Errorf("failed to create user profile: %w", err)
	}

	// Deploy profile to all machines
	deployedApps, syncStatus, err := s.deployProfileToAllMachines(userProfile, req.Name)
	if err != nil {
		logrus.Warnf("Failed to deploy profile to all machines: %v", err)
		// Continue even if deployment fails - profile is created
	}

	// Update UserProfile with deployment info
	// Parse JSON bytes back to JSON type
	var deployedMachinesMap models.JSON
	var syncStatusMap models.JSON
	json.Unmarshal(deployedApps, &deployedMachinesMap)
	json.Unmarshal(syncStatus, &syncStatusMap)

	userProfile.DeployedMachines = deployedMachinesMap
	userProfile.MachineSyncStatus = syncStatusMap
	if err := s.userProfileRepo.Update(userProfile); err != nil {
		logrus.Warnf("Failed to update user profile with deployment info: %v", err)
	}

	return userProfile, nil
}

// deployProfileToAllMachines deploys profile to machines that have active GeminiAccount
// Only deploys to machines with tunnel URLs AND have an active, non-locked GeminiAccount
func (s *UserProfileService) deployProfileToAllMachines(userProfile *models.UserProfile, profileName string) ([]byte, []byte, error) {
	// Get all apps with tunnel URLs
	allApps, err := s.appRepo.GetAll()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get all apps: %w", err)
	}

	var deployedAppIDs []string
	syncStatusMap := make(map[string]interface{})

	// Filter apps with tunnel URLs AND have active GeminiAccount
	var appsWithGeminiAccount []*models.App
	for _, app := range allApps {
		if app.TunnelURL == nil || *app.TunnelURL == "" {
			continue
		}

		// Get Box to get MachineID (string) from BoxID (UUID)
		box, err := s.boxRepo.GetByID(app.BoxID)
		if err != nil {
			logrus.Warnf("Failed to get Box %s for app %s: %v", app.BoxID, app.ID, err)
			continue
		}

		// Check if this machine has an active GeminiAccount
		// Note: GeminiAccount.MachineID is Box.MachineID (string), not Box.ID (UUID)
		accounts, err := s.geminiAccountRepo.GetActiveByMachineID(box.MachineID)
		if err != nil {
			logrus.Warnf("Failed to check GeminiAccount for machine %s (app %s): %v", box.MachineID, app.ID, err)
			continue
		}

		// Only include machines that have at least one active GeminiAccount
		if len(accounts) > 0 {
			appsWithGeminiAccount = append(appsWithGeminiAccount, app)
			logrus.Infof("Machine %s (app %s) has active GeminiAccount, will deploy profile", box.MachineID, app.ID)
		} else {
			// Debug: Check if machine has any GeminiAccount (even inactive)
			allAccounts, err := s.geminiAccountRepo.GetByMachineID(box.MachineID)
			if err == nil && len(allAccounts) > 0 {
				logrus.Warnf("Machine %s (app %s) has GeminiAccount but it's not active. Account status: is_active=%v, is_locked=%v, gemini_accessible=%v",
					box.MachineID, app.ID, allAccounts[0].IsActive, allAccounts[0].IsLocked, allAccounts[0].GeminiAccessible)
			} else {
				logrus.Infof("Machine %s (app %s) does not have GeminiAccount, skipping profile deployment", box.MachineID, app.ID)
			}
		}
	}

	if len(appsWithGeminiAccount) == 0 {
		logrus.Info("No machines with active GeminiAccount found, skipping deployment")
		deployedAppsJSON, _ := json.Marshal(deployedAppIDs)
		syncStatusJSON, _ := json.Marshal(syncStatusMap)
		return deployedAppsJSON, syncStatusJSON, nil
	}

	// Deploy to each machine that has GeminiAccount
	for _, app := range appsWithGeminiAccount {
		platformProfileID, err := s.createProfileOnMachine(*app.TunnelURL, profileName)
		if err != nil {
			logrus.Errorf("Failed to create profile on machine %s (app %s): %v", app.BoxID, app.ID, err)
			// Don't save error to DB - just log it
			// Only save successful deployments to MachineSyncStatus
			continue
		}

		// Success - add to deployed list and save to sync status
		deployedAppIDs = append(deployedAppIDs, app.ID)
		syncStatusMap[app.ID] = map[string]interface{}{
			"platform_profile_id": platformProfileID,
			"last_synced_at":      time.Now().Format(time.RFC3339),
			"sync_version":        1,
			"is_synced":           true,
		}
		logrus.Infof("Successfully created profile on machine %s (app %s), platform_profile_id: %s", app.BoxID, app.ID, platformProfileID)
	}

	// Convert to JSON
	deployedAppsJSON, _ := json.Marshal(deployedAppIDs)
	syncStatusJSON, _ := json.Marshal(syncStatusMap)

	return deployedAppsJSON, syncStatusJSON, nil
}

// createProfileOnMachine calls API to create profile on a specific machine
func (s *UserProfileService) createProfileOnMachine(tunnelURL, profileName string) (string, error) {
	// Build API URL: POST /chrome/profiles
	apiURL := fmt.Sprintf("%s/chrome/profiles", strings.TrimSuffix(tunnelURL, "/"))

	// Prepare request body
	requestBody := map[string]string{
		"name": profileName,
	}
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Green-Provider-Services/1.0")

	// Make request
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse response to get profile ID
	var responseData map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &responseData); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract profile ID from response
	// Response can be either:
	// 1. Direct: {"id": "...", ...}
	// 2. Nested: {"profile": {"id": "...", ...}}
	profileID := ""

	// First, try to get from nested "profile" object (most common case)
	if profileObj, ok := responseData["profile"].(map[string]interface{}); ok {
		if id, ok := profileObj["id"].(string); ok {
			profileID = id
		} else if uuid, ok := profileObj["uuid"].(string); ok {
			profileID = uuid
		} else if profileIDVal, ok := profileObj["profile_id"].(string); ok {
			profileID = profileIDVal
		}
	}

	// If not found in nested object, try root level
	if profileID == "" {
		if id, ok := responseData["id"].(string); ok {
			profileID = id
		} else if uuid, ok := responseData["uuid"].(string); ok {
			profileID = uuid
		} else if profileIDVal, ok := responseData["profile_id"].(string); ok {
			profileID = profileIDVal
		} else if profileIDVal, ok := responseData["profileId"].(string); ok {
			profileID = profileIDVal
		} else if profileIDVal, ok := responseData["profile_uuid"].(string); ok {
			profileID = profileIDVal
		} else if name, ok := responseData["name"].(string); ok {
			profileID = name
		}
	}

	if profileID == "" {
		return "", fmt.Errorf("profile ID not found in response. Raw response: %s", string(bodyBytes))
	}

	return profileID, nil
}

// GetByUserID retrieves a user profile by user ID
func (s *UserProfileService) GetByUserID(userID string) (*models.UserProfile, error) {
	return s.userProfileRepo.GetByUserID(userID)
}

// GetByID retrieves a user profile by ID
func (s *UserProfileService) GetByID(id string) (*models.UserProfile, error) {
	return s.userProfileRepo.GetByID(id)
}

// GetAll retrieves all user profiles (admin only)
func (s *UserProfileService) GetAll() ([]*models.UserProfile, error) {
	return s.userProfileRepo.GetAll()
}

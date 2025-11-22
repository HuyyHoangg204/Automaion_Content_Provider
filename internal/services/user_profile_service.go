package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/sirupsen/logrus"
)

type UserProfileService struct {
	userProfileRepo *repository.UserProfileRepository
	appRepo         *repository.AppRepository
}

func NewUserProfileService(userProfileRepo *repository.UserProfileRepository, appRepo *repository.AppRepository) *UserProfileService {
	return &UserProfileService{
		userProfileRepo: userProfileRepo,
		appRepo:         appRepo,
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

// deployProfileToAllMachines deploys profile to all machines with tunnel URLs
func (s *UserProfileService) deployProfileToAllMachines(userProfile *models.UserProfile, profileName string) ([]byte, []byte, error) {
	// Get all apps with tunnel URLs (all machines)
	allApps, err := s.appRepo.GetAll()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get all apps: %w", err)
	}

	var deployedAppIDs []string
	syncStatusMap := make(map[string]interface{})

	// Filter apps with tunnel URLs
	var appsWithTunnel []*models.App
	for _, app := range allApps {
		if app.TunnelURL != nil && *app.TunnelURL != "" {
			appsWithTunnel = append(appsWithTunnel, app)
		}
	}

	if len(appsWithTunnel) == 0 {
		logrus.Info("No apps with tunnel URLs found, skipping deployment")
		deployedAppsJSON, _ := json.Marshal(deployedAppIDs)
		syncStatusJSON, _ := json.Marshal(syncStatusMap)
		return deployedAppsJSON, syncStatusJSON, nil
	}

	// Deploy to each machine
	for _, app := range appsWithTunnel {
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
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Parse response to get profile ID
	var responseData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract profile ID from response (could be "id", "uuid", "profile_id", etc.)
	profileID := ""
	if id, ok := responseData["id"].(string); ok {
		profileID = id
	} else if uuid, ok := responseData["uuid"].(string); ok {
		profileID = uuid
	} else if profileIDVal, ok := responseData["profile_id"].(string); ok {
		profileID = profileIDVal
	}

	if profileID == "" {
		return "", errors.New("profile ID not found in response")
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

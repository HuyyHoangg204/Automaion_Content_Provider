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

// ChromeProfileService manages Chrome profile launching with lock mechanism
type ChromeProfileService struct {
	userProfileRepo *repository.UserProfileRepository
	appRepo         *repository.AppRepository
}

// NewChromeProfileService creates a new ChromeProfileService
func NewChromeProfileService(userProfileRepo *repository.UserProfileRepository, appRepo *repository.AppRepository) *ChromeProfileService {
	return &ChromeProfileService{
		userProfileRepo: userProfileRepo,
		appRepo:         appRepo,
	}
}

// LaunchChromeProfileRequest represents the request to launch Chrome
type LaunchChromeProfileRequest struct {
	UserProfileID string   `json:"user_profile_id"` // Optional: if not provided, will use user's profile
	ExtraArgs     []string `json:"extra_args,omitempty"`
	EnsureGmail   bool     `json:"ensure_gmail,omitempty"`
	EntityType    string   `json:"entity_type,omitempty"` // For logging: "topic", "script", etc.
	EntityID      string   `json:"entity_id,omitempty"`   // For logging: entity UUID
}

// LaunchChromeProfileResponse represents the response for Chrome launch
type LaunchChromeProfileResponse struct {
	Success      bool   `json:"success"`
	AppID        string `json:"app_id"`     // App/Machine ID that launched Chrome
	MachineID    string `json:"machine_id"` // Machine ID
	TunnelURL    string `json:"tunnel_url"` // Tunnel URL of the machine
	Message      string `json:"message"`
	LockAcquired bool   `json:"lock_acquired"` // Whether lock was acquired
}

// ReleaseChromeProfileRequest represents the request to release Chrome lock
type ReleaseChromeProfileRequest struct {
	UserProfileID string `json:"user_profile_id"` // Optional: if not provided, will use user's profile
	Force         bool   `json:"force,omitempty"` // Force release even if not owned by current app
}

// LaunchChromeProfile launches Chrome with profile and acquires lock
// This function can be reused by TopicService and other services
func (s *ChromeProfileService) LaunchChromeProfile(userID string, req *LaunchChromeProfileRequest) (*LaunchChromeProfileResponse, error) {
	// Get user profile
	var userProfile *models.UserProfile
	var err error

	if req.UserProfileID != "" {
		userProfile, err = s.userProfileRepo.GetByID(req.UserProfileID)
	} else {
		userProfile, err = s.userProfileRepo.GetByUserID(userID)
	}

	if err != nil {
		return nil, fmt.Errorf("user profile not found: %w", err)
	}

	// Check if profile is already locked
	if userProfile.CurrentAppID != nil {
		// Check if lock is expired (more than 1 hour)
		if userProfile.LastRunStartedAt != nil {
			timeSinceStart := time.Since(*userProfile.LastRunStartedAt)
			if timeSinceStart > 1*time.Hour {
				// Lock expired, release it
				logrus.Warnf("Lock expired for profile %s, releasing...", userProfile.ID)
				userProfile.CurrentAppID = nil
				userProfile.CurrentMachineID = nil
				userProfile.LastRunStartedAt = nil
			} else {
				// Profile is locked and not expired
				return nil, fmt.Errorf("profile is currently in use by another machine (app_id: %s)", *userProfile.CurrentAppID)
			}
		} else {
			// Lock exists but no start time, clear it
			userProfile.CurrentAppID = nil
			userProfile.CurrentMachineID = nil
		}
	}

	// Get available machines (apps with tunnel URLs)
	allApps, err := s.appRepo.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get apps: %w", err)
	}

	// Filter apps with tunnel URLs and find Automation app
	var automationApps []*models.App
	for _, app := range allApps {
		if app.TunnelURL != nil && *app.TunnelURL != "" {
			// Find Automation app (case-insensitive)
			if strings.ToLower(app.Name) == "automation" {
				automationApps = append(automationApps, app)
			}
		}
	}

	if len(automationApps) == 0 {
		return nil, errors.New("no automation machines available")
	}

	// Select first available machine (can be improved with load balancing)
	selectedApp := automationApps[0]

	// Acquire lock: Set CurrentAppID and LastRunStartedAt
	now := time.Now()
	userProfile.CurrentAppID = &selectedApp.ID
	userProfile.CurrentMachineID = &selectedApp.BoxID
	userProfile.LastRunStartedAt = &now
	userProfile.LastRunEndedAt = nil

	if err := s.userProfileRepo.Update(userProfile); err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	// Launch Chrome on selected machine
	tunnelURL := *selectedApp.TunnelURL
	launchURL := fmt.Sprintf("%s/chrome/profiles/launch", strings.TrimSuffix(tunnelURL, "/"))

	// TODO: Viết tiện ích extract userDataDir path từ profileDirName
	// Tạm thời dùng profileDirName làm userDataDir (sẽ được thay thế bằng tiện ích extract path)
	userDataDir := userProfile.ProfileDirName

	// Prepare request body
	requestBody := map[string]interface{}{
		"name":           userProfile.Name,
		"userDataDir":    userDataDir, // TODO: Thay bằng tiện ích extract path
		"profileDirName": userProfile.ProfileDirName,
		"ensureGmail":    req.EnsureGmail,
	}

	if len(req.ExtraArgs) > 0 {
		requestBody["extraArgs"] = req.ExtraArgs
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		// Release lock on error
		s.releaseLock(userProfile.ID, false)
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequest("POST", launchURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		s.releaseLock(userProfile.ID, false)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "Green-Provider-Services/1.0")

	// Send entity info headers for logging
	if req.EntityType != "" && req.EntityID != "" {
		httpReq.Header.Set("X-Entity-Type", req.EntityType)
		httpReq.Header.Set("X-Entity-ID", req.EntityID)
	}
	httpReq.Header.Set("X-User-ID", userID)

	// Make request
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		logrus.Errorf("HTTP request failed to automation backend %s: %v", launchURL, err)
		s.releaseLock(userProfile.ID, false)
		return nil, fmt.Errorf("failed to launch Chrome: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	responseBodyBytes, _ := io.ReadAll(resp.Body)

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.releaseLock(userProfile.ID, false)
		return nil, fmt.Errorf("chrome launch API returned status %d", resp.StatusCode)
	}

	// Parse response (reuse the body we already read)
	var responseData map[string]interface{}
	if len(responseBodyBytes) > 0 {
		if err := json.Unmarshal(responseBodyBytes, &responseData); err != nil {
			logrus.Warnf("Failed to parse Chrome launch response: %v", err)
		}
	}

	logrus.Infof("Successfully launched Chrome for profile %s on machine %s (app %s)", userProfile.ID, selectedApp.BoxID, selectedApp.ID)

	return &LaunchChromeProfileResponse{
		Success:      true,
		AppID:        selectedApp.ID,
		MachineID:    selectedApp.BoxID,
		TunnelURL:    tunnelURL,
		Message:      "Chrome launched successfully",
		LockAcquired: true,
	}, nil
}

// ReleaseChromeProfile releases the lock on a Chrome profile
func (s *ChromeProfileService) ReleaseChromeProfile(userID string, req *ReleaseChromeProfileRequest) error {
	// Get user profile
	var userProfile *models.UserProfile
	var err error

	if req.UserProfileID != "" {
		userProfile, err = s.userProfileRepo.GetByID(req.UserProfileID)
	} else {
		userProfile, err = s.userProfileRepo.GetByUserID(userID)
	}

	if err != nil {
		return fmt.Errorf("user profile not found: %w", err)
	}

	// Check if profile is locked
	if userProfile.CurrentAppID == nil {
		return errors.New("profile is not locked")
	}

	// If not force, check ownership (optional - can be removed if not needed)
	// For now, we'll allow release if user owns the profile

	// Release lock
	return s.releaseLock(userProfile.ID, req.Force)
}

// releaseLock releases the lock on a profile (internal helper)
func (s *ChromeProfileService) releaseLock(userProfileID string, force bool) error {
	userProfile, err := s.userProfileRepo.GetByID(userProfileID)
	if err != nil {
		return fmt.Errorf("user profile not found: %w", err)
	}

	// Release lock
	now := time.Now()
	userProfile.CurrentAppID = nil
	userProfile.CurrentMachineID = nil
	userProfile.LastRunEndedAt = &now

	if err := s.userProfileRepo.Update(userProfile); err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	logrus.Infof("Lock released for profile %s", userProfileID)
	return nil
}

// CheckProfileLock checks if a profile is currently locked
func (s *ChromeProfileService) CheckProfileLock(userID string, userProfileID string) (bool, *string, error) {
	var userProfile *models.UserProfile
	var err error

	if userProfileID != "" {
		userProfile, err = s.userProfileRepo.GetByID(userProfileID)
	} else {
		userProfile, err = s.userProfileRepo.GetByUserID(userID)
	}

	if err != nil {
		return false, nil, fmt.Errorf("user profile not found: %w", err)
	}

	isLocked := userProfile.CurrentAppID != nil
	var currentAppID *string
	if isLocked {
		currentAppID = userProfile.CurrentAppID
	}

	return isLocked, currentAppID, nil
}

// GetAvailableMachine selects an available machine for Chrome launch
// This can be used for load balancing
func (s *ChromeProfileService) GetAvailableMachine() (*models.App, error) {
	allApps, err := s.appRepo.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get apps: %w", err)
	}

	// Filter Automation apps with tunnel URLs
	var automationApps []*models.App
	for _, app := range allApps {
		if app.TunnelURL != nil && *app.TunnelURL != "" {
			if strings.ToLower(app.Name) == "automation" {
				automationApps = append(automationApps, app)
			}
		}
	}

	if len(automationApps) == 0 {
		return nil, errors.New("no automation machines available")
	}

	// TODO: Implement load balancing logic here
	// For now, return first available
	return automationApps[0], nil
}

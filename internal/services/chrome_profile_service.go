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
	userProfileRepo   *repository.UserProfileRepository
	appRepo           *repository.AppRepository
	boxRepo           *repository.BoxRepository
	geminiAccountRepo *repository.GeminiAccountRepository
	topicRepo         *repository.TopicRepository
}

// NewChromeProfileService creates a new ChromeProfileService
func NewChromeProfileService(userProfileRepo *repository.UserProfileRepository, appRepo *repository.AppRepository, boxRepo *repository.BoxRepository, geminiAccountRepo *repository.GeminiAccountRepository, topicRepo *repository.TopicRepository) *ChromeProfileService {
	return &ChromeProfileService{
		userProfileRepo:   userProfileRepo,
		appRepo:           appRepo,
		boxRepo:           boxRepo,
		geminiAccountRepo: geminiAccountRepo,
		topicRepo:         topicRepo,
	}
}

// LaunchChromeProfileRequest represents the request to launch Chrome
type LaunchChromeProfileRequest struct {
	UserProfileID string   `json:"user_profile_id"` // Optional: if not provided, will use user's profile
	ExtraArgs     []string `json:"extra_args,omitempty"`
	EnsureGmail   bool     `json:"ensure_gmail,omitempty"`
	EntityType    string   `json:"entity_type,omitempty"` // For logging: "topic", "script", etc.
	EntityID      string   `json:"entity_id,omitempty"`   // For logging: entity UUID
	DebugPort     int      `json:"debug_port,omitempty"`  // Chrome debug port - mỗi user 1 port riêng
}

// LaunchChromeProfileResponse represents the response for Chrome launch
type LaunchChromeProfileResponse struct {
	Success      bool   `json:"success"`
	AppID        string `json:"app_id"`     // App/Machine ID that launched Chrome
	MachineID    string `json:"machine_id"` // Machine ID
	TunnelURL    string `json:"tunnel_url"` // Tunnel URL of the machine
	DebugPort    int    `json:"debug_port"` // Chrome debug port (từ automation backend)
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

	// TODO: Tạm thời bỏ logic check lock để cho phép nhiều Chrome instances cùng profile
	// Sau này sẽ implement support multiple Chrome instances cho 1 profile
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
				// Profile is locked and not expired - TẠM THỜI: chỉ log warning, không block
				// Sau này sẽ support multiple Chrome instances cho 1 profile
				logrus.Warnf("Profile %s is currently in use by app %s, but allowing multiple instances (temporary)", userProfile.ID, *userProfile.CurrentAppID)
				// Không return error, cho phép launch tiếp
			}
		} else {
			// Lock exists but no start time, clear it
			userProfile.CurrentAppID = nil
			userProfile.CurrentMachineID = nil
		}
	}

	// Get available machines using load balancing (weighted score)
	// In the new model, machine selection is based on user profile (and per-project gem accounts for scripts),
	// so for topics/gemini we always use profile-based selection.
	var selectedApp *models.App
	selectedApp, err = s.selectBestMachineForProfile(userProfile)

	if err != nil {
		return nil, fmt.Errorf("failed to select machine: %w", err)
	}

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

	// Truyền debugPort để automation backend biết mở Chrome với port nào
	// Mỗi user sẽ có 1 port riêng, tránh conflict khi nhiều user dùng chung profile
	if req.DebugPort > 0 {
		requestBody["debugPort"] = req.DebugPort
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
	var debugPort int
	var responseData map[string]interface{}
	if len(responseBodyBytes) > 0 {
		if err := json.Unmarshal(responseBodyBytes, &responseData); err != nil {
			logrus.Warnf("Failed to parse Chrome launch response: %v", err)
		} else {
			// Extract debugPort từ response
			if dp, ok := responseData["debugPort"].(float64); ok {
				debugPort = int(dp)
				logrus.Infof("Got debugPort from launch response: %d", debugPort)
			}
		}
	}

	logrus.Infof("Successfully launched Chrome for profile %s on machine %s (app %s), debugPort=%d", userProfile.ID, selectedApp.BoxID, selectedApp.ID, debugPort)

	return &LaunchChromeProfileResponse{
		Success:      true,
		AppID:        selectedApp.ID,
		MachineID:    selectedApp.BoxID,
		TunnelURL:    tunnelURL,
		DebugPort:    debugPort,
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

	// Use weighted score load balancing
	return s.selectBestMachine(automationApps, nil)
}

// selectBestMachineForProfile selects the best machine for a specific profile using weighted score
func (s *ChromeProfileService) selectBestMachineForProfile(userProfile *models.UserProfile) (*models.App, error) {
	// Get all automation apps
	allApps, err := s.appRepo.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get apps: %w", err)
	}

	// Filter online automation apps with tunnel URLs
	var candidateApps []*models.App
	for _, app := range allApps {
		if app.TunnelURL == nil || *app.TunnelURL == "" {
			continue
		}
		if strings.ToLower(app.Name) != "automation" {
			continue
		}

		// Get box to check online status
		box, err := s.boxRepo.GetByID(app.BoxID)
		if err != nil {
			continue
		}

		// Only consider online machines
		if !box.IsOnline {
			continue
		}

		// Check if profile is deployed on this machine
		deployedMachines := userProfile.DeployedMachines
		if len(deployedMachines) > 0 {
			appIDStr := app.ID
			found := false

			// DeployedMachines is a JSON map, check if app ID exists in values
			for _, value := range deployedMachines {
				// Value could be a string or an array of strings
				if strValue, ok := value.(string); ok && strValue == appIDStr {
					found = true
					break
				}
				// If value is an array, check each element
				if arrValue, ok := value.([]interface{}); ok {
					for _, item := range arrValue {
						if itemStr, ok := item.(string); ok && itemStr == appIDStr {
							found = true
							break
						}
					}
					if found {
						break
					}
				}
			}

			if !found {
				continue // Profile not deployed on this machine
			}
		}

		candidateApps = append(candidateApps, app)
	}

	if len(candidateApps) == 0 {
		return nil, errors.New("no online automation machines available with profile deployed")
	}

	// Select best machine using weighted score
	return s.selectBestMachine(candidateApps, userProfile)
}

// selectBestMachineForTopic is deprecated in the new model (topics no longer bind to a single Gemini account/gem).
// Machine selection is handled per user profile (and per project for scripts).

// selectBestMachine selects machine with lowest weighted score
// Weighted Score = RunningProfiles * 10 + (CPUUsage / 10) - (MemoryFreeGB * 2)
func (s *ChromeProfileService) selectBestMachine(apps []*models.App, userProfile *models.UserProfile) (*models.App, error) {
	if len(apps) == 0 {
		return nil, errors.New("no machines available")
	}

	type machineScore struct {
		app   *models.App
		box   *models.Box
		score float64
	}

	var scoredMachines []machineScore

	// Calculate score for each machine
	for _, app := range apps {
		// Get box to access system metrics
		box, err := s.boxRepo.GetByID(app.BoxID)
		if err != nil {
			logrus.Warnf("Failed to get box %s for app %s: %v", app.BoxID, app.ID, err)
			continue
		}

		// Calculate weighted score
		// Score = RunningProfiles * 10 + (CPUUsage / 10) - (MemoryFreeGB * 2)
		score := float64(box.RunningProfiles) * 10.0

		if box.CPUUsage != nil {
			score += *box.CPUUsage / 10.0
		}

		if box.MemoryFreeGB != nil {
			score -= *box.MemoryFreeGB * 2.0
		}

		scoredMachines = append(scoredMachines, machineScore{
			app:   app,
			box:   box,
			score: score,
		})
	}

	if len(scoredMachines) == 0 {
		return nil, errors.New("no machines with valid metrics available")
	}

	// Find machine with lowest score (least loaded)
	bestMachine := scoredMachines[0]
	for _, machine := range scoredMachines[1:] {
		if machine.score < bestMachine.score {
			bestMachine = machine
		}
	}

	logrus.Infof("Selected machine %s (box %s) with score %.2f (RunningProfiles: %d, CPU: %.2f%%, Memory: %.2fGB)",
		bestMachine.app.ID, bestMachine.box.ID, bestMachine.score,
		bestMachine.box.RunningProfiles,
		getFloat64Value(bestMachine.box.CPUUsage),
		getFloat64Value(bestMachine.box.MemoryFreeGB))

	return bestMachine.app, nil
}

// getFloat64Value safely gets float64 value from pointer
func getFloat64Value(ptr *float64) float64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

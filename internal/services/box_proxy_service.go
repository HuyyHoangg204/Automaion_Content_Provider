package services

import (
	"fmt"
	"strings"

	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
)

type BoxProxyService struct {
	appRepo  *repository.AppRepository
	boxRepo  *repository.BoxRepository
	userRepo *repository.UserRepository
}

func NewBoxProxyService(appRepo *repository.AppRepository, boxRepo *repository.BoxRepository, userRepo *repository.UserRepository) *BoxProxyService {
	return &BoxProxyService{
		appRepo:  appRepo,
		boxRepo:  boxRepo,
		userRepo: userRepo,
	}
}

// ValidateBoxProxyRequest validates the box proxy request
func (s *BoxProxyService) ValidateBoxProxyRequest(userID, boxID, appID string) (*models.Box, *models.App, error) {
	// Check if user owns the box
	box, err := s.boxRepo.GetByUserIDAndID(userID, boxID)
	if err != nil {
		return nil, nil, fmt.Errorf("box not found or access denied")
	}

	// Get app details
	app, err := s.appRepo.GetByID(appID)
	if err != nil {
		return nil, nil, fmt.Errorf("app not found")
	}

	// Verify app belongs to the box
	if app.BoxID != boxID {
		return nil, nil, fmt.Errorf("app does not belong to the specified box")
	}

	// Check if tunnel URL is available
	if app.TunnelURL == nil || *app.TunnelURL == "" {
		return nil, nil, fmt.Errorf("tunnel URL not configured for this app")
	}

	return box, app, nil
}

// GetPlatformType determines the platform type from app name
func (s *BoxProxyService) GetPlatformType(appName string) string {
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

// BuildTargetURL builds the target URL for the platform
func (s *BoxProxyService) BuildTargetURL(baseURL, platformType, platformPath string) string {
	// Remove trailing slash from base URL if present
	baseURL = strings.TrimSuffix(baseURL, "/")

	// Add platform-specific base path if needed
	switch platformType {
	case "hidemium":
		// Hidemium typically uses root path
		return fmt.Sprintf("%s/%s", baseURL, platformPath)
	case "genlogin":
		// GenLogin might have different base path
		return fmt.Sprintf("%s/%s", baseURL, platformPath)
	default:
		return fmt.Sprintf("%s/%s", baseURL, platformPath)
	}
}

// GetPlatformConfig returns platform-specific configuration
func (s *BoxProxyService) GetPlatformConfig(platformType string) map[string]interface{} {
	switch platformType {
	case "hidemium":
		return map[string]interface{}{
			"name":      "Hidemium",
			"version":   "v4",
			"base_path": "/",
		}
	case "genlogin":
		return map[string]interface{}{
			"name":      "GenLogin",
			"version":   "latest",
			"base_path": "/",
		}
	default:
		return map[string]interface{}{
			"name":      "Unknown",
			"version":   "unknown",
			"base_path": "/",
		}
	}
}

// ValidatePlatformPath validates if the platform path is allowed
func (s *BoxProxyService) ValidatePlatformPath(platformType, platformPath string) bool {
	// Trim leading slash from platformPath for consistent comparison
	platformPath = strings.TrimPrefix(platformPath, "/")

	// Define allowed paths for each platform
	allowedPaths := map[string][]string{
		"hidemium": {
			"v1/browser/list",
			"v2/default-config",
			"v2/status-profile",
			"v2/tag",
			"v2/browser/get-list-version",
			"v2/browser/get-profile-by-uuid",
			"v1/folder/list",
			"create-profile-by-default",
			"create-profile-custom",
			"v2/browser/change-fingerprint",
			"v2/browser/update-note",
			"v2/browser/update-once",
			"v2/tag",
			"v2/status-profile/change-status",
			"v1/browser/destroy",
			"v1/folder/add-browser",
			"v2/proxy/quick-edit",
			"v2/browser/proxy/update",
			"automation/campaign",
			"automation/schedule",
			"automation/campaign/save-campaign-profile",
			"automation/campaign/save-auto-campaign",
			"automation/delete-campaign",
			"automation/campaign/delete-all-campaign-profile",
			"user-settings/token",
		},
		"genlogin": {
			"profiles",
			"profiles/create",
			"profiles/update",
			"profiles/delete",
			"profiles/list",
			"configs",
			"campaigns",
			"automation",
		},
	}

	allowedPathsForPlatform, exists := allowedPaths[platformType]
	if !exists {
		return false
	}

	// Check if the path starts with any allowed path
	for _, allowedPath := range allowedPathsForPlatform {
		if strings.HasPrefix(platformPath, allowedPath) {
			return true
		}
	}

	return false
}

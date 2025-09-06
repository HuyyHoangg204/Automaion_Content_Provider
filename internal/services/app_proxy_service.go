package services

import (
	"errors"
	"fmt"
	"strings"

	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
)

type AppProxyService struct {
	appRepo  *repository.AppRepository
	boxRepo  *repository.BoxRepository
	userRepo *repository.UserRepository
}

func NewAppProxyService(appRepo *repository.AppRepository, boxRepo *repository.BoxRepository, userRepo *repository.UserRepository) *AppProxyService {
	return &AppProxyService{
		appRepo:  appRepo,
		boxRepo:  boxRepo,
		userRepo: userRepo,
	}
}

// ValidateAppProxyRequest validates if user can access the app directly
func (s *AppProxyService) ValidateAppProxyRequest(userID, appID string) (*models.App, error) {
	// Get app by ID
	app, err := s.appRepo.GetByID(appID)
	if err != nil {
		return nil, errors.New("app not found")
	}

	// Get box that contains this app
	box, err := s.boxRepo.GetByID(app.BoxID)
	if err != nil {
		return nil, errors.New("box not found")
	}

	// Check if user owns the box
	if box.UserID != userID {
		return nil, errors.New("access denied: app does not belong to user")
	}

	// Check if tunnel URL is available
	if app.TunnelURL == nil || *app.TunnelURL == "" {
		return nil, errors.New("tunnel URL not configured for app")
	}

	return app, nil
}

// GetPlatformType determines the platform type from app name
func (s *AppProxyService) GetPlatformType(appName string) string {
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
func (s *AppProxyService) BuildTargetURL(baseURL, platformType, platformPath string) string {
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
func (s *AppProxyService) GetPlatformConfig(platformType string) map[string]interface{} {
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

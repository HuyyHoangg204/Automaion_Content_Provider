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

// BuildTargetURL builds the target URL for the platform
func (s *AppProxyService) BuildTargetURL(baseURL, appName, platformPath string) string {
	// Remove trailing slash from base URL if present
	baseURL = strings.TrimSuffix(baseURL, "/")

	// Add platform-specific base path if needed
	switch appName {
	case "Hidemium":
		// Hidemium typically uses root path
		return fmt.Sprintf("%s/%s", baseURL, platformPath)
	case "Genlogin":
		// GenLogin might have different base path
		return fmt.Sprintf("%s/%s", baseURL, platformPath)
	default:
		return fmt.Sprintf("%s/%s", baseURL, platformPath)
	}
}

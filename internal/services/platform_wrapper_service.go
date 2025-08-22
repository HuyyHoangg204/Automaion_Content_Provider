package services

import (
	"context"
	"fmt"

	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/onegreenvn/green-provider-services-backend/internal/services/service_platform/genlogin"
	"github.com/onegreenvn/green-provider-services-backend/internal/services/service_platform/hidemium"
)

// PlatformWrapperService wraps platform-specific services (simplified)
type PlatformWrapperService struct {
	appRepo repository.AppRepository
}

// NewPlatformWrapperService creates a new platform wrapper service
func NewPlatformWrapperService(appRepo repository.AppRepository) *PlatformWrapperService {
	return &PlatformWrapperService{
		appRepo: appRepo,
	}
}

// CreateProfileOnPlatform creates a profile on a specific platform
func (pws *PlatformWrapperService) CreateProfileOnPlatform(ctx context.Context, platformType string, appID string, profileData *models.CreateProfileRequest) (*models.ProfileResponse, error) {
	platform, err := pws.getProfilePlatform(platformType)
	if err != nil {
		return nil, err
	}

	return platform.CreateProfile(appID, profileData)
}

// UpdateProfileOnPlatform updates a profile on a specific platform
func (pws *PlatformWrapperService) UpdateProfileOnPlatform(ctx context.Context, platformType string, profileID string, profileData *models.UpdateProfileRequest) (*models.ProfileResponse, error) {
	platform, err := pws.getProfilePlatform(platformType)
	if err != nil {
		return nil, err
	}

	return platform.UpdateProfile(profileID, profileData)
}

// DeleteProfileOnPlatform deletes a profile on a specific platform
func (pws *PlatformWrapperService) DeleteProfileOnPlatform(ctx context.Context, platformType string, appID string, profile *models.Profile, machineID string) error {
	platform, err := pws.getProfilePlatform(platformType)
	if err != nil {
		return err
	}

	return platform.DeleteProfile(appID, profile, machineID)
}

// SyncProfilesFromPlatform syncs profiles from a specific platform
func (pws *PlatformWrapperService) SyncProfilesFromPlatform(ctx context.Context, platformType string, appID string, boxID string, machineID string) ([]models.HidemiumProfile, error) {
	platform, err := pws.getProfilePlatform(platformType)
	if err != nil {
		return nil, err
	}

	return platform.SyncProfilesFromPlatform(appID, boxID, machineID)
}

// Box Platform Operations

// SyncProfilesFromPlatformForBox syncs profiles from a specific platform for a box
func (pws *PlatformWrapperService) SyncProfilesFromPlatformForBox(ctx context.Context, platformType string, appID string, boxID string, machineID string) ([]models.HidemiumProfile, error) {
	platform, err := pws.getBoxPlatform(platformType)
	if err != nil {
		return nil, err
	}

	return platform.SyncBoxProfilesFromPlatform(appID, boxID, machineID)
}

// GetDefaultConfigsFromPlatform retrieves default configurations from a specific platform
func (pws *PlatformWrapperService) GetDefaultConfigsFromPlatform(ctx context.Context, platformType string, appID string, machineID string, page, limit int) (map[string]interface{}, error) {
	platform, err := pws.getProfilePlatform(platformType)
	if err != nil {
		return nil, err
	}

	// Type assertion to get the GetDefaultConfigs method
	if profilePlatform, ok := platform.(interface {
		GetDefaultConfigs(appID string, machineID string, page, limit int) (map[string]interface{}, error)
	}); ok {
		return profilePlatform.GetDefaultConfigs(appID, machineID, page, limit)
	}

	return nil, fmt.Errorf("platform %s does not support GetDefaultConfigs", platformType)
}

// getProfilePlatform gets a profile platform instance (simplified)
func (pws *PlatformWrapperService) getProfilePlatform(platformType string) (interface {
	CreateProfile(appID string, profileData *models.CreateProfileRequest) (*models.ProfileResponse, error)
	UpdateProfile(profileID string, profileData *models.UpdateProfileRequest) (*models.ProfileResponse, error)
	DeleteProfile(appID string, profile *models.Profile, machineID string) error
	SyncProfilesFromPlatform(appID string, boxID string, machineID string) ([]models.HidemiumProfile, error)
	GetDefaultConfigs(appID string, machineID string, page, limit int) (map[string]interface{}, error)
}, error) {
	switch platformType {
	case "hidemium":
		return hidemium.NewProfileService(pws.appRepo), nil
	case "genlogin":
		return genlogin.NewProfileService(pws.appRepo), nil
	default:
		return nil, fmt.Errorf("unsupported profile platform type: %s", platformType)
	}
}

// getBoxPlatform gets a box platform instance (simplified)
func (pws *PlatformWrapperService) getBoxPlatform(platformType string) (interface {
	SyncBoxProfilesFromPlatform(appID string, boxID string, machineID string) ([]models.HidemiumProfile, error)
}, error) {
	switch platformType {
	case "hidemium":
		return hidemium.NewBoxService(pws.appRepo), nil
	case "genlogin":
		return genlogin.NewBoxService(), nil
	default:
		return nil, fmt.Errorf("unsupported box platform type: %s", platformType)
	}
}

// GetSupportedProfilePlatforms returns list of supported profile platforms
func (pws *PlatformWrapperService) GetSupportedProfilePlatforms() []string {
	return []string{"hidemium", "genlogin"}
}

// GetSupportedBoxPlatforms returns list of supported box platforms
func (pws *PlatformWrapperService) GetSupportedBoxPlatforms() []string {
	return []string{"hidemium", "genlogin"}
}

// ValidateProfilePlatform validates if a profile platform type is supported
func (pws *PlatformWrapperService) ValidateProfilePlatform(platformType string) bool {
	supported := pws.GetSupportedProfilePlatforms()
	for _, p := range supported {
		if p == platformType {
			return true
		}
	}
	return false
}

// ValidateBoxPlatform validates if a box platform type is supported
func (pws *PlatformWrapperService) ValidateBoxPlatform(platformType string) bool {
	supported := pws.GetSupportedBoxPlatforms()
	for _, p := range supported {
		if p == platformType {
			return true
		}
	}
	return false
}

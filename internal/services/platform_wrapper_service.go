package services

import (
	"context"
	"fmt"

	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/onegreenvn/green-provider-services-backend/internal/services/service_platform"
	"github.com/onegreenvn/green-provider-services-backend/internal/services/service_platform/genlogin"
	"github.com/onegreenvn/green-provider-services-backend/internal/services/service_platform/hidemium"
)

// PlatformWrapperService wraps platform-specific services
type PlatformWrapperService struct {
	profilePlatformFactory service_platform.ProfilePlatformFactory
	boxPlatformFactory     service_platform.BoxPlatformFactory
}

// NewPlatformWrapperService creates a new platform wrapper service
func NewPlatformWrapperService() *PlatformWrapperService {
	return &PlatformWrapperService{
		profilePlatformFactory: service_platform.NewProfilePlatformFactory(),
		boxPlatformFactory:     service_platform.NewBoxPlatformFactory(),
	}
}

// Profile Platform Operations

// CreateProfileOnPlatform creates a profile on a specific platform
func (pws *PlatformWrapperService) CreateProfileOnPlatform(ctx context.Context, platformType string, profileData *models.CreateProfileRequest) (*models.ProfileResponse, error) {
	platform, err := pws.getProfilePlatform(platformType)
	if err != nil {
		return nil, err
	}

	return platform.CreateProfile(ctx, profileData)
}

// UpdateProfileOnPlatform updates a profile on a specific platform
func (pws *PlatformWrapperService) UpdateProfileOnPlatform(ctx context.Context, platformType string, profileID string, profileData *models.UpdateProfileRequest) (*models.ProfileResponse, error) {
	platform, err := pws.getProfilePlatform(platformType)
	if err != nil {
		return nil, err
	}

	return platform.UpdateProfile(ctx, profileID, profileData)
}

// DeleteProfileOnPlatform deletes a profile on a specific platform
func (pws *PlatformWrapperService) DeleteProfileOnPlatform(ctx context.Context, platformType string, profile *models.Profile, machineID string) error {
	platform, err := pws.getProfilePlatform(platformType)
	if err != nil {
		return err
	}

	return platform.DeleteProfile(ctx, profile, machineID)
}

// SyncProfilesFromPlatform syncs profiles from a specific platform
func (pws *PlatformWrapperService) SyncProfilesFromPlatform(ctx context.Context, platformType string, boxID string, machineID string) ([]models.HidemiumProfile, error) {
	platform, err := pws.getProfilePlatform(platformType)
	if err != nil {
		return nil, err
	}

	return platform.SyncProfilesFromPlatform(ctx, boxID, machineID)
}

// Box Platform Operations

// SyncBoxProfilesFromPlatform syncs profiles from a specific platform for a box
func (pws *PlatformWrapperService) SyncBoxProfilesFromPlatform(ctx context.Context, platformType string, boxID string, machineID string) (*models.SyncBoxProfilesResponse, error) {
	platform, err := pws.getBoxPlatform(platformType)
	if err != nil {
		return nil, err
	}

	return platform.SyncBoxProfilesFromPlatform(ctx, boxID, machineID)
}

// getProfilePlatform gets a profile platform instance
func (pws *PlatformWrapperService) getProfilePlatform(platformType string) (service_platform.ProfilePlatformInterface, error) {
	switch platformType {
	case "hidemium":
		return hidemium.NewProfileService(), nil
	case "genlogin":
		return genlogin.NewProfileService(), nil
	default:
		return nil, fmt.Errorf("unsupported profile platform type: %s", platformType)
	}
}

// getBoxPlatform gets a box platform instance
func (pws *PlatformWrapperService) getBoxPlatform(platformType string) (service_platform.BoxPlatformInterface, error) {
	switch platformType {
	case "hidemium":
		return hidemium.NewBoxService(), nil
	case "genlogin":
		return genlogin.NewBoxService(), nil
	default:
		return nil, fmt.Errorf("unsupported box platform type: %s", platformType)
	}
}

// GetSupportedProfilePlatforms returns list of supported profile platforms
func (pws *PlatformWrapperService) GetSupportedProfilePlatforms() []string {
	return pws.profilePlatformFactory.GetSupportedPlatforms()
}

// GetSupportedBoxPlatforms returns list of supported box platforms
func (pws *PlatformWrapperService) GetSupportedBoxPlatforms() []string {
	return pws.boxPlatformFactory.GetSupportedPlatforms()
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

package service_platform

import (
	"context"

	"github.com/onegreenvn/green-provider-services-backend/internal/models"
)

// ProfilePlatformInterface định nghĩa các method cần thiết cho mọi nền tảng profile
type ProfilePlatformInterface interface {
	// Profile operations
	CreateProfile(ctx context.Context, profileData *models.CreateProfileRequest) (*models.ProfileResponse, error)
	UpdateProfile(ctx context.Context, profileID string, profileData *models.UpdateProfileRequest) (*models.ProfileResponse, error)
	DeleteProfile(ctx context.Context, profile *models.Profile, machineID string) error
	GetProfile(ctx context.Context, profileID string) (*models.ProfileResponse, error)
	ListProfiles(ctx context.Context, filters map[string]interface{}) ([]*models.ProfileResponse, error)

	// Profile sync operations
	SyncProfilesFromPlatform(ctx context.Context, boxID string, machineID string) ([]models.HidemiumProfile, error)

	// Platform info
	GetPlatformName() string
	GetPlatformVersion() string
	ValidateProfileData(profileData *models.CreateProfileRequest) error
}

// ProfilePlatformFactory tạo instance của profile platform
type ProfilePlatformFactory interface {
	CreateProfilePlatform(platformType string) (ProfilePlatformInterface, error)
	GetSupportedPlatforms() []string
}

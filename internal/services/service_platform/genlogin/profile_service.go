package genlogin

import (
	"context"
	"fmt"

	"github.com/onegreenvn/green-provider-services-backend/internal/models"
)

// ProfileService implements ProfilePlatformInterface for Genlogin
type ProfileService struct{}

// NewProfileService creates a new Genlogin profile service
func NewProfileService() *ProfileService {
	return &ProfileService{}
}

// CreateProfile creates a new profile on Genlogin platform
func (s *ProfileService) CreateProfile(ctx context.Context, profileData *models.CreateProfileRequest) (*models.ProfileResponse, error) {
	return nil, fmt.Errorf("profile creation on Genlogin not implemented yet")
}

// UpdateProfile updates an existing profile on Genlogin platform
func (s *ProfileService) UpdateProfile(ctx context.Context, profileID string, profileData *models.UpdateProfileRequest) (*models.ProfileResponse, error) {
	return nil, fmt.Errorf("profile update on Genlogin not implemented yet")
}

// DeleteProfile deletes a profile on Genlogin platform
func (s *ProfileService) DeleteProfile(ctx context.Context, profile *models.Profile, machineID string) error {
	return fmt.Errorf("profile deletion on Genlogin not implemented yet")
}

// GetProfile retrieves profile information from Genlogin platform
func (s *ProfileService) GetProfile(ctx context.Context, profileID string) (*models.ProfileResponse, error) {
	return nil, fmt.Errorf("profile retrieval on Genlogin not implemented yet")
}

// ListProfiles lists all profiles from Genlogin platform
func (s *ProfileService) ListProfiles(ctx context.Context, filters map[string]interface{}) ([]*models.ProfileResponse, error) {
	return nil, fmt.Errorf("profile listing on Genlogin not implemented yet")
}

// SyncProfilesFromPlatform syncs profiles from Genlogin platform
func (s *ProfileService) SyncProfilesFromPlatform(ctx context.Context, boxID string, machineID string) ([]models.HidemiumProfile, error) {
	return nil, fmt.Errorf("profile sync from Genlogin not implemented yet")
}

// GetPlatformName returns the platform name
func (s *ProfileService) GetPlatformName() string {
	return "genlogin"
}

// GetPlatformVersion returns the platform version
func (s *ProfileService) GetPlatformVersion() string {
	return "0.0"
}

// ValidateProfileData validates profile data for Genlogin platform
func (s *ProfileService) ValidateProfileData(profileData *models.CreateProfileRequest) error {
	return fmt.Errorf("profile validation on Genlogin not implemented yet")
}

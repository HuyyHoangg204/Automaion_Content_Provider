package services

import (
	"fmt"

	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/onegreenvn/green-provider-services-backend/internal/utils"
)

type ProfileSyncService struct {
	profileRepo *repository.ProfileRepository
	appRepo     *repository.AppRepository
}

func NewProfileSyncService(profileRepo *repository.ProfileRepository, appRepo *repository.AppRepository) *ProfileSyncService {
	return &ProfileSyncService{
		profileRepo: profileRepo,
		appRepo:     appRepo,
	}
}

// ProcessSyncedProfiles processes synced profiles and returns response
func (s *ProfileSyncService) ProcessSyncedProfiles(appID string, profiles []map[string]interface{}) (*models.SyncBoxProfilesResponse, error) {
	result := &models.SyncBoxProfilesResponse{
		ProfilesSynced: len(profiles),
		Message:        "Profiles synced successfully",
	}

	// Get existing profiles for this app
	existingProfiles, err := s.profileRepo.GetByAppID(appID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing profiles: %w", err)
	}

	existingProfilesMap := make(map[string]*models.Profile)
	for _, profile := range existingProfiles {
		if uuid := utils.ExtractUUID(profile); uuid != "" {
			existingProfilesMap[uuid] = profile
		}
	}

	for _, platformProfile := range profiles {
		if err := s.processPlatformProfile(appID, platformProfile, existingProfilesMap, result); err != nil {
			continue
		}
	}

	s.MarkDeletedProfiles(existingProfilesMap, result)

	// Set message based on results
	if result.ProfilesCreated > 0 || result.ProfilesUpdated > 0 || result.ProfilesDeleted > 0 {
		result.Message = fmt.Sprintf("Sync completed: %d created, %d updated, %d deleted",
			result.ProfilesCreated, result.ProfilesUpdated, result.ProfilesDeleted)
	} else {
		result.Message = "No changes detected during sync"
	}
	return result, nil
}

// processPlatformProfile processes a single platform profile
func (s *ProfileSyncService) processPlatformProfile(appID string, platformProfile map[string]interface{}, existingProfilesMap map[string]*models.Profile, result *models.SyncBoxProfilesResponse) error {
	uuid := utils.ExtractUUIDFromPlatformProfile(platformProfile)
	if uuid == "" {
		return fmt.Errorf("platform profile has no UUID")
	}

	if existingProfile, exists := existingProfilesMap[uuid]; exists {
		// Update existing profile
		if err := s.UpdateExistingProfile(existingProfile, platformProfile); err != nil {
			return fmt.Errorf("failed to update profile: %w", err)
		}
		result.ProfilesUpdated++
		delete(existingProfilesMap, uuid)
	} else {
		// Create new profile using AppService method
		if err := s.CreateNewProfile(appID, platformProfile); err != nil {
			return fmt.Errorf("failed to create profile: %w", err)
		}
		result.ProfilesCreated++
	}
	return nil
}

// CreateNewProfile creates a new profile using provided repo
func (s *ProfileSyncService) CreateNewProfile(appID string, platformProfile map[string]interface{}) error {
	profileName, ok := platformProfile["name"].(string)
	if !ok {
		profileName = "Unknown Profile"
	}

	profile := &models.Profile{
		AppID: appID,
		Name:  profileName,
		Data:  platformProfile,
	}

	// Verify app exists before creating profile
	if _, err := s.appRepo.GetByID(appID); err != nil {
		return fmt.Errorf("app not found: %w", err)
	}

	// Try to create profile
	if err := s.profileRepo.Create(profile); err != nil {
		return fmt.Errorf("failed to create profile in database: %w", err)
	}

	return nil
}

// UpdateExistingProfile updates an existing profile
func (s *ProfileSyncService) UpdateExistingProfile(existingProfile *models.Profile, platformProfile map[string]interface{}) error {
	if existingProfile.Data == nil {
		existingProfile.Data = make(map[string]interface{})
	}

	// Update profile data
	for key, value := range platformProfile {
		existingProfile.Data[key] = value
	}

	// Update profile name if available
	if name, exists := platformProfile["name"]; exists {
		if nameStr, ok := name.(string); ok && nameStr != existingProfile.Name {
			existingProfile.Name = nameStr
		}
	}

	// Save updated profile to database
	if err := s.profileRepo.Update(existingProfile); err != nil {
		return fmt.Errorf("failed to update profile in database: %w", err)
	}

	return nil
}

// MarkDeletedProfiles clears associations and deletes missing profiles
func (s *ProfileSyncService) MarkDeletedProfiles(existingProfilesMap map[string]*models.Profile, result *models.SyncBoxProfilesResponse) {
	for _, profile := range existingProfilesMap {
		if err := s.profileRepo.ClearCampaignAssociations(profile.ID); err != nil {
			fmt.Printf("Warning: Failed to clear campaign associations for profile %s: %v\n", profile.Name, err)
			continue
		}
		if err := s.profileRepo.Delete(profile.ID); err != nil {
			fmt.Printf("Warning: Failed to delete profile %s: %v\n", profile.Name, err)
			continue
		}
		result.ProfilesDeleted++
	}
}

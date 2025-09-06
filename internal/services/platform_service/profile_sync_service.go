package platform_service

import (
	"fmt"

	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
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
func (s *ProfileSyncService) UpdateProfilesAfterSync(app *models.App, profiles []map[string]interface{}) (*models.SyncBoxProfilesResponse, error) {
	result := &models.SyncBoxProfilesResponse{
		ProfilesSynced: len(profiles),
		Message:        "Profiles synced successfully",
	}

	existingProfiles, err := s.profileRepo.GetByAppID(app.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing profiles: %w", err)
	}

	existingProfilesMap := make(map[string]*models.Profile)
	for _, prof := range existingProfiles {
		uuid, ok := prof.Data["uuid"].(string)
		if !ok || uuid == "" {
			continue
		}
		existingProfilesMap[uuid] = prof
	}

	profilesToUpdate := []*models.Profile{}
	profilesToCreate := []map[string]interface{}{}
	profilesToDelete := []*models.Profile{}
	seenProfiles := make(map[string]bool)

	// Identify profiles to create or update
	for _, platformProfile := range profiles {
		uuid, ok := platformProfile["uuid"].(string)
		if !ok || uuid == "" {
			continue
		}
		seenProfiles[uuid] = true

		if existingProfile, exists := existingProfilesMap[uuid]; exists {
			// Prepare for update
			existingProfile.Data = platformProfile
			existingProfile.Name = platformProfile["name"].(string)
			profilesToUpdate = append(profilesToUpdate, existingProfile)
		} else {
			// Prepare for creation
			profilesToCreate = append(profilesToCreate, platformProfile)
		}
	}

	// Identify profiles to delete
	for uuid, profile := range existingProfilesMap {
		if !seenProfiles[uuid] {
			profilesToDelete = append(profilesToDelete, profile)
		}
	}

	// Bulk update profiles
	if len(profilesToUpdate) > 0 {
		if err := s.profileRepo.UpdateMany(profilesToUpdate); err != nil {
			fmt.Printf("Warning: Failed to bulk update profiles: %v\n", err)
		}
		result.ProfilesUpdated = len(profilesToUpdate)
	}

	// Bulk create new profiles
	if len(profilesToCreate) > 0 {
		newProfiles := make([]*models.Profile, 0, len(profilesToCreate))
		for _, platformProfile := range profilesToCreate {
			newProfiles = append(newProfiles, &models.Profile{
				AppID: app.ID,
				Name:  platformProfile["name"].(string),
				Data:  platformProfile,
			})
		}
		if err := s.profileRepo.CreateMany(newProfiles); err != nil {
			fmt.Printf("Warning: Failed to bulk create profiles: %v\n", err)
		} else {
			result.ProfilesCreated = len(newProfiles)
		}
	}

	// Delete any profiles that were not in the synced list
	if len(profilesToDelete) > 0 {
		if err := s.profileRepo.DeleteMany(profilesToDelete); err != nil {
			fmt.Printf("Warning: Failed to delete profiles: %v\n", err)
		} else {
			result.ProfilesDeleted = len(profilesToDelete)
		}
	}

	if result.ProfilesCreated > 0 || result.ProfilesUpdated > 0 || result.ProfilesDeleted > 0 {
		result.Message = fmt.Sprintf("Sync completed: %d created, %d updated, %d deleted",
			result.ProfilesCreated, result.ProfilesUpdated, result.ProfilesDeleted)
	} else {
		result.Message = "No changes detected during sync"
	}
	return result, nil
}

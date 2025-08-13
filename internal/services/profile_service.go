package services

import (
	"errors"
	"fmt"
	"time"

	"green-anti-detect-browser-backend-v1/internal/database/repository"
	"green-anti-detect-browser-backend-v1/internal/models"
)

type ProfileService struct {
	profileRepo *repository.ProfileRepository
	appRepo     *repository.AppRepository
	userRepo    *repository.UserRepository
	boxRepo     *repository.BoxRepository
}

func NewProfileService(profileRepo *repository.ProfileRepository, appRepo *repository.AppRepository, userRepo *repository.UserRepository, boxRepo *repository.BoxRepository) *ProfileService {
	return &ProfileService{
		profileRepo: profileRepo,
		appRepo:     appRepo,
		userRepo:    userRepo,
		boxRepo:     boxRepo,
	}
}

// CreateProfile creates a new profile for a user
func (s *ProfileService) CreateProfile(userID string, req *models.CreateProfileRequest) (*models.ProfileResponse, error) {
	// Verify user exists
	_, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Verify app exists and belongs to user
	_, err = s.appRepo.GetByUserIDAndID(userID, req.AppID)
	if err != nil {
		return nil, errors.New("app not found or access denied")
	}

	// Validate data field is not empty
	if len(req.Data) == 0 {
		return nil, errors.New("profile data is required")
	}

	// Check if profile name already exists in this app
	exists, err := s.profileRepo.CheckNameExistsInApp(req.AppID, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check profile name: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("profile with name '%s' already exists in this app", req.Name)
	}

	// Create profile
	profile := &models.Profile{
		AppID: req.AppID,
		Name:  req.Name,
		Data:  req.Data,
	}

	if err := s.profileRepo.Create(profile); err != nil {
		return nil, fmt.Errorf("failed to create profile: %w", err)
	}

	return s.toResponse(profile), nil
}

// GetProfilesByUser retrieves all profiles for a specific user
func (s *ProfileService) GetProfilesByUser(userID string) ([]*models.ProfileResponse, error) {
	profiles, err := s.profileRepo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get profiles: %w", err)
	}

	responses := make([]*models.ProfileResponse, len(profiles))
	for i, profile := range profiles {
		responses[i] = s.toResponse(profile)
	}

	return responses, nil
}

// GetProfilesByBox retrieves all profiles for a specific box (user must own the box)
func (s *ProfileService) GetProfilesByBox(userID, boxID string) ([]*models.ProfileResponse, error) {
	// Verify box belongs to user
	_, err := s.boxRepo.GetByUserIDAndID(userID, boxID)
	if err != nil {
		return nil, errors.New("box not found or access denied")
	}

	profiles, err := s.profileRepo.GetByBoxID(boxID)
	if err != nil {
		return nil, fmt.Errorf("failed to get profiles: %w", err)
	}

	responses := make([]*models.ProfileResponse, len(profiles))
	for i, profile := range profiles {
		responses[i] = s.toResponse(profile)
	}

	return responses, nil
}

// GetProfilesByBoxPaginated retrieves paginated profiles for a specific box
func (s *ProfileService) GetProfilesByBoxPaginated(userID, boxID string, page, pageSize int) (*models.PaginatedProfileResponse, error) {
	// Verify box belongs to user
	_, err := s.boxRepo.GetByUserIDAndID(userID, boxID)
	if err != nil {
		return nil, errors.New("box not found or access denied")
	}

	// Validate pagination parameters
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// If pageSize is very large, get all profiles
	if pageSize >= 1000 {
		profiles, err := s.profileRepo.GetByBoxID(boxID)
		if err != nil {
			return nil, fmt.Errorf("failed to get profiles: %w", err)
		}

		responses := make([]*models.ProfileResponse, len(profiles))
		for i, profile := range profiles {
			responses[i] = s.toResponse(profile)
		}

		return &models.PaginatedProfileResponse{
			Profiles:    responses,
			Total:       len(profiles),
			Page:        1,
			PageSize:    len(profiles),
			TotalPages:  1,
			HasNext:     false,
			HasPrevious: false,
		}, nil
	}

	profiles, total, err := s.profileRepo.GetByBoxIDPaginated(boxID, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get profiles: %w", err)
	}

	responses := make([]*models.ProfileResponse, len(profiles))
	for i, profile := range profiles {
		responses[i] = s.toResponse(profile)
	}

	// Calculate pagination info
	totalPages := (total + pageSize - 1) / pageSize
	hasNext := page < totalPages
	hasPrevious := page > 1

	return &models.PaginatedProfileResponse{
		Profiles:    responses,
		Total:       total,
		Page:        page,
		PageSize:    pageSize,
		TotalPages:  totalPages,
		HasNext:     hasNext,
		HasPrevious: hasPrevious,
	}, nil
}

// GetProfilesByUserPaginated retrieves paginated profiles for a specific user
func (s *ProfileService) GetProfilesByUserPaginated(userID string, page, pageSize int) (*models.PaginatedProfileResponse, error) {
	// Validate pagination parameters
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// If pageSize is very large, get all profiles
	if pageSize >= 1000 {
		profiles, err := s.profileRepo.GetByUserID(userID)
		if err != nil {
			return nil, fmt.Errorf("failed to get profiles: %w", err)
		}

		responses := make([]*models.ProfileResponse, len(profiles))
		for i, profile := range profiles {
			responses[i] = s.toResponse(profile)
		}

		return &models.PaginatedProfileResponse{
			Profiles:    responses,
			Total:       len(profiles),
			Page:        1,
			PageSize:    len(profiles),
			TotalPages:  1,
			HasNext:     false,
			HasPrevious: false,
		}, nil
	}

	profiles, total, err := s.profileRepo.GetByUserIDPaginated(userID, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get profiles: %w", err)
	}

	responses := make([]*models.ProfileResponse, len(profiles))
	for i, profile := range profiles {
		responses[i] = s.toResponse(profile)
	}

	// Calculate pagination info
	totalPages := (total + pageSize - 1) / pageSize
	hasNext := page < totalPages
	hasPrevious := page > 1

	return &models.PaginatedProfileResponse{
		Profiles:    responses,
		Total:       total,
		Page:        page,
		PageSize:    pageSize,
		TotalPages:  totalPages,
		HasNext:     hasNext,
		HasPrevious: hasPrevious,
	}, nil
}

// GetProfilesByApp retrieves all profiles for a specific app (user must own the app)
func (s *ProfileService) GetProfilesByApp(userID, appID string) ([]*models.ProfileResponse, error) {
	// Verify app belongs to user
	_, err := s.appRepo.GetByUserIDAndID(userID, appID)
	if err != nil {
		return nil, errors.New("app not found or access denied")
	}

	profiles, err := s.profileRepo.GetByAppID(appID)
	if err != nil {
		return nil, fmt.Errorf("failed to get profiles: %w", err)
	}

	responses := make([]*models.ProfileResponse, len(profiles))
	for i, profile := range profiles {
		responses[i] = s.toResponse(profile)
	}

	return responses, nil
}

// GetProfilesByAppPaginated retrieves paginated profiles for a specific app
func (s *ProfileService) GetProfilesByAppPaginated(userID, appID string, page, pageSize int) (*models.PaginatedProfileResponse, error) {
	// Verify app belongs to user
	_, err := s.appRepo.GetByUserIDAndID(userID, appID)
	if err != nil {
		return nil, errors.New("app not found or access denied")
	}

	// Validate pagination parameters
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// If pageSize is very large, get all profiles
	if pageSize >= 1000 {
		profiles, err := s.profileRepo.GetByAppID(appID)
		if err != nil {
			return nil, fmt.Errorf("failed to get profiles: %w", err)
		}

		responses := make([]*models.ProfileResponse, len(profiles))
		for i, profile := range profiles {
			responses[i] = s.toResponse(profile)
		}

		return &models.PaginatedProfileResponse{
			Profiles:    responses,
			Total:       len(profiles),
			Page:        1,
			PageSize:    len(profiles),
			TotalPages:  1,
			HasNext:     false,
			HasPrevious: false,
		}, nil
	}

	profiles, total, err := s.profileRepo.GetByAppIDPaginated(appID, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get profiles: %w", err)
	}

	responses := make([]*models.ProfileResponse, len(profiles))
	for i, profile := range profiles {
		responses[i] = s.toResponse(profile)
	}

	// Calculate pagination info
	totalPages := (total + pageSize - 1) / pageSize
	hasNext := page < totalPages
	hasPrevious := page > 1

	return &models.PaginatedProfileResponse{
		Profiles:    responses,
		Total:       total,
		Page:        page,
		PageSize:    pageSize,
		TotalPages:  totalPages,
		HasNext:     hasNext,
		HasPrevious: hasPrevious,
	}, nil
}

// GetProfileByID retrieves a profile by ID (user must own it)
func (s *ProfileService) GetProfileByID(userID, profileID string) (*models.ProfileResponse, error) {
	profile, err := s.profileRepo.GetByUserIDAndID(userID, profileID)
	if err != nil {
		return nil, errors.New("profile not found")
	}

	return s.toResponse(profile), nil
}

// UpdateProfile updates a profile (user must own it)
func (s *ProfileService) UpdateProfile(userID, profileID string, req *models.UpdateProfileRequest) (*models.ProfileResponse, error) {
	profile, err := s.profileRepo.GetByUserIDAndID(userID, profileID)
	if err != nil {
		return nil, errors.New("profile not found")
	}

	// Check if new name already exists in this app (if name is being changed)
	if req.Name != profile.Name {
		exists, err := s.profileRepo.CheckNameExistsInApp(profile.AppID, req.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to check profile name: %w", err)
		}
		if exists {
			return nil, fmt.Errorf("profile with name '%s' already exists in this app", req.Name)
		}
	}

	// Update fields
	profile.Name = req.Name
	profile.Data = req.Data

	if err := s.profileRepo.Update(profile); err != nil {
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}

	return s.toResponse(profile), nil
}

// DeleteProfile deletes a profile (user must own it)
func (s *ProfileService) DeleteProfile(userID, profileID string) error {
	// Check if profile exists and belongs to user
	_, err := s.profileRepo.GetByUserIDAndID(userID, profileID)
	if err != nil {
		return errors.New("profile not found")
	}

	if err := s.profileRepo.DeleteByUserIDAndID(userID, profileID); err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}

	return nil
}

// GetAllProfiles retrieves all profiles (admin only)
func (s *ProfileService) GetAllProfiles() ([]*models.ProfileResponse, error) {
	profiles, err := s.profileRepo.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get all profiles: %w", err)
	}

	responses := make([]*models.ProfileResponse, len(profiles))
	for i, profile := range profiles {
		responses[i] = s.toResponse(profile)
	}

	return responses, nil
}

// toResponse converts Profile model to response DTO
func (s *ProfileService) toResponse(profile *models.Profile) *models.ProfileResponse {
	return &models.ProfileResponse{
		ID:        profile.ID,
		AppID:     profile.AppID,
		Name:      profile.Name,
		Data:      profile.Data,
		CreatedAt: profile.CreatedAt.Format(time.RFC3339),
		UpdatedAt: profile.UpdatedAt.Format(time.RFC3339),
	}
}

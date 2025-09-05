package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
)

type BoxService struct {
	boxRepo     *repository.BoxRepository
	appRepo     *repository.AppRepository
	userRepo    *repository.UserRepository
	profileRepo *repository.ProfileRepository
}

// NewBoxService creates a new box service
func NewBoxService(boxRepo *repository.BoxRepository, appRepo *repository.AppRepository, userRepo *repository.UserRepository, profileRepo *repository.ProfileRepository) *BoxService {
	return &BoxService{
		boxRepo:     boxRepo,
		appRepo:     appRepo,
		userRepo:    userRepo,
		profileRepo: profileRepo,
	}
}

// CreateBox creates a new box for a user
func (s *BoxService) CreateBox(userID string, req *models.CreateBoxRequest) (*models.BoxResponse, error) {
	// Check if machine ID already exists
	existingBox, err := s.boxRepo.GetByMachineID(req.MachineID)
	if err == nil {
		// Box exists, return error with box details
		return nil, &models.BoxAlreadyExistsError{
			BoxID:     existingBox.ID,
			MachineID: existingBox.MachineID,
			Message:   "machine ID already exists",
		}
	}

	// Verify user exists
	_, err = s.userRepo.GetByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Create box
	box := &models.Box{
		UserID:    userID,
		MachineID: req.MachineID,
		Name:      req.Name,
	}

	if err := s.boxRepo.Create(box); err != nil {
		return nil, fmt.Errorf("failed to create box: %w", err)
	}

	return s.toResponse(box), nil
}

// SyncAllProfilesInBox syncs all profiles from all apps in a specific box
func (s *BoxService) SyncAllProfilesInBox(userID, boxID string) (*models.SyncBoxProfilesResponse, error) {

	// Call app service to sync all apps in this box
	appService := NewAppService(s.appRepo, s.profileRepo, s.boxRepo, s.userRepo)
	syncResult, err := appService.SyncAllAppsInBox(boxID)
	if err != nil {
		return nil, fmt.Errorf("failed to sync box %s: %w", boxID, err)
	}

	return syncResult, nil
}

// GetBoxesByUserPaginated retrieves paginated boxes for a specific user
func (s *BoxService) GetBoxesByUserPaginated(userID string, page, pageSize int) ([]*models.BoxResponse, int, error) {
	boxes, total, err := s.boxRepo.GetByUserID(userID, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get boxes: %w", err)
	}

	responses := make([]*models.BoxResponse, len(boxes))
	for i, box := range boxes {
		responses[i] = s.toResponse(box)
	}

	return responses, total, nil
}

// GetBoxByID retrieves a box by ID (user must own it)
func (s *BoxService) GetBoxByID(userID, boxID string) (*models.BoxResponse, error) {
	box, err := s.boxRepo.GetByUserIDAndID(userID, boxID)
	if err != nil {
		return nil, errors.New("box not found")
	}

	return s.toResponse(box), nil
}

// UpdateBox updates a box (user must own it)
func (s *BoxService) UpdateBox(userID, boxID string, req *models.UpdateBoxRequest) (*models.BoxResponse, error) {
	// Get box by ID (no ownership check - allow claiming any box)
	box, err := s.boxRepo.GetByID(boxID)
	if err != nil {
		return nil, errors.New("box not found")
	}

	// Update both name and user_id (always set to current logged-in user)
	box.Name = req.Name
	box.UserID = userID

	if err := s.boxRepo.Update(box); err != nil {
		return nil, fmt.Errorf("failed to update box: %w", err)
	}

	// Get the updated box to verify changes
	updatedBox, err := s.boxRepo.GetByID(boxID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated box: %w", err)
	}

	return s.toResponse(updatedBox), nil
}

// DeleteBox deletes a box (user must own it)
func (s *BoxService) DeleteBox(userID, boxID string) error {
	// Check if box exists and belongs to user
	_, err := s.boxRepo.GetByUserIDAndID(userID, boxID)
	if err != nil {
		return errors.New("box not found")
	}

	if err := s.boxRepo.DeleteByUserIDAndID(userID, boxID); err != nil {
		return fmt.Errorf("failed to delete box: %w", err)
	}

	return nil
}

// GetAllBoxes retrieves all boxes (admin only)
func (s *BoxService) GetAllBoxes() ([]*models.BoxResponse, error) {
	boxes, err := s.boxRepo.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get all boxes: %w", err)
	}

	responses := make([]*models.BoxResponse, len(boxes))
	for i, box := range boxes {
		responses[i] = s.toResponse(box)
	}

	return responses, nil
}

// GetBoxByMachineID retrieves a box by machine ID
func (s *BoxService) GetBoxByMachineID(machineID string) (*models.BoxResponse, error) {
	box, err := s.boxRepo.GetByMachineID(machineID)
	if err != nil {
		return nil, errors.New("box not found")
	}

	return s.toResponse(box), nil
}

// toResponse converts Box model to response DTO
func (s *BoxService) toResponse(box *models.Box) *models.BoxResponse {
	return &models.BoxResponse{
		ID:        box.ID,
		UserID:    box.UserID,
		MachineID: box.MachineID,
		Name:      box.Name,
		CreatedAt: box.CreatedAt.Format(time.RFC3339),
		UpdatedAt: box.UpdatedAt.Format(time.RFC3339),
	}
}

// GetBoxRepo returns the box repository
func (s *BoxService) GetBoxRepo() *repository.BoxRepository {
	return s.boxRepo
}

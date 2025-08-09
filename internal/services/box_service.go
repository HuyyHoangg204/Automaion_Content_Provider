package services

import (
	"errors"
	"fmt"
	"time"

	"green-anti-detect-browser-backend-v1/internal/database/repository"
	"green-anti-detect-browser-backend-v1/internal/models"
)

type BoxService struct {
	boxRepo  *repository.BoxRepository
	userRepo *repository.UserRepository
}

func NewBoxService(boxRepo *repository.BoxRepository, userRepo *repository.UserRepository) *BoxService {
	return &BoxService{
		boxRepo:  boxRepo,
		userRepo: userRepo,
	}
}

// CreateBox creates a new box for a user
func (s *BoxService) CreateBox(userID uint, req *models.CreateBoxRequest) (*models.BoxResponse, error) {
	// Check if machine ID already exists
	exists, err := s.boxRepo.CheckMachineIDExists(req.MachineID)
	if err != nil {
		return nil, fmt.Errorf("failed to check machine ID: %w", err)
	}
	if exists {
		return nil, errors.New("machine ID already exists")
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

// GetBoxesByUser retrieves all boxes for a specific user
func (s *BoxService) GetBoxesByUser(userID uint) ([]*models.BoxResponse, error) {
	boxes, err := s.boxRepo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get boxes: %w", err)
	}

	responses := make([]*models.BoxResponse, len(boxes))
	for i, box := range boxes {
		responses[i] = s.toResponse(box)
	}

	return responses, nil
}

// GetBoxByID retrieves a box by ID (user must own it)
func (s *BoxService) GetBoxByID(userID, boxID uint) (*models.BoxResponse, error) {
	box, err := s.boxRepo.GetByUserIDAndID(userID, boxID)
	if err != nil {
		return nil, errors.New("box not found")
	}

	return s.toResponse(box), nil
}

// UpdateBox updates a box (user must own it)
func (s *BoxService) UpdateBox(userID, boxID uint, req *models.UpdateBoxRequest) (*models.BoxResponse, error) {
	box, err := s.boxRepo.GetByUserIDAndID(userID, boxID)
	if err != nil {
		return nil, errors.New("box not found")
	}

	// Update fields
	box.Name = req.Name

	if err := s.boxRepo.Update(box); err != nil {
		return nil, fmt.Errorf("failed to update box: %w", err)
	}

	return s.toResponse(box), nil
}

// DeleteBox deletes a box (user must own it)
func (s *BoxService) DeleteBox(userID, boxID uint) error {
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

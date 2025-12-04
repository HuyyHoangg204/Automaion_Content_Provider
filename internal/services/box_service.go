package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
)

type BoxService struct {
	boxRepo    *repository.BoxRepository
	userRepo   *repository.UserRepository
	appRepo    *repository.AppRepository
	appService *AppService
}

// NewBoxService creates a new box service
func NewBoxService(boxRepo *repository.BoxRepository, userRepo *repository.UserRepository) *BoxService {
	return &BoxService{
		boxRepo:  boxRepo,
		userRepo: userRepo,
	}
}

// SetAppService sets the app service (for checking tunnel status)
func (s *BoxService) SetAppService(appService *AppService) {
	s.appService = appService
}

// SetAppRepo sets the app repository
func (s *BoxService) SetAppRepo(appRepo *repository.AppRepository) {
	s.appRepo = appRepo
}

// CreateBox creates a new box for a user
func (s *BoxService) CreateBoxByUserID(userID string, req *models.CreateBoxRequest) (*models.BoxResponse, error) {
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

// GetBoxesByUserPaginated retrieves paginated boxes for a specific user
func (s *BoxService) GetBoxesByUserIDPaginated(userID string, page, pageSize int) ([]*models.BoxResponse, int, error) {
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
func (s *BoxService) GetBoxByUserIDAndID(userID, boxID string) (*models.BoxResponse, error) {
	box, err := s.boxRepo.GetByUserIDAndID(userID, boxID)
	if err != nil {
		return nil, errors.New("box not found")
	}

	return s.toResponse(box), nil
}

// UpdateBox updates a box (user must own it)
func (s *BoxService) UpdateBoxByUserIDAndID(userID, boxID string, req *models.UpdateBoxRequest) (*models.BoxResponse, error) {
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
func (s *BoxService) DeleteBoxByUserIDAndID(userID, boxID string) error {
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

// GetAllBoxesWithStatus retrieves all boxes with online/offline status (admin only)
func (s *BoxService) GetAllBoxesWithStatus() ([]*models.BoxWithStatusResponse, error) {
	boxes, err := s.boxRepo.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get all boxes: %w", err)
	}

	if s.appRepo == nil || s.appService == nil {
		return nil, fmt.Errorf("app service or repository not initialized")
	}

	responses := make([]*models.BoxWithStatusResponse, 0, len(boxes))

	for _, box := range boxes {
		// Get apps for this box
		apps, err := s.appRepo.GetByBoxID(box.ID)
		if err != nil {
			// Log error but continue
			continue
		}

		// Find Automation app with tunnel URL
		var automationApp *models.App
		for _, app := range apps {
			if app.TunnelURL != nil && *app.TunnelURL != "" {
				// Check if it's Automation app (case-insensitive)
				if strings.ToLower(app.Name) == "automation" {
					automationApp = app
					break
				}
			}
		}

		// Check online status
		// Priority: Use is_online from DB if heartbeat is recent, otherwise perform health check
		now := time.Now()
		timeSinceLastSeen := now.Sub(box.UpdatedAt)
		isOnline := box.IsOnline // Use value from DB
		var statusCheck *models.StatusCheckInfo

		// If heartbeat is recent (< 5 minutes), trust DB value
		if timeSinceLastSeen < 5*time.Minute {
			// Use is_online from DB (set by heartbeat)
			if isOnline {
				statusCheck = &models.StatusCheckInfo{
					IsAccessible: true,
					ResponseTime: 0,
					Message:      fmt.Sprintf("Machine is online (last heartbeat %d seconds ago)", int(timeSinceLastSeen.Seconds())),
				}
			} else {
				statusCheck = &models.StatusCheckInfo{
					IsAccessible: false,
					ResponseTime: 0,
					Message:      fmt.Sprintf("Machine is offline (last heartbeat %d seconds ago)", int(timeSinceLastSeen.Seconds())),
				}
			}
		} else if automationApp != nil && automationApp.TunnelURL != nil {
			// If heartbeat is old, perform health check
			checkResult, err := s.appService.CheckTunnelURLSimple(*automationApp.TunnelURL)
			if err == nil && checkResult != nil {
				isOnline = checkResult.IsAccessible
				statusCheck = &models.StatusCheckInfo{
					IsAccessible: checkResult.IsAccessible,
					ResponseTime: checkResult.ResponseTime,
					Message:      checkResult.Message,
					StatusCode:   checkResult.StatusCode,
					Error:        checkResult.Error,
				}
			} else {
				// Health check failed
				isOnline = false
				errorMsg := "Health check failed"
				if err != nil {
					errorMsg = err.Error()
				}
				statusCheck = &models.StatusCheckInfo{
					IsAccessible: false,
					ResponseTime: 0,
					Message:      fmt.Sprintf("Machine offline (last heartbeat %d minutes ago)", int(timeSinceLastSeen.Minutes())),
					Error:        &errorMsg,
				}
			}
		} else {
			// No tunnel URL or automation app
			isOnline = false
			statusCheck = &models.StatusCheckInfo{
				IsAccessible: false,
				ResponseTime: 0,
				Message:      fmt.Sprintf("No automation app or tunnel URL found (last heartbeat %d minutes ago)", int(timeSinceLastSeen.Minutes())),
			}
		}

		// Convert to response
		response := &models.BoxWithStatusResponse{
			ID:          box.ID,
			UserID:      box.UserID,
			MachineID:   box.MachineID,
			Name:        box.Name,
			IsOnline:    isOnline,
			LastSeen:    box.UpdatedAt.Format(time.RFC3339),
			StatusCheck: statusCheck,
			CreatedAt:   box.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   box.UpdatedAt.Format(time.RFC3339),
		}

		responses = append(responses, response)
	}

	return responses, nil
}

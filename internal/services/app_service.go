package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
)

type AppService struct {
	appRepo  *repository.AppRepository
	boxRepo  *repository.BoxRepository
	userRepo *repository.UserRepository
}

func NewAppService(appRepo *repository.AppRepository, boxRepo *repository.BoxRepository, userRepo *repository.UserRepository) *AppService {
	return &AppService{
		appRepo:  appRepo,
		boxRepo:  boxRepo,
		userRepo: userRepo,
	}
}

// CreateApp creates a new app for a user
func (s *AppService) CreateApp(userID string, req *models.CreateAppRequest) (*models.AppResponse, error) {
	// Verify user exists
	_, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Verify box exists and belongs to user
	_, err = s.boxRepo.GetByUserIDAndID(userID, req.BoxID)
	if err != nil {
		return nil, errors.New("box not found or access denied")
	}

	// Check if app name already exists in this box
	exists, err := s.appRepo.CheckNameExistsInBox(req.BoxID, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check app name: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("app with name '%s' already exists in this box", req.Name)
	}

	// Create app
	app := &models.App{
		BoxID: req.BoxID,
		Name:  req.Name,
	}

	if err := s.appRepo.Create(app); err != nil {
		return nil, fmt.Errorf("failed to create app: %w", err)
	}

	return s.toResponse(app), nil
}

// GetAppsByUser retrieves all apps for a specific user
func (s *AppService) GetAppsByUser(userID string) ([]*models.AppResponse, error) {
	apps, err := s.appRepo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get apps: %w", err)
	}

	responses := make([]*models.AppResponse, len(apps))
	for i, app := range apps {
		responses[i] = s.toResponse(app)
	}

	return responses, nil
}

// GetAppsByBox retrieves all apps for a specific box (user must own the box)
func (s *AppService) GetAppsByBox(userID, boxID string) ([]*models.AppResponse, error) {
	// Verify box belongs to user
	_, err := s.boxRepo.GetByUserIDAndID(userID, boxID)
	if err != nil {
		return nil, errors.New("box not found or access denied")
	}

	apps, err := s.appRepo.GetByBoxID(boxID)
	if err != nil {
		return nil, fmt.Errorf("failed to get apps: %w", err)
	}

	responses := make([]*models.AppResponse, len(apps))
	for i, app := range apps {
		responses[i] = s.toResponse(app)
	}

	return responses, nil
}

// GetAppByID retrieves an app by ID (user must own it)
func (s *AppService) GetAppByID(userID, appID string) (*models.AppResponse, error) {
	app, err := s.appRepo.GetByUserIDAndID(userID, appID)
	if err != nil {
		return nil, errors.New("app not found")
	}

	return s.toResponse(app), nil
}

// UpdateApp updates an app (user must own it)
func (s *AppService) UpdateApp(userID, appID string, req *models.UpdateAppRequest) (*models.AppResponse, error) {
	app, err := s.appRepo.GetByUserIDAndID(userID, appID)
	if err != nil {
		return nil, errors.New("app not found")
	}

	// Check if new name already exists in this box (if name is being changed)
	if req.Name != app.Name {
		exists, err := s.appRepo.CheckNameExistsInBox(app.BoxID, req.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to check app name: %w", err)
		}
		if exists {
			return nil, fmt.Errorf("app with name '%s' already exists in this box", req.Name)
		}
	}

	// Update fields
	app.Name = req.Name

	if err := s.appRepo.Update(app); err != nil {
		return nil, fmt.Errorf("failed to update app: %w", err)
	}

	return s.toResponse(app), nil
}

// DeleteApp deletes an app (user must own it)
func (s *AppService) DeleteApp(userID, appID string) error {
	// Check if app exists and belongs to user
	_, err := s.appRepo.GetByUserIDAndID(userID, appID)
	if err != nil {
		return errors.New("app not found")
	}

	if err := s.appRepo.DeleteByUserIDAndID(userID, appID); err != nil {
		return fmt.Errorf("failed to delete app: %w", err)
	}

	return nil
}

// GetAllApps retrieves all apps (admin only)
func (s *AppService) GetAllApps() ([]*models.AppResponse, error) {
	apps, err := s.appRepo.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get all apps: %w", err)
	}

	responses := make([]*models.AppResponse, len(apps))
	for i, app := range apps {
		responses[i] = s.toResponse(app)
	}

	return responses, nil
}

// toResponse converts App model to response DTO
func (s *AppService) toResponse(app *models.App) *models.AppResponse {
	return &models.AppResponse{
		ID:        app.ID,
		BoxID:     app.BoxID,
		Name:      app.Name,
		CreatedAt: app.CreatedAt.Format(time.RFC3339),
		UpdatedAt: app.UpdatedAt.Format(time.RFC3339),
	}
}

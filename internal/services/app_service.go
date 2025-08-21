package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/onegreenvn/green-provider-services-backend/internal/config"
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
		BoxID:     req.BoxID,
		Name:      req.Name,
		TunnelURL: req.TunnelURL,
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
	if req.TunnelURL != nil {
		app.TunnelURL = req.TunnelURL
	}

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

// GetRegisterAppDomains generates subdomain and FRP configuration for app registration
func (s *AppService) GetRegisterAppDomains(userID, boxID, platformNames string) (*models.RegisterAppResponse, error) {
	// Get user to verify it exists
	_, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Verify box belongs to user and get machine_id
	box, err := s.boxRepo.GetByUserIDAndID(userID, boxID)
	if err != nil {
		return nil, fmt.Errorf("box not found or access denied: %w", err)
	}

	machineID := box.MachineID
	if machineID == "" {
		return nil, fmt.Errorf("box has no machine_id")
	}

	// Create response
	response := &models.RegisterAppResponse{}

	// Create dynamic subdomain map for all requested platforms
	response.SubDomain = make(map[string]string)

	// Split platform names by comma and create subdomain for each
	platformList := strings.Split(platformNames, ",")
	for _, platform := range platformList {
		platform = strings.TrimSpace(platform) // Remove whitespace
		if platform != "" {
			response.SubDomain[platform] = fmt.Sprintf("%s-%s-%s", machineID, platform, userID)
		}
	}

	// Set FRP configuration from environment variables
	frpConfig := config.GetFrpConfig()
	response.FrpDomain = frpConfig.Domain
	response.FrpServerPort = frpConfig.Port
	response.FrpToken = frpConfig.Token
	response.FrpProtocol = frpConfig.Protocol
	return response, nil
}

// CheckTunnelURL checks if a tunnel URL is accessible and ready for Hidemium
func (s *AppService) CheckTunnelURL(tunnelURL string) (*models.CheckTunnelResponse, error) {
	if tunnelURL == "" {
		return nil, errors.New("tunnel URL is empty")
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Record start time for response time measurement
	startTime := time.Now()

	// Test tunnel by calling /user-settings/token endpoint
	testURL := fmt.Sprintf("%s/user-settings/token", strings.TrimSuffix(tunnelURL, "/"))

	resp, err := client.Get(testURL)
	responseTime := time.Since(startTime).Milliseconds()

	if err != nil {
		errorMsg := err.Error()
		return &models.CheckTunnelResponse{
			IsAccessible: false,
			ResponseTime: responseTime,
			Message:      "Tunnel is not accessible",
			Error:        &errorMsg,
		}, nil
	}
	defer resp.Body.Close()

	// Check if response is successful
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &models.CheckTunnelResponse{
			IsAccessible: false,
			ResponseTime: responseTime,
			Message:      fmt.Sprintf("Tunnel returned status code: %d", resp.StatusCode),
			StatusCode:   &resp.StatusCode,
		}, nil
	}

	// Try to parse response to check if it contains token data
	var responseData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		return &models.CheckTunnelResponse{
			IsAccessible: false,
			ResponseTime: responseTime,
			Message:      "Tunnel accessible but response is not valid JSON",
			StatusCode:   &resp.StatusCode,
		}, nil
	}

	// Check if response contains token data
	if _, hasToken := responseData["token"]; hasToken {
		return &models.CheckTunnelResponse{
			IsAccessible: true,
			ResponseTime: responseTime,
			Message:      "Tunnel is accessible and /user-settings/token endpoint is working",
			StatusCode:   &resp.StatusCode,
		}, nil
	}

	// Check for other possible token fields
	tokenFields := []string{"access_token", "api_token", "auth_token", "key"}
	for _, field := range tokenFields {
		if _, hasField := responseData[field]; hasField {
			return &models.CheckTunnelResponse{
				IsAccessible: true,
				ResponseTime: responseTime,
				Message:      fmt.Sprintf("Tunnel is accessible and contains %s", field),
				StatusCode:   &resp.StatusCode,
			}, nil
		}
	}

	return &models.CheckTunnelResponse{
		IsAccessible: false,
		ResponseTime: responseTime,
		Message:      "Tunnel accessible but /user-settings/token endpoint does not return token data",
		StatusCode:   &resp.StatusCode,
	}, nil
}

// toResponse converts App model to response DTO
func (s *AppService) toResponse(app *models.App) *models.AppResponse {
	return &models.AppResponse{
		ID:        app.ID,
		BoxID:     app.BoxID,
		Name:      app.Name,
		TunnelURL: app.TunnelURL,
		CreatedAt: app.CreatedAt.Format(time.RFC3339),
		UpdatedAt: app.UpdatedAt.Format(time.RFC3339),
	}
}

package services

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/onegreenvn/green-provider-services-backend/internal/config"
	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/sirupsen/logrus"
)

type MachineService struct {
	boxRepo  *repository.BoxRepository
	appRepo  *repository.AppRepository
	userRepo *repository.UserRepository
}

func NewMachineService(boxRepo *repository.BoxRepository, appRepo *repository.AppRepository, userRepo *repository.UserRepository) *MachineService {
	return &MachineService{
		boxRepo:  boxRepo,
		appRepo:  appRepo,
		userRepo: userRepo,
	}
}

// RegisterMachine registers a machine (creates box if not exists, returns existing if exists)
func (s *MachineService) RegisterMachine(machineID, name string) (*models.RegisterMachineResponse, error) {
	// Check if machine already exists
	existingBox, err := s.boxRepo.GetByMachineID(machineID)
	if err == nil {
		// Machine already exists, return existing box info
		userID := existingBox.UserID
		return &models.RegisterMachineResponse{
			BoxID:   existingBox.ID,
			UserID:  &userID,
			Message: "Machine already registered",
		}, nil
	}

	// Machine doesn't exist, create new box
	// Try to get default user ID from environment (optional)
	defaultUserID := getMachineEnv("DEFAULT_USER_ID", "")
	var userID *string
	if defaultUserID != "" {
		// Verify user exists
		_, err := s.userRepo.GetByID(defaultUserID)
		if err == nil {
			userID = &defaultUserID
		}
	}

	// Create box (userID can be null initially, user can claim it later)
	box := &models.Box{
		MachineID: machineID,
		Name:      name,
	}

	// If we have a default user, assign it
	if userID != nil {
		box.UserID = *userID
	} else {
		// Create with empty user_id (will be set when user claims it)
		// But GORM requires not null, so we need a placeholder
		// Get first admin user or create a system user
		adminUsers, _, err := s.userRepo.GetAllUsers(1, 1, "")
		if err == nil && len(adminUsers) > 0 {
			box.UserID = adminUsers[0].ID
		} else {
			// If no users exist, we can't create box without user_id
			// Return error or create a system user
			return nil, errors.New("no users found in system, please create a user first")
		}
	}

	if err := s.boxRepo.Create(box); err != nil {
		return nil, fmt.Errorf("failed to create box: %w", err)
	}

	return &models.RegisterMachineResponse{
		BoxID:   box.ID,
		UserID:  &box.UserID,
		Message: "Machine registered successfully",
	}, nil
}

// GetFrpConfigByMachineID gets FRP configuration for a machine
func (s *MachineService) GetFrpConfigByMachineID(machineID string) (*models.RegisterAppResponse, error) {
	// Get box by machine ID
	box, err := s.boxRepo.GetByMachineID(machineID)
	if err != nil {
		return nil, errors.New("machine not found")
	}

	// Create response
	response := &models.RegisterAppResponse{}

	// Create subdomain for "automation" platform
	response.SubDomain = make(map[string]string)
	response.SubDomain["automation"] = fmt.Sprintf("%s-automation-%s", box.MachineID, box.UserID)

	// Set FRP configuration from environment variables
	frpConfig := config.GetFrpConfig()
	response.FrpServerDomain = frpConfig.Domain
	response.FrpServerPort = frpConfig.Port
	response.FrpToken = frpConfig.Token
	response.FrpProtocol = frpConfig.Protocol
	response.FrpCustomDomain = frpConfig.CustomDomain

	return response, nil
}

// normalizeTunnelURL adds port to tunnel URL if not present
func (s *MachineService) normalizeTunnelURL(tunnelURL string) string {
	// Parse URL to check if port is already present
	parsedURL, err := url.Parse(tunnelURL)
	if err != nil {
		return tunnelURL
	}

	// If port is already present, return as-is
	if parsedURL.Port() != "" {
		return tunnelURL
	}

	// Get FRP_TUNNEL_PORT from environment (default: 8085)
	tunnelPort := os.Getenv("FRP_TUNNEL_PORT")
	if tunnelPort == "" {
		tunnelPort = "8085" // Default port
	}

	// Validate port is a number
	if _, err := strconv.Atoi(tunnelPort); err != nil {
		logrus.Warnf("Invalid FRP_TUNNEL_PORT: %s, using default 8085", tunnelPort)
		tunnelPort = "8085"
	}

	// Add port to URL
	parsedURL.Host = parsedURL.Host + ":" + tunnelPort
	normalizedURL := parsedURL.String()
	logrus.Infof("Normalized tunnel URL: %s -> %s", tunnelURL, normalizedURL)
	return normalizedURL
}

// UpdateTunnelURLByMachineID updates tunnel URL for a machine's app
func (s *MachineService) UpdateTunnelURLByMachineID(machineID, tunnelURL string) (*models.UpdateTunnelURLResponse, error) {
	// Normalize tunnel URL (add port if not present)
	normalizedURL := s.normalizeTunnelURL(tunnelURL)

	// Get box by machine ID
	box, err := s.boxRepo.GetByMachineID(machineID)
	if err != nil {
		return nil, errors.New("machine not found")
	}

	// Find or create app with name "Automation" for this box
	apps, err := s.appRepo.GetByBoxID(box.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get apps: %w", err)
	}

	var app *models.App
	appName := "Automation"

	// Find existing Automation app
	for _, a := range apps {
		if strings.EqualFold(a.Name, appName) {
			app = a
			break
		}
	}

	// Create app if not exists
	if app == nil {
		app = &models.App{
			BoxID:     box.ID,
			Name:      appName,
			TunnelURL: &normalizedURL,
		}
		if err := s.appRepo.Create(app); err != nil {
			return nil, fmt.Errorf("failed to create app: %w", err)
		}
	} else {
		// Update existing app
		app.TunnelURL = &normalizedURL
		if err := s.appRepo.Update(app); err != nil {
			return nil, fmt.Errorf("failed to update app: %w", err)
		}
	}

	return &models.UpdateTunnelURLResponse{
		Success: true,
		AppID:   app.ID,
		Message: "Tunnel URL updated successfully",
	}, nil
}

// SendHeartbeat updates machine status and last seen time
func (s *MachineService) SendHeartbeat(machineID string, req *models.HeartbeatRequest) (*models.HeartbeatResponse, error) {
	// Get box by machine ID
	box, err := s.boxRepo.GetByMachineID(machineID)
	if err != nil {
		return nil, errors.New("machine not found")
	}

	// Update tunnel URL if provided and different
	if req.TunnelURL != "" {
		_, err := s.UpdateTunnelURLByMachineID(machineID, req.TunnelURL)
		if err != nil {
			// Log error but don't fail heartbeat
			fmt.Printf("Failed to update tunnel URL in heartbeat: %v\n", err)
		}
	}

	// Update box updated_at (acts as last_seen) and set is_online = true
	box.UpdatedAt = time.Now()
	box.IsOnline = true // Set online when receiving heartbeat
	if err := s.boxRepo.Update(box); err != nil {
		return nil, fmt.Errorf("failed to update box: %w", err)
	}

	return &models.HeartbeatResponse{
		Success:  true,
		LastSeen: box.UpdatedAt.Format(time.RFC3339),
		Message:  "Heartbeat received",
	}, nil
}

// getMachineEnv gets environment variable with fallback default value
func getMachineEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

package genlogin

import (
	"context"
	"fmt"

	"github.com/onegreenvn/green-provider-services-backend/internal/models"
)

// BoxService implements box operations for Genlogin platform
type BoxService struct{}

// NewBoxService creates a new Genlogin box service
func NewBoxService() *BoxService {
	return &BoxService{}
}

// CreateBox creates a new box on Genlogin platform
func (s *BoxService) CreateBox(ctx context.Context, boxData *models.CreateBoxRequest) (*models.BoxResponse, error) {
	return nil, fmt.Errorf("box creation on Genlogin not implemented yet")
}

// UpdateBox updates an existing box on Genlogin platform
func (s *BoxService) UpdateBox(ctx context.Context, boxID string, boxData *models.UpdateBoxRequest) (*models.BoxResponse, error) {
	return nil, fmt.Errorf("box update on Genlogin not implemented yet")
}

// DeleteBox deletes a box on Genlogin platform
func (s *BoxService) DeleteBox(ctx context.Context, boxID string) error {
	return fmt.Errorf("box deletion on Genlogin not implemented yet")
}

// GetBox retrieves box information from Genlogin platform
func (s *BoxService) GetBox(ctx context.Context, boxID string) (*models.BoxResponse, error) {
	return nil, fmt.Errorf("box retrieval on Genlogin not implemented yet")
}

// ListBoxes lists all boxes from Genlogin platform
func (s *BoxService) ListBoxes(ctx context.Context, filters map[string]interface{}) ([]*models.BoxResponse, error) {
	return nil, fmt.Errorf("box listing on Genlogin not implemented yet")
}

// SyncBoxProfilesFromPlatform syncs profiles from Genlogin platform for a specific box
func (s *BoxService) SyncBoxProfilesFromPlatform(ctx context.Context, boxID string, machineID string) ([]models.HidemiumProfile, error) {
	return nil, fmt.Errorf("profile sync from Genlogin not implemented yet")
}

// GetPlatformName returns the platform name
func (s *BoxService) GetPlatformName() string {
	return "genlogin"
}

// GetPlatformVersion returns the platform version
func (s *BoxService) GetPlatformVersion() string {
	return "0.0"
}

// ValidateBoxData validates box data for Genlogin platform
func (s *BoxService) ValidateBoxData(boxData *models.CreateBoxRequest) error {
	return fmt.Errorf("box validation on Genlogin not implemented yet")
}

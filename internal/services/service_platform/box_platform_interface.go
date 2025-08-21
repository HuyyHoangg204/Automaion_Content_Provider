package service_platform

import (
	"context"

	"github.com/onegreenvn/green-provider-services-backend/internal/models"
)

// BoxPlatformInterface định nghĩa các method cần thiết cho mọi nền tảng box
type BoxPlatformInterface interface {
	// Box operations
	CreateBox(ctx context.Context, boxData *models.CreateBoxRequest) (*models.BoxResponse, error)
	UpdateBox(ctx context.Context, boxID string, boxData *models.UpdateBoxRequest) (*models.BoxResponse, error)
	DeleteBox(ctx context.Context, boxID string) error
	GetBox(ctx context.Context, boxID string) (*models.BoxResponse, error)
	ListBoxes(ctx context.Context, filters map[string]interface{}) ([]*models.BoxResponse, error)

	// Box sync operations
	SyncBoxProfilesFromPlatform(ctx context.Context, boxID string, machineID string) (*models.SyncBoxProfilesResponse, error)

	// Platform info
	GetPlatformName() string
	GetPlatformVersion() string
	ValidateBoxData(boxData *models.CreateBoxRequest) error
}

// BoxPlatformFactory tạo instance của box platform
type BoxPlatformFactory interface {
	CreateBoxPlatform(platformType string) (BoxPlatformInterface, error)
	GetSupportedPlatforms() []string
}

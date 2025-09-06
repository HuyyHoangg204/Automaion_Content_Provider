package repository

import (
	"fmt"

	"github.com/onegreenvn/green-provider-services-backend/internal/models"

	"gorm.io/gorm"
)

type FlowRepository struct {
	db *gorm.DB
}

func NewFlowRepository(db *gorm.DB) *FlowRepository {
	return &FlowRepository{db: db}
}

// Create creates a new flow
func (r *FlowRepository) Create(flow *models.Flow) error {
	return r.db.Create(flow).Error
}

// GetByID retrieves a flow by ID
func (r *FlowRepository) GetByID(id string) (*models.Flow, error) {
	var flow models.Flow
	err := r.db.Preload("FlowGroup").Preload("Profile").First(&flow, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &flow, nil
}

// GetByProfileID retrieves all flows for a specific profile
func (r *FlowRepository) GetByProfileID(profileID string) ([]*models.Flow, error) {
	var flows []*models.Flow
	err := r.db.Where("profile_id = ?", profileID).Preload("FlowGroup").Preload("Profile").Find(&flows).Error
	return flows, err
}

// GetByProfileIDPaginated retrieves paginated flows for a specific profile
func (r *FlowRepository) GetByProfileIDPaginated(profileID string, page, pageSize int) ([]*models.Flow, int, error) {
	var flows []*models.Flow
	var total int64

	// Count total records
	err := r.db.Where("profile_id = ?", profileID).
		Model(&models.Flow{}).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Get paginated results
	err = r.db.Where("profile_id = ?", profileID).
		Preload("FlowGroup").
		Preload("Profile").
		Offset(offset).
		Limit(pageSize).
		Order("created_at DESC").
		Find(&flows).Error

	return flows, int(total), err
}

// GetByFlowGroupID retrieves all flows for a specific group campaign
func (r *FlowRepository) GetByFlowGroupID(flowGroupID string) ([]*models.Flow, error) {
	var flows []*models.Flow
	err := r.db.Where("flow_group_id = ?", flowGroupID).Preload("Profile").Preload("FlowGroup").Find(&flows).Error
	return flows, err
}

// GetByFlowGroupIDPaginated retrieves paginated flows for a specific group campaign
func (r *FlowRepository) GetByFlowGroupIDPaginated(flowGroupID string, page, pageSize int) ([]*models.Flow, int, error) {
	var flows []*models.Flow
	var total int64

	// Count total records
	err := r.db.Where("flow_group_id = ?", flowGroupID).
		Model(&models.Flow{}).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Get paginated results
	err = r.db.Where("flow_group_id = ?", flowGroupID).
		Preload("Profile").
		Preload("FlowGroup").
		Offset(offset).
		Limit(pageSize).
		Order("created_at DESC").
		Find(&flows).Error

	return flows, int(total), err
}

// GetByCampaignIDPaginated retrieves paginated flows for a specific campaign
func (r *FlowRepository) GetByCampaignIDPaginated(campaignID string, page, pageSize int) ([]*models.Flow, int, error) {
	var flows []*models.Flow
	var total int64

	// Count total records
	err := r.db.Joins("JOIN flow_groups ON flows.flow_group_id = flow_groups.id").
		Where("flow_groups.campaign_id = ?", campaignID).
		Model(&models.Flow{}).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Get paginated results
	err = r.db.Joins("JOIN flow_groups ON flows.flow_group_id = flow_groups.id").
		Where("flow_groups.campaign_id = ?", campaignID).
		Preload("FlowGroup").
		Preload("Profile").
		Offset(offset).
		Limit(pageSize).
		Order("flows.created_at DESC").
		Find(&flows).Error

	return flows, int(total), err
}

// GetByUserIDPaginated retrieves paginated flows for a specific user
func (r *FlowRepository) GetByUserIDPaginated(userID string, page, pageSize int) ([]*models.Flow, int, error) {
	var flows []*models.Flow
	var total int64

	// Count total records
	err := r.db.Joins("JOIN flow_groups ON flows.flow_group_id = flow_groups.id").
		Joins("JOIN campaigns ON flow_groups.campaign_id = campaigns.id").
		Where("campaigns.user_id = ?", userID).
		Model(&models.Flow{}).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Get paginated results
	err = r.db.Joins("JOIN flow_groups ON flows.flow_group_id = flow_groups.id").
		Joins("JOIN campaigns ON flow_groups.campaign_id = campaigns.id").
		Where("campaigns.user_id = ?", userID).
		Preload("FlowGroup").
		Preload("Profile").
		Offset(offset).
		Limit(pageSize).
		Order("flows.created_at DESC").
		Find(&flows).Error

	return flows, int(total), err
}

// GetByUserIDAndID retrieves a flow by user ID and flow ID
func (r *FlowRepository) GetByUserIDAndID(userID, flowID string) (*models.Flow, error) {
	var flow models.Flow
	err := r.db.Joins("JOIN flow_groups ON flows.flow_group_id = flow_groups.id").
		Joins("JOIN campaigns ON flow_groups.campaign_id = campaigns.id").
		Where("campaigns.user_id = ? AND flows.id = ?", userID, flowID).
		Preload("FlowGroup").
		Preload("Profile").
		First(&flow).Error
	if err != nil {
		return nil, err
	}
	return &flow, nil
}

// Update updates a flow
func (r *FlowRepository) Update(flow *models.Flow) error {
	return r.db.Save(flow).Error
}

// DeleteByUserIDAndID deletes a flow by user ID and flow ID
func (r *FlowRepository) DeleteByUserIDAndID(userID, flowID string) error {
	result := r.db.Unscoped().Where("user_id = ?", userID).
		Delete(&models.Flow{ID: flowID})
	if result.Error != nil {
		return fmt.Errorf("failed to delete flow: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("flow not found or access denied")
	}
	return nil
}

// GetByStatus retrieves flows by status
func (r *FlowRepository) GetByStatus(status string) ([]*models.Flow, error) {
	var flows []*models.Flow
	err := r.db.Where("status = ?", status).Preload("FlowGroup").Preload("Profile").Find(&flows).Error
	return flows, err
}

// GetByUserIDAndStatus retrieves flows by user ID and status
func (r *FlowRepository) GetByUserIDAndStatus(userID, status string) ([]*models.Flow, error) {
	var flows []*models.Flow
	err := r.db.Joins("JOIN flow_groups ON flows.flow_group_id = flow_groups.id").
		Joins("JOIN campaigns ON flow_groups.campaign_id = campaigns.id").
		Where("campaigns.user_id = ? AND flows.status = ?", userID, status).
		Preload("FlowGroup").
		Preload("Profile").
		Find(&flows).Error
	return flows, err
}

// GetByUserIDAndStatusPaginated retrieves paginated flows by user ID and status
func (r *FlowRepository) GetByUserIDAndStatusPaginated(userID, status string, page, pageSize int) ([]*models.Flow, int, error) {
	var flows []*models.Flow
	var total int64

	// Count total records
	err := r.db.Joins("JOIN flow_groups ON flows.flow_group_id = flow_groups.id").
		Joins("JOIN campaigns ON flow_groups.campaign_id = campaigns.id").
		Where("campaigns.user_id = ? AND flows.status = ?", userID, status).
		Model(&models.Flow{}).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Get paginated results
	err = r.db.Joins("JOIN flow_groups ON flows.flow_group_id = flow_groups.id").
		Joins("JOIN campaigns ON flow_groups.campaign_id = campaigns.id").
		Where("campaigns.user_id = ? AND flows.status = ?", userID, status).
		Preload("FlowGroup").
		Preload("Profile").
		Offset(offset).
		Limit(pageSize).
		Order("flows.created_at DESC").
		Find(&flows).Error

	return flows, int(total), err
}

// GetAll retrieves all flows (admin only)
func (r *FlowRepository) GetAll() ([]*models.Flow, error) {
	var flows []*models.Flow
	err := r.db.Preload("FlowGroup").Preload("Profile").Find(&flows).Error
	return flows, err
}

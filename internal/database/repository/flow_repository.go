package repository

import (
	"green-provider-services-backend/internal/models"

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
	err := r.db.Preload("GroupCampaign").Preload("Profile").First(&flow, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &flow, nil
}

// GetByProfileID retrieves all flows for a specific profile
func (r *FlowRepository) GetByProfileID(profileID string) ([]*models.Flow, error) {
	var flows []*models.Flow
	err := r.db.Where("profile_id = ?", profileID).Preload("GroupCampaign").Preload("Profile").Find(&flows).Error
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
		Preload("GroupCampaign").
		Preload("Profile").
		Offset(offset).
		Limit(pageSize).
		Order("created_at DESC").
		Find(&flows).Error

	return flows, int(total), err
}

// GetByGroupCampaignID retrieves all flows for a specific group campaign
func (r *FlowRepository) GetByGroupCampaignID(groupCampaignID string) ([]*models.Flow, error) {
	var flows []*models.Flow
	err := r.db.Where("group_campaign_id = ?", groupCampaignID).Preload("Profile").Preload("GroupCampaign").Find(&flows).Error
	return flows, err
}

// GetByGroupCampaignIDPaginated retrieves paginated flows for a specific group campaign
func (r *FlowRepository) GetByGroupCampaignIDPaginated(groupCampaignID string, page, pageSize int) ([]*models.Flow, int, error) {
	var flows []*models.Flow
	var total int64

	// Count total records
	err := r.db.Where("group_campaign_id = ?", groupCampaignID).
		Model(&models.Flow{}).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Get paginated results
	err = r.db.Where("group_campaign_id = ?", groupCampaignID).
		Preload("Profile").
		Preload("GroupCampaign").
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
	err := r.db.Joins("JOIN group_campaigns ON flows.group_campaign_id = group_campaigns.id").
		Where("group_campaigns.campaign_id = ?", campaignID).
		Model(&models.Flow{}).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Get paginated results
	err = r.db.Joins("JOIN group_campaigns ON flows.group_campaign_id = group_campaigns.id").
		Where("group_campaigns.campaign_id = ?", campaignID).
		Preload("GroupCampaign").
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
	err := r.db.Joins("JOIN group_campaigns ON flows.group_campaign_id = group_campaigns.id").
		Joins("JOIN campaigns ON group_campaigns.campaign_id = campaigns.id").
		Where("campaigns.user_id = ?", userID).
		Model(&models.Flow{}).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Get paginated results
	err = r.db.Joins("JOIN group_campaigns ON flows.group_campaign_id = group_campaigns.id").
		Joins("JOIN campaigns ON group_campaigns.campaign_id = campaigns.id").
		Where("campaigns.user_id = ?", userID).
		Preload("GroupCampaign").
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
	err := r.db.Joins("JOIN group_campaigns ON flows.group_campaign_id = group_campaigns.id").
		Joins("JOIN campaigns ON group_campaigns.campaign_id = campaigns.id").
		Where("campaigns.user_id = ? AND flows.id = ?", userID, flowID).
		Preload("GroupCampaign").
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

// Delete deletes a flow
func (r *FlowRepository) Delete(id string) error {
	return r.db.Delete(&models.Flow{}, "id = ?", id).Error
}

// DeleteByUserIDAndID deletes a flow by user ID and flow ID
func (r *FlowRepository) DeleteByUserIDAndID(userID, flowID string) error {
	return r.db.Joins("JOIN group_campaigns ON flows.group_campaign_id = group_campaigns.id").
		Joins("JOIN campaigns ON group_campaigns.campaign_id = campaigns.id").
		Where("campaigns.user_id = ? AND flows.id = ?", userID, flowID).
		Delete(&models.Flow{}).Error
}

// GetByStatus retrieves flows by status
func (r *FlowRepository) GetByStatus(status string) ([]*models.Flow, error) {
	var flows []*models.Flow
	err := r.db.Where("status = ?", status).Preload("GroupCampaign").Preload("Profile").Find(&flows).Error
	return flows, err
}

// GetByUserIDAndStatus retrieves flows by user ID and status
func (r *FlowRepository) GetByUserIDAndStatus(userID, status string) ([]*models.Flow, error) {
	var flows []*models.Flow
	err := r.db.Joins("JOIN group_campaigns ON flows.group_campaign_id = group_campaigns.id").
		Joins("JOIN campaigns ON group_campaigns.campaign_id = campaigns.id").
		Where("campaigns.user_id = ? AND flows.status = ?", userID, status).
		Preload("GroupCampaign").
		Preload("Profile").
		Find(&flows).Error
	return flows, err
}

// GetByUserIDAndStatusPaginated retrieves paginated flows by user ID and status
func (r *FlowRepository) GetByUserIDAndStatusPaginated(userID, status string, page, pageSize int) ([]*models.Flow, int, error) {
	var flows []*models.Flow
	var total int64

	// Count total records
	err := r.db.Joins("JOIN group_campaigns ON flows.group_campaign_id = group_campaigns.id").
		Joins("JOIN campaigns ON group_campaigns.campaign_id = campaigns.id").
		Where("campaigns.user_id = ? AND flows.status = ?", userID, status).
		Model(&models.Flow{}).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Get paginated results
	err = r.db.Joins("JOIN group_campaigns ON flows.group_campaign_id = group_campaigns.id").
		Joins("JOIN campaigns ON group_campaigns.campaign_id = campaigns.id").
		Where("campaigns.user_id = ? AND flows.status = ?", userID, status).
		Preload("GroupCampaign").
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
	err := r.db.Preload("GroupCampaign").Preload("Profile").Find(&flows).Error
	return flows, err
}

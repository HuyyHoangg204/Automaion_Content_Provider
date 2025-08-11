package repository

import (
	"green-anti-detect-browser-backend-v1/internal/models"

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
	err := r.db.Preload("Campaign").Preload("Profile").First(&flow, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &flow, nil
}

// GetByCampaignID retrieves all flows for a specific campaign
func (r *FlowRepository) GetByCampaignID(campaignID string) ([]*models.Flow, error) {
	var flows []*models.Flow
	err := r.db.Where("campaign_id = ?", campaignID).Preload("Profile").Find(&flows).Error
	return flows, err
}

// GetByProfileID retrieves all flows for a specific profile
func (r *FlowRepository) GetByProfileID(profileID string) ([]*models.Flow, error) {
	var flows []*models.Flow
	err := r.db.Where("profile_id = ?", profileID).Preload("Campaign").Find(&flows).Error
	return flows, err
}

// GetByUserID retrieves all flows for a specific user (through campaigns)
func (r *FlowRepository) GetByUserID(userID string) ([]*models.Flow, error) {
	var flows []*models.Flow
	err := r.db.Joins("JOIN campaigns ON flows.campaign_id = campaigns.id").
		Where("campaigns.user_id = ?", userID).
		Preload("Campaign").
		Preload("Profile").
		Find(&flows).Error
	return flows, err
}

// GetByUserIDAndID retrieves a flow by user ID and flow ID
func (r *FlowRepository) GetByUserIDAndID(userID, flowID string) (*models.Flow, error) {
	var flow models.Flow
	err := r.db.Joins("JOIN campaigns ON flows.campaign_id = campaigns.id").
		Where("campaigns.user_id = ? AND flows.id = ?", userID, flowID).
		Preload("Campaign").
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
	return r.db.Joins("JOIN campaigns ON flows.campaign_id = campaigns.id").
		Where("campaigns.user_id = ? AND flows.id = ?", userID, flowID).
		Delete(&models.Flow{}).Error
}

// GetByStatus retrieves flows by status
func (r *FlowRepository) GetByStatus(status string) ([]*models.Flow, error) {
	var flows []*models.Flow
	err := r.db.Where("status = ?", status).Preload("Campaign").Preload("Profile").Find(&flows).Error
	return flows, err
}

// GetByUserIDAndStatus retrieves flows by user ID and status
func (r *FlowRepository) GetByUserIDAndStatus(userID, status string) ([]*models.Flow, error) {
	var flows []*models.Flow
	err := r.db.Joins("JOIN campaigns ON flows.campaign_id = campaigns.id").
		Where("campaigns.user_id = ? AND flows.status = ?", userID, status).
		Preload("Campaign").
		Preload("Profile").
		Find(&flows).Error
	return flows, err
}

// GetAll retrieves all flows (admin only)
func (r *FlowRepository) GetAll() ([]*models.Flow, error) {
	var flows []*models.Flow
	err := r.db.Preload("Campaign").Preload("Profile").Find(&flows).Error
	return flows, err
}

package repository

import (
	"github.com/onegreenvn/green-provider-services-backend/internal/models"

	"gorm.io/gorm"
)

type FlowGroupRepository struct {
	db *gorm.DB
}

func NewFlowGroupRepository(db *gorm.DB) *FlowGroupRepository {
	return &FlowGroupRepository{db: db}
}

// Create creates a new group campaign
func (r *FlowGroupRepository) Create(flowGroup *models.FlowGroup) error {
	return r.db.Create(flowGroup).Error
}

// GetByID retrieves a group campaign by ID
func (r *FlowGroupRepository) GetByID(id string) (*models.FlowGroup, error) {
	var flowGroup models.FlowGroup
	err := r.db.Preload("Campaign").Preload("Flows").First(&flowGroup, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &flowGroup, nil
}

// GetByCampaignID retrieves all group campaigns for a specific campaign
func (r *FlowGroupRepository) GetByCampaignID(campaignID string) ([]*models.FlowGroup, error) {
	var flowGroups []*models.FlowGroup
	err := r.db.Where("campaign_id = ?", campaignID).Order("created_at DESC").Find(&flowGroups).Error
	if err != nil {
		return nil, err
	}
	return flowGroups, nil
}

// GetByCampaignIDAndUserID retrieves group campaigns for a specific campaign and user
func (r *FlowGroupRepository) GetByCampaignIDAndUserID(campaignID, userID string) ([]*models.FlowGroup, error) {
	var flowGroups []*models.FlowGroup
	err := r.db.Joins("JOIN campaigns ON flow_groups.campaign_id = campaigns.id").
		Where("flow_groups.campaign_id = ? AND campaigns.user_id = ?", campaignID, userID).
		Order("flow_groups.created_at DESC").
		Find(&flowGroups).Error
	if err != nil {
		return nil, err
	}
	return flowGroups, nil
}

// Update updates a group campaign
func (r *FlowGroupRepository) Update(flowGroup *models.FlowGroup) error {
	return r.db.Save(flowGroup).Error
}

// Delete deletes a group campaign
func (r *FlowGroupRepository) Delete(id string) error {
	return r.db.Delete(&models.FlowGroup{}, "id = ?", id).Error
}

// GetStats retrieves statistics for a group campaign
func (r *FlowGroupRepository) GetStats(id string) (*models.FlowGroupStats, error) {
	var flowGroup models.FlowGroup
	err := r.db.First(&flowGroup, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	// Count flows by status
	var totalFlows, completedFlows, failedFlows int64
	err = r.db.Model(&models.Flow{}).Where("flow_group_id = ?", id).Count(&totalFlows).Error
	if err != nil {
		return nil, err
	}

	err = r.db.Model(&models.Flow{}).Where("flow_group_id = ? AND status = ?", id, "completed").Count(&completedFlows).Error
	if err != nil {
		return nil, err
	}

	err = r.db.Model(&models.Flow{}).Where("flow_group_id = ? AND status = ?", id, "failed").Count(&failedFlows).Error
	if err != nil {
		return nil, err
	}

	// Calculate success rate
	var successRate float64
	if totalFlows > 0 {
		successRate = float64(completedFlows) / float64(totalFlows) * 100
	}

	stats := &models.FlowGroupStats{
		ID:          flowGroup.ID,
		Name:        flowGroup.Name,
		Status:      flowGroup.Status,
		SuccessRate: successRate,
	}

	return stats, nil
}

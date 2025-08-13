package repository

import (
	"green-anti-detect-browser-backend-v1/internal/models"

	"gorm.io/gorm"
)

type GroupCampaignRepository struct {
	db *gorm.DB
}

func NewGroupCampaignRepository(db *gorm.DB) *GroupCampaignRepository {
	return &GroupCampaignRepository{db: db}
}

// Create creates a new group campaign
func (r *GroupCampaignRepository) Create(groupCampaign *models.GroupCampaign) error {
	return r.db.Create(groupCampaign).Error
}

// GetByID retrieves a group campaign by ID
func (r *GroupCampaignRepository) GetByID(id string) (*models.GroupCampaign, error) {
	var groupCampaign models.GroupCampaign
	err := r.db.Preload("Campaign").Preload("Flows").First(&groupCampaign, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &groupCampaign, nil
}

// GetByCampaignID retrieves all group campaigns for a specific campaign
func (r *GroupCampaignRepository) GetByCampaignID(campaignID string) ([]*models.GroupCampaign, error) {
	var groupCampaigns []*models.GroupCampaign
	err := r.db.Where("campaign_id = ?", campaignID).Order("created_at DESC").Find(&groupCampaigns).Error
	if err != nil {
		return nil, err
	}
	return groupCampaigns, nil
}

// GetByCampaignIDAndUserID retrieves group campaigns for a specific campaign and user
func (r *GroupCampaignRepository) GetByCampaignIDAndUserID(campaignID, userID string) ([]*models.GroupCampaign, error) {
	var groupCampaigns []*models.GroupCampaign
	err := r.db.Joins("JOIN campaigns ON group_campaigns.campaign_id = campaigns.id").
		Where("group_campaigns.campaign_id = ? AND campaigns.user_id = ?", campaignID, userID).
		Order("group_campaigns.created_at DESC").
		Find(&groupCampaigns).Error
	if err != nil {
		return nil, err
	}
	return groupCampaigns, nil
}

// Update updates a group campaign
func (r *GroupCampaignRepository) Update(groupCampaign *models.GroupCampaign) error {
	return r.db.Save(groupCampaign).Error
}

// Delete deletes a group campaign
func (r *GroupCampaignRepository) Delete(id string) error {
	return r.db.Delete(&models.GroupCampaign{}, "id = ?", id).Error
}

// GetStats retrieves statistics for a group campaign
func (r *GroupCampaignRepository) GetStats(id string) (*models.GroupCampaignStats, error) {
	var groupCampaign models.GroupCampaign
	err := r.db.First(&groupCampaign, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	// Count flows by status
	var totalFlows, completedFlows, failedFlows int64
	err = r.db.Model(&models.Flow{}).Where("group_campaign_id = ?", id).Count(&totalFlows).Error
	if err != nil {
		return nil, err
	}

	err = r.db.Model(&models.Flow{}).Where("group_campaign_id = ? AND status = ?", id, "completed").Count(&completedFlows).Error
	if err != nil {
		return nil, err
	}

	err = r.db.Model(&models.Flow{}).Where("group_campaign_id = ? AND status = ?", id, "failed").Count(&failedFlows).Error
	if err != nil {
		return nil, err
	}

	// Calculate success rate
	var successRate float64
	if totalFlows > 0 {
		successRate = float64(completedFlows) / float64(totalFlows) * 100
	}

	// Calculate duration
	var duration string
	if groupCampaign.StartedAt != nil && groupCampaign.FinishedAt != nil {
		duration = groupCampaign.FinishedAt.Sub(*groupCampaign.StartedAt).String()
	}

	stats := &models.GroupCampaignStats{
		ID:          groupCampaign.ID,
		Name:        groupCampaign.Name,
		Status:      groupCampaign.Status,
		SuccessRate: successRate,
		Duration:    duration,
		StartedAt:   groupCampaign.StartedAt.Format("2006-01-02T15:04:05Z"),
	}

	if groupCampaign.FinishedAt != nil {
		stats.FinishedAt = groupCampaign.FinishedAt.Format("2006-01-02T15:04:05Z")
	}

	return stats, nil
}

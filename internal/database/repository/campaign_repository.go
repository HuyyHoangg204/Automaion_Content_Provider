package repository

import (
	"fmt"

	"github.com/onegreenvn/green-provider-services-backend/internal/models"

	"gorm.io/gorm"
)

type CampaignRepository struct {
	db *gorm.DB
}

func NewCampaignRepository(db *gorm.DB) *CampaignRepository {
	return &CampaignRepository{db: db}
}

// Create creates a new campaign
func (r *CampaignRepository) Create(campaign *models.Campaign) error {
	return r.db.Create(campaign).Error
}

// GetByID retrieves a campaign by ID
func (r *CampaignRepository) GetByID(id string) (*models.Campaign, error) {
	var campaign models.Campaign
	err := r.db.Preload("FlowGroups").
		Preload("Profiles.App.Box").
		First(&campaign, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &campaign, nil
}

// GetByUserID retrieves all campaigns for a specific user
func (r *CampaignRepository) GetByUserID(userID string) ([]*models.Campaign, error) {
	var campaigns []*models.Campaign
	err := r.db.Where("user_id = ?", userID).
		Preload("FlowGroups").
		Preload("Profiles.App.Box").
		Find(&campaigns).Error
	return campaigns, err
}

// GetByUserIDAndID retrieves a campaign by user ID and campaign ID
func (r *CampaignRepository) GetByUserIDAndID(userID, campaignID string) (*models.Campaign, error) {
	var campaign models.Campaign
	err := r.db.Where("user_id = ? AND id = ?", userID, campaignID).
		Preload("FlowGroups").
		Preload("Profiles.App.Box").
		First(&campaign).Error
	if err != nil {
		return nil, err
	}
	return &campaign, nil
}

// Update updates a campaign
func (r *CampaignRepository) Update(campaign *models.Campaign) error {
	return r.db.Save(campaign).Error
}

// DeleteByUserIDAndID deletes a campaign by user ID and campaign ID
func (r *CampaignRepository) DeleteByUserIDAndID(userID, campaignID string) error {
	result := r.db.Unscoped().Where("user_id = ?", userID).Delete(&models.Campaign{ID: campaignID})
	if result.Error != nil {
		return fmt.Errorf("failed to delete campaign: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("campaign not found or access denied")
	}
	return nil
}

// GetAll retrieves all campaigns (admin only)
func (r *CampaignRepository) GetAll() ([]*models.Campaign, error) {
	var campaigns []*models.Campaign
	err := r.db.Preload("FlowGroups").
		Preload("Profiles.App.Box").
		Find(&campaigns).Error
	return campaigns, err
}

func (r *CampaignRepository) UpdateAssociations(campaign *models.Campaign, profiles []*models.Profile) error {
	return r.db.Model(campaign).Association("Profiles").Replace(profiles)
}

// ClearProfileAssociations removes profile associations for a campaign (doesn't delete profiles)
func (r *CampaignRepository) ClearProfileAssociations(campaignID string) error {
	// First get the campaign to have the full model
	var campaign models.Campaign
	if err := r.db.First(&campaign, "id = ?", campaignID).Error; err != nil {
		return err
	}

	// Clear the associations using the loaded model
	return r.db.Model(&campaign).Association("Profiles").Clear()
}

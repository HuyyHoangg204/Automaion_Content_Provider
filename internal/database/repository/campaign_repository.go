package repository

import (
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
	err := r.db.Preload("FlowGroups").First(&campaign, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &campaign, nil
}

// GetByUserID retrieves all campaigns for a specific user
func (r *CampaignRepository) GetByUserID(userID string) ([]*models.Campaign, error) {
	var campaigns []*models.Campaign
	err := r.db.Where("user_id = ?", userID).Preload("FlowGroups").Find(&campaigns).Error
	return campaigns, err
}

// GetByUserIDAndID retrieves a campaign by user ID and campaign ID
func (r *CampaignRepository) GetByUserIDAndID(userID, campaignID string) (*models.Campaign, error) {
	var campaign models.Campaign
	err := r.db.Where("user_id = ? AND id = ?", userID, campaignID).Preload("FlowGroups").First(&campaign).Error
	if err != nil {
		return nil, err
	}
	return &campaign, nil
}

// Update updates a campaign
func (r *CampaignRepository) Update(campaign *models.Campaign) error {
	return r.db.Save(campaign).Error
}

// Delete deletes a campaign
func (r *CampaignRepository) Delete(id string) error {
	return r.db.Delete(&models.Campaign{}, "id = ?", id).Error
}

// DeleteByUserIDAndID deletes a campaign by user ID and campaign ID
func (r *CampaignRepository) DeleteByUserIDAndID(userID, campaignID string) error {
	return r.db.Where("user_id = ? AND id = ?", userID, campaignID).Delete(&models.Campaign{}).Error
}

// CheckNameExistsForUser checks if a campaign name already exists for a specific user
func (r *CampaignRepository) CheckNameExistsForUser(userID, name string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Campaign{}).Where("user_id = ? AND name = ?", userID, name).Count(&count).Error
	return count > 0, err
}

// GetAll retrieves all campaigns (admin only)
func (r *CampaignRepository) GetAll() ([]*models.Campaign, error) {
	var campaigns []*models.Campaign
	err := r.db.Preload("FlowGroups").Find(&campaigns).Error
	return campaigns, err
}

func (r *CampaignRepository) UpdateAssociations(campaign *models.Campaign, profiles []*models.Profile) error {
	return r.db.Model(campaign).Association("Profiles").Replace(profiles)
}

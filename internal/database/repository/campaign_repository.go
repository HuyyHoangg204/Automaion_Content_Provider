package repository

import (
	"green-anti-detect-browser-backend-v1/internal/models"

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
	err := r.db.Preload("Flows").First(&campaign, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &campaign, nil
}

// GetByUserID retrieves all campaigns for a specific user
func (r *CampaignRepository) GetByUserID(userID string) ([]*models.Campaign, error) {
	var campaigns []*models.Campaign
	err := r.db.Where("user_id = ?", userID).Preload("Flows").Find(&campaigns).Error
	return campaigns, err
}

// GetByUserIDAndID retrieves a campaign by user ID and campaign ID
func (r *CampaignRepository) GetByUserIDAndID(userID, campaignID string) (*models.Campaign, error) {
	var campaign models.Campaign
	err := r.db.Where("user_id = ? AND id = ?", userID, campaignID).Preload("Flows").First(&campaign).Error
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

// GetByStatus retrieves campaigns by status
func (r *CampaignRepository) GetByStatus(status string) ([]*models.Campaign, error) {
	var campaigns []*models.Campaign
	err := r.db.Where("status = ?", status).Preload("Flows").Find(&campaigns).Error
	return campaigns, err
}

// GetByUserIDAndStatus retrieves campaigns by user ID and status
func (r *CampaignRepository) GetByUserIDAndStatus(userID, status string) ([]*models.Campaign, error) {
	var campaigns []*models.Campaign
	err := r.db.Where("user_id = ? AND status = ?", userID, status).Preload("Flows").Find(&campaigns).Error
	return campaigns, err
}

// GetAll retrieves all campaigns (admin only)
func (r *CampaignRepository) GetAll() ([]*models.Campaign, error) {
	var campaigns []*models.Campaign
	err := r.db.Preload("Flows").Find(&campaigns).Error
	return campaigns, err
}

package services

import (
	"errors"

	"green-anti-detect-browser-backend-v1/internal/database/repository"
	"green-anti-detect-browser-backend-v1/internal/models"
)

type GroupCampaignService struct {
	groupCampaignRepo *repository.GroupCampaignRepository
	campaignRepo      *repository.CampaignRepository
	flowRepo          *repository.FlowRepository
}

func NewGroupCampaignService(
	groupCampaignRepo *repository.GroupCampaignRepository,
	campaignRepo *repository.CampaignRepository,
	flowRepo *repository.FlowRepository,
) *GroupCampaignService {
	return &GroupCampaignService{
		groupCampaignRepo: groupCampaignRepo,
		campaignRepo:      campaignRepo,
		flowRepo:          flowRepo,
	}
}

// GetGroupCampaignByID retrieves a group campaign by ID
func (s *GroupCampaignService) GetGroupCampaignByID(userID, groupCampaignID string) (*models.GroupCampaign, error) {
	groupCampaign, err := s.groupCampaignRepo.GetByID(groupCampaignID)
	if err != nil {
		return nil, err
	}

	// Validate that the campaign belongs to the user
	_, err = s.campaignRepo.GetByUserIDAndID(userID, groupCampaign.CampaignID)
	if err != nil {
		return nil, errors.New("access denied")
	}

	return groupCampaign, nil
}

// GetGroupCampaignsByCampaign retrieves all group campaigns for a specific campaign
func (s *GroupCampaignService) GetGroupCampaignsByCampaign(userID, campaignID string) ([]*models.GroupCampaign, error) {
	// Validate that the campaign belongs to the user
	_, err := s.campaignRepo.GetByUserIDAndID(userID, campaignID)
	if err != nil {
		return nil, errors.New("campaign not found or access denied")
	}

	return s.groupCampaignRepo.GetByCampaignIDAndUserID(campaignID, userID)
}

// GetGroupCampaignStats retrieves statistics for a group campaign
func (s *GroupCampaignService) GetGroupCampaignStats(userID, groupCampaignID string) (*models.GroupCampaignStats, error) {
	groupCampaign, err := s.groupCampaignRepo.GetByID(groupCampaignID)
	if err != nil {
		return nil, err
	}

	// Validate that the campaign belongs to the user
	_, err = s.campaignRepo.GetByUserIDAndID(userID, groupCampaign.CampaignID)
	if err != nil {
		return nil, errors.New("access denied")
	}

	return s.groupCampaignRepo.GetStats(groupCampaignID)
}

package services

import (
	"errors"

	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
)

type FlowGroupService struct {
	flowGroupRepo *repository.FlowGroupRepository
	campaignRepo  *repository.CampaignRepository
	flowRepo      *repository.FlowRepository
}

func NewFlowGroupService(
	flowGroupRepo *repository.FlowGroupRepository,
	campaignRepo *repository.CampaignRepository,
	flowRepo *repository.FlowRepository,
) *FlowGroupService {
	return &FlowGroupService{
		flowGroupRepo: flowGroupRepo,
		campaignRepo:  campaignRepo,
		flowRepo:      flowRepo,
	}
}

// GetFlowGroupByID retrieves a group campaign by ID
func (s *FlowGroupService) GetFlowGroupByUserIDAndID(userID, flowGroupID string) (*models.FlowGroup, error) {
	flowGroup, err := s.flowGroupRepo.GetByID(flowGroupID)
	if err != nil {
		return nil, err
	}

	// Validate that the campaign belongs to the user
	_, err = s.campaignRepo.GetByUserIDAndID(userID, flowGroup.CampaignID)
	if err != nil {
		return nil, errors.New("access denied")
	}

	return flowGroup, nil
}

// GetFlowGroupsByCampaign retrieves all group campaigns for a specific campaign
func (s *FlowGroupService) GetFlowGroupsByUserIDAndCampaignID(userID, campaignID string) ([]*models.FlowGroup, error) {
	// Validate that the campaign belongs to the user
	_, err := s.campaignRepo.GetByUserIDAndID(userID, campaignID)
	if err != nil {
		return nil, errors.New("campaign not found or access denied")
	}

	return s.flowGroupRepo.GetByCampaignIDAndUserID(campaignID, userID)
}

// GetFlowGroupStats retrieves statistics for a group campaign
func (s *FlowGroupService) GetFlowGroupStatsByUserIDAndID(userID, flowGroupID string) (*models.FlowGroupStats, error) {
	flowGroup, err := s.flowGroupRepo.GetByID(flowGroupID)
	if err != nil {
		return nil, err
	}

	// Validate that the campaign belongs to the user
	_, err = s.campaignRepo.GetByUserIDAndID(userID, flowGroup.CampaignID)
	if err != nil {
		return nil, errors.New("access denied")
	}

	return s.flowGroupRepo.GetStats(flowGroupID)
}

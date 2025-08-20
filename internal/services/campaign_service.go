package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
)

type CampaignService struct {
	campaignRepo  *repository.CampaignRepository
	flowGroupRepo *repository.FlowGroupRepository
	userRepo      *repository.UserRepository
}

func NewCampaignService(
	campaignRepo *repository.CampaignRepository,
	flowGroupRepo *repository.FlowGroupRepository,
	userRepo *repository.UserRepository,
) *CampaignService {
	return &CampaignService{
		campaignRepo:  campaignRepo,
		flowGroupRepo: flowGroupRepo,
		userRepo:      userRepo,
	}
}

// CreateCampaign creates a new campaign for a user
func (s *CampaignService) CreateCampaign(userID string, req *models.CreateCampaignRequest) (*models.CampaignResponse, error) {
	// Verify user exists
	_, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Check if campaign name already exists for this user
	exists, err := s.campaignRepo.CheckNameExistsForUser(userID, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check campaign name: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("campaign with name '%s' already exists", req.Name)
	}

	// Create campaign
	campaign := &models.Campaign{
		UserID:           userID,
		Name:             req.Name,
		Description:      req.Description,
		ScriptName:       req.ScriptName,
		ScriptVariables:  req.ScriptVariables,
		ConcurrentPhones: req.ConcurrentPhones,
		Schedule:         req.Schedule,
	}
	if req.IsActive != nil {
		campaign.IsActive = *req.IsActive
	}

	if err := s.campaignRepo.Create(campaign); err != nil {
		return nil, fmt.Errorf("failed to create campaign: %w", err)
	}

	return s.toResponse(campaign), nil
}

// CompleteCampaign creates a group campaign record when a campaign is completed
func (s *CampaignService) CompleteCampaign(campaignID string, startedAt *time.Time) error {
	// Get campaign to validate it exists
	_, err := s.campaignRepo.GetByID(campaignID)
	if err != nil {
		return fmt.Errorf("campaign not found: %w", err)
	}

	now := time.Now()
	flowGroup := &models.FlowGroup{
		CampaignID: campaignID,
		Name:       "Lần chạy " + now.Format("2006-01-02 15:04:05"),
		Status:     "completed",
	}

	err = s.flowGroupRepo.Create(flowGroup)
	if err != nil {
		return fmt.Errorf("failed to create group campaign: %w", err)
	}

	return nil
}

// GetCampaignsByUser retrieves all campaigns for a specific user
func (s *CampaignService) GetCampaignsByUser(userID string) ([]*models.CampaignResponse, error) {
	campaigns, err := s.campaignRepo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get campaigns: %w", err)
	}

	responses := make([]*models.CampaignResponse, len(campaigns))
	for i, campaign := range campaigns {
		responses[i] = s.toResponse(campaign)
	}

	return responses, nil
}

// GetCampaignByID retrieves a campaign by ID (user must own it)
func (s *CampaignService) GetCampaignByID(userID, campaignID string) (*models.CampaignResponse, error) {
	campaign, err := s.campaignRepo.GetByUserIDAndID(userID, campaignID)
	if err != nil {
		return nil, errors.New("campaign not found")
	}

	return s.toResponse(campaign), nil
}

// UpdateCampaign updates a campaign (user must own it)
func (s *CampaignService) UpdateCampaign(userID, campaignID string, req *models.UpdateCampaignRequest) (*models.CampaignResponse, error) {
	campaign, err := s.campaignRepo.GetByUserIDAndID(userID, campaignID)
	if err != nil {
		return nil, errors.New("campaign not found")
	}

	// Check if new name already exists for this user (if name is being changed)
	if req.Name != campaign.Name {
		exists, err := s.campaignRepo.CheckNameExistsForUser(userID, req.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to check campaign name: %w", err)
		}
		if exists {
			return nil, fmt.Errorf("campaign with name '%s' already exists", req.Name)
		}
	}

	// Update fields
	campaign.Name = req.Name
	campaign.Description = req.Description
	campaign.ScriptName = req.ScriptName
	campaign.ScriptVariables = req.ScriptVariables
	campaign.ConcurrentPhones = req.ConcurrentPhones
	campaign.Schedule = req.Schedule
	if req.IsActive != nil {
		campaign.IsActive = *req.IsActive
	}

	if err := s.campaignRepo.Update(campaign); err != nil {
		return nil, fmt.Errorf("failed to update campaign: %w", err)
	}

	return s.toResponse(campaign), nil
}

// DeleteCampaign deletes a campaign (user must own it)
func (s *CampaignService) DeleteCampaign(userID, campaignID string) error {
	// Check if campaign exists and belongs to user
	_, err := s.campaignRepo.GetByUserIDAndID(userID, campaignID)
	if err != nil {
		return errors.New("campaign not found")
	}

	if err := s.campaignRepo.DeleteByUserIDAndID(userID, campaignID); err != nil {
		return fmt.Errorf("failed to delete campaign: %w", err)
	}

	return nil
}

// GetAllCampaigns retrieves all campaigns (admin only)
func (s *CampaignService) GetAllCampaigns() ([]*models.CampaignResponse, error) {
	campaigns, err := s.campaignRepo.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get all campaigns: %w", err)
	}

	responses := make([]*models.CampaignResponse, len(campaigns))
	for i, campaign := range campaigns {
		responses[i] = s.toResponse(campaign)
	}

	return responses, nil
}

// toResponse converts Campaign model to response DTO
func (s *CampaignService) toResponse(campaign *models.Campaign) *models.CampaignResponse {
	return &models.CampaignResponse{
		ID:               campaign.ID,
		UserID:           campaign.UserID,
		Name:             campaign.Name,
		Description:      campaign.Description,
		ScriptName:       campaign.ScriptName,
		ScriptVariables:  campaign.ScriptVariables,
		ConcurrentPhones: campaign.ConcurrentPhones,
		Schedule:         campaign.Schedule,
		IsActive:         campaign.IsActive,
		Status:           campaign.Status,
		CreatedAt:        campaign.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        campaign.UpdatedAt.Format(time.RFC3339),
	}
}

package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/onegreenvn/green-provider-services-backend/internal/utils"
)

type FlowService struct {
	flowRepo      *repository.FlowRepository
	campaignRepo  *repository.CampaignRepository
	flowGroupRepo *repository.FlowGroupRepository
	profileRepo   *repository.ProfileRepository
	userRepo      *repository.UserRepository
}

func NewFlowService(
	flowRepo *repository.FlowRepository,
	campaignRepo *repository.CampaignRepository,
	flowGroupRepo *repository.FlowGroupRepository,
	profileRepo *repository.ProfileRepository,
	userRepo *repository.UserRepository,
) *FlowService {
	return &FlowService{
		flowRepo:      flowRepo,
		campaignRepo:  campaignRepo,
		flowGroupRepo: flowGroupRepo,
		profileRepo:   profileRepo,
		userRepo:      userRepo,
	}
}

// CreateFlow creates a new flow for a user
func (s *FlowService) CreateFlow(userID string, req *models.CreateFlowRequest) (*models.FlowResponse, error) {
	// Verify user exists
	_, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Verify group campaign exists and belongs to user's campaign
	flowGroup, err := s.flowGroupRepo.GetByID(req.FlowGroupID)
	if err != nil {
		return nil, errors.New("group campaign not found")
	}

	// Verify the campaign belongs to user
	_, err = s.campaignRepo.GetByUserIDAndID(userID, flowGroup.CampaignID)
	if err != nil {
		return nil, errors.New("group campaign access denied")
	}

	// Verify profile exists and belongs to user (through apps and boxes)
	_, err = s.profileRepo.GetByUserIDAndID(userID, req.ProfileID)
	if err != nil {
		return nil, errors.New("profile not found or access denied")
	}

	// Create flow
	flow := &models.Flow{
		FlowGroupID: req.FlowGroupID,
		ProfileID:   req.ProfileID,
		Status:      req.Status,
	}

	// Set StartedAt if status indicates flow has started
	if req.Status == "Started" || req.Status == "Running" || req.Status == "Completed" || req.Status == "Failed" {
		now := time.Now()
		flow.StartedAt = &now
	}

	if err := s.flowRepo.Create(flow); err != nil {
		return nil, fmt.Errorf("failed to create flow: %w", err)
	}

	return s.toResponse(flow), nil
}

// GetFlowsByUser retrieves paginated flows for a specific user
func (s *FlowService) GetFlowsByUser(userID string, page, pageSize int) ([]*models.FlowResponse, int, error) {
	// Validate and normalize pagination parameters
	page, pageSize = utils.ValidateAndNormalizePagination(page, pageSize)

	flows, total, err := s.flowRepo.GetByUserIDPaginated(userID, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get flows: %w", err)
	}

	responses := make([]*models.FlowResponse, len(flows))
	for i, flow := range flows {
		responses[i] = s.toResponse(flow)
	}

	return responses, total, nil
}

// GetFlowsByCampaign retrieves paginated flows for a specific campaign
func (s *FlowService) GetFlowsByCampaign(userID, campaignID string, page, pageSize int) ([]*models.FlowResponse, int, error) {
	// Verify campaign belongs to user
	_, err := s.campaignRepo.GetByUserIDAndID(userID, campaignID)
	if err != nil {
		return nil, 0, errors.New("campaign not found or access denied")
	}

	// Validate and normalize pagination parameters
	page, pageSize = utils.ValidateAndNormalizePagination(page, pageSize)

	flows, total, err := s.flowRepo.GetByCampaignIDPaginated(campaignID, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get flows: %w", err)
	}

	responses := make([]*models.FlowResponse, len(flows))
	for i, flow := range flows {
		responses[i] = s.toResponse(flow)
	}

	return responses, total, nil
}

// GetFlowsByFlowGroup retrieves paginated flows for a specific group campaign
func (s *FlowService) GetFlowsByFlowGroup(userID, flowGroupID string, page, pageSize int) ([]*models.FlowResponse, int, error) {
	// Verify group campaign exists and belongs to user's campaign
	flowGroup, err := s.flowGroupRepo.GetByID(flowGroupID)
	if err != nil {
		return nil, 0, errors.New("group campaign not found")
	}

	// Verify the campaign belongs to user
	_, err = s.campaignRepo.GetByUserIDAndID(userID, flowGroup.CampaignID)
	if err != nil {
		return nil, 0, errors.New("group campaign access denied")
	}

	// Validate and normalize pagination parameters
	page, pageSize = utils.ValidateAndNormalizePagination(page, pageSize)

	flows, total, err := s.flowRepo.GetByFlowGroupIDPaginated(flowGroupID, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get flows: %w", err)
	}

	responses := make([]*models.FlowResponse, len(flows))
	for i, flow := range flows {
		responses[i] = s.toResponse(flow)
	}

	return responses, total, nil
}

// GetFlowsByProfile retrieves paginated flows for a specific profile (user must own the profile)
func (s *FlowService) GetFlowsByProfile(userID, profileID string, page, pageSize int) ([]*models.FlowResponse, int, error) {
	// Verify profile belongs to user
	_, err := s.profileRepo.GetByUserIDAndID(userID, profileID)
	if err != nil {
		return nil, 0, errors.New("profile not found or access denied")
	}

	// Validate and normalize pagination parameters
	page, pageSize = utils.ValidateAndNormalizePagination(page, pageSize)

	flows, total, err := s.flowRepo.GetByProfileIDPaginated(profileID, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get flows: %w", err)
	}

	responses := make([]*models.FlowResponse, len(flows))
	for i, flow := range flows {
		responses[i] = s.toResponse(flow)
	}

	return responses, total, nil
}

// GetFlowByID retrieves a flow by ID (user must own it)
func (s *FlowService) GetFlowByID(userID, flowID string) (*models.FlowResponse, error) {
	flow, err := s.flowRepo.GetByUserIDAndID(userID, flowID)
	if err != nil {
		return nil, errors.New("flow not found")
	}

	return s.toResponse(flow), nil
}

// UpdateFlow updates a flow (user must own it)
func (s *FlowService) UpdateFlow(userID, flowID string, req *models.UpdateFlowRequest) (*models.FlowResponse, error) {
	flow, err := s.flowRepo.GetByUserIDAndID(userID, flowID)
	if err != nil {
		return nil, errors.New("flow not found")
	}

	// Update status
	flow.Status = req.Status

	// Update timestamps based on status
	now := time.Now()
	if req.Status == "Started" || req.Status == "Running" {
		if flow.StartedAt == nil {
			flow.StartedAt = &now
		}
	} else if req.Status == "Completed" || req.Status == "Failed" || req.Status == "Stopped" {
		if flow.StartedAt == nil {
			flow.StartedAt = &now
		}
		flow.FinishedAt = &now
	}

	if err := s.flowRepo.Update(flow); err != nil {
		return nil, fmt.Errorf("failed to update flow: %w", err)
	}

	return s.toResponse(flow), nil
}

// DeleteFlow deletes a flow (user must own it)
func (s *FlowService) DeleteFlow(userID, flowID string) error {
	// Check if flow exists and belongs to user
	_, err := s.flowRepo.GetByUserIDAndID(userID, flowID)
	if err != nil {
		return errors.New("flow not found")
	}

	if err := s.flowRepo.DeleteByUserIDAndID(userID, flowID); err != nil {
		return fmt.Errorf("failed to delete flow: %w", err)
	}

	return nil
}

// GetFlowsByStatus retrieves paginated flows by status for a user
func (s *FlowService) GetFlowsByStatus(userID, status string, page, pageSize int) ([]*models.FlowResponse, int, error) {
	// Validate and normalize pagination parameters
	page, pageSize = utils.ValidateAndNormalizePagination(page, pageSize)

	flows, total, err := s.flowRepo.GetByUserIDAndStatusPaginated(userID, status, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get flows: %w", err)
	}

	responses := make([]*models.FlowResponse, len(flows))
	for i, flow := range flows {
		responses[i] = s.toResponse(flow)
	}

	return responses, total, nil
}

// GetAllFlows retrieves all flows (admin only)
func (s *FlowService) GetAllFlows() ([]*models.FlowResponse, error) {
	flows, err := s.flowRepo.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get all flows: %w", err)
	}

	responses := make([]*models.FlowResponse, len(flows))
	for i, flow := range flows {
		responses[i] = s.toResponse(flow)
	}

	return responses, nil
}

// toResponse converts Flow model to response DTO
func (s *FlowService) toResponse(flow *models.Flow) *models.FlowResponse {
	response := &models.FlowResponse{
		ID:        flow.ID,
		ProfileID: flow.ProfileID,
		Status:    flow.Status,
		CreatedAt: flow.CreatedAt.Format(time.RFC3339),
		UpdatedAt: flow.UpdatedAt.Format(time.RFC3339),
	}

	if flow.StartedAt != nil {
		response.StartedAt = flow.StartedAt.Format(time.RFC3339)
	}

	if flow.FinishedAt != nil {
		response.FinishedAt = flow.FinishedAt.Format(time.RFC3339)
	}

	return response
}

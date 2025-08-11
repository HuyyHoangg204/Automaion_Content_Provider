package services

import (
	"errors"
	"fmt"
	"time"

	"green-anti-detect-browser-backend-v1/internal/database/repository"
	"green-anti-detect-browser-backend-v1/internal/models"
)

type FlowService struct {
	flowRepo     *repository.FlowRepository
	campaignRepo *repository.CampaignRepository
	profileRepo  *repository.ProfileRepository
	userRepo     *repository.UserRepository
}

func NewFlowService(flowRepo *repository.FlowRepository, campaignRepo *repository.CampaignRepository, profileRepo *repository.ProfileRepository, userRepo *repository.UserRepository) *FlowService {
	return &FlowService{
		flowRepo:     flowRepo,
		campaignRepo: campaignRepo,
		profileRepo:  profileRepo,
		userRepo:     userRepo,
	}
}

// CreateFlow creates a new flow for a user
func (s *FlowService) CreateFlow(userID string, req *models.CreateFlowRequest) (*models.FlowResponse, error) {
	// Verify user exists
	_, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Verify campaign exists and belongs to user
	_, err = s.campaignRepo.GetByUserIDAndID(userID, req.CampaignID)
	if err != nil {
		return nil, errors.New("campaign not found or access denied")
	}

	// Verify profile exists and belongs to user (through apps and boxes)
	_, err = s.profileRepo.GetByUserIDAndID(userID, req.ProfileID)
	if err != nil {
		return nil, errors.New("profile not found or access denied")
	}

	// Create flow
	flow := &models.Flow{
		CampaignID: req.CampaignID,
		ProfileID:  req.ProfileID,
		Status:     req.Status,
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

// GetFlowsByUser retrieves all flows for a specific user
func (s *FlowService) GetFlowsByUser(userID string) ([]*models.FlowResponse, error) {
	flows, err := s.flowRepo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get flows: %w", err)
	}

	responses := make([]*models.FlowResponse, len(flows))
	for i, flow := range flows {
		responses[i] = s.toResponse(flow)
	}

	return responses, nil
}

// GetFlowsByCampaign retrieves all flows for a specific campaign (user must own the campaign)
func (s *FlowService) GetFlowsByCampaign(userID, campaignID string) ([]*models.FlowResponse, error) {
	// Verify campaign belongs to user
	_, err := s.campaignRepo.GetByUserIDAndID(userID, campaignID)
	if err != nil {
		return nil, errors.New("campaign not found or access denied")
	}

	flows, err := s.flowRepo.GetByCampaignID(campaignID)
	if err != nil {
		return nil, fmt.Errorf("failed to get flows: %w", err)
	}

	responses := make([]*models.FlowResponse, len(flows))
	for i, flow := range flows {
		responses[i] = s.toResponse(flow)
	}

	return responses, nil
}

// GetFlowsByProfile retrieves all flows for a specific profile (user must own the profile)
func (s *FlowService) GetFlowsByProfile(userID, profileID string) ([]*models.FlowResponse, error) {
	// Verify profile belongs to user
	_, err := s.profileRepo.GetByUserIDAndID(userID, profileID)
	if err != nil {
		return nil, errors.New("profile not found or access denied")
	}

	flows, err := s.flowRepo.GetByProfileID(profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get flows: %w", err)
	}

	responses := make([]*models.FlowResponse, len(flows))
	for i, flow := range flows {
		responses[i] = s.toResponse(flow)
	}

	return responses, nil
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

// GetFlowsByStatus retrieves flows by status for a user
func (s *FlowService) GetFlowsByStatus(userID, status string) ([]*models.FlowResponse, error) {
	flows, err := s.flowRepo.GetByUserIDAndStatus(userID, status)
	if err != nil {
		return nil, fmt.Errorf("failed to get flows: %w", err)
	}

	responses := make([]*models.FlowResponse, len(flows))
	for i, flow := range flows {
		responses[i] = s.toResponse(flow)
	}

	return responses, nil
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
		ID:         flow.ID,
		CampaignID: flow.CampaignID,
		ProfileID:  flow.ProfileID,
		Status:     flow.Status,
		CreatedAt:  flow.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  flow.UpdatedAt.Format(time.RFC3339),
	}

	if flow.StartedAt != nil {
		response.StartedAt = flow.StartedAt.Format(time.RFC3339)
	}

	if flow.FinishedAt != nil {
		response.FinishedAt = flow.FinishedAt.Format(time.RFC3339)
	}

	return response
}

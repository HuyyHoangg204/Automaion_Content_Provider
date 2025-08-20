package handlers

import (
	"net/http"
	"time"

	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/onegreenvn/green-provider-services-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type FlowGroupHandler struct {
	flowGroupService *services.FlowGroupService
}

func NewFlowGroupHandler(flowGroupService *services.FlowGroupService) *FlowGroupHandler {
	return &FlowGroupHandler{
		flowGroupService: flowGroupService,
	}
}

// GetFlowGroupByID godoc
// @Summary Get a group campaign by ID
// @Description Get a specific group campaign by its ID
// @Tags flow-groups
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Group campaign ID"
// @Success 200 {object} models.FlowGroupResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/flow-groups/{id} [get]
func (h *FlowGroupHandler) GetFlowGroupByID(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	flowGroupID := c.Param("id")

	flowGroup, err := h.flowGroupService.GetFlowGroupByID(userID, flowGroupID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	response := &models.FlowGroupResponse{
		ID:         flowGroup.ID,
		CampaignID: flowGroup.CampaignID,
		Name:       flowGroup.Name,
		Status:     flowGroup.Status,
		CreatedAt:  flowGroup.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  flowGroup.UpdatedAt.Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, response)
}

// GetFlowGroupsByCampaign godoc
// @Summary Get all group campaigns for a campaign
// @Description Get all group campaigns for a specific campaign
// @Tags flow-groups
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Campaign ID"
// @Success 200 {array} models.FlowGroupResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/v1/campaigns/{id}/flow-groups [get]
func (h *FlowGroupHandler) GetFlowGroupsByCampaign(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	campaignID := c.Param("id")

	flowGroups, err := h.flowGroupService.GetFlowGroupsByCampaign(userID, campaignID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	responses := make([]*models.FlowGroupResponse, len(flowGroups))
	for i, flowGroup := range flowGroups {
		response := &models.FlowGroupResponse{
			ID:         flowGroup.ID,
			CampaignID: flowGroup.CampaignID,
			Name:       flowGroup.Name,
			Status:     flowGroup.Status,
			CreatedAt:  flowGroup.CreatedAt.Format(time.RFC3339),
			UpdatedAt:  flowGroup.UpdatedAt.Format(time.RFC3339),
		}
		responses[i] = response
	}

	c.JSON(http.StatusOK, responses)
}

// GetFlowGroupStats godoc
// @Summary Get group campaign statistics
// @Description Get statistics for a specific group campaign
// @Tags flow-groups
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Group campaign ID"
// @Success 200 {object} models.FlowGroupStats
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/flow-groups/{id}/stats [get]
func (h *FlowGroupHandler) GetFlowGroupStats(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	flowGroupID := c.Param("id")

	stats, err := h.flowGroupService.GetFlowGroupStats(userID, flowGroupID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

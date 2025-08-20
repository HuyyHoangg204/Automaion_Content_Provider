package handlers

import (
	"net/http"
	"time"

	"green-provider-services-backend/internal/models"
	"green-provider-services-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type GroupCampaignHandler struct {
	groupCampaignService *services.GroupCampaignService
}

func NewGroupCampaignHandler(groupCampaignService *services.GroupCampaignService) *GroupCampaignHandler {
	return &GroupCampaignHandler{
		groupCampaignService: groupCampaignService,
	}
}

// GetGroupCampaignByID godoc
// @Summary Get a group campaign by ID
// @Description Get a specific group campaign by its ID
// @Tags group-campaigns
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Group campaign ID"
// @Success 200 {object} models.GroupCampaignResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/group-campaigns/{id} [get]
func (h *GroupCampaignHandler) GetGroupCampaignByID(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	groupCampaignID := c.Param("id")

	groupCampaign, err := h.groupCampaignService.GetGroupCampaignByID(userID, groupCampaignID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	response := &models.GroupCampaignResponse{
		ID:         groupCampaign.ID,
		CampaignID: groupCampaign.CampaignID,
		Name:       groupCampaign.Name,
		Status:     groupCampaign.Status,
		CreatedAt:  groupCampaign.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  groupCampaign.UpdatedAt.Format(time.RFC3339),
	}

	if groupCampaign.StartedAt != nil {
		response.StartedAt = groupCampaign.StartedAt.Format(time.RFC3339)
	}
	if groupCampaign.FinishedAt != nil {
		response.FinishedAt = groupCampaign.FinishedAt.Format(time.RFC3339)
	}

	c.JSON(http.StatusOK, response)
}

// GetGroupCampaignsByCampaign godoc
// @Summary Get all group campaigns for a campaign
// @Description Get all group campaigns for a specific campaign
// @Tags group-campaigns
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Campaign ID"
// @Success 200 {array} models.GroupCampaignResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/v1/campaigns/{id}/group-campaigns [get]
func (h *GroupCampaignHandler) GetGroupCampaignsByCampaign(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	campaignID := c.Param("id")

	groupCampaigns, err := h.groupCampaignService.GetGroupCampaignsByCampaign(userID, campaignID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	responses := make([]*models.GroupCampaignResponse, len(groupCampaigns))
	for i, groupCampaign := range groupCampaigns {
		response := &models.GroupCampaignResponse{
			ID:         groupCampaign.ID,
			CampaignID: groupCampaign.CampaignID,
			Name:       groupCampaign.Name,
			Status:     groupCampaign.Status,
			CreatedAt:  groupCampaign.CreatedAt.Format(time.RFC3339),
			UpdatedAt:  groupCampaign.UpdatedAt.Format(time.RFC3339),
		}

		if groupCampaign.StartedAt != nil {
			response.StartedAt = groupCampaign.StartedAt.Format(time.RFC3339)
		}
		if groupCampaign.FinishedAt != nil {
			response.FinishedAt = groupCampaign.FinishedAt.Format(time.RFC3339)
		}

		responses[i] = response
	}

	c.JSON(http.StatusOK, responses)
}

// GetGroupCampaignStats godoc
// @Summary Get group campaign statistics
// @Description Get statistics for a specific group campaign
// @Tags group-campaigns
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Group campaign ID"
// @Success 200 {object} models.GroupCampaignStats
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/group-campaigns/{id}/stats [get]
func (h *GroupCampaignHandler) GetGroupCampaignStats(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	groupCampaignID := c.Param("id")

	stats, err := h.groupCampaignService.GetGroupCampaignStats(userID, groupCampaignID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

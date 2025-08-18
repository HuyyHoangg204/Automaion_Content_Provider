package handlers

import (
	"net/http"
	"strings"

	"green-provider-services-backend/internal/models"
	"green-provider-services-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type CampaignHandler struct {
	campaignService *services.CampaignService
}

func NewCampaignHandler(campaignService *services.CampaignService) *CampaignHandler {
	return &CampaignHandler{
		campaignService: campaignService,
	}
}

// CreateCampaign godoc
// @Summary Create a new campaign
// @Description Create a new campaign for the authenticated user
// @Tags campaigns
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.CreateCampaignRequest true "Create campaign request"
// @Success 201 {object} models.CampaignResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/campaigns [post]
func (h *CampaignHandler) CreateCampaign(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	var req models.CreateCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	response, err := h.campaignService.CreateCampaign(userID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create campaign", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// GetMyCampaigns godoc
// @Summary Get user's campaigns
// @Description Get all campaigns belonging to the authenticated user
// @Tags campaigns
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.CampaignResponse
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/campaigns [get]
func (h *CampaignHandler) GetMyCampaigns(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	campaigns, err := h.campaignService.GetCampaignsByUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get campaigns", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, campaigns)
}

// GetCampaignByID godoc
// @Summary Get campaign by ID
// @Description Get a specific campaign by ID (user must own it)
// @Tags campaigns
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Campaign ID"
// @Success 200 {object} models.CampaignResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/campaigns/{id} [get]
func (h *CampaignHandler) GetCampaignByID(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	campaignID := c.Param("id")

	campaign, err := h.campaignService.GetCampaignByID(userID, campaignID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Campaign not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get campaign", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, campaign)
}

// UpdateCampaign godoc
// @Summary Update campaign
// @Description Update a campaign (user must own it)
// @Tags campaigns
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Campaign ID"
// @Param request body models.UpdateCampaignRequest true "Update campaign request"
// @Success 200 {object} models.CampaignResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/campaigns/{id} [put]
func (h *CampaignHandler) UpdateCampaign(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	campaignID := c.Param("id")

	var req models.UpdateCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	response, err := h.campaignService.UpdateCampaign(userID, campaignID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Campaign not found"})
			return
		}
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update campaign", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// DeleteCampaign godoc
// @Summary Delete campaign
// @Description Delete a campaign (user must own it)
// @Tags campaigns
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Campaign ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/campaigns/{id} [delete]
func (h *CampaignHandler) DeleteCampaign(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	campaignID := c.Param("id")

	err := h.campaignService.DeleteCampaign(userID, campaignID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Campaign not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete campaign", "details": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// AdminGetAllCampaigns godoc
// @Summary Get all campaigns (Admin only)
// @Description Get all campaigns in the system (Admin privileges required)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.CampaignResponse
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/admin/campaigns [get]
func (h *CampaignHandler) AdminGetAllCampaigns(c *gin.Context) {
	// Check if user is admin
	user := c.MustGet("user").(*models.User)
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}

	campaigns, err := h.campaignService.GetAllCampaigns()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get campaigns", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, campaigns)
}

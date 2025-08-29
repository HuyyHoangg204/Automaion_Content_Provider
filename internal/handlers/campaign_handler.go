package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/onegreenvn/green-provider-services-backend/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CampaignHandler struct {
	campaignService *services.CampaignService
	rabbitMQService *services.RabbitMQService
}

func NewCampaignHandler(db *gorm.DB, rabbitMQService *services.RabbitMQService) *CampaignHandler {
	userRepo := repository.NewUserRepository(db)
	campaignRepo := repository.NewCampaignRepository(db)
	flowGroupRepo := repository.NewFlowGroupRepository(db)
	profileRepo := repository.NewProfileRepository(db)

	campaignService := services.NewCampaignService(campaignRepo, flowGroupRepo, userRepo, profileRepo)
	return &CampaignHandler{
		campaignService: campaignService,
		rabbitMQService: rabbitMQService,
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

// RunCampaign godoc
// @Summary Run a campaign
// @Description Execute a campaign by sending execution request to queue
// @Tags campaigns
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Campaign ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/campaigns/{id}/run [post]
func (h *CampaignHandler) RunCampaign(c *gin.Context) {
	// Get campaign ID from URL
	campaignID := c.Param("id")
	if campaignID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Campaign ID is required",
		})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID := c.MustGet("user_id").(string)

	// Check if campaign exists in the database for the user
	_, err := h.campaignService.GetCampaignByID(userID, campaignID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Campaign with ID %s not found", campaignID),
		})
		return
	}

	// Send execution request to event handler service
	if h.rabbitMQService != nil {
		message := map[string]interface{}{
			"type":        "execute_campaign",
			"campaign_id": campaignID,
		}
		if err := h.rabbitMQService.PublishMessage(c, "campaign_executor", message); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send execution request"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Campaign execution request sent"})
	} else {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Campaign execution service is not available"})
	}
}

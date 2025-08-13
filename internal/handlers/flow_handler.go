package handlers

import (
	"net/http"
	"strings"

	"green-anti-detect-browser-backend-v1/internal/models"
	"green-anti-detect-browser-backend-v1/internal/services"

	"github.com/gin-gonic/gin"
)

type FlowHandler struct {
	flowService *services.FlowService
}

func NewFlowHandler(flowService *services.FlowService) *FlowHandler {
	return &FlowHandler{
		flowService: flowService,
	}
}

// CreateFlow godoc
// @Summary Create a new flow
// @Description Create a new flow for the authenticated user
// @Tags flows
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.CreateFlowRequest true "Create flow request"
// @Success 201 {object} models.FlowResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/flows [post]
func (h *FlowHandler) CreateFlow(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	var req models.CreateFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	response, err := h.flowService.CreateFlow(userID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "access denied") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create flow", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// GetMyFlows godoc
// @Summary Get user's flows
// @Description Get all flows belonging to the authenticated user
// @Tags flows
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.FlowResponse
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/flows [get]
func (h *FlowHandler) GetMyFlows(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	flows, err := h.flowService.GetFlowsByUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get flows", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, flows)
}

// GetFlowsByCampaign godoc
// @Summary Get flows by campaign
// @Description Get all flows for a specific campaign (user must own the campaign)
// @Tags flows
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param campaign_id path string true "Campaign ID" format(uuid)
// @Success 200 {array} models.FlowResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/campaign-flows/{campaign_id}/flows [get]
func (h *FlowHandler) GetFlowsByCampaign(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	campaignID := c.Param("campaign_id")

	// Validate UUID format
	if !isValidUUID(campaignID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid campaign ID format"})
		return
	}

	flows, err := h.flowService.GetFlowsByCampaign(userID, campaignID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "access denied") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get flows", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, flows)
}

// GetFlowsByGroupCampaign godoc
// @Summary Get flows by group campaign
// @Description Get all flows for a specific group campaign (user must own the campaign)
// @Tags flows
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param group_campaign_id path string true "Group Campaign ID" format(uuid)
// @Success 200 {array} models.FlowResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/group-campaign-flows/{group_campaign_id}/flows [get]
func (h *FlowHandler) GetFlowsByGroupCampaign(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	groupCampaignID := c.Param("group_campaign_id")

	// Validate UUID format
	if !isValidUUID(groupCampaignID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group campaign ID format"})
		return
	}

	flows, err := h.flowService.GetFlowsByGroupCampaign(userID, groupCampaignID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "access denied") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get flows", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, flows)
}

// GetFlowsByProfile godoc
// @Summary Get flows by profile
// @Description Get all flows for a specific profile (user must own the profile)
// @Tags flows
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param profile_id path string true "Profile ID" format(uuid)
// @Success 200 {array} models.FlowResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/profile-flows/{profile_id}/flows [get]
func (h *FlowHandler) GetFlowsByProfile(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	profileID := c.Param("profile_id")

	// Validate UUID format
	if !isValidUUID(profileID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid profile ID format"})
		return
	}

	flows, err := h.flowService.GetFlowsByProfile(userID, profileID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "access denied") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get flows", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, flows)
}

// GetFlowByID godoc
// @Summary Get flow by ID
// @Description Get a specific flow by ID (user must own it)
// @Tags flows
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Flow ID" format(uuid)
// @Success 200 {object} models.FlowResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/flows/{id} [get]
func (h *FlowHandler) GetFlowByID(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	flowID := c.Param("id")

	// Validate UUID format
	if !isValidUUID(flowID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid flow ID format"})
		return
	}

	flow, err := h.flowService.GetFlowByID(userID, flowID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Flow not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get flow", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, flow)
}

// UpdateFlow godoc
// @Summary Update flow
// @Description Update a flow (user must own it)
// @Tags flows
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Flow ID" format(uuid)
// @Param request body models.UpdateFlowRequest true "Update flow request"
// @Success 200 {object} models.FlowResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/flows/{id} [put]
func (h *FlowHandler) UpdateFlow(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	flowID := c.Param("id")

	// Validate UUID format
	if !isValidUUID(flowID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid flow ID format"})
		return
	}

	var req models.UpdateFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	response, err := h.flowService.UpdateFlow(userID, flowID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Flow not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update flow", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// DeleteFlow godoc
// @Summary Delete flow
// @Description Delete a flow (user must own it)
// @Tags flows
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Flow ID" format(uuid)
// @Success 204 "No Content"
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/flows/{id} [delete]
func (h *FlowHandler) DeleteFlow(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	flowID := c.Param("id")

	// Validate UUID format
	if !isValidUUID(flowID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid flow ID format"})
		return
	}

	err := h.flowService.DeleteFlow(userID, flowID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Flow not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete flow", "details": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetFlowsByStatus godoc
// @Summary Get flows by status
// @Description Get all flows for a specific status (user must own them)
// @Tags flows
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param status path string true "Flow Status" Enums(Started, Running, Completed, Failed, Stopped)
// @Success 200 {array} models.FlowResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/flows/status/{status} [get]
func (h *FlowHandler) GetFlowsByStatus(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	status := c.Param("status")

	// Validate status
	validStatuses := []string{"Started", "Running", "Completed", "Failed", "Stopped"}
	isValidStatus := false
	for _, validStatus := range validStatuses {
		if status == validStatus {
			isValidStatus = true
			break
		}
	}
	if !isValidStatus {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status. Must be one of: Started, Running, Completed, Failed, Stopped"})
		return
	}

	flows, err := h.flowService.GetFlowsByStatus(userID, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get flows", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, flows)
}

// AdminGetAllFlows godoc
// @Summary Get all flows (Admin only)
// @Description Get all flows in the system (Admin privileges required)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.FlowResponse
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/admin/flows [get]
func (h *FlowHandler) AdminGetAllFlows(c *gin.Context) {
	// Check if user is admin
	user := c.MustGet("user").(*models.User)
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}

	flows, err := h.flowService.GetAllFlows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get flows", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, flows)
}

// isValidUUID checks if a string is a valid UUID
func isValidUUID(uuid string) bool {
	// Simple UUID validation - check length and format
	if len(uuid) != 36 {
		return false
	}

	// Check if it matches UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	for i, char := range uuid {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if char != '-' {
				return false
			}
		} else {
			if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
				return false
			}
		}
	}
	return true
}

package handlers

import (
	"net/http"
	"strings"

	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/onegreenvn/green-provider-services-backend/internal/services"
	"github.com/onegreenvn/green-provider-services-backend/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type FlowHandler struct {
	flowService *services.FlowService
}

func NewFlowHandler(db *gorm.DB) *FlowHandler {
	userRepo := repository.NewUserRepository(db)
	campaignRepo := repository.NewCampaignRepository(db)
	flowGroupRepo := repository.NewFlowGroupRepository(db)
	flowRepo := repository.NewFlowRepository(db)
	profileRepo := repository.NewProfileRepository(db)

	flowService := services.NewFlowService(flowRepo, campaignRepo, flowGroupRepo, profileRepo, userRepo)
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
// @Description Get all flows belonging to the authenticated user with pagination
// @Tags flows
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)" minimum(1)
// @Param limit query int false "Number of items per page (default: 20, max: 100)" minimum(1) maximum(100)
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/flows [get]
func (h *FlowHandler) GetMyFlows(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	// Parse query parameters
	page, pageSize := utils.ParsePaginationFromQuery(c.Query("page"), c.Query("limit"))

	flows, total, err := h.flowService.GetFlowsByUser(userID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get flows", "details": err.Error()})
		return
	}

	// Calculate pagination info
	paginationInfo := utils.CalculatePaginationInfo(total, page, pageSize)

	response := gin.H{
		"data":         flows,
		"total":        total,
		"page":         paginationInfo.Page,
		"limit":        paginationInfo.PageSize,
		"total_pages":  paginationInfo.TotalPages,
		"has_next":     paginationInfo.HasNext,
		"has_previous": paginationInfo.HasPrevious,
	}
	c.JSON(http.StatusOK, response)
}

// GetFlowsByCampaign godoc
// @Summary Get flows by campaign
// @Description Get all flows for a specific campaign (user must own the campaign) with pagination
// @Tags flows
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param campaign_id path string true "Campaign ID"
// @Param page query int false "Page number (default: 1)" minimum(1)
// @Param limit query int false "Number of items per page (default: 20, max: 100)" minimum(1) maximum(100)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/campaign-flows/{campaign_id}/flows [get]
func (h *FlowHandler) GetFlowsByCampaign(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	campaignID := c.Param("campaign_id")

	// Parse query parameters
	page, pageSize := utils.ParsePaginationFromQuery(c.Query("page"), c.Query("limit"))

	flows, total, err := h.flowService.GetFlowsByCampaign(userID, campaignID, page, pageSize)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "access denied") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get flows", "details": err.Error()})
		return
	}

	// Calculate pagination info
	paginationInfo := utils.CalculatePaginationInfo(total, page, pageSize)

	response := gin.H{
		"data":         flows,
		"total":        total,
		"page":         paginationInfo.Page,
		"limit":        paginationInfo.PageSize,
		"total_pages":  paginationInfo.TotalPages,
		"has_next":     paginationInfo.HasNext,
		"has_previous": paginationInfo.HasPrevious,
	}
	c.JSON(http.StatusOK, response)
}

// GetFlowsByFlowGroup godoc
// @Summary Get flows by group campaign
// @Description Get all flows for a specific group campaign (user must own the campaign) with pagination
// @Tags flows
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param flow_group_id path string true "Group Campaign ID"
// @Param page query int false "Page number (default: 1)" minimum(1)
// @Param limit query int false "Number of items per page (default: 20, max: 100)" minimum(1) maximum(100)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/flow-group-flows/{flow_group_id}/flows [get]
func (h *FlowHandler) GetFlowsByFlowGroup(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	flowGroupID := c.Param("flow_group_id")

	// Parse query parameters
	page, pageSize := utils.ParsePaginationFromQuery(c.Query("page"), c.Query("limit"))

	flows, total, err := h.flowService.GetFlowsByFlowGroup(userID, flowGroupID, page, pageSize)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "access denied") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get flows", "details": err.Error()})
		return
	}

	// Calculate pagination info
	paginationInfo := utils.CalculatePaginationInfo(total, page, pageSize)

	response := gin.H{
		"data":         flows,
		"total":        total,
		"page":         paginationInfo.Page,
		"limit":        paginationInfo.PageSize,
		"total_pages":  paginationInfo.TotalPages,
		"has_next":     paginationInfo.HasNext,
		"has_previous": paginationInfo.HasPrevious,
	}
	c.JSON(http.StatusOK, response)
}

// GetFlowsByProfile godoc
// @Summary Get flows by profile
// @Description Get all flows for a specific profile (user must own the profile) with pagination
// @Tags flows
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param profile_id path string true "Profile ID"
// @Param page query int false "Page number (default: 1)" minimum(1)
// @Param limit query int false "Number of items per page (default: 20, max: 100)" minimum(1) maximum(100)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/profile-flows/{profile_id}/flows [get]
func (h *FlowHandler) GetFlowsByProfile(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	profileID := c.Param("profile_id")

	// Parse query parameters
	page, pageSize := utils.ParsePaginationFromQuery(c.Query("page"), c.Query("limit"))

	flows, total, err := h.flowService.GetFlowsByProfile(userID, profileID, page, pageSize)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "access denied") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get flows", "details": err.Error()})
		return
	}

	// Calculate pagination info
	paginationInfo := utils.CalculatePaginationInfo(total, page, pageSize)

	response := gin.H{
		"data":         flows,
		"total":        total,
		"page":         paginationInfo.Page,
		"limit":        paginationInfo.PageSize,
		"total_pages":  paginationInfo.TotalPages,
		"has_next":     paginationInfo.HasNext,
		"has_previous": paginationInfo.HasPrevious,
	}
	c.JSON(http.StatusOK, response)
}

// GetFlowByID godoc
// @Summary Get flow by ID
// @Description Get a specific flow by ID (user must own it)
// @Tags flows
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Flow ID"
// @Success 200 {object} models.FlowResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/flows/{id} [get]
func (h *FlowHandler) GetFlowByID(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	flowID := c.Param("id")

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
// @Param id path string true "Flow ID"
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
// @Param id path string true "Flow ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/flows/{id} [delete]
func (h *FlowHandler) DeleteFlow(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	flowID := c.Param("id")

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
// @Description Get all flows for a specific status (user must own them) with pagination
// @Tags flows
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param status path string true "Flow Status" Enums(Started, Running, Completed, Failed, Stopped)
// @Param page query int false "Page number (default: 1)" minimum(1)
// @Param limit query int false "Number of items per page (default: 20, max: 100)" minimum(1) maximum(100)
// @Success 200 {object} map[string]interface{}
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

	// Parse query parameters
	page, pageSize := utils.ParsePaginationFromQuery(c.Query("page"), c.Query("limit"))

	flows, total, err := h.flowService.GetFlowsByStatus(userID, status, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get flows", "details": err.Error()})
		return
	}

	// Calculate pagination info
	paginationInfo := utils.CalculatePaginationInfo(total, page, pageSize)

	response := gin.H{
		"data":         flows,
		"total":        total,
		"page":         paginationInfo.Page,
		"limit":        paginationInfo.PageSize,
		"total_pages":  paginationInfo.TotalPages,
		"has_next":     paginationInfo.HasNext,
		"has_previous": paginationInfo.HasPrevious,
	}
	c.JSON(http.StatusOK, response)
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

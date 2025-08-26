package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/onegreenvn/green-provider-services-backend/internal/services"
	"github.com/onegreenvn/green-provider-services-backend/internal/utils"
)

type BoxHandler struct {
	boxService *services.BoxService
}

func NewBoxHandler(boxService *services.BoxService) *BoxHandler {
	return &BoxHandler{
		boxService: boxService,
	}
}

// CreateBox godoc
// @Summary Create a new box
// @Description Create a new box for the authenticated user
// @Tags boxes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.CreateBoxRequest true "Create box request"
// @Success 201 {object} models.BoxResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/boxes [post]
func (h *BoxHandler) CreateBox(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	var req models.CreateBoxRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	response, err := h.boxService.CreateBox(userID, &req)
	if err != nil {
		// Check if it's a BoxAlreadyExistsError
		if boxExistsErr, ok := err.(*models.BoxAlreadyExistsError); ok {
			c.JSON(http.StatusConflict, gin.H{
				"error":      boxExistsErr.Message,
				"box_id":     boxExistsErr.BoxID,
				"machine_id": boxExistsErr.MachineID,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create box", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// GetMyBoxes godoc
// @Summary Get user's boxes
// @Description Get all boxes belonging to the authenticated user with pagination
// @Tags boxes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)" minimum(1)
// @Param limit query int false "Number of items per page (default: 20, max: 100)" minimum(1) maximum(100)
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/boxes [get]
func (h *BoxHandler) GetMyBoxes(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	// Parse query parameters
	page, pageSize := utils.ParsePaginationFromQuery(c.Query("page"), c.Query("limit"))

	boxes, total, err := h.boxService.GetBoxesByUserPaginated(userID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get boxes", "details": err.Error()})
		return
	}

	// Calculate pagination info
	paginationInfo := utils.CalculatePaginationInfo(total, page, pageSize)

	response := gin.H{
		"data":         boxes,
		"total":        total,
		"page":         paginationInfo.Page,
		"limit":        paginationInfo.PageSize,
		"total_pages":  paginationInfo.TotalPages,
		"has_next":     paginationInfo.HasNext,
		"has_previous": paginationInfo.HasPrevious,
	}
	c.JSON(http.StatusOK, response)
}

// GetBoxByID godoc
// @Summary Get box by ID
// @Description Get a specific box by ID (user must own it)
// @Tags boxes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Box ID"
// @Success 200 {object} models.BoxResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/boxes/{id} [get]
func (h *BoxHandler) GetBoxByID(c *gin.Context) {
	// Get user ID from context
	userID := c.MustGet("user_id").(string)

	// Get box ID from URL
	boxID := c.Param("id")

	box, err := h.boxService.GetBoxByID(userID, boxID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Box not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get box", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, box)
}

// UpdateBox godoc
// @Summary Update box
// @Description Update a box. Can update both name and user_id. If user_id is provided, it must be a valid user ID.
// @Tags boxes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Box ID"
// @Param request body models.UpdateBoxRequest true "Update box request - name is required, user_id is optional for ownership transfer"
// @Success 200 {object} models.BoxResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/boxes/{id} [put]
func (h *BoxHandler) UpdateBox(c *gin.Context) {
	// Get user ID from context
	userID := c.MustGet("user_id").(string)

	// Get box ID from URL
	boxID := c.Param("id")

	var req models.UpdateBoxRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	response, err := h.boxService.UpdateBox(userID, boxID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "box not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Box not found"})
			return
		}
		if strings.Contains(err.Error(), "new user not found") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update box", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// DeleteBox godoc
// @Summary Delete box
// @Description Delete a box (user must own it)
// @Tags boxes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Box ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/boxes/{id} [delete]
func (h *BoxHandler) DeleteBox(c *gin.Context) {
	// Get user ID from context
	userID := c.MustGet("user_id").(string)

	// Get box ID from URL
	boxID := c.Param("id")

	err := h.boxService.DeleteBox(userID, boxID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Box not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete box", "details": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// AdminGetAllBoxes godoc
// @Summary Get all boxes (Admin only)
// @Description Get all boxes in the system (Admin privileges required)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.BoxResponse
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/admin/boxes [get]
func (h *BoxHandler) AdminGetAllBoxes(c *gin.Context) {
	// Check if user is admin
	user := c.MustGet("user").(*models.User)
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}

	boxes, err := h.boxService.GetAllBoxes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get boxes", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, boxes)
}

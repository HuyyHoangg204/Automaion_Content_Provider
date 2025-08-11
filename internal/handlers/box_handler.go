package handlers

import (
	"net/http"
	"strings"

	"green-anti-detect-browser-backend-v1/internal/models"
	"green-anti-detect-browser-backend-v1/internal/services"

	"github.com/gin-gonic/gin"
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
		if strings.Contains(err.Error(), "machine ID already exists") {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create box", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// GetMyBoxes godoc
// @Summary Get user's boxes
// @Description Get all boxes belonging to the authenticated user
// @Tags boxes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.BoxResponse
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/boxes [get]
func (h *BoxHandler) GetMyBoxes(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	boxes, err := h.boxService.GetBoxesByUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get boxes", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, boxes)
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
// @Description Update a box (user must own it)
// @Tags boxes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Box ID"
// @Param request body models.UpdateBoxRequest true "Update box request"
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
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Box not found"})
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

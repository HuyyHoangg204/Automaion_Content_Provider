package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/onegreenvn/green-provider-services-backend/internal/services"
	"github.com/onegreenvn/green-provider-services-backend/internal/utils"
	"gorm.io/gorm"
)

type BoxHandler struct {
	boxService *services.BoxService
	appService *services.AppService
}

func NewBoxHandler(db *gorm.DB) *BoxHandler {
	// Create repositories
	userRepo := repository.NewUserRepository(db)
	boxRepo := repository.NewBoxRepository(db)
	appRepo := repository.NewAppRepository(db)
	profileRepo := repository.NewProfileRepository(db)

	// Create service
	boxService := services.NewBoxService(boxRepo, userRepo, profileRepo)
	appService := services.NewAppService(appRepo, profileRepo, boxRepo, userRepo)
	return &BoxHandler{
		boxService: boxService,
		appService: appService,
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

	response, err := h.boxService.CreateBoxByUserID(userID, &req)
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

	boxes, total, err := h.boxService.GetBoxesByUserIDPaginated(userID, page, pageSize)
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

	box, err := h.boxService.GetBoxByUserIDAndID(userID, boxID)
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
// @Summary Update box name and ownership
// @Description Update a box name and automatically assign ownership to the current logged-in user
// @Tags boxes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Box ID"
// @Param request body models.UpdateBoxRequest true "Update box request - name is required"
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

	response, err := h.boxService.UpdateBoxByUserIDAndID(userID, boxID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "box not found") {
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

	err := h.boxService.DeleteBoxByUserIDAndID(userID, boxID)
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

// SyncAllProfilesInBox syncs all profiles from all apps in a specific box
// @Summary Sync all profiles from all apps in a box
// @Description Sync all profiles from all apps in a specific box
// @Tags boxes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Box ID to sync profiles from"
// @Success 200 {object} models.SyncBoxProfilesResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/boxes/{id}/sync-profiles [post]
func (h *BoxHandler) SyncAllProfilesInBox(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	boxID := c.Param("id")

	// Sync all profiles from all apps in the box
	syncResult, err := h.appService.SyncAllAppsInBox(userID, boxID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, syncResult)
}

// GetAppsByBox godoc
// @Summary Get apps by box
// @Description Get all apps for a specific box (user must own the box)
// @Tags boxes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Box ID"
// @Success 200 {array} models.App
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/boxes/{id}/apps [get]
func (h *BoxHandler) GetAppsByBox(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	boxID := c.Param("id")

	apps, err := h.appService.GetAppsByUserIDAndBoxID(userID, boxID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "access denied") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get apps", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, apps)
}

package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/onegreenvn/green-provider-services-backend/internal/services"
	"github.com/onegreenvn/green-provider-services-backend/internal/utils"
)

type ProfileHandler struct {
	profileService *services.ProfileService
}

func NewProfileHandler(profileService *services.ProfileService) *ProfileHandler {
	return &ProfileHandler{
		profileService: profileService,
	}
}

// CreateProfile godoc
// @Summary Create a new profile
// @Description Create a new browser profile for the authenticated user. Profile data is required and must contain 'name' field along with configuration from anti-detect browser.
// @Tags profiles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.CreateProfileRequest true "Create profile request (data field must include 'name' and can contain other configuration parameters)"
// @Success 201 {object} models.ProfileResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/profiles [post]
func (h *ProfileHandler) CreateProfile(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	var req models.CreateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	response, err := h.profileService.CreateProfile(context.Background(), userID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "access denied") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "profile data is required") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create profile", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// GetMyProfiles godoc
// @Summary Get user's profiles
// @Description Get all profiles for a specific box (user must own the box) with pagination
// @Tags profiles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param box_id query string true "Box ID"
// @Param page query int false "Page number (default: 1)" minimum(1)
// @Param limit query int false "Number of items per page (default: 20, max: 100)" minimum(1) maximum(100)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/profiles [get]
func (h *ProfileHandler) GetMyProfiles(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	boxID := c.Query("box_id")

	if boxID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "box_id is required"})
		return
	}

	// Parse query parameters
	page, pageSize := utils.ParsePaginationFromQuery(c.Query("page"), c.Query("limit"))

	profiles, total, err := h.profileService.GetProfilesByBoxPaginated(context.Background(), userID, boxID, page, pageSize)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "access denied") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get profiles", "details": err.Error()})
		return
	}

	// Calculate pagination info
	paginationInfo := utils.CalculatePaginationInfo(total, page, pageSize)

	response := gin.H{
		"data":         profiles,
		"total":        total,
		"page":         paginationInfo.Page,
		"limit":        paginationInfo.PageSize,
		"total_pages":  paginationInfo.TotalPages,
		"has_next":     paginationInfo.HasNext,
		"has_previous": paginationInfo.HasPrevious,
	}
	c.JSON(http.StatusOK, response)
}

// GetProfilesByApp godoc
// @Summary Get profiles by app
// @Description Get all profiles for a specific app (user must own the app) with pagination
// @Tags profiles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param app_id path string true "App ID"
// @Param page query int false "Page number (default: 1)" minimum(1)
// @Param limit query int false "Number of items per page (default: 20, max: 100)" minimum(1) maximum(100)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/app-profiles/{app_id}/profiles [get]
func (h *ProfileHandler) GetProfilesByApp(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	appID := c.Param("app_id")

	// Parse query parameters
	page, pageSize := utils.ParsePaginationFromQuery(c.Query("page"), c.Query("limit"))

	profiles, total, err := h.profileService.GetProfilesByAppPaginated(context.Background(), userID, appID, page, pageSize)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "access denied") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get profiles", "details": err.Error()})
		return
	}

	// Calculate pagination info
	paginationInfo := utils.CalculatePaginationInfo(total, page, pageSize)

	response := gin.H{
		"data":         profiles,
		"total":        total,
		"page":         paginationInfo.Page,
		"limit":        paginationInfo.PageSize,
		"total_pages":  paginationInfo.TotalPages,
		"has_next":     paginationInfo.HasNext,
		"has_previous": paginationInfo.HasPrevious,
	}
	c.JSON(http.StatusOK, response)
}

// GetProfileByID godoc
// @Summary Get profile by ID
// @Description Get a specific profile by ID (user must own it)
// @Tags profiles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Profile ID"
// @Success 200 {object} models.ProfileResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/profiles/{id} [get]
func (h *ProfileHandler) GetProfileByID(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	profileID := c.Param("id")

	profile, err := h.profileService.GetProfileByID(context.Background(), userID, profileID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get profile", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, profile)
}

// UpdateProfile godoc
// @Summary Update profile
// @Description Update a profile (user must own it)
// @Tags profiles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Profile ID"
// @Param request body models.UpdateProfileRequest true "Update profile request"
// @Success 200 {object} models.ProfileResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/profiles/{id} [put]
func (h *ProfileHandler) UpdateProfile(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	profileID := c.Param("id")

	var req models.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	response, err := h.profileService.UpdateProfile(context.Background(), userID, profileID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found"})
			return
		}
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// DeleteProfile godoc
// @Summary Delete profile
// @Description Delete a profile (user must own it)
// @Tags profiles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Profile ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/profiles/{id} [delete]
func (h *ProfileHandler) DeleteProfile(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	profileID := c.Param("id")

	err := h.profileService.DeleteProfile(context.Background(), userID, profileID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete profile", "details": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// AdminGetAllProfiles godoc
// @Summary Get all profiles (Admin only)
// @Description Get all profiles in the system (Admin privileges required)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.ProfileResponse
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/admin/profiles [get]
func (h *ProfileHandler) AdminGetAllProfiles(c *gin.Context) {
	// Check if user is admin
	user := c.MustGet("user").(*models.User)
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}

	profiles, err := h.profileService.GetAllProfiles()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get profiles", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, profiles)
}

// GetDefaultConfigs godoc
// @Summary Get default configurations from platform
// @Description Get default configuration options available for creating profiles on a specific platform (Hidemium, Genlogin, etc.)
// @Tags profiles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param platform query string true "Platform type (hidemium, genlogin)"
// @Param box_id query string true "Box ID (used to resolve machine_id for tunnel)"
// @Param page query int false "Page number (default: 1)" minimum(1)
// @Param limit query int false "Number of items per page (default: 10, max: 100)" minimum(1) maximum(100)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/profiles/default-configs [get]
func (h *ProfileHandler) GetDefaultConfigs(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	platformType := c.Query("platform")
	boxID := c.Query("box_id")

	if platformType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "platform parameter is required"})
		return
	}
	if boxID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "box_id parameter is required"})
		return
	}

	// Validate platform type
	validPlatforms := []string{"hidemium", "genlogin"}
	isValid := false
	for _, valid := range validPlatforms {
		if platformType == valid {
			isValid = true
			break
		}
	}

	if !isValid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid platform type. Supported platforms: hidemium, genlogin"})
		return
	}

	// Parse pagination parameters
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}

	// Get default configs from platform
	configs, err := h.profileService.GetDefaultConfigsFromPlatform(context.Background(), userID, platformType, boxID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get default configs", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, configs)
}

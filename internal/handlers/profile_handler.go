package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/services"
	"github.com/onegreenvn/green-provider-services-backend/internal/utils"
	"gorm.io/gorm"
)

type ProfileHandler struct {
	profileService *services.ProfileService
	profileRepo    *repository.ProfileRepository
}

func NewProfileHandler(db *gorm.DB) *ProfileHandler {
	userRepo := repository.NewUserRepository(db)
	boxRepo := repository.NewBoxRepository(db)
	appRepo := repository.NewAppRepository(db)
	profileRepo := repository.NewProfileRepository(db)

	profileService := services.NewProfileService(profileRepo, appRepo, userRepo, boxRepo)
	return &ProfileHandler{
		profileService: profileService,
		profileRepo:    profileRepo,
	}
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

	profiles, total, err := h.profileService.GetProfilesByBoxPaginated(userID, boxID, page, pageSize)
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
// @Param id path string true "App ID"
// @Param page query int false "Page number (default: 1)" minimum(1)
// @Param limit query int false "Number of items per page (default: 20, max: 100)" minimum(1) maximum(100)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/apps/{id}/profiles [get]
func (h *ProfileHandler) GetProfilesByApp(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	appID := c.Param("id")

	// Parse query parameters
	page, pageSize := utils.ParsePaginationFromQuery(c.Query("page"), c.Query("limit"))

	profiles, total, err := h.profileService.GetProfilesByAppPaginated(userID, appID, page, pageSize)
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

	profile, err := h.profileService.GetProfileByID(userID, profileID)
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

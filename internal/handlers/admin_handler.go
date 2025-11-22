package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/onegreenvn/green-provider-services-backend/internal/services"
	"github.com/onegreenvn/green-provider-services-backend/internal/services/auth"
	"github.com/onegreenvn/green-provider-services-backend/internal/utils"
	"gorm.io/gorm"
)

type AdminHandler struct {
	authService *auth.AuthService
	boxService  *services.BoxService
	appService  *services.AppService
}

func NewAdminHandler(authService *auth.AuthService, db *gorm.DB) *AdminHandler {
	// Create repositories
	userRepo := repository.NewUserRepository(db)
	boxRepo := repository.NewBoxRepository(db)
	appRepo := repository.NewAppRepository(db)

	return &AdminHandler{
		authService: authService,
		boxService:  services.NewBoxService(boxRepo, userRepo),
		appService:  services.NewAppService(appRepo, boxRepo, userRepo),
	}
}

// Register godoc
// @Summary Register a new user (Admin only)
// @Description Register a new user account with username and password (Admin privileges required)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.RegisterRequest true "Registration request (username, password, first_name, last_name)"
// @Success 201 {object} models.AuthResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/admin/register [post]
func (h *AdminHandler) Register(c *gin.Context) {
	// Check if user is admin
	user := c.MustGet("user").(*models.User)
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}

	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	response, err := h.authService.Register(&req)
	if err != nil {
		if strings.Contains(err.Error(), "username already exists") {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// GetAllUsers godoc
// @Summary Get all users (Admin only)
// @Description Get list of all users in the system with pagination and search (Admin privileges required)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)" minimum(1)
// @Param limit query int false "Number of items per page (default: 10, max: 100)" minimum(1) maximum(100)
// @Param search query string false "Search term for username"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/admin/users [get]
func (h *AdminHandler) GetAllUsers(c *gin.Context) {
	// Check if user is admin
	user := c.MustGet("user").(*models.User)
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}

	// Parse query parameters
	page, pageSize := utils.ParsePaginationFromQuery(c.Query("page"), c.Query("limit"))
	search := c.DefaultQuery("search", "")

	users, total, err := h.authService.GetAllUsers(page, pageSize, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get users", "details": err.Error()})
		return
	}

	// Calculate pagination info
	paginationInfo := utils.CalculatePaginationInfo(int(total), page, pageSize)

	response := gin.H{
		"data":         users,
		"total":        total,
		"page":         paginationInfo.Page,
		"limit":        paginationInfo.PageSize,
		"total_pages":  paginationInfo.TotalPages,
		"has_next":     paginationInfo.HasNext,
		"has_previous": paginationInfo.HasPrevious,
	}
	c.JSON(http.StatusOK, response)
}

// SetUserStatus godoc
// @Summary Set user active status (Admin only)
// @Description Set the active status of a user account (Admin privileges required)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param request body map[string]bool true "Status request {\"is_active\": true/false}"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/admin/users/{id}/status [put]
func (h *AdminHandler) SetUserStatus(c *gin.Context) {
	// Check if user is admin
	user := c.MustGet("user").(*models.User)
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}

	// Get user ID from URL
	userID := c.Param("id")

	// Parse request body
	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Set user status
	err := h.authService.SetUserActive(userID, req.IsActive)
	if err != nil {
		if strings.Contains(err.Error(), "user not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user status", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User status updated successfully"})
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
func (h *AdminHandler) AdminGetAllBoxes(c *gin.Context) {
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

// AdminGetAllApps godoc
// @Summary Get all apps (Admin only)
// @Description Get all apps in the system (Admin privileges required)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.App
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/admin/apps [get]
func (h *AdminHandler) AdminGetAllApps(c *gin.Context) {
	// Check if user is admin
	user := c.MustGet("user").(*models.User)
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}

	apps, err := h.appService.GetAllApps()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get apps", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, apps)
}

// ResetPassword godoc
// @Summary Reset user password (Admin only)
// @Description Reset a user's password to a new password specified by admin (Admin privileges required)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param request body models.ResetPasswordRequest true "New password request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/admin/users/{id}/reset-password [post]
func (h *AdminHandler) ResetPassword(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}
	var req models.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	err := h.authService.ResetPassword(userID, req.NewPassword)
	if err != nil {
		if strings.Contains(err.Error(), "user not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset password", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Password reset successfully",
		"user_id": userID,
	})
}

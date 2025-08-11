package handlers

import (
	"net/http"
	"strings"

	"green-anti-detect-browser-backend-v1/internal/models"
	"green-anti-detect-browser-backend-v1/internal/services/auth"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *auth.AuthService
}

func NewAuthHandler(authService *auth.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Login godoc
// @Summary Login user
// @Description Authenticate user with username and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.LoginRequest true "Login request (username and password)"
// @Success 200 {object} models.AuthResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Get user agent and IP address
	userAgent := c.GetHeader("User-Agent")
	ipAddress := c.ClientIP()

	response, err := h.authService.Login(&req, userAgent, ipAddress)
	if err != nil {
		if strings.Contains(err.Error(), "invalid credentials") || strings.Contains(err.Error(), "deactivated") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to login", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// RefreshToken godoc
// @Summary Refresh access token
// @Description Refresh access token using refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.RefreshTokenRequest true "Refresh token request"
// @Success 200 {object} models.AuthResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req models.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Get user agent and IP address
	userAgent := c.GetHeader("User-Agent")
	ipAddress := c.ClientIP()

	response, err := h.authService.RefreshToken(req.RefreshToken, userAgent, ipAddress)
	if err != nil {
		if strings.Contains(err.Error(), "invalid refresh token") || strings.Contains(err.Error(), "expired") || strings.Contains(err.Error(), "deactivated") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh token", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// Logout godoc
// @Summary Logout user
// @Description Logout the current user (revoke refresh token)
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.LogoutRequest true "Logout request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	// Get user ID from context (middleware already verified authentication)
	userID := c.MustGet("user_id").(string)

	var req models.LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// If no refresh token provided, logout from all sessions
		req.RefreshToken = ""
	}

	err := h.authService.Logout(req.RefreshToken, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to logout", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// GetProfile godoc
// @Summary Get user profile
// @Description Get current user profile information
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.User
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/auth/profile [get]
func (h *AuthHandler) GetProfile(c *gin.Context) {
	// Get user from context (middleware already verified authentication)
	user := c.MustGet("user").(*models.User)

	c.JSON(http.StatusOK, user)
}

// ChangePassword godoc
// @Summary Change user password
// @Description Change the current user's password
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.ChangePasswordRequest true "Change password request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/auth/change-password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	// Get user ID from context (middleware already verified authentication)
	userID := c.MustGet("user_id").(string)

	// Parse request body
	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Change password
	err := h.authService.ChangePassword(userID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		if strings.Contains(err.Error(), "current password is incorrect") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Current password is incorrect"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to change password", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// AdminRegister godoc
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
func (h *AuthHandler) AdminRegister(c *gin.Context) {
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

	// Get user agent and IP address
	userAgent := c.GetHeader("User-Agent")
	ipAddress := c.ClientIP()

	response, err := h.authService.Register(&req, userAgent, ipAddress)
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
// @Description Get list of all users in the system (Admin privileges required)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.User
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/admin/users [get]
func (h *AuthHandler) GetAllUsers(c *gin.Context) {
	// Check if user is admin
	user := c.MustGet("user").(*models.User)
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}

	users, err := h.authService.GetAllUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get users", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
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
func (h *AuthHandler) SetUserStatus(c *gin.Context) {
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

// DeleteUser godoc
// @Summary Delete a user (Admin only)
// @Description Delete a user account (Admin privileges required)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/admin/users/{id} [delete]
func (h *AuthHandler) DeleteUser(c *gin.Context) {
	// Check if user is admin
	user := c.MustGet("user").(*models.User)
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}

	// Get user ID from URL
	userID := c.Param("id")

	// Prevent admin from deleting themselves
	if userID == user.ID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete your own account"})
		return
	}

	// Delete user
	err := h.authService.DeleteUser(userID)
	if err != nil {
		if strings.Contains(err.Error(), "user not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

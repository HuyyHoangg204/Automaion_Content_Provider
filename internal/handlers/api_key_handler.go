package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/onegreenvn/green-provider-services-backend/internal/services/api_key"
	"gorm.io/gorm"
)

// APIKeyStatusRequest represents the request body for updating API key status
type APIKeyStatusRequest struct {
	IsActive bool `json:"is_active"`
}

// APIKeyHandler handles HTTP requests related to API keys
type APIKeyHandler struct {
	apiKeyService *api_key.Service
}

// NewAPIKeyHandler creates a new APIKeyHandler instance
func NewAPIKeyHandler(db *gorm.DB) *APIKeyHandler {
	apiKeyService := api_key.NewService(db)

	return &APIKeyHandler{
		apiKeyService: apiKeyService,
	}
}

// Generate handles POST /api/v1/api-key/generate
// @Summary Generate API key
// @Description Generate a new API key for the authenticated user
// @Tags api-key
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 201 {object} map[string]interface{} "success: true, api_key: models.APIKey"
// @Failure 400 {object} map[string]interface{} "success: false, error: error message"
// @Failure 500 {object} map[string]interface{} "success: false, error: error message"
// @Router /api/v1/api-key/generate [post]
func (h *APIKeyHandler) Generate(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID := c.MustGet("user_id").(string)

	// Generate API key
	apiKey, err := h.apiKeyService.GenerateAPIKey(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "API key generated successfully",
		"api_key": apiKey,
	})
}

// Get handles GET /api/v1/api-key
// @Summary Get API key
// @Description Get the API key for the authenticated user
// @Tags api-key
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "success: true, api_key: models.APIKey"
// @Failure 404 {object} map[string]interface{} "success: false, error: error message"
// @Failure 500 {object} map[string]interface{} "success: false, error: error message"
// @Router /api/v1/api-key [get]
func (h *APIKeyHandler) Get(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID := c.MustGet("user_id").(string)

	// Get API key
	apiKey, err := h.apiKeyService.GetAPIKeyByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	if apiKey == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "API key not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"api_key": apiKey,
	})
}

// UpdateStatus handles PUT /api/v1/api-key/status
// @Summary Update API key status
// @Description Enable or disable the API key for the authenticated user
// @Tags api-key
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param status body handlers.APIKeyStatusRequest true "Status object with is_active field"
// @Success 200 {object} map[string]interface{} "success: true, api_key: models.APIKey"
// @Failure 400 {object} map[string]interface{} "success: false, error: error message"
// @Failure 404 {object} map[string]interface{} "success: false, error: error message"
// @Failure 500 {object} map[string]interface{} "success: false, error: error message"
// @Router /api/v1/api-key/status [put]
func (h *APIKeyHandler) UpdateStatus(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID := c.MustGet("user_id").(string)

	// Parse request body
	var request APIKeyStatusRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Update API key status
	apiKey, err := h.apiKeyService.UpdateAPIKeyStatus(userID, request.IsActive)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	status := "enabled"
	if !request.IsActive {
		status = "disabled"
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "API key " + status + " successfully",
		"api_key": apiKey,
	})
}

// Delete handles DELETE /api/v1/api-key
// @Summary Delete API key
// @Description Delete the API key for the authenticated user
// @Tags api-key
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "success: true, message: string"
// @Failure 404 {object} map[string]interface{} "success: false, error: error message"
// @Failure 500 {object} map[string]interface{} "success: false, error: error message"
// @Router /api/v1/api-key [delete]
func (h *APIKeyHandler) Delete(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID := c.MustGet("user_id").(string)

	// Delete API key
	err := h.apiKeyService.DeleteAPIKey(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "API key deleted successfully",
	})
}

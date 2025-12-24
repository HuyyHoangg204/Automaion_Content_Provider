package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/onegreenvn/green-provider-services-backend/internal/services"
)

type GeminiAccountHandler struct {
	geminiAccountService *services.GeminiAccountService
	topicService         *services.TopicService
}

func NewGeminiAccountHandler(geminiAccountService *services.GeminiAccountService, topicService *services.TopicService) *GeminiAccountHandler {
	return &GeminiAccountHandler{
		geminiAccountService: geminiAccountService,
		topicService:         topicService,
	}
}

// SetupGeminiAccount sets up a Gemini account on a specific machine
// @Summary Setup Gemini account (Admin only)
// @Description Setup a Gemini account (Gmail) on a specific machine for creating Gemini Gems (Admin privileges required)
// @Tags gemini-accounts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.SetupGeminiAccountRequest true "Setup Gemini Account Request"
// @Success 201 {object} models.GeminiAccountResponse
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 403 {object} map[string]interface{} "Admin privileges required"
// @Failure 404 {object} map[string]interface{} "Machine not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/gemini/accounts/setup [post]
func (h *GeminiAccountHandler) SetupGeminiAccount(c *gin.Context) {
	// Check if user is admin
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	userModel, ok := user.(*models.User)
	if !ok || !userModel.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}

	var req models.SetupGeminiAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	account, err := h.geminiAccountService.SetupGeminiAccount(&req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to setup Gemini account", "details": err.Error()})
		}
		return
	}

	response := toGeminiAccountResponse(account)
	c.JSON(http.StatusCreated, response)
}

// GetAllAccounts retrieves all Gemini accounts with optional filters
// @Summary Get all Gemini accounts
// @Description Get all Gemini accounts with optional filters (machine_id, is_active, is_locked)
// @Tags gemini-accounts
// @Produce json
// @Param machine_id query string false "Filter by machine ID"
// @Param is_active query bool false "Filter by active status"
// @Param is_locked query bool false "Filter by locked status"
// @Success 200 {array} models.GeminiAccountResponse
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/gemini/accounts [get]
func (h *GeminiAccountHandler) GetAllAccounts(c *gin.Context) {
	machineID := c.Query("machine_id")
	var isActive *bool
	var isLocked *bool

	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		if val, err := strconv.ParseBool(isActiveStr); err == nil {
			isActive = &val
		}
	}

	if isLockedStr := c.Query("is_locked"); isLockedStr != "" {
		if val, err := strconv.ParseBool(isLockedStr); err == nil {
			isLocked = &val
		}
	}

	accounts, err := h.geminiAccountService.GetAllAccounts(machineID, isActive, isLocked)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get accounts", "details": err.Error()})
		return
	}

	responses := make([]models.GeminiAccountResponse, len(accounts))
	for i, account := range accounts {
		responses[i] = toGeminiAccountResponse(account)
	}

	c.JSON(http.StatusOK, responses)
}

// GetAccountByID retrieves a Gemini account by ID
// @Summary Get Gemini account by ID
// @Description Get a specific Gemini account by its ID
// @Tags gemini-accounts
// @Produce json
// @Param id path string true "Account ID"
// @Success 200 {object} models.GeminiAccountResponse
// @Failure 404 {object} map[string]interface{} "Account not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/gemini/accounts/{id} [get]
func (h *GeminiAccountHandler) GetAccountByID(c *gin.Context) {
	accountID := c.Param("id")

	account, err := h.geminiAccountService.GetAccountByID(accountID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found", "details": err.Error()})
		return
	}

	response := toGeminiAccountResponse(account)
	c.JSON(http.StatusOK, response)
}

// GetAccountsByMachineID retrieves all Gemini accounts for a specific machine
// @Summary Get Gemini accounts by machine ID
// @Description Get all Gemini accounts for a specific machine
// @Tags gemini-accounts
// @Produce json
// @Param machine_id path string true "Machine ID"
// @Success 200 {array} models.GeminiAccountResponse
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/gemini/accounts/machine/{machine_id} [get]
func (h *GeminiAccountHandler) GetAccountsByMachineID(c *gin.Context) {
	machineID := c.Param("machine_id")

	accounts, err := h.geminiAccountService.GetAccountsByMachineID(machineID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get accounts", "details": err.Error()})
		return
	}

	responses := make([]models.GeminiAccountResponse, len(accounts))
	for i, account := range accounts {
		responses[i] = toGeminiAccountResponse(account)
	}

	c.JSON(http.StatusOK, responses)
}

// LockAccount locks a Gemini account and deletes all topics created with it
// @Summary Lock Gemini account (Admin only)
// @Description Lock a Gemini account and delete all topics created with it (Admin privileges required)
// @Tags gemini-accounts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Account ID"
// @Param request body models.LockGeminiAccountRequest false "Lock reason (optional)"
// @Success 200 {object} map[string]interface{} "{\"message\": \"Account locked successfully\", \"topics_deleted\": 5}"
// @Failure 403 {object} map[string]interface{} "Admin privileges required"
// @Failure 404 {object} map[string]interface{} "Account not found"
// @Failure 400 {object} map[string]interface{} "Account already locked"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/gemini/accounts/{id}/lock [put]
func (h *GeminiAccountHandler) LockAccount(c *gin.Context) {
	// Check if user is admin
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	userModel, ok := user.(*models.User)
	if !ok || !userModel.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}

	accountID := c.Param("id")

	var req models.LockGeminiAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Optional request body, use empty reason if not provided
		req.Reason = ""
	}

	topicsDeleted, err := h.geminiAccountService.LockAccount(accountID, req.Reason)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else if strings.Contains(err.Error(), "already locked") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to lock account", "details": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Account locked successfully",
		"topics_deleted": topicsDeleted,
	})
}

// UnlockAccount unlocks a Gemini account
// @Summary Unlock Gemini account (Admin only)
// @Description Unlock a previously locked Gemini account (Admin privileges required)
// @Tags gemini-accounts
// @Produce json
// @Security BearerAuth
// @Param id path string true "Account ID"
// @Success 200 {object} map[string]interface{} "{\"message\": \"Account unlocked successfully\"}"
// @Failure 403 {object} map[string]interface{} "Admin privileges required"
// @Failure 404 {object} map[string]interface{} "Account not found"
// @Failure 400 {object} map[string]interface{} "Account not locked"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/gemini/accounts/{id}/unlock [put]
func (h *GeminiAccountHandler) UnlockAccount(c *gin.Context) {
	// Check if user is admin
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	userModel, ok := user.(*models.User)
	if !ok || !userModel.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}

	accountID := c.Param("id")

	err := h.geminiAccountService.UnlockAccount(accountID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else if strings.Contains(err.Error(), "not locked") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unlock account", "details": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Account unlocked successfully"})
}

// GetTopicsByAccountID (legacy) - in the new model, gems are attached to ScriptProjects, not Topics.
// This endpoint is kept for backward compatibility but now returns an empty topics list.
// @Summary Get topics by Gemini account ID (deprecated)
// @Description [DEPRECATED] Topics are no longer directly bound to Gemini accounts. Use project-level APIs instead.
// @Tags gemini-accounts
// @Produce json
// @Security BearerAuth
// @Param id path string true "Gemini Account ID"
// @Success 200 {array} models.TopicResponse
// @Failure 404 {object} map[string]interface{} "Account not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/gemini/accounts/{id}/topics [get]
func (h *GeminiAccountHandler) GetTopicsByAccountID(c *gin.Context) {
	accountID := c.Param("id")

	// Verify account exists
	account, err := h.geminiAccountService.GetAccountByID(accountID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"account_id": account.ID,
		"email":      account.Email,
		"topics":     []gin.H{},
		"total":      0,
	})
}

// toGeminiAccountResponse converts GeminiAccount to GeminiAccountResponse
func toGeminiAccountResponse(account *models.GeminiAccount) models.GeminiAccountResponse {
	response := models.GeminiAccountResponse{
		ID:               account.ID,
		MachineID:        account.MachineID,
		AppID:            account.AppID,
		Email:            account.Email,
		IsActive:         account.IsActive,
		IsLocked:         account.IsLocked,
		LockedReason:     account.LockedReason,
		GeminiAccessible: account.GeminiAccessible,
		TopicsCount:      account.TopicsCount,
		CreatedAt:        account.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        account.UpdatedAt.Format(time.RFC3339),
	}

	if account.LockedAt != nil {
		lockedAtStr := account.LockedAt.Format(time.RFC3339)
		response.LockedAt = &lockedAtStr
	}

	if account.LastUsedAt != nil {
		lastUsedAtStr := account.LastUsedAt.Format(time.RFC3339)
		response.LastUsedAt = &lastUsedAtStr
	}

	if account.LastCheckedAt != nil {
		lastCheckedAtStr := account.LastCheckedAt.Format(time.RFC3339)
		response.LastCheckedAt = &lastCheckedAtStr
	}

	return response
}


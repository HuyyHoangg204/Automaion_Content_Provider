package handlers

import (
	"net/http"
	"strings"

	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/onegreenvn/green-provider-services-backend/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AppHandler struct {
	appService *services.AppService
}

func NewAppHandler(db *gorm.DB) *AppHandler {
	userRepo := repository.NewUserRepository(db)
	boxRepo := repository.NewBoxRepository(db)
	appRepo := repository.NewAppRepository(db)
	profileRepo := repository.NewProfileRepository(db)

	appService := services.NewAppService(appRepo, profileRepo, boxRepo, userRepo)
	return &AppHandler{
		appService: appService,
	}
}

// CreateApp godoc
// @Summary Create a new app
// @Description Create a new anti-detect browser app for the authenticated user
// @Tags apps
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.CreateAppRequest true "Create app request"
// @Success 201 {object} models.AppResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/apps [post]
func (h *AppHandler) CreateApp(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	var req models.CreateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	response, err := h.appService.CreateApp(userID, &req)
	if err != nil {
		// Check if it's an AppAlreadyExistsError
		if appExistsErr, ok := err.(*services.AppAlreadyExistsError); ok {
			c.JSON(http.StatusConflict, gin.H{
				"error":           appExistsErr.Message,
				"existing_app_id": appExistsErr.ExistingAppID,
			})
			return
		}
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "access denied") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create app", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// GetMyApps godoc
// @Summary Get user's apps
// @Description Get all apps belonging to the authenticated user
// @Tags apps
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.AppResponse
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/apps [get]
func (h *AppHandler) GetMyApps(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	apps, err := h.appService.GetAppsByUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get apps", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, apps)
}

// GetAppsByBox godoc
// @Summary Get apps by box
// @Description Get all apps for a specific box (user must own the box)
// @Tags apps
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Box ID"
// @Success 200 {array} models.AppResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/boxes/{id}/apps [get]
func (h *AppHandler) GetAppsByBox(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	boxID := c.Param("id")

	apps, err := h.appService.GetAppsByBox(userID, boxID)
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

// GetAppByID godoc
// @Summary Get app by ID
// @Description Get a specific app by ID (user must own it)
// @Tags apps
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "App ID"
// @Success 200 {object} models.AppResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/apps/{id} [get]
func (h *AppHandler) GetAppByID(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	appID := c.Param("id")

	app, err := h.appService.GetAppByID(userID, appID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "App not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get app", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, app)
}

// UpdateApp godoc
// @Summary Update app
// @Description Update an app (user must own it)
// @Tags apps
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "App ID"
// @Param request body models.UpdateAppRequest true "Update app request"
// @Success 200 {object} models.AppResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/apps/{id} [put]
func (h *AppHandler) UpdateApp(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	appID := c.Param("id")

	var req models.UpdateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	response, err := h.appService.UpdateApp(userID, appID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "App not found"})
			return
		}
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update app", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// DeleteApp godoc
// @Summary Delete app
// @Description Delete an app (user must own it)
// @Tags apps
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "App ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/apps/{id} [delete]
func (h *AppHandler) DeleteApp(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	appID := c.Param("id")

	err := h.appService.DeleteApp(userID, appID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "App not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete app", "details": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// AdminGetAllApps godoc
// @Summary Get all apps (Admin only)
// @Description Get all apps in the system (Admin privileges required)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.AppResponse
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/admin/apps [get]
func (h *AppHandler) AdminGetAllApps(c *gin.Context) {
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

// GetRegisterAppDomains godoc
// @Summary Get subdomain and FRP configuration for app registration
// @Description Get subdomain configuration and FRP settings for multiple platforms based on box_id and platform_name (comma-separated)
// @Tags apps
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param box_id query string true "Box ID to get machine_id"
// @Param platform_name query string true "Platform names separated by comma (e.g., hidemium,genlogin,adspower)"
// @Success 200 {object} models.RegisterAppResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/apps/register-app [get]
func (h *AppHandler) GetRegisterAppDomains(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	// Get query parameters
	boxID := c.Query("box_id")
	platformNames := c.Query("platform_name")

	// Validate required parameters
	if boxID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "box_id parameter is required"})
		return
	}

	if platformNames == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "platform_name parameter is required"})
		return
	}

	response, err := h.appService.GetRegisterAppDomains(userID, boxID, platformNames)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "access denied") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get register app domains", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// CheckTunnelURL godoc
// @Summary Check if tunnel URL is accessible for Hidemium
// @Description Check if a tunnel URL is accessible by testing the /user-settings/token endpoint. Returns true if the endpoint is accessible and returns token data.
// @Tags apps
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param tunnel_url query string true "Tunnel URL to check (e.g., http://machineid-hidemium-userid.agent-controller.onegreen.cloud)"
// @Success 200 {object} models.CheckTunnelResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/apps/check-tunnel [get]
func (h *AppHandler) CheckTunnelURL(c *gin.Context) {
	// Get tunnel URL from query parameter
	tunnelURL := c.Query("tunnel_url")

	// Validate required parameter
	if tunnelURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tunnel_url parameter is required"})
		return
	}

	// Check tunnel accessibility
	response, err := h.appService.CheckTunnelURL(tunnelURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check tunnel URL", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// SyncAppProfiles syncs profiles from a specific app
// @Summary Sync profiles from a specific app
// @Description Sync all profiles from a specific app
// @Tags apps
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "App ID to sync profiles from"
// @Success 200 {object} models.SyncBoxProfilesResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/apps/sync/{id} [post]
func (h *AppHandler) SyncAppProfiles(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	appID := c.Param("id")

	// Get app by ID and verify ownership
	app, err := h.appService.GetAppByUserIDAndID(userID, appID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "App not found"})
		return
	}

	// Sync profiles from the app
	syncResult, err := h.appService.SyncAppProfiles(app)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, syncResult)
}

// SyncAllUserApps syncs all apps owned by the user
// @Summary Sync all profiles from all apps owned by the user
// @Description Sync all profiles from all apps owned by the user
// @Tags apps
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.SyncBoxProfilesResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/apps/sync/all-apps [post]
func (h *AppHandler) SyncAllUserApps(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	// Sync all user apps
	syncResult, err := h.appService.SyncAllAppsByUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, syncResult)
}

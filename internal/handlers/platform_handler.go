package handlers

import (
	"net/http"

	"green-anti-detect-browser-backend-v1/internal/config"

	"github.com/gin-gonic/gin"
)

type PlatformHandler struct{}

func NewPlatformHandler() *PlatformHandler {
	return &PlatformHandler{}
}

// GetPlatformRoutes godoc
// @Summary Get platform routes
// @Description Get all available platform routes and endpoints
// @Tags platforms
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} config.PlatformRoutes
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/platforms/routes [get]
func (h *PlatformHandler) GetPlatformRoutes(c *gin.Context) {
	routes := config.GetPlatformRoutes()
	c.JSON(http.StatusOK, routes)
}

// GetSupportedPlatforms godoc
// @Summary Get supported platforms
// @Description Get list of supported platforms
// @Tags platforms
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} string
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/platforms/supported [get]
func (h *PlatformHandler) GetSupportedPlatforms(c *gin.Context) {
	platforms := config.GetSupportedPlatforms()
	c.JSON(http.StatusOK, platforms)
}

// GetPlatformEndpointInfo godoc
// @Summary Get platform endpoint info
// @Description Get information about a specific platform endpoint
// @Tags platforms
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param platform path string true "Platform name (hidemium, genlogin)"
// @Param endpoint path string true "Endpoint name"
// @Success 200 {object} interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/platforms/{platform}/endpoints/{endpoint} [get]
func (h *PlatformHandler) GetPlatformEndpointInfo(c *gin.Context) {
	platform := c.Param("platform")
	endpoint := c.Param("endpoint")

	// Check if platform is supported
	if !config.IsPlatformSupported(platform) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":               "Platform not supported",
			"supported_platforms": config.GetSupportedPlatforms(),
		})
		return
	}

	// Get endpoint info
	endpointInfo, err := config.GetEndpointInfo(platform, endpoint)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":    err.Error(),
			"platform": platform,
			"endpoint": endpoint,
		})
		return
	}

	c.JSON(http.StatusOK, endpointInfo)
}

// GetPlatformEndpointURL godoc
// @Summary Get platform endpoint URL
// @Description Get the full URL for a specific platform endpoint
// @Tags platforms
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param platform path string true "Platform name (hidemium, genlogin)"
// @Param endpoint path string true "Endpoint name"
// @Param machine_id query string true "Machine ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/platforms/{platform}/endpoints/{endpoint}/url [get]
func (h *PlatformHandler) GetPlatformEndpointURL(c *gin.Context) {
	platform := c.Param("platform")
	endpoint := c.Param("endpoint")
	machineID := c.Query("machine_id")

	if machineID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "machine_id query parameter is required",
		})
		return
	}

	// Check if platform is supported
	if !config.IsPlatformSupported(platform) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":               "Platform not supported",
			"supported_platforms": config.GetSupportedPlatforms(),
		})
		return
	}

	// Get endpoint URL
	url, err := config.GetEndpointURL(platform, machineID, endpoint)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":      err.Error(),
			"platform":   platform,
			"endpoint":   endpoint,
			"machine_id": machineID,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"platform":   platform,
		"endpoint":   endpoint,
		"machine_id": machineID,
		"url":        url,
	})
}

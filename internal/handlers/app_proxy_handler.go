package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/services"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type AppProxyHandler struct {
	appProxyService *services.AppProxyService
}

func NewAppProxyHandler(db *gorm.DB) *AppProxyHandler {
	userRepo := repository.NewUserRepository(db)
	boxRepo := repository.NewBoxRepository(db)
	appRepo := repository.NewAppRepository(db)

	return &AppProxyHandler{
		appProxyService: services.NewAppProxyService(appRepo, boxRepo, userRepo),
	}
}

// ProxyRequest handles all proxy requests to anti-detect browser platforms
// @Summary Proxy request to anti-detect browser platform
// @Description Forwards requests to the appropriate platform based on app. Supports Hidemium and GenLogin platforms.
// @Tags app-proxy
// @Accept json
// @Produce json
// @Param app_id path string true "App ID (UUID)" example:"550e8400-e29b-41d4-a716-446655440001"
// @Param platform_path path string true "Platform-specific API path" example:"v1/browser/list"
// @Param request body object false "Request body for POST/PUT requests (platform-specific format)"
// @Success 200 {object} map[string]interface{} "Platform response (varies by platform)"
// @Success 201 {object} map[string]interface{} "Resource created successfully"
// @Success 204 "No content (for DELETE requests)"
// @Failure 400 {object} map[string]string "Bad request (invalid path, missing tunnel URL, etc.)"
// @Failure 401 {object} map[string]string "Unauthorized (invalid/missing JWT token)"
// @Failure 403 {object} map[string]string "Forbidden (app doesn't belong to user)"
// @Failure 404 {object} map[string]string "Not found (app not found)"
// @Failure 500 {object} map[string]string "Internal server error (forwarding failed)"
// @Security BearerAuth
// @Router /api/v1/app-proxy/{app_id}/{platform_path} [get]
// @Router /api/v1/app-proxy/{app_id}/{platform_path} [post]
// @Router /api/v1/app-proxy/{app_id}/{platform_path} [put]
// @Router /api/v1/app-proxy/{app_id}/{platform_path} [delete]
func (h *AppProxyHandler) ProxyRequest(c *gin.Context) {
	userID := c.GetString("user_id")
	appID := c.Param("app_id")
	platformPath := c.Param("platform_path")

	// Validate request using service - check app ownership
	app, err := h.appProxyService.ValidateAppProxyRequest(userID, appID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build target URL
	targetURL := h.appProxyService.BuildTargetURL(*app.TunnelURL, app.Name, platformPath)

	// Forward the request
	response, err := h.forwardRequest(c, targetURL, app.Name)
	if err != nil {
		logrus.Errorf("Failed to forward request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to forward request to platform"})
		return
	}

	// Return the platform response
	c.Data(http.StatusOK, "application/json", response)
}

// forwardRequest forwards the HTTP request to the target platform
func (h *AppProxyHandler) forwardRequest(c *gin.Context, targetURL, appName string) ([]byte, error) {
	// Get request method and headers
	method := c.Request.Method
	headers := c.Request.Header

	// Create new request
	var req *http.Request
	var err error

	if method == "GET" {
		req, err = http.NewRequest(method, targetURL, nil)
	} else {
		// For POST/PUT/DELETE, read body
		body, readErr := io.ReadAll(c.Request.Body)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read request body: %w", readErr)
		}
		req, err = http.NewRequest(method, targetURL, strings.NewReader(string(body)))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Copy headers (excluding some that shouldn't be forwarded)
	for key, values := range headers {
		if !h.shouldSkipHeader(key) {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
	}

	// Set platform-specific headers
	h.setPlatformHeaders(req, appName)

	// Make the request
	client := &http.Client{
		Timeout: 30 * time.Second, // Default timeout
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Set response status code
	c.Status(resp.StatusCode)

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}

	return responseBody, nil
}

// shouldSkipHeader determines if a header should be skipped when forwarding
func (h *AppProxyHandler) shouldSkipHeader(key string) bool {
	skipHeaders := []string{
		"Host",
		"Content-Length",
		"Transfer-Encoding",
		"Connection",
		"Upgrade",
		"Sec-WebSocket-Key",
		"Sec-WebSocket-Version",
		"Sec-WebSocket-Protocol",
	}

	for _, skipHeader := range skipHeaders {
		if strings.EqualFold(key, skipHeader) {
			return true
		}
	}
	return false
}

// setPlatformHeaders sets platform-specific headers
func (h *AppProxyHandler) setPlatformHeaders(req *http.Request, appName string) {
	switch appName {
	case "Hidemium":
		// Hidemium specific headers if needed
		req.Header.Set("User-Agent", "Green-Controller/1.0")
	case "Genlogin":
		// GenLogin specific headers if needed
		req.Header.Set("User-Agent", "Green-Controller/1.0")
	}
}

package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/onegreenvn/green-provider-services-backend/internal/services/api_key"
)

// APIKeyMiddleware handles API key authentication
type APIKeyMiddleware struct {
	apiKeyService *api_key.Service
}

// NewAPIKeyMiddleware creates a new API key middleware
func NewAPIKeyMiddleware(apiKeyService *api_key.Service) *APIKeyMiddleware {
	return &APIKeyMiddleware{
		apiKeyService: apiKeyService,
	}
}

// APIKeyAuthMiddleware validates API key and sets user context
func (m *APIKeyMiddleware) APIKeyAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get API key from header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Authorization header is required",
			})
			c.Abort()
			return
		}

		// Check if it's an API key (starts with "ApiKey ")
		if !strings.HasPrefix(authHeader, "ApiKey ") {
			// Not an API key, let other middleware handle it
			c.Next()
			return
		}

		// Extract the API key
		apiKey := strings.TrimPrefix(authHeader, "ApiKey ")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid API key format",
			})
			c.Abort()
			return
		}

		// Validate the API key
		user, err := m.apiKeyService.ValidateAPIKey(apiKey)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   err.Error(),
			})
			c.Abort()
			return
		}

		// Set user information in context
		c.Set("user_id", user.ID)
		c.Set("user", user)
		c.Set("is_admin", user.IsAdmin)
		c.Set("auth_type", "api_key")

		c.Next()
	}
}

package middleware

import (
	"net/http"
	"strings"

	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/services/auth"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type BearerTokenMiddleware struct {
	authService *auth.AuthService
	userRepo    *repository.UserRepository
	db          *gorm.DB
}

func NewBearerTokenMiddleware(authService *auth.AuthService, db *gorm.DB) *BearerTokenMiddleware {
	// Create repositories
	userRepo := repository.NewUserRepository(db)

	return &BearerTokenMiddleware{
		authService: authService,
		userRepo:    userRepo,
		db:          db,
	}
}

// BearerTokenAuthMiddleware validates JWT token and sets user info in context
func (m *BearerTokenMiddleware) BearerTokenAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		// If user_id is already set, skip authentication
		_, exists := c.Get("user_id")
		if exists {
			c.Next()
			return
		}

		// Get Authorization header
		authHeader := c.GetHeader("Authorization")

		// Check if it's Bearer token
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		// Extract token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Validate token
		tokenInfo, err := m.authService.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Get user from database
		user, err := m.userRepo.GetByID(tokenInfo.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}

		// Set user info in context
		c.Set("user_id", user.ID)
		c.Set("user", user)
		c.Set("is_admin", user.IsAdmin)
		c.Set("token_info", tokenInfo)

		c.Next()
	}
}

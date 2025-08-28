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
	db *gorm.DB
}

func NewBearerTokenMiddleware(db *gorm.DB) *BearerTokenMiddleware {
	return &BearerTokenMiddleware{db: db}
}

// Helper method to create auth service
func (m *BearerTokenMiddleware) createAuthService() *auth.AuthService {
	userRepo := repository.NewUserRepository(m.db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(m.db)
	return auth.NewAuthService(userRepo, refreshTokenRepo)
}

// Helper method to create user repository
func (m *BearerTokenMiddleware) createUserRepo() *repository.UserRepository {
	return repository.NewUserRepository(m.db)
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
		tokenInfo, err := m.createAuthService().ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Get user from database
		user, err := m.createUserRepo().GetByID(tokenInfo.UserID)
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

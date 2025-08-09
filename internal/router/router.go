package router

import (
	"time"

	"green-anti-detect-browser-backend-v1/internal/database/repository"
	"green-anti-detect-browser-backend-v1/internal/handlers"
	"green-anti-detect-browser-backend-v1/internal/middleware"
	"green-anti-detect-browser-backend-v1/internal/services/auth"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

// SetupRouter configures the Gin router with user authentication routes
func SetupRouter(db *gorm.DB, basePath string) *gin.Engine {
	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	// Create a new router
	r := gin.New()

	// Use middleware
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())

	// Configure CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)

	// Create auth service and middleware
	authService := auth.NewAuthService(userRepo, refreshTokenRepo)
	bearerTokenMiddleware := middleware.NewBearerTokenMiddleware(authService, userRepo)

	// Create auth handler
	authHandler := handlers.NewAuthHandler(authService)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	logrus.Info("Swagger UI endpoint registered at /swagger/index.html")

	// API v1 routes
	api := r.Group("/api/v1")
	{
		// Health check
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status": "ok",
				"time":   time.Now().Format(time.RFC3339),
			})
		})

		// Auth routes (public)
		auth := api.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
		}

		// Protected routes
		protected := api.Group("")
		protected.Use(bearerTokenMiddleware.BearerTokenAuthMiddleware())
		{
			// Auth protected routes
			authProtected := protected.Group("/auth")
			{
				authProtected.POST("/logout", authHandler.Logout)
				authProtected.GET("/profile", authHandler.GetProfile)
				authProtected.POST("/change-password", authHandler.ChangePassword)
			}

			// User routes
			users := protected.Group("/users")
			{
				// Get current user info
				users.GET("/me", authHandler.GetProfile)
			}

			// Admin routes (requires admin privileges)
			admin := protected.Group("/admin")
			{
				admin.POST("/register", authHandler.AdminRegister)
				admin.GET("/users", authHandler.GetAllUsers)
				admin.PUT("/users/:id/status", authHandler.SetUserStatus)
				admin.DELETE("/users/:id", authHandler.DeleteUser)
			}
		}
	}

	return r
}

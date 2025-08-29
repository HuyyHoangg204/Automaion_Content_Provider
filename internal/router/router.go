package router

import (
	"time"

	"github.com/onegreenvn/green-provider-services-backend/internal/handlers"
	"github.com/onegreenvn/green-provider-services-backend/internal/middleware"
	"github.com/onegreenvn/green-provider-services-backend/internal/services"
	"github.com/onegreenvn/green-provider-services-backend/internal/services/auth"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

// SetupRouter configures the Gin router with user authentication routes
func SetupRouter(db *gorm.DB) *gin.Engine {
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

	// Create auth middleware
	// Create services
	authService := auth.NewAuthService(db)

	// Initialize RabbitMQ service
	rabbitMQService, err := services.NewRabbitMQService()
	if err != nil {
		logrus.Warnf("Failed to initialize RabbitMQ: %v", err)
	} else {
		logrus.Info("RabbitMQ service initialized in router")
	}

	// Create middleware with services
	bearerTokenMiddleware := middleware.NewBearerTokenMiddleware(authService, db)

	// Create handlers with services
	authHandler := handlers.NewAuthHandler(authService)
	boxHandler := handlers.NewBoxHandler(db)
	appHandler := handlers.NewAppHandler(db)
	profileHandler := handlers.NewProfileHandler(db)
	campaignHandler := handlers.NewCampaignHandler(db, rabbitMQService)
	flowGroupHandler := handlers.NewFlowGroupHandler(db)
	flowHandler := handlers.NewFlowHandler(db)
	appProxyHandler := handlers.NewAppProxyHandler(db)

	// Create admin handler with services
	adminHandler := handlers.NewAdminHandler(authService, db)

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

			// Box routes
			boxes := protected.Group("/boxes")
			{
				boxes.POST("", boxHandler.CreateBox)
				boxes.GET("", boxHandler.GetMyBoxes)
				boxes.GET("/:id", boxHandler.GetBoxByID)
				boxes.PUT("/:id", boxHandler.UpdateBox)
				boxes.DELETE("/:id", boxHandler.DeleteBox)
				boxes.GET("/:id/apps", appHandler.GetAppsByBox)
				boxes.POST("/sync-profiles/:id", boxHandler.SyncAllProfilesInBox)
			}

			// Box Proxy routes - for direct platform operations
			appProxy := protected.Group("/app-proxy")
			{
				appProxy.Any("/:app_id/*platform_path", appProxyHandler.ProxyRequest)
			}

			// App routes
			apps := protected.Group("/apps")
			{
				apps.POST("", appHandler.CreateApp)
				apps.GET("", appHandler.GetMyApps)
				apps.GET("/:id/profiles", profileHandler.GetProfilesByApp)
				apps.GET("/register-app", appHandler.GetRegisterAppDomains)
				apps.GET("/check-tunnel", appHandler.CheckTunnelURL)
				apps.GET("/:id", appHandler.GetAppByID)
				apps.PUT("/:id", appHandler.UpdateApp)
				apps.DELETE("/:id", appHandler.DeleteApp)
				apps.POST("/sync/:id", appHandler.SyncAppProfiles)
				apps.POST("/sync/all-apps", appHandler.SyncAllUserApps)
			}

			// Campaign routes
			campaigns := protected.Group("/campaigns")
			{
				campaigns.POST("", campaignHandler.CreateCampaign)
				campaigns.GET("", campaignHandler.GetMyCampaigns)
				campaigns.GET("/:id", campaignHandler.GetCampaignByID)
				campaigns.PUT("/:id", campaignHandler.UpdateCampaign)
				campaigns.DELETE("/:id", campaignHandler.DeleteCampaign)
				campaigns.POST("/:id/run", campaignHandler.RunCampaign)
				campaigns.GET("/:id/flow-groups", flowGroupHandler.GetFlowGroupsByCampaign)
				campaigns.GET("/:id/flows", flowHandler.GetFlowsByCampaign)
			}

			// Flow Group routes
			flowGroups := protected.Group("/flow-groups")
			{
				flowGroups.GET("/:id", flowGroupHandler.GetFlowGroupByID)
				flowGroups.GET("/:id/stats", flowGroupHandler.GetFlowGroupStats)
				flowGroups.GET("/:id/flows", flowHandler.GetFlowsByFlowGroup)
			}

			// Profile routes
			profiles := protected.Group("/profiles")
			{
				profiles.POST("", profileHandler.CreateProfile)
				profiles.GET("", profileHandler.GetMyProfiles)
				profiles.GET("/default-configs", profileHandler.GetDefaultConfigs)
				profiles.GET("/:id", profileHandler.GetProfileByID)
				profiles.PUT("/:id", profileHandler.UpdateProfile)
				profiles.DELETE("/:id", profileHandler.DeleteProfile)
				profiles.GET("/:id/flows", flowHandler.GetFlowsByProfile)
			}

			// Flow routes
			flows := protected.Group("/flows")
			{
				flows.POST("", flowHandler.CreateFlow)
				flows.GET("", flowHandler.GetMyFlows)
				flows.GET("/:id", flowHandler.GetFlowByID)
				flows.PUT("/:id", flowHandler.UpdateFlow)
				flows.DELETE("/:id", flowHandler.DeleteFlow)
				flows.GET("/status/:status", flowHandler.GetFlowsByStatus)
			}

			// Admin routes (requires admin privileges)
			admin := protected.Group("/admin")
			{
				admin.POST("/register", adminHandler.Register)
				admin.GET("/users", adminHandler.GetAllUsers)
				admin.PUT("/users/:id/status", adminHandler.SetUserStatus)
				admin.DELETE("/users/:id", adminHandler.DeleteUser)
				admin.POST("/users/:id/reset-password", adminHandler.ResetPassword)
				admin.GET("/boxes", adminHandler.AdminGetAllBoxes)
				admin.GET("/apps", adminHandler.AdminGetAllApps)
				admin.GET("/profiles", adminHandler.AdminGetAllProfiles)
				admin.GET("/campaigns", adminHandler.AdminGetAllCampaigns)
				admin.GET("/flows", adminHandler.AdminGetAllFlows)
			}
		}
	}

	return r
}

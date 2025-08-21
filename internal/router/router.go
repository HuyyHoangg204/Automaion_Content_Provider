package router

import (
	"time"

	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
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
	boxRepo := repository.NewBoxRepository(db)
	appRepo := repository.NewAppRepository(db)
	profileRepo := repository.NewProfileRepository(db)
	campaignRepo := repository.NewCampaignRepository(db)
	flowGroupRepo := repository.NewFlowGroupRepository(db)
	flowRepo := repository.NewFlowRepository(db)

	// Create auth service and middleware
	authService := auth.NewAuthService(userRepo, refreshTokenRepo)
	bearerTokenMiddleware := middleware.NewBearerTokenMiddleware(authService, userRepo)

	// Create services
	boxService := services.NewBoxService(boxRepo, userRepo, appRepo, profileRepo)
	appService := services.NewAppService(appRepo, boxRepo, userRepo)
	profileService := services.NewProfileService(profileRepo, appRepo, userRepo, boxRepo)
	campaignService := services.NewCampaignService(campaignRepo, flowGroupRepo, userRepo, profileRepo)
	flowGroupService := services.NewFlowGroupService(flowGroupRepo, campaignRepo, flowRepo)
	flowService := services.NewFlowService(flowRepo, campaignRepo, flowGroupRepo, profileRepo, userRepo)

	// Create handlers
	authHandler := handlers.NewAuthHandler(authService)
	boxHandler := handlers.NewBoxHandler(boxService)
	appHandler := handlers.NewAppHandler(appService)
	profileHandler := handlers.NewProfileHandler(profileService)
	campaignHandler := handlers.NewCampaignHandler(campaignService)
	flowGroupHandler := handlers.NewFlowGroupHandler(flowGroupService)
	flowHandler := handlers.NewFlowHandler(flowService)

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

			// Box routes
			boxes := protected.Group("/boxes")
			{
				boxes.POST("", boxHandler.CreateBox)
				boxes.GET("", boxHandler.GetMyBoxes)
				boxes.GET("/:id", boxHandler.GetBoxByID)
				boxes.PUT("/:id", boxHandler.UpdateBox)
				boxes.DELETE("/:id", boxHandler.DeleteBox)
				boxes.POST("/:id/sync-profiles", boxHandler.SyncBoxProfilesFromPlatform)
				boxes.POST("/sync-all", boxHandler.SyncAllUserBoxes)
			}

			// Box Apps routes (separate to avoid conflict)
			boxApps := protected.Group("/box-apps")
			{
				boxApps.GET("/:box_id/apps", appHandler.GetAppsByBox)
			}

			// App routes
			apps := protected.Group("/apps")
			{
				apps.POST("", appHandler.CreateApp)
				apps.GET("", appHandler.GetMyApps)
				apps.GET("/register-app", appHandler.GetRegisterAppDomains)
				apps.GET("/:id", appHandler.GetAppByID)
				apps.PUT("/:id", appHandler.UpdateApp)
				apps.DELETE("/:id", appHandler.DeleteApp)
			}

			// App Profiles routes (separate to avoid conflict)
			appProfiles := protected.Group("/app-profiles")
			{
				appProfiles.GET("/:app_id/profiles", profileHandler.GetProfilesByApp)
			}

			// Campaign routes
			campaigns := protected.Group("/campaigns")
			{
				campaigns.POST("", campaignHandler.CreateCampaign)
				campaigns.GET("", campaignHandler.GetMyCampaigns)
				campaigns.GET("/:id", campaignHandler.GetCampaignByID)
				campaigns.PUT("/:id", campaignHandler.UpdateCampaign)
				campaigns.DELETE("/:id", campaignHandler.DeleteCampaign)
				campaigns.GET("/:id/flow-groups", flowGroupHandler.GetFlowGroupsByCampaign)
			}

			// Group Campaign routes
			flowGroups := protected.Group("/flow-groups")
			{
				flowGroups.GET("/:id", flowGroupHandler.GetFlowGroupByID)
				flowGroups.GET("/:id/stats", flowGroupHandler.GetFlowGroupStats)
			}

			// Group Campaign Flows routes
			flowGroupFlows := protected.Group("/flow-group-flows")
			{
				flowGroupFlows.GET("/:flow_group_id/flows", flowHandler.GetFlowsByFlowGroup)
			}

			// Campaign Flows routes (separate to avoid conflict)
			campaignFlows := protected.Group("/campaign-flows")
			{
				campaignFlows.GET("/:campaign_id/flows", flowHandler.GetFlowsByCampaign)
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
			}

			// Profile Flows routes (separate to avoid conflict)
			profileFlows := protected.Group("/profile-flows")
			{
				profileFlows.GET("/:profile_id/flows", flowHandler.GetFlowsByProfile)
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
				admin.POST("/register", authHandler.AdminRegister)
				admin.GET("/users", authHandler.GetAllUsers)
				admin.PUT("/users/:id/status", authHandler.SetUserStatus)
				admin.DELETE("/users/:id", authHandler.DeleteUser)
				admin.GET("/boxes", boxHandler.AdminGetAllBoxes)
				admin.GET("/apps", appHandler.AdminGetAllApps)
				admin.GET("/profiles", profileHandler.AdminGetAllProfiles)
				admin.GET("/campaigns", campaignHandler.AdminGetAllCampaigns)
				admin.GET("/flows", flowHandler.AdminGetAllFlows)
			}
		}
	}

	return r
}

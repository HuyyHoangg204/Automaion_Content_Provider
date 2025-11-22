package router

import (
	"time"

	"github.com/onegreenvn/green-provider-services-backend/internal/handlers"
	"github.com/onegreenvn/green-provider-services-backend/internal/middleware"
	"github.com/onegreenvn/green-provider-services-backend/internal/services"
	"github.com/onegreenvn/green-provider-services-backend/internal/services/api_key"
	"github.com/onegreenvn/green-provider-services-backend/internal/services/auth"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

// SetupRouter configures the Gin router with user authentication routes
func SetupRouter(db *gorm.DB, rabbitMQService *services.RabbitMQService, basePath string) *gin.Engine {
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
	apiKeyService := api_key.NewService(db)

	// Create middleware with services
	bearerTokenMiddleware := middleware.NewBearerTokenMiddleware(authService, db)
	apiKeyMiddleware := middleware.NewAPIKeyMiddleware(apiKeyService)

	// Create SSE Hub for real-time log streaming
	sseHub := services.NewSSEHub()

	// Create handlers with services
	authHandler := handlers.NewAuthHandler(authService)
	boxHandler := handlers.NewBoxHandler(db)
	appHandler := handlers.NewAppHandler(db)
	appProxyHandler := handlers.NewAppProxyHandler(db)
	apiKeyHandler := handlers.NewAPIKeyHandler(db)
	machineHandler := handlers.NewMachineHandler(db)
	topicHandler := handlers.NewTopicHandler(db, sseHub, rabbitMQService)
	processLogHandler := handlers.NewProcessLogHandler(db, sseHub, rabbitMQService)

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

		// Machine routes (public - for machine self-registration)
		machines := api.Group("/machines")
		{
			machines.POST("/register", machineHandler.RegisterMachine)
			machines.GET("/:machine_id/frp-config", machineHandler.GetFrpConfigByMachineID)
			machines.PUT("/:machine_id/tunnel-url", machineHandler.UpdateTunnelURLByMachineID)
			machines.POST("/:machine_id/heartbeat", machineHandler.SendHeartbeat)
		}

		// Protected routes
		protected := api.Group("")
		protected.Use(apiKeyMiddleware.APIKeyAuthMiddleware(), bearerTokenMiddleware.BearerTokenAuthMiddleware())
		{
			// Auth protected routes
			authProtected := protected.Group("/auth")
			{
				authProtected.POST("/logout", authHandler.Logout)
				authProtected.GET("/profile", authHandler.GetProfile)
				authProtected.POST("/change-password", authHandler.ChangePassword)
			}

			// API Key routes
			apiKeys := protected.Group("/api-key")
			{
				apiKeys.GET("", apiKeyHandler.Get)
				apiKeys.POST("/generate", apiKeyHandler.Generate)
				apiKeys.PUT("/status", apiKeyHandler.UpdateStatus)
				apiKeys.DELETE("", apiKeyHandler.Delete)
			}

			// Box routes
			boxes := protected.Group("/boxes")
			{
				boxes.POST("", boxHandler.CreateBox)
				boxes.GET("", boxHandler.GetMyBoxes)
				boxes.GET("/:id", boxHandler.GetBoxByID)
				boxes.PUT("/:id", boxHandler.UpdateBox)
				boxes.DELETE("/:id", boxHandler.DeleteBox)
				boxes.GET("/:id/apps", boxHandler.GetAppsByBox)
			}

			// App Proxy routes - for direct platform operations
			appProxy := protected.Group("/app-proxy")
			{
				appProxy.Any("/:app_id/*platform_path", appProxyHandler.ProxyRequest)
			}

			// App routes
			apps := protected.Group("/apps")
			{
				apps.POST("", appHandler.CreateApp)
				apps.GET("", appHandler.GetMyApps)
				apps.GET("/:id", appHandler.GetAppByID)
				apps.PUT("/:id", appHandler.UpdateApp)
				apps.DELETE("/:id", appHandler.DeleteApp)
				apps.GET("/register-app", appHandler.GetRegisterAppDomains)
				apps.GET("/check-tunnel", appHandler.CheckTunnelURL)
			}

			// Topic routes
			topics := protected.Group("/topics")
			{
				topics.POST("", topicHandler.CreateTopic)
				topics.GET("", topicHandler.GetAllTopics)
				topics.GET("/:id", topicHandler.GetTopicByID)
				topics.PUT("/:id", topicHandler.UpdateTopic)
				topics.DELETE("/:id", topicHandler.DeleteTopic)
				// topics.POST("/:id/sync", topicHandler.SyncTopicWithGemini) // TODO: Implement later
			}

			// Process Log routes
			processLogs := api.Group("/process-logs")
			{
				// Public endpoint for automation backend to send logs
				processLogs.POST("", processLogHandler.CreateLog)

				// Protected routes
				processLogsProtected := processLogs.Group("")
				processLogsProtected.Use(apiKeyMiddleware.APIKeyAuthMiddleware(), bearerTokenMiddleware.BearerTokenAuthMiddleware())
				{
					processLogsProtected.GET("", processLogHandler.GetLogsByUser)
					processLogsProtected.GET("/:entity_type/:entity_id", processLogHandler.GetLogsByEntity)
					processLogsProtected.GET("/:entity_type/:entity_id/stream", processLogHandler.StreamLogsSSE)
				}
			}

			// Admin routes (requires admin privileges)
			admin := protected.Group("/admin")
			{
				admin.POST("/register", adminHandler.Register)
				admin.GET("/users", adminHandler.GetAllUsers)
				admin.PUT("/users/:id/status", adminHandler.SetUserStatus)
				admin.POST("/users/:id/reset-password", adminHandler.ResetPassword)
				admin.GET("/boxes", adminHandler.AdminGetAllBoxes)
				admin.GET("/apps", adminHandler.AdminGetAllApps)
			}
		}

	}

	return r
}

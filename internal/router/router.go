package router

import (
	"fmt"
	"os"
	"time"

	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
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

// getEnv gets environment variable or returns default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// SetupRouter configures the Gin router with user authentication routes
func SetupRouter(db *gorm.DB, rabbitMQService *services.RabbitMQService, sseHub *services.SSEHub, basePath string) *gin.Engine {
	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	// Create a new router
	r := gin.New()

	// Set max memory for multipart form (32MB for file uploads)
	r.MaxMultipartMemory = 32 << 20 // 32 MB

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

	// Get base URL from environment
	baseURL := getEnv("BASE_URL", "")
	if baseURL == "" {
		port := getEnv("PORT", "8080")
		baseURL = fmt.Sprintf("http://localhost:%s", port)
		logrus.Warnf("BASE_URL not set, using default: %s", baseURL)
	}

	// Create FileService first (needed by TopicService)
	fileRepo := repository.NewFileRepository(db)
	fileService := services.NewFileService(fileRepo, baseURL)

	// Create TopicService (needed by FileHandler để cache file IDs và TopicHandler)
	topicRepo := repository.NewTopicRepository(db)
	userProfileRepo := repository.NewUserProfileRepository(db)
	appRepo := repository.NewAppRepository(db)
	chromeProfileService := services.NewChromeProfileService(userProfileRepo, appRepo)
	logRepo := repository.NewProcessLogRepository(db)
	processLogService := services.NewProcessLogService(logRepo, sseHub, rabbitMQService, db)
	topicService := services.NewTopicService(topicRepo, userProfileRepo, appRepo, chromeProfileService, processLogService, fileService)
	geminiService := services.NewGeminiService(userProfileRepo, appRepo, topicRepo, chromeProfileService)

	// Create handlers with services
	authHandler := handlers.NewAuthHandler(authService)
	boxHandler := handlers.NewBoxHandler(db)
	appHandler := handlers.NewAppHandler(db)
	appProxyHandler := handlers.NewAppProxyHandler(db)
	apiKeyHandler := handlers.NewAPIKeyHandler(db)
	machineHandler := handlers.NewMachineHandler(db)
	topicHandler := handlers.NewTopicHandler(topicService)
	processLogHandler := handlers.NewProcessLogHandler(db, sseHub, rabbitMQService)
	fileHandler := handlers.NewFileHandler(db, baseURL, topicService)
	geminiHandler := handlers.NewGeminiHandler(geminiService)

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

		// File download route (public - supports token in query param)
		api.GET("/files/:id/download", fileHandler.DownloadFile)

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
				topics.GET("/:id/prompts", topicHandler.GetTopicPrompts)
				topics.PUT("/:id/prompts", topicHandler.UpdateTopicPrompts)
				topics.DELETE("/:id", topicHandler.DeleteTopic)
				// topics.POST("/:id/sync", topicHandler.SyncTopicWithGemini) // TODO: Implement later
			}

			// Gemini routes
			gemini := protected.Group("/gemini")
			{
				gemini.POST("/topics/:topic_id/generate-outline-and-upload", geminiHandler.GenerateOutlineAndUpload)
			}

			// File routes (upload and list require auth)
			files := protected.Group("/files")
			{
				files.POST("/upload", fileHandler.UploadFile)
				files.GET("", fileHandler.GetMyFiles)
				// Download endpoint moved to public routes (supports token in query param)
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
				// IMPORTANT: More specific routes must come before less specific ones
				admin.GET("/boxes/status", adminHandler.AdminGetAllBoxesWithStatus)
				admin.GET("/boxes", adminHandler.AdminGetAllBoxes)
				admin.GET("/apps", adminHandler.AdminGetAllApps)
			}
		}

	}

	return r
}

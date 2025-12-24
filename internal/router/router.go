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

// getEnvAsInt gets environment variable as int or returns default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, fmt.Sprintf("%d", defaultValue))
	var value int
	if _, err := fmt.Sscanf(valueStr, "%d", &value); err != nil {
		return defaultValue
	}
	return value
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
	// Create repositories first
	userRepo := repository.NewUserRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	userProfileRepo := repository.NewUserProfileRepository(db)
	appRepo := repository.NewAppRepository(db)
	geminiAccountRepo := repository.NewGeminiAccountRepository(db)
	
	// Create UserProfileService for RoleService (needed to create profile when assigning topic_creator role)
	// Note: boxRepo will be created later, but we need it here for UserProfileService
	boxRepoForUserProfile := repository.NewBoxRepository(db)
	userProfileService := services.NewUserProfileService(userProfileRepo, appRepo, geminiAccountRepo, boxRepoForUserProfile)
	
	roleService := services.NewRoleService(roleRepo, userRepo, userProfileService)

	// Create auth middleware
	// Create services
	authService := auth.NewAuthService(db, roleService)
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
	topicUserRepo := repository.NewTopicUserRepository(db) // New: For topic assignments
	boxRepo := boxRepoForUserProfile // Reuse the same instance
	topicRepo := repository.NewTopicRepository(db)
	chromeProfileService := services.NewChromeProfileService(userProfileRepo, appRepo, boxRepo, geminiAccountRepo, topicRepo)
	logRepo := repository.NewProcessLogRepository(db)
	processLogService := services.NewProcessLogService(logRepo, sseHub, rabbitMQService, db)
	geminiAccountService := services.NewGeminiAccountService(geminiAccountRepo, appRepo, boxRepo, topicRepo, topicUserRepo)
	topicService := services.NewTopicService(topicRepo, topicUserRepo, userProfileRepo, appRepo, boxRepo, chromeProfileService, processLogService, fileService, geminiAccountService)
	geminiService := services.NewGeminiService(userProfileRepo, appRepo, topicRepo, topicService, chromeProfileService)
	
	// Create ScriptService
	scriptRepo := repository.NewScriptRepository(db)
	scriptService := services.NewScriptService(
		scriptRepo,
		topicRepo,
		userProfileRepo,
		boxRepo,
		chromeProfileService,
		geminiAccountService,
		fileService,
	)
	
	// Create ScriptExecutionService
	scriptExecutionService := services.NewScriptExecutionService(
		scriptRepo,
		topicRepo,
		userProfileRepo,
		chromeProfileService,
		rabbitMQService,
	)

	// Inject ScriptExecutionService into ProcessLogService
	processLogService.SetScriptExecutionService(scriptExecutionService)

	// Start ProcessLogService RabbitMQ consumer (sau khi inject ScriptExecutionService)
	if rabbitMQService != nil {
		if err := processLogService.StartRabbitMQConsumer(); err != nil {
			logrus.Warnf("[Router] Failed to start RabbitMQ log consumer: %v", err)
		} else {
			logrus.Info("[Router] ✅ RabbitMQ log consumer started (with ScriptExecutionService)")
		}

		// Start log cleanup service (cleanup every 6 hours, keep logs for 1 day)
		logRetentionDays := getEnvAsInt("LOG_RETENTION_DAYS", 1)
		cleanupInterval := 6 * time.Hour
		processLogService.StartLogCleanup(cleanupInterval, logRetentionDays)
		logrus.Infof("[Router] Log cleanup service started (retention: %d days)", logRetentionDays)
	}

	// Create handlers with services
	authHandler := handlers.NewAuthHandler(authService, roleService)
	boxHandler := handlers.NewBoxHandler(db)
	appHandler := handlers.NewAppHandler(db)
	appProxyHandler := handlers.NewAppProxyHandler(db)
	apiKeyHandler := handlers.NewAPIKeyHandler(db)
	machineHandler := handlers.NewMachineHandler(db)
	topicHandler := handlers.NewTopicHandler(topicService, roleService)
	processLogHandler := handlers.NewProcessLogHandler(db, sseHub, rabbitMQService)
	fileHandler := handlers.NewFileHandler(db, baseURL, scriptService)
	geminiHandler := handlers.NewGeminiHandler(geminiService)
	geminiAccountHandler := handlers.NewGeminiAccountHandler(geminiAccountService, topicService)
	scriptHandler := handlers.NewScriptHandler(scriptService, scriptExecutionService, topicService)

	// Create admin handler with services
	adminHandler := handlers.NewAdminHandler(authService, db, topicService, scriptService)

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
				
				// Script routes (1-1 với user + topic) - phải đặt trước /:id để tránh conflict
				topics.POST("/:id/projects", scriptHandler.CreateProject) // New: Create project (and gem)
				topics.POST("/:id/scripts", scriptHandler.SaveScript)
				topics.GET("/:id/scripts", scriptHandler.GetScript)
				topics.DELETE("/:id/scripts", scriptHandler.DeleteScript)
				topics.POST("/:id/scripts/execute", scriptHandler.ExecuteScript)
				
				topics.GET("/:id", topicHandler.GetTopicByID)
				topics.PUT("/:id", topicHandler.UpdateTopic)
				topics.DELETE("/:id", topicHandler.DeleteTopic)
				// topics.POST("/:id/sync", topicHandler.SyncTopicWithGemini) // TODO: Implement later
			}

			// Gemini routes
			gemini := protected.Group("/gemini")
			{
				gemini.POST("/topics/:topic_id/generate-outline-and-upload", geminiHandler.GenerateOutlineAndUpload)
				// Gemini Account management routes
				geminiAccounts := gemini.Group("/accounts")
				{
					geminiAccounts.POST("/setup", geminiAccountHandler.SetupGeminiAccount)
					geminiAccounts.GET("", geminiAccountHandler.GetAllAccounts)
					geminiAccounts.GET("/:id", geminiAccountHandler.GetAccountByID)
				geminiAccounts.GET("/machine/:machine_id", geminiAccountHandler.GetAccountsByMachineID)
				geminiAccounts.GET("/:id/topics", geminiAccountHandler.GetTopicsByAccountID)
				geminiAccounts.PUT("/:id/lock", geminiAccountHandler.LockAccount)
				geminiAccounts.PUT("/:id/unlock", geminiAccountHandler.UnlockAccount)
				}
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
				// Role management routes
				admin.GET("/roles", adminHandler.GetAllRoles)
				admin.GET("/users/:id/roles", adminHandler.GetUserRoles)
				admin.POST("/users/:id/roles", adminHandler.AssignRoleToUser)
				admin.DELETE("/users/:id/roles", adminHandler.RemoveRoleFromUser)
				// Topic management routes
				// IMPORTANT: More specific routes must come before less specific ones
				admin.GET("/topics", adminHandler.GetAllTopics) // Must come before /topics/:id routes
				// Topic assignment management routes
				admin.POST("/topics/:id/assign", adminHandler.AssignTopicToUser)
				admin.GET("/topics/:id/users", adminHandler.GetTopicAssignedUsers)
				admin.DELETE("/topics/:id/users/:user_id", adminHandler.RemoveTopicAssignment)
				// IMPORTANT: More specific routes must come before less specific ones
				admin.GET("/boxes/status", adminHandler.AdminGetAllBoxesWithStatus)
				admin.GET("/boxes", adminHandler.AdminGetAllBoxes)
				admin.GET("/apps", adminHandler.AdminGetAllApps)
			}
		}

	}

	// Start script execution workers if RabbitMQ is available
	if rabbitMQService != nil {
		logrus.Info("[Router] RabbitMQ available, starting workers...")

		// Start project worker (new event-driven approach)
		if err := scriptExecutionService.StartProjectWorker(); err != nil {
			logrus.Errorf("[Router] Failed to start script project worker: %v", err)
		} else {
			logrus.Info("[Router] ✅ Script project worker started successfully")
		}

		// Start old execution worker (for backward compatibility, can be removed later)
		if err := scriptExecutionService.StartWorker(); err != nil {
			logrus.Errorf("[Router] Failed to start script execution worker: %v", err)
		} else {
			logrus.Info("[Router] ✅ Script execution worker started successfully")
		}

		// NOTE: Không dùng defer StopWorker() ở đây vì nó sẽ stop workers ngay khi SetupRouter() return!
		// Workers sẽ chạy trong background cho đến khi server shutdown
	} else {
		logrus.Warn("[Router] RabbitMQ not available, workers not started")
	}

	return r
}

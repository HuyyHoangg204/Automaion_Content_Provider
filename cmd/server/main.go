package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/onegreenvn/green-provider-services-backend/docs"
	"github.com/onegreenvn/green-provider-services-backend/internal/database"
	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/router"
	"github.com/onegreenvn/green-provider-services-backend/internal/services"
	"github.com/onegreenvn/green-provider-services-backend/internal/services/auth"
	"github.com/onegreenvn/green-provider-services-backend/internal/utils"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"

	// Import Swagger docs
	_ "github.com/onegreenvn/green-provider-services-backend/docs"
)

// @title User Management API
// @version 1.0
// @description User Management API with JWT Authentication
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter `Bearer ` followed by your JWT token (e.g. "Bearer <token>") or `ApiKey ` followed by your API key (e.g. "ApiKey <key>")

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Set Swagger base path dynamically
	basePath := getEnv("BASE_PATH", "/green-provider-services-api")
	docs.SwaggerInfo.BasePath = basePath

	// Configure logging
	configureLogging()

	// Initialize Sentry
	utils.InitSentry()

	// Initialize database connection
	db, err := database.InitDB()
	if err != nil {
		logrus.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize role service (needed by auth service)
	userRepo := repository.NewUserRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	roleService := services.NewRoleService(roleRepo, userRepo)

	// Initialize auth service
	authService := auth.NewAuthService(db, roleService)

	// Create SSE Hub (shared instance for both ProcessLogService and ProcessLogHandler)
	sseHub := services.NewSSEHub()

	// Initialize RabbitMQ service
	rabbitMQService, err := services.NewRabbitMQService()
	if err != nil {
		logrus.Warnf("Failed to initialize RabbitMQ: %v", err)
	} else {
		logrus.Info("RabbitMQ service initialized")
		defer rabbitMQService.Close()

		// Initialize ProcessLogService and start RabbitMQ consumer
		logRepo := repository.NewProcessLogRepository(db)
		processLogService := services.NewProcessLogService(logRepo, sseHub, rabbitMQService, db)

		if err := processLogService.StartRabbitMQConsumer(); err != nil {
			logrus.Warnf("Failed to start RabbitMQ log consumer: %v", err)
		} else {
			logrus.Info("RabbitMQ log consumer started")
			defer processLogService.StopRabbitMQConsumer()
		}

		// Start log cleanup service (cleanup every 6 hours, keep logs for 1 day)
		logRetentionDays := getEnvAsInt("LOG_RETENTION_DAYS", 1)
		cleanupInterval := 6 * time.Hour
		processLogService.StartLogCleanup(cleanupInterval, logRetentionDays)
		defer processLogService.StopLogCleanup()
	}

	// Create admin user if not exists
	if err := authService.CreateAdminUser(); err != nil {
		logrus.Warnf("Failed to create admin user: %v", err)
	} else {
		logrus.Info("Admin user check completed")
	}

	// Initialize token cleanup service
	tokenCleanupService := auth.NewTokenCleanupService(db)
	tokenCleanupService.Start()
	defer tokenCleanupService.Stop()

	// Initialize box status update service (update offline boxes every 1 minute)
	boxStatusService := services.NewBoxStatusUpdateService(db)
	boxStatusService.Start()
	defer boxStatusService.Stop()

	// Initialize router with RabbitMQ service and SSE Hub
	r := router.SetupRouter(db, rabbitMQService, sseHub, basePath)

	// Configure HTTP server
	port := getEnv("PORT", "8080")
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		logrus.Infof("Server starting on port %s", port)
		logrus.Infof("API Health Check: http://localhost:%s/api/v1/health", port)
		logrus.Infof("Swagger UI: http://localhost:%s/swagger/index.html", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logrus.Info("Shutting down server...")

	// Create a deadline for server shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown the server
	if err := srv.Shutdown(ctx); err != nil {
		logrus.Fatalf("Server forced to shutdown: %v", err)
	}

	logrus.Info("Server exited properly")
}

func configureLogging() {
	logLevel := getEnv("LOG_LEVEL", "info")
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, fmt.Sprintf("%d", defaultValue))
	var value int
	if _, err := fmt.Sscanf(valueStr, "%d", &value); err != nil {
		return defaultValue
	}
	return value
}

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

	"github.com/onegreenvn/green-provider-services-backend/internal/database"
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
// @description Enter `Bearer ` followed by your JWT token (e.g. "Bearer <token>")

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Configure logging
	configureLogging()

	// Initialize Sentry
	utils.InitSentry()

	// Initialize database connection
	db, err := database.InitDB()
	if err != nil {
		logrus.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize auth service
	authService := auth.NewAuthService(db)

	// Initialize RabbitMQ service
	rabbitMQService, err := services.NewRabbitMQService()
	if err != nil {
		logrus.Warnf("Failed to initialize RabbitMQ: %v", err)
	} else {
		logrus.Info("RabbitMQ service initialized")
		defer rabbitMQService.Close()
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

	// Initialize router với RabbitMQ service
	r := router.SetupRouter(db, rabbitMQService)

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

package database

import (
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/onegreenvn/green-provider-services-backend/internal/models"
)

// DB is the global database instance
var DB *gorm.DB

// InitDB initializes the database connection and performs migrations
func InitDB() (*gorm.DB, error) {
	// Get database connection parameters from environment variables
	host := getEnv("DB_HOST", "")
	port := getEnv("DB_PORT", "")
	user := getEnv("DB_USER", "")
	password := getEnv("DB_PASSWORD", "")
	dbname := getEnv("DB_NAME", "")
	sslmode := getEnv("DB_SSLMODE", "disable")

	// Validate required environment variables
	if host == "" || port == "" || user == "" || password == "" || dbname == "" {
		return nil, fmt.Errorf("missing required database environment variables. Please check your .env file")
	}

	// Create DSN string
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	// Configure GORM logger
	gormLogger := logger.New(
		logrus.New(),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Error,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	// Open database connection
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Set connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Create public schema if it doesn't exist
	err = db.Exec("CREATE SCHEMA IF NOT EXISTS public").Error
	if err != nil {
		return nil, fmt.Errorf("failed to create public schema: %w", err)
	}

	// Set search_path to public
	err = db.Exec("SET search_path TO public").Error
	if err != nil {
		return nil, fmt.Errorf("failed to set search_path: %w", err)
	}

	// Enable UUID extension
	err = db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\" SCHEMA public").Error
	if err != nil {
		return nil, fmt.Errorf("failed to enable UUID extension: %w", err)
	}

	// Auto migrate the schema
	err = db.AutoMigrate(
		&models.User{},
		&models.RefreshToken{},
		&models.Box{},
		&models.App{},
		&models.UserProfile{},
		&models.Topic{},
		&models.ProcessLog{},
		&models.APIKey{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	// Migration: Drop user_data_dir column from user_profiles table if it exists
	// This column is no longer needed as automation backend resolves path automatically
	var columnExists bool
	err = db.Raw(`
		SELECT EXISTS (
			SELECT 1 
			FROM information_schema.columns 
			WHERE table_schema = 'public' 
			AND table_name = 'user_profiles' 
			AND column_name = 'user_data_dir'
		)
	`).Scan(&columnExists).Error
	if err != nil {
		logrus.Warnf("Failed to check if user_data_dir column exists: %v", err)
	} else if columnExists {
		logrus.Info("Dropping user_data_dir column from user_profiles table...")
		err = db.Exec("ALTER TABLE user_profiles DROP COLUMN IF EXISTS user_data_dir").Error
		if err != nil {
			logrus.Warnf("Failed to drop user_data_dir column: %v", err)
		} else {
			logrus.Info("Successfully dropped user_data_dir column")
		}
	}

	// Set global DB instance
	DB = db

	logrus.Info("Database connection established and migrations completed")
	return db, nil
}

// GetDB returns the global database instance
func GetDB() *gorm.DB {
	return DB
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

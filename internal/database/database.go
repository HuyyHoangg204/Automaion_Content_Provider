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
		Logger:                                   gormLogger,
		DisableForeignKeyConstraintWhenMigrating: true, // Disable FK constraints during migration to avoid order issues
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
		&models.TopicUser{}, // New: Topic-User assignment table
		&models.ProcessLog{},
		&models.APIKey{},
		&models.File{},
		&models.Role{},
		&models.GeminiAccount{}, // New: Gemini accounts table
	)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	// Migrate script tables separately to ensure correct order for foreign keys
	// Order: Script -> ScriptProject -> ScriptPrompt, ScriptEdge
	// Check if tables exist before migrating to avoid foreign key constraint issues
	var scriptsTableExists bool
	err = db.Raw(`
		SELECT EXISTS (
			SELECT 1 
			FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'scripts'
		)
	`).Scan(&scriptsTableExists).Error
	if err == nil && !scriptsTableExists {
		err = db.AutoMigrate(&models.Script{})
		if err != nil {
			return nil, fmt.Errorf("failed to migrate scripts table: %w", err)
		}
		logrus.Info("Successfully migrated scripts table")
		scriptsTableExists = true // Update flag after migration
	}

	var scriptProjectsTableExists bool
	err = db.Raw(`
		SELECT EXISTS (
			SELECT 1 
			FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'script_projects'
		)
	`).Scan(&scriptProjectsTableExists).Error
	if err == nil && !scriptProjectsTableExists {
		err = db.AutoMigrate(&models.ScriptProject{})
		if err != nil {
			return nil, fmt.Errorf("failed to migrate script_projects table: %w", err)
		}
		logrus.Info("Successfully migrated script_projects table")
		scriptProjectsTableExists = true // Update flag after migration
	}

	// Migrate ScriptPrompt separately (depends on ScriptProject - must exist first)
	var scriptPromptsTableExists bool
	err = db.Raw(`
		SELECT EXISTS (
			SELECT 1 
			FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'script_prompts'
		)
	`).Scan(&scriptPromptsTableExists).Error
	if err == nil && !scriptPromptsTableExists {
		err = db.AutoMigrate(&models.ScriptPrompt{})
		if err != nil {
			return nil, fmt.Errorf("failed to migrate script_prompts table: %w", err)
		}
		logrus.Info("Successfully migrated script_prompts table")
		scriptPromptsTableExists = true // Update flag after migration
	}

	// Migrate ScriptEdge separately (depends on Script - must exist first)
	var scriptEdgesTableExists bool
	err = db.Raw(`
		SELECT EXISTS (
			SELECT 1 
			FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'script_edges'
		)
	`).Scan(&scriptEdgesTableExists).Error
	if err == nil && !scriptEdgesTableExists {
		err = db.AutoMigrate(&models.ScriptEdge{})
		if err != nil {
			return nil, fmt.Errorf("failed to migrate script_edges table: %w", err)
		}
		logrus.Info("Successfully migrated script_edges table")
		scriptEdgesTableExists = true // Update flag after migration
	}

	// Migrate ScriptExecution separately (depends on Script - must exist first)
	var scriptExecutionsTableExists bool
	err = db.Raw(`
		SELECT EXISTS (
			SELECT 1 
			FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'script_executions'
		)
	`).Scan(&scriptExecutionsTableExists).Error
	if err == nil && !scriptExecutionsTableExists {
		err = db.AutoMigrate(&models.ScriptExecution{})
		if err != nil {
			return nil, fmt.Errorf("failed to migrate script_executions table: %w", err)
		}
		logrus.Info("Successfully migrated script_executions table")
		scriptExecutionsTableExists = true // Update flag after migration
	}

	// Migration: Add debug_port column to script_executions table if it doesn't exist
	var debugPortColumnExists bool
	err = db.Raw(`
		SELECT EXISTS (
			SELECT 1 
			FROM information_schema.columns 
			WHERE table_schema = 'public' 
			AND table_name = 'script_executions' 
			AND column_name = 'debug_port'
		)
	`).Scan(&debugPortColumnExists).Error
	if err != nil {
		logrus.Warnf("Failed to check if debug_port column exists: %v", err)
	} else if !debugPortColumnExists {
		logrus.Info("Adding debug_port column to script_executions table...")
		err = db.Exec("ALTER TABLE script_executions ADD COLUMN IF NOT EXISTS debug_port INTEGER DEFAULT 0").Error
		if err != nil {
			logrus.Warnf("Failed to add debug_port column: %v", err)
		} else {
			logrus.Info("Successfully added debug_port column")
		}
	}

	// Migrate ScriptProjectExecution separately (depends on ScriptExecution - must exist first)
	var scriptProjectExecutionsTableExists bool
	err = db.Raw(`
		SELECT EXISTS (
			SELECT 1 
			FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'script_project_executions'
		)
	`).Scan(&scriptProjectExecutionsTableExists).Error
	if err == nil && !scriptProjectExecutionsTableExists {
		err = db.AutoMigrate(&models.ScriptProjectExecution{})
		if err != nil {
			return nil, fmt.Errorf("failed to migrate script_project_executions table: %w", err)
		}
		logrus.Info("Successfully migrated script_project_executions table")
	}

	// Note: We don't create foreign key constraints for script_prompts -> script_projects
	// because script_projects uses composite primary key (script_id, project_id) and GORM doesn't handle composite FK well.
	// We rely on application logic for referential integrity.
	// Similarly, script_edges uses varchar for source/target, so no FK constraints are needed.

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

	// Migration: Add is_online column to boxes table if it doesn't exist
	var isOnlineColumnExists bool
	err = db.Raw(`
		SELECT EXISTS (
			SELECT 1 
			FROM information_schema.columns 
			WHERE table_schema = 'public' 
			AND table_name = 'boxes' 
			AND column_name = 'is_online'
		)
	`).Scan(&isOnlineColumnExists).Error
	if err != nil {
		logrus.Warnf("Failed to check if is_online column exists: %v", err)
	} else if !isOnlineColumnExists {
		logrus.Info("Adding is_online column to boxes table...")
		err = db.Exec("ALTER TABLE boxes ADD COLUMN IF NOT EXISTS is_online BOOLEAN DEFAULT false").Error
		if err != nil {
			logrus.Warnf("Failed to add is_online column: %v", err)
		} else {
			logrus.Info("Successfully added is_online column")
			// Create index for better query performance
			err = db.Exec("CREATE INDEX IF NOT EXISTS idx_boxes_is_online ON boxes(is_online)").Error
			if err != nil {
				logrus.Warnf("Failed to create index on is_online: %v", err)
			}
		}
	}

	// Migration: Add system metrics columns to boxes table if they don't exist
	migrations := []struct {
		columnName string
		columnType string
		index      bool
	}{
		{"cpu_usage", "DECIMAL(5,2)", false},
		{"memory_free_gb", "DECIMAL(5,2)", false},
		{"running_profiles", "INTEGER DEFAULT 0", true},
	}

	for _, migration := range migrations {
		var columnExists bool
		err = db.Raw(`
			SELECT EXISTS (
				SELECT 1 
				FROM information_schema.columns 
				WHERE table_schema = 'public' 
				AND table_name = 'boxes' 
				AND column_name = ?
			)
		`, migration.columnName).Scan(&columnExists).Error
		if err != nil {
			logrus.Warnf("Failed to check if %s column exists: %v", migration.columnName, err)
			continue
		}
		if !columnExists {
			logrus.Infof("Adding %s column to boxes table...", migration.columnName)
			err = db.Exec(fmt.Sprintf("ALTER TABLE boxes ADD COLUMN IF NOT EXISTS %s %s", migration.columnName, migration.columnType)).Error
			if err != nil {
				logrus.Warnf("Failed to add %s column: %v", migration.columnName, err)
			} else {
				logrus.Infof("Successfully added %s column", migration.columnName)
				if migration.index {
					indexName := fmt.Sprintf("idx_boxes_%s", migration.columnName)
					err = db.Exec(fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON boxes(%s)", indexName, migration.columnName)).Error
					if err != nil {
						logrus.Warnf("Failed to create index on %s: %v", migration.columnName, err)
					}
				}
			}
		}
	}

	// Migration: Drop legacy Gemini-related columns from topics (moved to script_projects in new model)
	var legacyCols []string
	err = db.Raw(`SELECT column_name FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'topics' AND column_name IN ('gemini_gem_id', 'gemini_gem_name', 'gemini_account_id')`).Scan(&legacyCols).Error
	if err != nil {
		logrus.Warnf("Failed to check for legacy gemini columns on topics: %v", err)
	} else {
		for _, col := range legacyCols {
			logrus.Infof("Dropping legacy column %s from topics table...", col)
			if err := db.Exec(fmt.Sprintf("ALTER TABLE topics DROP COLUMN IF EXISTS %s CASCADE", col)).Error; err != nil {
				logrus.Warnf("Failed to drop legacy column %s from topics: %v", col, err)
			}
		}
	}

	// Migration: Drop old unique constraint on machine_id (nếu có)
	// Và thêm unique constraint trên (email, machine_id) để đảm bảo 1 machine không có 2 accounts cùng email
	// Nhưng cho phép nhiều machines có cùng email (cùng account)
	var oldUniqueIndexExists bool
	err = db.Raw(`
		SELECT EXISTS (
			SELECT 1 
			FROM pg_indexes 
			WHERE schemaname = 'public' 
			AND tablename = 'gemini_accounts' 
			AND indexname = 'idx_gemini_accounts_machine_id_unique'
		)
	`).Scan(&oldUniqueIndexExists).Error
	if err != nil {
		logrus.Warnf("Failed to check if old unique index exists: %v", err)
	} else if oldUniqueIndexExists {
		logrus.Info("Dropping old unique index on machine_id...")
		err = db.Exec("DROP INDEX IF EXISTS idx_gemini_accounts_machine_id_unique").Error
		if err != nil {
			logrus.Warnf("Failed to drop old unique index: %v", err)
		} else {
			logrus.Info("Successfully dropped old unique index on machine_id")
		}
	}

	// Migration: Add unique constraint on (email, machine_id)
	// Đảm bảo 1 machine không có 2 accounts cùng email, nhưng cho phép nhiều machines có cùng email
	var emailMachineUniqueExists bool
	err = db.Raw(`
		SELECT EXISTS (
			SELECT 1 
			FROM pg_indexes 
			WHERE schemaname = 'public' 
			AND tablename = 'gemini_accounts' 
			AND indexname = 'idx_gemini_accounts_email_machine'
		)
	`).Scan(&emailMachineUniqueExists).Error
	if err != nil {
		logrus.Warnf("Failed to check if email_machine unique index exists: %v", err)
	} else if !emailMachineUniqueExists {
		logrus.Info("Creating unique index on (email, machine_id)...")
		err = db.Exec(`
			CREATE UNIQUE INDEX IF NOT EXISTS idx_gemini_accounts_email_machine 
			ON gemini_accounts(email, machine_id)
		`).Error
		if err != nil {
			logrus.Warnf("Failed to create unique index on (email, machine_id): %v", err)
		} else {
			logrus.Info("Successfully created unique index on (email, machine_id)")
		}
	}

	// Migration: Drop account_index column from gemini_accounts table if it exists (1 machine = 1 account, không cần account_index)
	var accountIndexColumnExists bool
	err = db.Raw(`
		SELECT EXISTS (
			SELECT 1 
			FROM information_schema.columns 
			WHERE table_schema = 'public' 
			AND table_name = 'gemini_accounts' 
			AND column_name = 'account_index'
		)
	`).Scan(&accountIndexColumnExists).Error
	if err != nil {
		logrus.Warnf("Failed to check if account_index column exists: %v", err)
	} else if accountIndexColumnExists {
		logrus.Info("Dropping account_index column from gemini_accounts table...")
		// Drop unique constraint on (machine_id, account_index) if exists
		err = db.Exec("ALTER TABLE gemini_accounts DROP CONSTRAINT IF EXISTS gemini_accounts_machine_id_account_index_key").Error
		if err != nil {
			logrus.Warnf("Failed to drop unique index on (machine_id, account_index): %v", err)
		}
		// Drop column
		err = db.Exec("ALTER TABLE gemini_accounts DROP COLUMN IF NOT EXISTS account_index").Error
		if err != nil {
			logrus.Warnf("Failed to drop account_index column: %v", err)
		} else {
			logrus.Info("Successfully dropped account_index column")
		}
	}

	// Migration: Create default roles if they don't exist
	defaultRoles := []struct {
		name        string
		description string
	}{
		{"topic_creator", "Can create topics"},
		{"topic_user", "Can access and use topics"},
	}

	for _, roleData := range defaultRoles {
		var roleExists bool
		err = db.Raw(`
			SELECT EXISTS (
				SELECT 1 
				FROM roles 
				WHERE name = ?
			)
		`, roleData.name).Scan(&roleExists).Error
		if err != nil {
			logrus.Warnf("Failed to check if %s role exists: %v", roleData.name, err)
			continue
		}
		if !roleExists {
			logrus.Infof("Creating default role '%s'...", roleData.name)
			role := &models.Role{
				Name:        roleData.name,
				Description: roleData.description,
			}
			if err := db.Create(role).Error; err != nil {
				logrus.Warnf("Failed to create %s role: %v", roleData.name, err)
			} else {
				logrus.Infof("Successfully created %s role", roleData.name)
			}
		}
	}

	// Migration: Add unique constraint on scripts (topic_id, user_id) for 1-1 relationship
	var scriptsUniqueIndexExists bool
	err = db.Raw(`
		SELECT EXISTS (
			SELECT 1 
			FROM pg_indexes 
			WHERE schemaname = 'public' 
			AND tablename = 'scripts' 
			AND indexname = 'idx_scripts_topic_user_unique'
		)
	`).Scan(&scriptsUniqueIndexExists).Error
	if err != nil {
		logrus.Warnf("Failed to check if scripts unique index exists: %v", err)
	} else if !scriptsUniqueIndexExists {
		logrus.Info("Creating unique index on scripts (topic_id, user_id)...")
		err = db.Exec(`
			CREATE UNIQUE INDEX IF NOT EXISTS idx_scripts_topic_user_unique 
			ON scripts(topic_id, user_id)
		`).Error
		if err != nil {
			logrus.Warnf("Failed to create unique index on scripts (topic_id, user_id): %v", err)
		} else {
			logrus.Info("Successfully created unique index on scripts (topic_id, user_id)")
		}
	}

	// Migration: Add filename column to script_projects table if it doesn't exist
	var scriptProjectsFilenameColumnExists bool
	err = db.Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = 'public'
			AND table_name = 'script_projects'
			AND column_name = 'filename'
		)
	`).Scan(&scriptProjectsFilenameColumnExists).Error
	if err != nil {
		logrus.Warnf("Failed to check if filename column exists on script_projects: %v", err)
	} else if !scriptProjectsFilenameColumnExists {
		logrus.Info("Adding filename column to script_projects table...")
		err = db.Exec("ALTER TABLE script_projects ADD COLUMN IF NOT EXISTS filename VARCHAR(255)").Error
		if err != nil {
			logrus.Warnf("Failed to add filename column to script_projects: %v", err)
		} else {
			logrus.Info("Successfully added filename column to script_projects")
		}
	}

	// Migration: Drop gem_name column from script_projects table (không lưu trong DB nữa, generate từ name khi cần)
	var scriptProjectsGemNameColumnExists bool
	err = db.Raw(`
		SELECT EXISTS (
			SELECT 1 
			FROM information_schema.columns 
			WHERE table_schema = 'public' 
			AND table_name = 'script_projects'
			AND column_name = 'gem_name'
		)
	`).Scan(&scriptProjectsGemNameColumnExists).Error
	if err != nil {
		logrus.Warnf("Failed to check if gem_name column exists on script_projects: %v", err)
	} else if scriptProjectsGemNameColumnExists {
		logrus.Info("Dropping gem_name column from script_projects table (no longer stored in DB)...")
		err = db.Exec("ALTER TABLE script_projects DROP COLUMN IF EXISTS gem_name").Error
		if err != nil {
			logrus.Warnf("Failed to drop gem_name column from script_projects: %v", err)
		} else {
			logrus.Info("Successfully dropped gem_name column from script_projects")
		}
	}

	// Migration: Add gemini_account_id column to script_projects table if it doesn't exist
	var scriptProjectsGeminiAccountColumnExists bool
	err = db.Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = 'public'
			AND table_name = 'script_projects'
			AND column_name = 'gemini_account_id'
		)
	`).Scan(&scriptProjectsGeminiAccountColumnExists).Error
	if err != nil {
		logrus.Warnf("Failed to check if gemini_account_id column exists on script_projects: %v", err)
	} else if !scriptProjectsGeminiAccountColumnExists {
		logrus.Info("Adding gemini_account_id column to script_projects table...")
		err = db.Exec("ALTER TABLE script_projects ADD COLUMN IF NOT EXISTS gemini_account_id UUID").Error
		if err != nil {
			logrus.Warnf("Failed to add gemini_account_id column to script_projects: %v", err)
		} else {
			logrus.Info("Successfully added gemini_account_id column to script_projects")
		}
	}

	// Migration: Add description and instructions columns to script_projects table if they don't exist
	var projectDescriptionColumnExists bool
	err = db.Raw(`
		SELECT EXISTS (
			SELECT 1 
			FROM information_schema.columns 
			WHERE table_schema = 'public' 
			AND table_name = 'script_projects' 
			AND column_name = 'description'
		)
	`).Scan(&projectDescriptionColumnExists).Error
	if err != nil {
		logrus.Warnf("Failed to check if description column exists on script_projects: %v", err)
	} else if !projectDescriptionColumnExists {
		logrus.Info("Adding description column to script_projects table...")
		err = db.Exec("ALTER TABLE script_projects ADD COLUMN IF NOT EXISTS description TEXT").Error
		if err != nil {
			logrus.Warnf("Failed to add description column to script_projects: %v", err)
		} else {
			logrus.Info("Successfully added description column to script_projects")
		}
	}

	var projectInstructionsColumnExists bool
	err = db.Raw(`
		SELECT EXISTS (
			SELECT 1 
			FROM information_schema.columns 
			WHERE table_schema = 'public' 
			AND table_name = 'script_projects' 
			AND column_name = 'instructions'
		)
	`).Scan(&projectInstructionsColumnExists).Error
	if err != nil {
		logrus.Warnf("Failed to check if instructions column exists on script_projects: %v", err)
	} else if !projectInstructionsColumnExists {
		logrus.Info("Adding instructions column to script_projects table...")
		err = db.Exec("ALTER TABLE script_projects ADD COLUMN IF NOT EXISTS instructions TEXT").Error
		if err != nil {
			logrus.Warnf("Failed to add instructions column to script_projects: %v", err)
		} else {
			logrus.Info("Successfully added instructions column to script_projects")
		}
	}

	// Migration: Add filename column to script_prompts table if it doesn't exist
	var scriptPromptsFilenameColumnExists bool
	err = db.Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = 'public'
			AND table_name = 'script_prompts'
			AND column_name = 'filename'
		)
	`).Scan(&scriptPromptsFilenameColumnExists).Error
	if err != nil {
		logrus.Warnf("Failed to check if filename column exists on script_prompts: %v", err)
	} else if !scriptPromptsFilenameColumnExists {
		logrus.Info("Adding filename column to script_prompts table...")
		err = db.Exec("ALTER TABLE script_prompts ADD COLUMN IF NOT EXISTS filename VARCHAR(255)").Error
		if err != nil {
			logrus.Warnf("Failed to add filename column to script_prompts: %v", err)
		} else {
			logrus.Info("Successfully added filename column to script_prompts")
		}
	}

	// Migration: Add input_files column to script_prompts table if it doesn't exist
	var scriptPromptsInputFilesColumnExists bool
	err = db.Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = 'public'
			AND table_name = 'script_prompts'
			AND column_name = 'input_files'
		)
	`).Scan(&scriptPromptsInputFilesColumnExists).Error
	if err != nil {
		logrus.Warnf("Failed to check if input_files column exists on script_prompts: %v", err)
	} else if !scriptPromptsInputFilesColumnExists {
		logrus.Info("Adding input_files column to script_prompts table...")
		err = db.Exec("ALTER TABLE script_prompts ADD COLUMN IF NOT EXISTS input_files JSONB").Error
		if err != nil {
			logrus.Warnf("Failed to add input_files column to script_prompts: %v", err)
		} else {
			logrus.Info("Successfully added input_files column to script_prompts")
		}
	}

	// Migration: Add temp_prompt_id column to script_prompts table if it doesn't exist
	var tempPromptIDColumnExists bool
	err = db.Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = 'public'
			AND table_name = 'script_prompts'
			AND column_name = 'temp_prompt_id'
		)
	`).Scan(&tempPromptIDColumnExists).Error
	if err != nil {
		logrus.Warnf("Failed to check if temp_prompt_id column exists on script_prompts: %v", err)
	} else if !tempPromptIDColumnExists {
		logrus.Info("Adding temp_prompt_id column to script_prompts table...")
		err = db.Exec("ALTER TABLE script_prompts ADD COLUMN IF NOT EXISTS temp_prompt_id VARCHAR(255)").Error
		if err != nil {
			logrus.Warnf("Failed to add temp_prompt_id column to script_prompts: %v", err)
		} else {
			logrus.Info("Successfully added temp_prompt_id column to script_prompts")
		}
		// Add index for better query performance
		err = db.Exec("CREATE INDEX IF NOT EXISTS idx_script_prompts_temp_prompt_id ON script_prompts(temp_prompt_id)").Error
		if err != nil {
			logrus.Warnf("Failed to create index on temp_prompt_id: %v", err)
		} else {
			logrus.Info("Successfully created index on temp_prompt_id")
		}
	}

	// Migration: Add project_id and temp_prompt_id columns to files table if they don't exist
	var filesProjectIDColumnExists bool
	err = db.Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = 'public'
			AND table_name = 'files'
			AND column_name = 'project_id'
		)
	`).Scan(&filesProjectIDColumnExists).Error
	if err != nil {
		logrus.Warnf("Failed to check if project_id column exists on files: %v", err)
	} else if !filesProjectIDColumnExists {
		logrus.Info("Adding project_id column to files table...")
		err = db.Exec("ALTER TABLE files ADD COLUMN IF NOT EXISTS project_id VARCHAR(255)").Error
		if err != nil {
			logrus.Warnf("Failed to add project_id column to files: %v", err)
		} else {
			logrus.Info("Successfully added project_id column to files")
			// Add index for better query performance
			err = db.Exec("CREATE INDEX IF NOT EXISTS idx_files_project_id ON files(project_id)").Error
			if err != nil {
				logrus.Warnf("Failed to create index on files.project_id: %v", err)
			}
		}
	}

	var filesTempPromptIDColumnExists bool
	err = db.Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = 'public'
			AND table_name = 'files'
			AND column_name = 'temp_prompt_id'
		)
	`).Scan(&filesTempPromptIDColumnExists).Error
	if err != nil {
		logrus.Warnf("Failed to check if temp_prompt_id column exists on files: %v", err)
	} else if !filesTempPromptIDColumnExists {
		logrus.Info("Adding temp_prompt_id column to files table...")
		err = db.Exec("ALTER TABLE files ADD COLUMN IF NOT EXISTS temp_prompt_id VARCHAR(255)").Error
		if err != nil {
			logrus.Warnf("Failed to add temp_prompt_id column to files: %v", err)
		} else {
			logrus.Info("Successfully added temp_prompt_id column to files")
			// Add index for better query performance
			err = db.Exec("CREATE INDEX IF NOT EXISTS idx_files_temp_prompt_id ON files(temp_prompt_id)").Error
			if err != nil {
				logrus.Warnf("Failed to create index on files.temp_prompt_id: %v", err)
			}
			// Add composite index for query performance
			err = db.Exec("CREATE INDEX IF NOT EXISTS idx_files_user_project_prompt ON files(user_id, project_id, temp_prompt_id)").Error
			if err != nil {
				logrus.Warnf("Failed to create composite index on files: %v", err)
			}
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

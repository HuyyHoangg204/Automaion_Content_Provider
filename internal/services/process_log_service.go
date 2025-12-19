package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type ProcessLogService struct {
	logRepo                *repository.ProcessLogRepository
	topicRepo              *repository.TopicRepository
	sseHub                 *SSEHub
	rabbitMQ               *RabbitMQService
	db                     *gorm.DB
	scriptExecutionService *ScriptExecutionService // Optional: injected later
	stopChan               chan bool
	cleanupStopChan        chan bool
}

func NewProcessLogService(logRepo *repository.ProcessLogRepository, sseHub *SSEHub, rabbitMQ *RabbitMQService, db *gorm.DB) *ProcessLogService {
	return &ProcessLogService{
		logRepo:         logRepo,
		topicRepo:       repository.NewTopicRepository(db),
		sseHub:          sseHub,
		rabbitMQ:        rabbitMQ,
		db:              db,
		stopChan:        make(chan bool),
		cleanupStopChan: make(chan bool),
	}
}

// SetScriptExecutionService sets the script execution service (injected after creation to avoid circular dependency)
func (s *ProcessLogService) SetScriptExecutionService(scriptExecutionService *ScriptExecutionService) {
	s.scriptExecutionService = scriptExecutionService
}

// StartRabbitMQConsumer starts consuming logs from RabbitMQ queue
func (s *ProcessLogService) StartRabbitMQConsumer() error {
	// Declare queue
	queueName := "process_logs"
	_, err := s.rabbitMQ.channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Consume messages
	msgs, err := s.rabbitMQ.channel.Consume(
		queueName, // queue
		"",        // consumer
		true,      // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	logrus.Info("RabbitMQ consumer started for process_logs queue")

	// Process messages in goroutine
	go func() {
		for {
			select {
			case <-s.stopChan:
				logrus.Info("RabbitMQ consumer stopped")
				return
			case msg, ok := <-msgs:
				if !ok {
					logrus.Warn("RabbitMQ channel closed")
					return
				}

				// Process message
				if err := s.processLogMessage(msg.Body); err != nil {
					logrus.Errorf("Failed to process log message: %v", err)
				}
			}
		}
	}()

	return nil
}

// StopRabbitMQConsumer stops the consumer
func (s *ProcessLogService) StopRabbitMQConsumer() {
	close(s.stopChan)
}

// processLogMessage processes a log message from RabbitMQ
func (s *ProcessLogService) processLogMessage(body []byte) error {
	var req models.ProcessLogRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return fmt.Errorf("failed to unmarshal log message: %w", err)
	}

	// Validate UUIDs - skip logs with "unknown" values
	if req.EntityID == "unknown" || req.UserID == "unknown" {
		logrus.Warnf("Skipping log with unknown IDs: entity_id=%s, user_id=%s", req.EntityID, req.UserID)
		return nil // Skip this log instead of failing
	}

	// Convert metadata
	var metadataJSON models.JSON
	if req.Metadata != nil {
		metadataBytes, _ := json.Marshal(req.Metadata)
		json.Unmarshal(metadataBytes, &metadataJSON)
	}

	// Create log entry
	log := &models.ProcessLog{
		EntityType: req.EntityType,
		EntityID:   req.EntityID,
		UserID:     req.UserID,
		MachineID:  req.MachineID,
		Stage:      req.Stage,
		Status:     req.Status,
		Message:    req.Message,
		Metadata:   metadataJSON,
		CreatedAt:  time.Now(),
	}

	// Save to database
	if err := s.logRepo.Create(log); err != nil {
		return fmt.Errorf("failed to save log to database: %w", err)
	}

	// Broadcast via SSE
	s.sseHub.BroadcastLog(log)

	s.handleTopicLog(req.EntityType, req.EntityID, req.Stage, req.Status, req.Metadata)

	// Handle script execution logs
	s.handleScriptExecutionLog(req.EntityType, req.EntityID, req.Stage, req.Status, req.Metadata)

	return nil
}

// CreateLog creates a log entry (can be called directly or via RabbitMQ)
func (s *ProcessLogService) CreateLog(req *models.ProcessLogRequest) (*models.ProcessLog, error) {
	// Convert metadata
	var metadataJSON models.JSON
	if req.Metadata != nil {
		metadataBytes, _ := json.Marshal(req.Metadata)
		json.Unmarshal(metadataBytes, &metadataJSON)
	}

	// Create log entry
	log := &models.ProcessLog{
		EntityType: req.EntityType,
		EntityID:   req.EntityID,
		UserID:     req.UserID,
		MachineID:  req.MachineID,
		Stage:      req.Stage,
		Status:     req.Status,
		Message:    req.Message,
		Metadata:   metadataJSON,
		CreatedAt:  time.Now(),
	}

	// Save to database
	if err := s.logRepo.Create(log); err != nil {
		return nil, fmt.Errorf("failed to create log: %w", err)
	}

	// Broadcast via SSE
	s.sseHub.BroadcastLog(log)

	s.handleTopicLog(req.EntityType, req.EntityID, req.Stage, req.Status, req.Metadata)

	// Handle script execution logs
	s.handleScriptExecutionLog(req.EntityType, req.EntityID, req.Stage, req.Status, req.Metadata)

	return log, nil
}

// updateTopicOnCompletion updates topic when automation backend reports completion
func (s *ProcessLogService) updateTopicOnCompletion(topicID string, metadata map[string]interface{}) {
	// Get topic
	topic, err := s.topicRepo.GetByID(topicID)
	if err != nil {
		logrus.Errorf("Failed to get topic %s for completion update: %v", topicID, err)
		return
	}

	// Update SyncStatus
	topic.SyncStatus = "synced"
	now := time.Now()
	topic.LastSyncedAt = &now
	topic.SyncError = ""

	// Update GeminiGemName từ metadata nếu có
	if metadata != nil {
		if gemName, ok := metadata["gem_name"].(string); ok && gemName != "" {
			topic.GeminiGemName = gemName
		}
	}

	// Save to database
	if err := s.topicRepo.Update(topic); err != nil {
		logrus.Errorf("Failed to update topic %s on completion: %v", topicID, err)
	} else {
		logrus.Infof("Topic %s marked as synced after receiving completion log from automation backend", topicID)
	}
}

// handleTopicLog xử lý log liên quan đến topic (thành công hoặc thất bại)
func (s *ProcessLogService) handleTopicLog(entityType, topicID, stage, status string, metadata map[string]interface{}) {
	if entityType != "topic" {
		return
	}

	// Thành công → update sync status
	if status == "success" && (stage == "completed" || stage == "create_gem_completed") {
		s.updateTopicOnCompletion(topicID, metadata)
		return
	}

	// Thất bại → xóa topic
	if status == "failed" || status == "error" {
		if deleteErr := s.topicRepo.Delete(topicID); deleteErr != nil {
			logrus.Errorf("Failed to delete topic %s after automation %s (%s): %v", topicID, stage, status, deleteErr)
		} else {
			logrus.Warnf("Deleted topic %s due to automation %s (%s)", topicID, stage, status)
		}
	}
}

// handleScriptExecutionLog xử lý log liên quan đến script execution
// entityID ở đây là topic.ID (vì automation backend nhận X-Entity-ID = topic.ID)
// Nếu metadata chứa execution_id → dùng trực tiếp để match đúng execution
func (s *ProcessLogService) handleScriptExecutionLog(entityType, entityID, stage, status string, metadata map[string]interface{}) {
	if entityType != "script_execution" {
		return
	}

	// Handle project_completed log
	if stage == "project_completed" && (status == "success" || status == "info") {
		if s.scriptExecutionService == nil {
			return
		}

		var projectID, executionID string
		if metadata != nil {
			if pid, ok := metadata["project_id"].(string); ok {
				projectID = pid
			} else if pid, ok := metadata["project"].(string); ok {
				projectID = pid
			}
			if eid, ok := metadata["execution_id"].(string); ok {
				executionID = eid
			}
		}

		if projectID == "" {
			return
		}

		if executionID != "" {
			if err := s.scriptExecutionService.MarkProjectCompleted(executionID, projectID); err != nil {
				logrus.Errorf("[Log] Failed to mark project completed: %v", err)
			}
		} else {
			if err := s.scriptExecutionService.MarkProjectCompletedByTopicID(entityID, projectID); err != nil {
				logrus.Errorf("[Log] Failed to mark project completed: %v", err)
			}
		}
		return
	}

	// Handle project_failed log
	if stage == "project_failed" && (status == "failed" || status == "error") {
		if s.scriptExecutionService == nil {
			return
		}

		var projectID, executionID string
		if metadata != nil {
			if pid, ok := metadata["project_id"].(string); ok {
				projectID = pid
			} else if pid, ok := metadata["project"].(string); ok {
				projectID = pid
			}
			if eid, ok := metadata["execution_id"].(string); ok {
				executionID = eid
			}
		}

		if projectID == "" {
			return
		}

		logrus.Warnf("[Log] Project %s failed (execution=%s)", projectID, executionID)
		// TODO: Implement mark project as failed
	}
}

// GetLogsByEntity retrieves logs for a specific entity
func (s *ProcessLogService) GetLogsByEntity(entityType, entityID string, limit, offset int) ([]*models.ProcessLog, error) {
	return s.logRepo.GetByEntity(entityType, entityID, limit, offset)
}

// GetLogsByUserID retrieves logs for a specific user
func (s *ProcessLogService) GetLogsByUserID(userID string, limit, offset int) ([]*models.ProcessLog, error) {
	return s.logRepo.GetByUserID(userID, limit, offset)
}

// GetLatestLog retrieves the latest log for an entity
func (s *ProcessLogService) GetLatestLog(entityType, entityID string) (*models.ProcessLog, error) {
	return s.logRepo.GetLatestByEntity(entityType, entityID)
}

// CountLogs counts logs for an entity
func (s *ProcessLogService) CountLogs(entityType, entityID string) (int64, error) {
	return s.logRepo.CountByEntity(entityType, entityID)
}

// Log is a convenience method to create a log entry
func (s *ProcessLogService) Log(entityType, entityID, userID, machineID, stage, status, message string, metadata map[string]interface{}) error {
	req := &models.ProcessLogRequest{
		EntityType: entityType,
		EntityID:   entityID,
		UserID:     userID,
		MachineID:  machineID,
		Stage:      stage,
		Status:     status,
		Message:    message,
		Metadata:   metadata,
	}

	_, err := s.CreateLog(req)
	return err
}

// StartLogCleanup starts a background goroutine to periodically clean up old logs
func (s *ProcessLogService) StartLogCleanup(interval time.Duration, retentionDays int) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Run initial cleanup
		s.cleanupOldLogs(retentionDays)

		for {
			select {
			case <-ticker.C:
				s.cleanupOldLogs(retentionDays)
			case <-s.cleanupStopChan:
				return
			}
		}
	}()
	logrus.Infof("Log cleanup service started (interval: %v, retention: %d days)", interval, retentionDays)
}

// StopLogCleanup stops the log cleanup service
func (s *ProcessLogService) StopLogCleanup() {
	select {
	case s.cleanupStopChan <- true:
	default:
	}
}

// cleanupOldLogs deletes logs older than the specified number of days
// Chỉ xóa log cũ, không ảnh hưởng đến log mới đang được tạo liên tục
func (s *ProcessLogService) cleanupOldLogs(retentionDays int) {
	deletedCount, err := s.logRepo.DeleteOldLogs(retentionDays)
	if err != nil {
		logrus.Errorf("Failed to cleanup old logs: %v", err)
		return
	}

	if deletedCount > 0 {
		logrus.Infof("Log cleanup completed: deleted %d log entries older than %d day(s)", deletedCount, retentionDays)
	} else {
		logrus.Debugf("Log cleanup completed: no logs to delete (all logs are within %d day(s))", retentionDays)
	}
}

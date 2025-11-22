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
	logRepo  *repository.ProcessLogRepository
	sseHub   *SSEHub
	rabbitMQ *RabbitMQService
	db       *gorm.DB
	stopChan chan bool
}

func NewProcessLogService(logRepo *repository.ProcessLogRepository, sseHub *SSEHub, rabbitMQ *RabbitMQService, db *gorm.DB) *ProcessLogService {
	return &ProcessLogService{
		logRepo:  logRepo,
		sseHub:   sseHub,
		rabbitMQ: rabbitMQ,
		db:       db,
		stopChan: make(chan bool),
	}
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

	logrus.Debugf("Processed log: %s/%s - %s", req.EntityType, req.EntityID, req.Stage)
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

	return log, nil
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

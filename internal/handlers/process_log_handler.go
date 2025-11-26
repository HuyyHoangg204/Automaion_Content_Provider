package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/onegreenvn/green-provider-services-backend/internal/services"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type ProcessLogHandler struct {
	processLogService *services.ProcessLogService
	sseHub            *services.SSEHub
}

func NewProcessLogHandler(db *gorm.DB, sseHub *services.SSEHub, rabbitMQ *services.RabbitMQService) *ProcessLogHandler {
	logRepo := repository.NewProcessLogRepository(db)
	processLogService := services.NewProcessLogService(logRepo, sseHub, rabbitMQ, db)

	return &ProcessLogHandler{
		processLogService: processLogService,
		sseHub:            sseHub,
	}
}

// CreateLog godoc
// @Summary Create a process log (for automation backend)
// @Description Create a process log entry. Can be called by automation backend via HTTP or RabbitMQ
// @Tags process-logs
// @Accept json
// @Produce json
// @Param request body models.ProcessLogRequest true "Process log request"
// @Success 201 {object} models.ProcessLogResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/process-logs [post]
func (h *ProcessLogHandler) CreateLog(c *gin.Context) {
	var req models.ProcessLogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	log, err := h.processLogService.CreateLog(&req)
	if err != nil {
		logrus.Errorf("Failed to create log: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create log", "details": err.Error()})
		return
	}

	response := h.logToResponse(log)
	c.JSON(http.StatusCreated, response)
}

// GetLogsByEntity godoc
// @Summary Get logs for a specific entity
// @Description Get paginated logs for a specific entity (e.g., topic)
// @Tags process-logs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param entity_type path string true "Entity type" example:"topic"
// @Param entity_id path string true "Entity ID" example:"550e8400-e29b-41d4-a716-446655440000"
// @Param limit query int false "Limit" default(100)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} models.ProcessLogResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/process-logs/{entity_type}/{entity_id} [get]
func (h *ProcessLogHandler) GetLogsByEntity(c *gin.Context) {
	entityType := c.Param("entity_type")
	entityID := c.Param("entity_id")

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 1000 {
		limit = 1000
	}

	logs, err := h.processLogService.GetLogsByEntity(entityType, entityID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get logs", "details": err.Error()})
		return
	}

	responses := make([]models.ProcessLogResponse, len(logs))
	for i, log := range logs {
		responses[i] = h.logToResponse(log)
	}

	c.JSON(http.StatusOK, responses)
}

// StreamLogsSSE godoc
// @Summary Stream logs via Server-Sent Events (SSE)
// @Description Stream real-time logs for a specific entity via SSE
// @Tags process-logs
// @Accept json
// @Produce text/event-stream
// @Security BearerAuth
// @Param entity_type path string true "Entity type" example:"topic"
// @Param entity_id path string true "Entity ID" example:"550e8400-e29b-41d4-a716-446655440000"
// @Success 200 "SSE stream"
// @Router /api/v1/process-logs/{entity_type}/{entity_id}/stream [get]
func (h *ProcessLogHandler) StreamLogsSSE(c *gin.Context) {
	entityType := c.Param("entity_type")
	entityID := c.Param("entity_id")

	// Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // Disable buffering for nginx

	// Register client
	clientChan := h.sseHub.RegisterClient(entityType, entityID)
	defer h.sseHub.UnregisterClient(entityType, entityID, clientChan)

	// Send initial connection message
	c.SSEvent("connected", gin.H{
		"entity_type": entityType,
		"entity_id":   entityID,
		"message":     "Connected to log stream",
	})
	c.Writer.Flush()

	// Send existing logs from database (so client can see logs that were created before connection)
	existingLogs, err := h.processLogService.GetLogsByEntity(entityType, entityID, 100, 0)
	if err == nil {
		for _, log := range existingLogs {
			// Format log as SSE message with event type
			logJSON, err := json.Marshal(log)
			if err != nil {
				continue
			}
			message := fmt.Sprintf("event: log\ndata: %s\n\n", string(logJSON))
			if _, err := c.Writer.Write([]byte(message)); err != nil {
				return
			}
			c.Writer.Flush()
		}
	}

	// Send logs as they arrive (new logs)
	for {
		select {
		case <-c.Request.Context().Done():
			logrus.Infof("SSE client disconnected: %s/%s", entityType, entityID)
			return
		case message, ok := <-clientChan:
			if !ok {
				return
			}
			if _, err := c.Writer.Write(message); err != nil {
				logrus.Errorf("Failed to write SSE message: %v", err)
				return
			}
			c.Writer.Flush()
		}
	}
}

// GetLogsByUser godoc
// @Summary Get logs for the current user
// @Description Get paginated logs for all entities belonging to the current user
// @Tags process-logs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Limit" default(100)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} models.ProcessLogResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/process-logs [get]
func (h *ProcessLogHandler) GetLogsByUser(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 1000 {
		limit = 1000
	}

	logs, err := h.processLogService.GetLogsByUserID(userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get logs", "details": err.Error()})
		return
	}

	responses := make([]models.ProcessLogResponse, len(logs))
	for i, log := range logs {
		responses[i] = h.logToResponse(log)
	}

	c.JSON(http.StatusOK, responses)
}

// logToResponse converts a ProcessLog model to ProcessLogResponse
func (h *ProcessLogHandler) logToResponse(log *models.ProcessLog) models.ProcessLogResponse {
	return models.ProcessLogResponse{
		ID:         log.ID,
		EntityType: log.EntityType,
		EntityID:   log.EntityID,
		UserID:     log.UserID,
		MachineID:  log.MachineID,
		Stage:      log.Stage,
		Status:     log.Status,
		Message:    log.Message,
		Metadata:   log.Metadata,
		CreatedAt:  log.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

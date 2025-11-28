package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/onegreenvn/green-provider-services-backend/internal/services"
	"github.com/sirupsen/logrus"
)

type TopicHandler struct {
	topicService *services.TopicService
}

func NewTopicHandler(topicService *services.TopicService) *TopicHandler {
	return &TopicHandler{
		topicService: topicService,
	}
}

// CreateTopic godoc
// @Summary Create a new topic
// @Description Create a new topic and automatically create a Gem on Gemini
// @Tags topics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.CreateTopicRequest true "Topic creation request"
// @Success 201 {object} models.TopicResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/topics [post]
func (h *TopicHandler) CreateTopic(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	var req models.CreateTopicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Log để debug - xem request có knowledge_files không
	logrus.Infof("CreateTopic request - KnowledgeFiles received: %v (length: %d)", req.KnowledgeFiles, len(req.KnowledgeFiles))

	topic, err := h.topicService.CreateTopic(userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create topic", "details": err.Error()})
		return
	}

	// Convert to response
	response := h.topicToResponse(topic)
	c.JSON(http.StatusCreated, response)
}

// GetAllTopics godoc
// @Summary Get all topics for the current user
// @Description Get all topics belonging to the authenticated user
// @Tags topics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.TopicResponse
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/topics [get]
func (h *TopicHandler) GetAllTopics(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	topics, err := h.topicService.GetAllTopicsByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get topics", "details": err.Error()})
		return
	}

	// Convert to responses
	responses := make([]models.TopicResponse, len(topics))
	for i, topic := range topics {
		responses[i] = h.topicToResponse(topic)
	}

	c.JSON(http.StatusOK, responses)
}

// GetTopicByID godoc
// @Summary Get a topic by ID
// @Description Get a specific topic by its ID
// @Tags topics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Topic ID"
// @Success 200 {object} models.TopicResponse
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/topics/{id} [get]
func (h *TopicHandler) GetTopicByID(c *gin.Context) {
	id := c.Param("id")

	topic, err := h.topicService.GetTopicByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Topic not found"})
		return
	}

	response := h.topicToResponse(topic)
	c.JSON(http.StatusOK, response)
}

// UpdateTopic godoc
// @Summary Update a topic
// @Description Update a topic's information
// @Tags topics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Topic ID"
// @Param request body models.UpdateTopicRequest true "Topic update request"
// @Success 200 {object} models.TopicResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/topics/{id} [put]
func (h *TopicHandler) UpdateTopic(c *gin.Context) {
	id := c.Param("id")

	var req models.UpdateTopicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	topic, err := h.topicService.UpdateTopic(id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update topic", "details": err.Error()})
		return
	}

	response := h.topicToResponse(topic)
	c.JSON(http.StatusOK, response)
}

// GetTopicPrompts godoc
// @Summary Get topic prompts
// @Description Get notebooklm_prompt and send_prompt_text for a topic
// @Tags topics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Topic ID"
// @Success 200 {object} models.TopicPromptsResponse
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/topics/{id}/prompts [get]
func (h *TopicHandler) GetTopicPrompts(c *gin.Context) {
	id := c.Param("id")

	topic, err := h.topicService.GetTopicByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Topic not found"})
		return
	}

	response := models.TopicPromptsResponse{
		ID:               topic.ID,
		NotebooklmPrompt: topic.NotebooklmPrompt,
		SendPromptText:   topic.SendPromptText,
	}

	c.JSON(http.StatusOK, response)
}

// UpdateTopicPrompts godoc
// @Summary Update topic prompts
// @Description Update notebooklm_prompt and send_prompt_text for a topic
// @Tags topics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Topic ID"
// @Param request body models.UpdateTopicPromptsRequest true "Topic prompts update request"
// @Success 200 {object} models.TopicResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/topics/{id}/prompts [put]
func (h *TopicHandler) UpdateTopicPrompts(c *gin.Context) {
	id := c.Param("id")

	var req models.UpdateTopicPromptsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	topic, err := h.topicService.UpdateTopicPrompts(id, &req)
	if err != nil {
		if err.Error() == "topic not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Topic not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update topic prompts", "details": err.Error()})
		return
	}

	response := h.topicToResponse(topic)
	c.JSON(http.StatusOK, response)
}

// DeleteTopic godoc
// @Summary Delete a topic
// @Description Delete a topic by its ID
// @Tags topics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Topic ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/topics/{id} [delete]
func (h *TopicHandler) DeleteTopic(c *gin.Context) {
	id := c.Param("id")

	if err := h.topicService.DeleteTopic(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete topic", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Topic deleted successfully"})
}

// SyncTopicWithGemini - TODO: Implement later
// func (h *TopicHandler) SyncTopicWithGemini(c *gin.Context) {
// 	// Implementation will be added later
// }

// topicToResponse converts a Topic model to TopicResponse
func (h *TopicHandler) topicToResponse(topic *models.Topic) models.TopicResponse {
	response := models.TopicResponse{
		ID:               topic.ID,
		UserProfileID:    topic.UserProfileID,
		Name:             topic.Name,
		GeminiGemID:      topic.GeminiGemID,
		GeminiGemName:    topic.GeminiGemName,
		Description:      topic.Description,
		Instructions:     topic.Instructions,
		KnowledgeFiles:   topic.KnowledgeFiles,
		NotebooklmPrompt: topic.NotebooklmPrompt,
		SendPromptText:   topic.SendPromptText,
		IsActive:         topic.IsActive,
		SyncStatus:       topic.SyncStatus,
		SyncError:        topic.SyncError,
		CreatedAt:        topic.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:        topic.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if topic.LastSyncedAt != nil {
		lastSyncedAt := topic.LastSyncedAt.Format("2006-01-02T15:04:05Z07:00")
		response.LastSyncedAt = &lastSyncedAt
	}

	return response
}

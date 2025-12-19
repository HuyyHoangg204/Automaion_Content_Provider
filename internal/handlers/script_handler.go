package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/onegreenvn/green-provider-services-backend/internal/services"
	"github.com/sirupsen/logrus"
)

type ScriptHandler struct {
	scriptService         *services.ScriptService
	scriptExecutionService *services.ScriptExecutionService
	topicService          *services.TopicService
}

func NewScriptHandler(scriptService *services.ScriptService, scriptExecutionService *services.ScriptExecutionService, topicService *services.TopicService) *ScriptHandler {
	return &ScriptHandler{
		scriptService:         scriptService,
		scriptExecutionService: scriptExecutionService,
		topicService:          topicService,
	}
}

// SaveScript godoc
// @Summary Save or update a script for a topic
// @Description Save or update a script (projects + edges) for a topic. 1 script = 1 user + 1 topic (1-1 relationship)
// @Tags scripts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Topic ID"
// @Param request body models.SaveScriptRequest true "Script data"
// @Success 200 {object} models.ScriptResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/topics/{id}/scripts [post]
func (h *ScriptHandler) SaveScript(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	topicID := c.Param("id")

	// Check if user has permission to access this topic
	canAccess, _, err := h.topicService.CanUserAccessTopic(userID, topicID, false)
	if err != nil {
		logrus.Errorf("Failed to check topic access for user %s, topic %s: %v", userID, topicID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check topic access", "details": err.Error()})
		return
	}
	if !canAccess {
		logrus.Errorf("User %s does not have permission to access topic %s", userID, topicID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Topic not found"})
		return
	}

	var req models.SaveScriptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	response, err := h.scriptService.SaveScript(topicID, userID, &req)
	if err != nil {
		logrus.Errorf("Failed to save script for user %s, topic %s: %v", userID, topicID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save script", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetScript godoc
// @Summary Get a script for a topic
// @Description Get a script (projects + edges) for a topic and current user
// @Tags scripts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Topic ID"
// @Success 200 {object} models.ScriptResponse
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/topics/{id}/scripts [get]
func (h *ScriptHandler) GetScript(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	topicID := c.Param("id")

	// Check if user has permission to access this topic
	canAccess, _, err := h.topicService.CanUserAccessTopic(userID, topicID, false)
	if err != nil {
		logrus.Errorf("Failed to check topic access for user %s, topic %s: %v", userID, topicID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check topic access", "details": err.Error()})
		return
	}
	if !canAccess {
		logrus.Errorf("User %s does not have permission to access topic %s", userID, topicID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Topic not found"})
		return
	}

	response, err := h.scriptService.GetScript(topicID, userID)
	if err != nil {
		logrus.Errorf("Failed to get script for user %s, topic %s: %v", userID, topicID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Script not found", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// DeleteScript godoc
// @Summary Delete a script for a topic
// @Description Delete a script (projects + edges) for a topic and current user
// @Tags scripts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Topic ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/topics/{id}/scripts [delete]
func (h *ScriptHandler) DeleteScript(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	topicID := c.Param("id")

	// Check if user has permission to access this topic
	canAccess, _, err := h.topicService.CanUserAccessTopic(userID, topicID, false)
	if err != nil {
		logrus.Errorf("Failed to check topic access for user %s, topic %s: %v", userID, topicID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check topic access", "details": err.Error()})
		return
	}
	if !canAccess {
		logrus.Errorf("User %s does not have permission to access topic %s", userID, topicID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Topic not found"})
		return
	}

	if err := h.scriptService.DeleteScript(topicID, userID); err != nil {
		logrus.Errorf("Failed to delete script for user %s, topic %s: %v", userID, topicID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete script", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Script deleted successfully"})
}

// ExecuteScript godoc
// @Summary Execute a script
// @Description Execute a script by running projects in topological order. Execution is queued and processed asynchronously.
// @Tags scripts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Topic ID"
// @Success 202 {object} models.ExecuteScriptResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/topics/{id}/scripts/execute [post]
func (h *ScriptHandler) ExecuteScript(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	topicID := c.Param("id")

	// Check if user has permission to access this topic
	canAccess, _, err := h.topicService.CanUserAccessTopic(userID, topicID, false)
	if err != nil {
		logrus.Errorf("Failed to check topic access for user %s, topic %s: %v", userID, topicID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check topic access", "details": err.Error()})
		return
	}
	if !canAccess {
		logrus.Errorf("User %s does not have permission to access topic %s", userID, topicID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Topic not found"})
		return
	}

	response, err := h.scriptExecutionService.ExecuteScript(topicID, userID)
	if err != nil {
		logrus.Errorf("Failed to execute script for user %s, topic %s: %v", userID, topicID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to execute script", "details": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, response)
}


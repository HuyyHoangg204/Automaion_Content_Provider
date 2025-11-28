package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/onegreenvn/green-provider-services-backend/internal/services"
	"github.com/sirupsen/logrus"
)

type GeminiHandler struct {
	geminiService *services.GeminiService
}

func NewGeminiHandler(geminiService *services.GeminiService) *GeminiHandler {
	return &GeminiHandler{
		geminiService: geminiService,
	}
}

// GenerateOutlineAndUpload godoc
// @Summary Generate outline using NotebookLM and upload to Gemini
// @Description Kết hợp NotebookLM và Gemini: tạo dàn ý bằng NotebookLM, lưu vào folder outlines trong profile, sau đó upload file dàn ý lên Gemini Gem và gửi prompt.
// @Tags gemini
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param topic_id path string true "Topic ID" example:"550e8400-e29b-41d4-a716-446655440000"
// @Param request body models.GenerateOutlineRequest true "Generate outline request"
// @Success 200 {object} models.GenerateOutlineResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/gemini/topics/{topic_id}/generate-outline-and-upload [post]
func (h *GeminiHandler) GenerateOutlineAndUpload(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	topicID := c.Param("topic_id")

	var req models.GenerateOutlineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	response, err := h.geminiService.GenerateOutlineAndUpload(userID, topicID, &req)
	if err != nil {
		logrus.Errorf("Failed to generate outline for user %s, topic %s: %v", userID, topicID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate outline", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

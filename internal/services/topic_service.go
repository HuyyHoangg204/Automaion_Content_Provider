package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/sirupsen/logrus"
)

type TopicService struct {
	topicRepo            *repository.TopicRepository
	userProfileRepo      *repository.UserProfileRepository
	appRepo              *repository.AppRepository
	chromeProfileService *ChromeProfileService
	processLogService    *ProcessLogService
	baseURL              string // Base URL để generate download URLs cho files
}

func NewTopicService(topicRepo *repository.TopicRepository, userProfileRepo *repository.UserProfileRepository, appRepo *repository.AppRepository, chromeProfileService *ChromeProfileService, processLogService *ProcessLogService) *TopicService {
	// Get base URL from environment
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		baseURL = fmt.Sprintf("http://localhost:%s", port)
		logrus.Warnf("BASE_URL not set, using default: %s", baseURL)
	}

	return &TopicService{
		topicRepo:            topicRepo,
		userProfileRepo:      userProfileRepo,
		appRepo:              appRepo,
		chromeProfileService: chromeProfileService,
		processLogService:    processLogService,
		baseURL:              baseURL,
	}
}

// CreateTopic creates a new topic and calls Gemini API to create a Gem
func (s *TopicService) CreateTopic(userID string, req *models.CreateTopicRequest) (*models.Topic, error) {
	// Get user profile
	userProfile, err := s.userProfileRepo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("user profile not found: %w", err)
	}

	// Convert knowledge files to JSON
	var knowledgeFilesJSON models.JSON
	if len(req.KnowledgeFiles) > 0 {
		knowledgeFilesBytes, _ := json.Marshal(req.KnowledgeFiles)
		json.Unmarshal(knowledgeFilesBytes, &knowledgeFilesJSON)
	} else {
		emptyArray := []interface{}{}
		knowledgeFilesBytes, _ := json.Marshal(emptyArray)
		json.Unmarshal(knowledgeFilesBytes, &knowledgeFilesJSON)
	}

	// Create topic in database (initially with sync_status = "pending")
	topic := &models.Topic{
		UserProfileID:  userProfile.ID,
		Name:           req.Name,
		Description:    req.Description,
		Instructions:   req.Instructions,
		KnowledgeFiles: knowledgeFilesJSON,
		IsActive:       true,
		SyncStatus:     "pending",
	}

	if err := s.topicRepo.Create(topic); err != nil {
		return nil, fmt.Errorf("failed to create topic: %w", err)
	}

	// Call Gemini API to create Gem (in background)
	// First launch Chrome, then create Gem
	go func() {
		// Log: Topic creation started
		s.processLogService.Log("topic", topic.ID, userID, "", "chrome_launching", "info", "Starting Chrome launch process...", map[string]interface{}{
			"topic_name": topic.Name,
		})

		// Step 1: Launch Chrome with lock
		launchReq := &LaunchChromeProfileRequest{
			UserProfileID: userProfile.ID,
			EnsureGmail:   true, // Mặc định true để đảm bảo Gmail đã login
			EntityType:    "topic",
			EntityID:      topic.ID,
		}

		launchResp, err := s.chromeProfileService.LaunchChromeProfile(userID, launchReq)
		if err != nil {
			logrus.Errorf("Failed to launch Chrome for topic %s: %v", topic.ID, err)
			s.processLogService.Log("topic", topic.ID, userID, "", "chrome_launching", "error", fmt.Sprintf("Failed to launch Chrome: %v", err), map[string]interface{}{
				"error": err.Error(),
			})
			topic.SyncStatus = "failed"
			topic.SyncError = fmt.Sprintf("Failed to launch Chrome: %v", err)
			s.topicRepo.Update(topic)
			return
		}

		// Log: Chrome launched successfully (Gmail login đã được đảm bảo trong launch với ensureGmail: true)
		s.processLogService.Log("topic", topic.ID, userID, launchResp.MachineID, "chrome_launched", "success", "Chrome launched successfully with Gmail login ensured", map[string]interface{}{
			"machine_id": launchResp.MachineID,
			"app_id":     launchResp.AppID,
			"tunnel_url": launchResp.TunnelURL,
		})

		// Step 2: Create Gem on Gemini
		s.processLogService.Log("topic", topic.ID, userID, launchResp.MachineID, "gem_creating", "info", "Creating Gem on Gemini...", map[string]interface{}{
			"gem_name": req.Name,
		})

		geminiGemID, geminiGemName, err := s.createGemOnGemini(userProfile, req, launchResp.TunnelURL)
		if err != nil {
			logrus.Errorf("Failed to create Gem on Gemini for topic %s: %v", topic.ID, err)
			s.processLogService.Log("topic", topic.ID, userID, launchResp.MachineID, "gem_creating", "error", fmt.Sprintf("Failed to create Gem: %v", err), map[string]interface{}{
				"error": err.Error(),
			})
			// Release lock on error
			s.chromeProfileService.ReleaseChromeProfile(userID, &ReleaseChromeProfileRequest{
				UserProfileID: userProfile.ID,
			})
			// Update topic with error
			topic.SyncStatus = "failed"
			topic.SyncError = err.Error()
			s.topicRepo.Update(topic)
			return
		}

		// Log: Gem created successfully
		logData := map[string]interface{}{
			"gemini_gem_name": geminiGemName,
		}
		if geminiGemID != "" {
			logData["gemini_gem_id"] = geminiGemID
		}
		s.processLogService.Log("topic", topic.ID, userID, launchResp.MachineID, "gem_created", "success", "Gem created successfully on Gemini", logData)

		// Step 3: Release lock after successful Gem creation
		if err := s.chromeProfileService.ReleaseChromeProfile(userID, &ReleaseChromeProfileRequest{
			UserProfileID: userProfile.ID,
		}); err != nil {
			logrus.Warnf("Failed to release lock for topic %s: %v", topic.ID, err)
		} else {
			s.processLogService.Log("topic", topic.ID, userID, launchResp.MachineID, "lock_releasing", "success", "Chrome lock released", nil)
		}

		// Update topic with Gemini info
		// Note: gemini_gem_id là optional - chỉ lưu khi có giá trị
		// Logic hiện tại xác định gem bằng name, không cần ID
		if geminiGemID != "" {
			topic.GeminiGemID = &geminiGemID
		} else {
			topic.GeminiGemID = nil
		}
		topic.GeminiGemName = geminiGemName
		topic.SyncStatus = "synced"
		now := time.Now()
		topic.LastSyncedAt = &now
		topic.SyncError = ""

		if err := s.topicRepo.Update(topic); err != nil {
			logrus.Errorf("Failed to update topic with Gemini info: %v", err)
		} else {
			// Log: Topic creation completed
			completeLogData := map[string]interface{}{
				"gemini_gem_name": geminiGemName,
			}
			if geminiGemID != "" {
				completeLogData["gemini_gem_id"] = geminiGemID
				logrus.Infof("Successfully created Gem on Gemini for topic %s, gem_id: %s, gem_name: %s", topic.ID, geminiGemID, geminiGemName)
			} else {
				logrus.Infof("Successfully created Gem on Gemini for topic %s, gem_name: %s (no ID returned, using name for identification)", topic.ID, geminiGemName)
			}
			s.processLogService.Log("topic", topic.ID, userID, launchResp.MachineID, "completed", "success", "Topic created successfully", completeLogData)
		}
	}()

	return topic, nil
}

// convertFileIDsToURLs converts file IDs (UUIDs) to download URLs
// Logic đơn giản: Nếu là file ID (UUID format) → convert thành URL
// Automation backend sẽ tự download files từ URLs này
func (s *TopicService) convertFileIDsToURLs(fileInputs []string) []string {
	if len(fileInputs) == 0 {
		return []string{}
	}

	urls := make([]string, 0, len(fileInputs))
	for _, fileInput := range fileInputs {
		// Check if it's a file ID (UUID format: 8-4-4-4-12 hex digits)
		if s.isFileID(fileInput) {
			// Convert file ID to download URL
			downloadURL := fmt.Sprintf("%s/api/v1/files/%s/download", strings.TrimSuffix(s.baseURL, "/"), fileInput)
			urls = append(urls, downloadURL)
		} else {
			// Already a URL or local path - keep as is
			urls = append(urls, fileInput)
		}
	}

	return urls
}

// isFileID checks if input is a file ID (UUID format)
func (s *TopicService) isFileID(input string) bool {
	// Basic UUID format check (8-4-4-4-12 hex digits)
	if len(input) == 36 {
		parts := strings.Split(input, "-")
		if len(parts) == 5 && len(parts[0]) == 8 && len(parts[1]) == 4 && len(parts[2]) == 4 && len(parts[3]) == 4 && len(parts[4]) == 12 {
			return true
		}
	}
	return false
}

// createGemOnGemini calls the Gemini API to create a Gem
// tunnelURL is passed from LaunchChromeProfileResponse to use the same machine
func (s *TopicService) createGemOnGemini(userProfile *models.UserProfile, req *models.CreateTopicRequest, tunnelURL string) (string, string, error) {
	// Build API URL: POST /gemini/gems
	apiURL := fmt.Sprintf("%s/gemini/gems", strings.TrimSuffix(tunnelURL, "/"))

	// Convert file IDs to download URLs nếu có
	// Automation backend sẽ tự download files từ URLs này
	knowledgeFiles := s.convertFileIDsToURLs(req.KnowledgeFiles)

	// Prepare request body (without debugPort)
	// Note: Không gửi userDataDir - automation backend tự resolve path
	requestBody := map[string]interface{}{
		"name":           req.Name,
		"profileDirName": userProfile.ProfileDirName, // Chỉ gửi profileDirName, backend tự resolve path
		"gemName":        req.Name,
		"description":    req.Description,
		"instructions":   req.Instructions,
		"knowledgeFiles": knowledgeFiles, // URLs để automation backend download
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "Green-Provider-Services/1.0")

	// Make request
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return "", "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Parse response to get Gem ID
	var responseData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		return "", "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract Gem ID and name from response
	gemID := ""
	gemName := req.Name

	if id, ok := responseData["id"].(string); ok {
		gemID = id
	} else if id, ok := responseData["gem_id"].(string); ok {
		gemID = id
	} else if id, ok := responseData["uuid"].(string); ok {
		gemID = id
	}

	if name, ok := responseData["name"].(string); ok {
		gemName = name
	}

	if gemID == "" {
		return "", "", errors.New("gem ID not found in response")
	}

	return gemID, gemName, nil
}

// GetAllTopicsByUserID retrieves all topics for a user
func (s *TopicService) GetAllTopicsByUserID(userID string) ([]*models.Topic, error) {
	return s.topicRepo.GetByUserID(userID)
}

// GetTopicByID retrieves a topic by ID
func (s *TopicService) GetTopicByID(id string) (*models.Topic, error) {
	return s.topicRepo.GetByID(id)
}

// UpdateTopic updates a topic and optionally syncs with Gemini
func (s *TopicService) UpdateTopic(id string, req *models.UpdateTopicRequest) (*models.Topic, error) {
	topic, err := s.topicRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("topic not found: %w", err)
	}

	// Update fields
	if req.Name != "" {
		topic.Name = req.Name
	}
	if req.Description != "" {
		topic.Description = req.Description
	}
	if req.Instructions != "" {
		topic.Instructions = req.Instructions
	}
	if req.KnowledgeFiles != nil {
		knowledgeFilesBytes, _ := json.Marshal(req.KnowledgeFiles)
		json.Unmarshal(knowledgeFilesBytes, &topic.KnowledgeFiles)
	}
	if req.IsActive != nil {
		topic.IsActive = *req.IsActive
	}

	if err := s.topicRepo.Update(topic); err != nil {
		return nil, fmt.Errorf("failed to update topic: %w", err)
	}

	return topic, nil
}

// DeleteTopic deletes a topic
func (s *TopicService) DeleteTopic(id string) error {
	return s.topicRepo.Delete(id)
}

// SyncTopicWithGemini syncs a topic with Gemini (calls /gemini/gems/sync)
// TODO: Implement sync logic later
// func (s *TopicService) SyncTopicWithGemini(topicID string) error {
// 	// Implementation will be added later
// 	return nil
// }

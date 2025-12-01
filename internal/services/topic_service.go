package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"

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
	fileService          *FileService // FileService để lấy files mới nhất của user
	baseURL              string       // Base URL để generate download URLs cho files
	// In-memory cache để lưu file IDs vừa upload theo userID
	// Key: userID, Value: []string (file IDs)
	recentUploadedFiles sync.Map // map[string][]string
}

func NewTopicService(topicRepo *repository.TopicRepository, userProfileRepo *repository.UserProfileRepository, appRepo *repository.AppRepository, chromeProfileService *ChromeProfileService, processLogService *ProcessLogService, fileService *FileService) *TopicService {
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
		fileService:          fileService,
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

	// Bỏ field KnowledgeFiles khỏi Topic model
	// Files sẽ được lấy tự động khi tạo Gem
	emptyArray := []interface{}{}
	knowledgeFilesBytes, _ := json.Marshal(emptyArray)
	var knowledgeFilesJSON models.JSON
	json.Unmarshal(knowledgeFilesBytes, &knowledgeFilesJSON)

	// Create topic in database (initially with sync_status = "pending")
	topic := &models.Topic{
		UserProfileID:    userProfile.ID,
		Name:             req.Name,
		Description:      req.Description,
		Instructions:     req.Instructions,
		KnowledgeFiles:   knowledgeFilesJSON, // Luôn là empty array, không lưu files
		NotebooklmPrompt: req.NotebooklmPrompt,
		SendPromptText:   req.SendPromptText,
		IsActive:         true,
		SyncStatus:       "pending",
	}

	if err := s.topicRepo.Create(topic); err != nil {
		return nil, fmt.Errorf("failed to create topic: %w", err)
	}

	// Call Gemini API to create Gem (in background)
	// First launch Chrome, then create Gem
	// NOTE: Tất cả log sẽ được gửi từ automation backend, không tạo log ở đây
	go func() {
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
			// Xóa topic khi automation fail
			if deleteErr := s.topicRepo.Delete(topic.ID); deleteErr != nil {
				logrus.Errorf("Failed to delete topic %s after Chrome launch failure: %v", topic.ID, deleteErr)
			} else {
				logrus.Infof("Deleted topic %s due to Chrome launch failure", topic.ID)
			}
			return
		}

		// Step 2: Create Gem on Gemini
		geminiGemName, err := s.createGemOnGemini(userProfile, req, launchResp.TunnelURL, userID)
		if err != nil {
			logrus.Errorf("Failed to create Gem on Gemini for topic %s: %v", topic.ID, err)
			// Release lock on error
			s.chromeProfileService.ReleaseChromeProfile(userID, &ReleaseChromeProfileRequest{
				UserProfileID: userProfile.ID,
			})
			// Xóa topic khi automation fail
			if deleteErr := s.topicRepo.Delete(topic.ID); deleteErr != nil {
				logrus.Errorf("Failed to delete topic %s after Gem creation failure: %v", topic.ID, deleteErr)
			} else {
				logrus.Infof("Deleted topic %s due to Gem creation failure", topic.ID)
			}
			return
		}

		// Step 3: Release lock (automation backend sẽ xử lý việc tạo Gem và gửi log)
		if err := s.chromeProfileService.ReleaseChromeProfile(userID, &ReleaseChromeProfileRequest{
			UserProfileID: userProfile.ID,
		}); err != nil {
			logrus.Warnf("Failed to release lock for topic %s: %v", topic.ID, err)
		}

		// Lưu geminiGemName tạm thời (từ response của automation backend)
		// Reload topic trước khi update để tránh overwrite sync_status đã được automation backend update
		// Automation backend có thể đã update sync_status = "synced" qua log "completed"
		currentTopic, err := s.topicRepo.GetByID(topic.ID)
		if err != nil {
			logrus.Errorf("Failed to reload topic %s before update: %v", topic.ID, err)
			return
		}

		// Chỉ update geminiGemName và syncError, giữ nguyên SyncStatus (có thể đã được automation backend update)
		currentTopic.GeminiGemID = nil
		currentTopic.GeminiGemName = geminiGemName
		currentTopic.SyncError = ""
		// KHÔNG update SyncStatus - để automation backend tự update qua log

		s.topicRepo.Update(currentTopic)
	}()

	return topic, nil
}

// AddUploadedFiles lưu file IDs vừa upload vào cache
// Được gọi từ FileHandler sau khi upload thành công
func (s *TopicService) AddUploadedFiles(userID string, fileIDs []string) {
	if len(fileIDs) == 0 {
		return
	}

	// Lấy file IDs hiện có (nếu có)
	existing, _ := s.recentUploadedFiles.LoadOrStore(userID, []string{})
	existingIDs := existing.([]string)

	// Thêm file IDs mới vào (append, không duplicate)
	existingMap := make(map[string]bool)
	for _, id := range existingIDs {
		existingMap[id] = true
	}

	newIDs := make([]string, 0, len(existingIDs)+len(fileIDs))
	newIDs = append(newIDs, existingIDs...)

	for _, id := range fileIDs {
		if !existingMap[id] {
			newIDs = append(newIDs, id)
		}
	}

	s.recentUploadedFiles.Store(userID, newIDs)
}

// GetAndClearUploadedFiles lấy file IDs từ cache và xóa sau khi lấy
// Được gọi khi tạo Gem để lấy chính xác files vừa upload
func (s *TopicService) GetAndClearUploadedFiles(userID string) []string {
	// Lấy file IDs từ cache
	value, ok := s.recentUploadedFiles.LoadAndDelete(userID)
	if !ok {
		return []string{}
	}

	fileIDs := value.([]string)
	return fileIDs
}

// getRecentUserFilesAsURLs lấy files vừa upload từ cache và convert thành download URLs
func (s *TopicService) getRecentUserFilesAsURLs(userID string) []string {
	// Lấy file IDs từ cache (chính xác files vừa upload)
	fileIDs := s.GetAndClearUploadedFiles(userID)

	if len(fileIDs) == 0 {
		return []string{}
	}

	// Convert file IDs thành download URLs
	urls := make([]string, 0, len(fileIDs))
	for _, fileID := range fileIDs {
		downloadURL := fmt.Sprintf("%s/api/v1/files/%s/download", strings.TrimSuffix(s.baseURL, "/"), fileID)
		urls = append(urls, downloadURL)
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
// Returns gemName only (gemID is not needed - gem is identified by name with username prefix)
func (s *TopicService) createGemOnGemini(userProfile *models.UserProfile, req *models.CreateTopicRequest, tunnelURL string, userID string) (string, error) {
	// Build API URL: POST /gemini/gems
	apiURL := fmt.Sprintf("%s/gemini/gems", strings.TrimSuffix(tunnelURL, "/"))

	// Tự động lấy files mới nhất của user (upload trong 10 phút gần nhất)
	// Convert file IDs thành download URLs
	knowledgeFiles := s.getRecentUserFilesAsURLs(userID)

	// Đảm bảo knowledgeFiles luôn là array (không phải null)
	if knowledgeFiles == nil {
		knowledgeFiles = []string{}
	}

	// Lấy username từ userProfile (đã được preload)
	username := "user" // Default fallback
	if userProfile.User.ID != "" {
		if userProfile.User.Username != "" {
			username = normalizeUsername(userProfile.User.Username)
		} else {
			logrus.Warnf("User.Username is empty for userID %s, using default 'user' prefix", userID)
		}
	} else {
		logrus.Warnf("User not preloaded for userID %s, using default 'user' prefix", userID)
	}

	// Thêm tiền tố username vào gemName
	prefixedGemName := fmt.Sprintf("%s_%s", username, req.Name)

	// Prepare request body (without debugPort)
	// Note: Không gửi userDataDir - automation backend tự resolve path
	requestBody := map[string]interface{}{
		"name":           req.Name,
		"profileDirName": userProfile.ProfileDirName, // Chỉ gửi profileDirName, backend tự resolve path
		"gemName":        prefixedGemName,            // Gem name với tiền tố username
		"description":    req.Description,
		"instructions":   req.Instructions,
		"knowledgeFiles": knowledgeFiles, // URLs để automation backend download (always array, never null)
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "Green-Provider-Services/1.0")

	// Make request
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Parse response to get Gem name
	var responseData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract Gem name from response (gemID không cần thiết - xác định gem bằng name với tiền tố username)
	gemName := prefixedGemName // Default to prefixed name we sent

	if name, ok := responseData["name"].(string); ok {
		gemName = name
	}

	return gemName, nil
}

// normalizeUsername normalizes username for use as prefix
// Converts to lowercase, replaces spaces and special chars with underscore
func normalizeUsername(username string) string {
	// Convert to lowercase
	normalized := strings.ToLower(username)

	// Replace spaces and special characters with underscore
	var result strings.Builder
	for _, r := range normalized {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			result.WriteRune(r)
		} else {
			result.WriteRune('_')
		}
	}

	// Remove consecutive underscores
	resultStr := result.String()
	for strings.Contains(resultStr, "__") {
		resultStr = strings.ReplaceAll(resultStr, "__", "_")
	}

	// Remove leading/trailing underscores
	resultStr = strings.Trim(resultStr, "_")

	// If empty after normalization, use "user" as default
	if resultStr == "" {
		resultStr = "user"
	}

	return resultStr
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
	if req.NotebooklmPrompt != "" {
		topic.NotebooklmPrompt = req.NotebooklmPrompt
	}
	if req.SendPromptText != "" {
		topic.SendPromptText = req.SendPromptText
	}
	if req.IsActive != nil {
		topic.IsActive = *req.IsActive
	}

	if err := s.topicRepo.Update(topic); err != nil {
		return nil, fmt.Errorf("failed to update topic: %w", err)
	}

	return topic, nil
}

// UpdateTopicPrompts updates only the prompt fields of a topic
func (s *TopicService) UpdateTopicPrompts(id string, req *models.UpdateTopicPromptsRequest) (*models.Topic, error) {
	topic, err := s.topicRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("topic not found: %w", err)
	}

	topic.NotebooklmPrompt = req.NotebooklmPrompt
	topic.SendPromptText = req.SendPromptText

	if err := s.topicRepo.Update(topic); err != nil {
		return nil, fmt.Errorf("failed to update topic prompts: %w", err)
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

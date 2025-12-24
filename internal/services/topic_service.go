package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	topicUserRepo        *repository.TopicUserRepository // New: For topic assignments
	userProfileRepo      *repository.UserProfileRepository
	appRepo              *repository.AppRepository
	boxRepo              *repository.BoxRepository // For getting machine_id from BoxID
	chromeProfileService *ChromeProfileService
	processLogService    *ProcessLogService
	fileService          *FileService          // FileService để lấy files mới nhất của user
	geminiAccountService *GeminiAccountService // For getting available Gemini account
	baseURL              string                // Base URL để generate download URLs cho files
	// In-memory cache để lưu file IDs vừa upload theo userID
	// Key: userID, Value: []string (file IDs)
	recentUploadedFiles sync.Map // map[string][]string
}

func NewTopicService(topicRepo *repository.TopicRepository, topicUserRepo *repository.TopicUserRepository, userProfileRepo *repository.UserProfileRepository, appRepo *repository.AppRepository, boxRepo *repository.BoxRepository, chromeProfileService *ChromeProfileService, processLogService *ProcessLogService, fileService *FileService, geminiAccountService *GeminiAccountService) *TopicService {
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
		topicUserRepo:        topicUserRepo,
		userProfileRepo:      userProfileRepo,
		appRepo:              appRepo,
		boxRepo:              boxRepo,
		chromeProfileService: chromeProfileService,
		processLogService:    processLogService,
		fileService:          fileService,
		geminiAccountService: geminiAccountService,
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

	// Create topic in database (initially with sync_status = "pending")
	topic := &models.Topic{
		UserProfileID: userProfile.ID,
		Name:          req.Name,
		Description:   req.Description,
		IsActive:      true,
		SyncStatus:    "pending",
	}

	if err := s.topicRepo.Create(topic); err != nil {
		return nil, fmt.Errorf("failed to create topic: %w", err)
	}

	// NOTE: Logic tạo gem đã được chuyển sang API tạo project (SaveScript)
	// Mỗi project trong script sẽ tương ứng với 1 gem
	// Comment lại logic cũ để không gọi automation backend khi tạo topic
	/*
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

			// Step 1.5: Get Gemini account for this machine (if available)
			var geminiAccount *models.GeminiAccount
			if s.geminiAccountService != nil && launchResp.MachineID != "" {
				// Get Box to get machine_id (string) from BoxID (UUID)
				box, err := s.boxRepo.GetByID(launchResp.MachineID)
				if err == nil && box != nil {
					// Get available Gemini account for this machine
					account, err := s.geminiAccountService.GetAvailableAccountForMachine(box.MachineID)
					if err == nil && account != nil {
						geminiAccount = account
						// Update account: increment topics_count and last_used_at (will be done in service method)
						logrus.Infof("Using Gemini account %s (email: %s) for topic %s on machine %s", account.ID, account.Email, topic.ID, box.MachineID)
					} else {
						logrus.Warnf("No available Gemini account found for machine %s, topic will be created without account association", box.MachineID)
					}
				}
			}

			// Step 2: Trigger Gem creation on Gemini (fire-and-forget)
			// Automation backend sẽ xử lý việc tạo Gem và gửi log qua RabbitMQ
			// Chỉ xóa topic khi nhận log "failed" từ automation backend, không xóa khi API timeout
			err = s.triggerGemCreation(userProfile, req, launchResp.TunnelURL, userID, geminiAccount)
			if err != nil {
				// Chỉ xóa topic nếu không gửi được request (network error, không phải timeout)
				// Nếu timeout → automation backend vẫn đang chạy, sẽ gửi log sau
				if isNetworkError(err) {
					logrus.Errorf("Failed to trigger Gem creation for topic %s (network error): %v", topic.ID, err)
					// Release lock on error
					s.chromeProfileService.ReleaseChromeProfile(userID, &ReleaseChromeProfileRequest{
						UserProfileID: userProfile.ID,
					})
					// Xóa topic khi không gửi được request
					if deleteErr := s.topicRepo.Delete(topic.ID); deleteErr != nil {
						logrus.Errorf("Failed to delete topic %s after trigger failure: %v", topic.ID, deleteErr)
					} else {
						logrus.Infof("Deleted topic %s due to trigger failure (network error)", topic.ID)
					}
					return
				}
				// Timeout hoặc lỗi khác → automation backend có thể vẫn đang chạy
				// Không xóa topic, đợi log từ automation backend
				logrus.Warnf("Gem creation trigger returned error for topic %s (may be timeout, automation backend still running): %v", topic.ID, err)
				logrus.Infof("Topic %s will be updated/deleted based on logs from automation backend", topic.ID)
			}

			// Step 3: Release lock (automation backend sẽ xử lý việc tạo Gem và gửi log)
			if err := s.chromeProfileService.ReleaseChromeProfile(userID, &ReleaseChromeProfileRequest{
				UserProfileID: userProfile.ID,
			}); err != nil {
				logrus.Warnf("Failed to release lock for topic %s: %v", topic.ID, err)
			}

		}()
	*/

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

// isNetworkError checks if error is a network error (not timeout)
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Network errors: connection refused, no such host, etc.
	// NOT timeout errors: context deadline exceeded, timeout
	return (strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "dial tcp") ||
		strings.Contains(errStr, "connectex")) &&
		!strings.Contains(errStr, "timeout") &&
		!strings.Contains(errStr, "deadline exceeded")
}

// triggerGemCreation triggers Gem creation on automation backend (fire-and-forget)
// Returns error only if cannot send request (network error)
// Timeout errors are ignored - automation backend will send logs via RabbitMQ
func (s *TopicService) triggerGemCreation(userProfile *models.UserProfile, req *models.CreateTopicRequest, tunnelURL string, userID string, geminiAccount *models.GeminiAccount) error {
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
	// Note: Không gửi account_index vì 1 machine = 1 account, automation backend tự biết account nào
	requestBody := map[string]interface{}{
		"name":           req.Name,
		"profileDirName": userProfile.ProfileDirName, // Chỉ gửi profileDirName, backend tự resolve path
		"gemName":        prefixedGemName,            // Gem name với tiền tố username
		"description":    req.Description,
		"knowledgeFiles": knowledgeFiles, // URLs để automation backend download (always array, never null)
	}

	// Note: Không cần gửi account_index vì 1 machine = 1 account
	// Automation backend sẽ tự biết account nào được setup trên machine đó
	if geminiAccount != nil {
		logrus.Infof("Using Gemini account %s (email: %s) for topic on machine", geminiAccount.ID, geminiAccount.Email)
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "Green-Provider-Services/1.0")

	// Make request with short timeout (chỉ để trigger, không đợi response)
	// Automation backend sẽ xử lý và gửi log qua RabbitMQ
	client := &http.Client{
		Timeout: 10 * time.Second, // Short timeout, chỉ để trigger
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		// Network error hoặc timeout đều return error
		// Caller sẽ phân biệt network error vs timeout
		return fmt.Errorf("failed to trigger Gem creation: %w", err)
	}
	defer resp.Body.Close()

	// Check response status (nếu có response)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// HTTP error status → automation backend đã nhận request nhưng trả về lỗi
		// Đọc response body để log
		bodyBytes, _ := io.ReadAll(resp.Body)
		logrus.Warnf("Automation backend returned status %d for Gem creation trigger: %s", resp.StatusCode, string(bodyBytes))
		// Không return error vì automation backend có thể vẫn xử lý được
		// Sẽ đợi log từ automation backend
	}

	// Không cần parse response, automation backend sẽ gửi log qua RabbitMQ
	logrus.Infof("Gem creation triggered for topic, waiting for logs from automation backend")
	return nil
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

// GetAllTopicsByUserID retrieves all topics for a user (owned + assigned) - DEPRECATED: Use GetAllTopicsByUserIDPaginated
func (s *TopicService) GetAllTopicsByUserID(userID string) ([]*models.Topic, error) {
	// Get assigned topic IDs
	assignedTopicIDs, err := s.topicUserRepo.GetTopicIDsAssignedToUser(userID)
	if err != nil {
		logrus.Warnf("Failed to get assigned topic IDs for user %s: %v", userID, err)
		assignedTopicIDs = []string{} // Continue with empty list
	}

	// Get topics (owned + assigned)
	return s.topicRepo.GetByUserIDIncludingAssigned(userID, assignedTopicIDs)
}

// GetAllTopicsByUserIDPaginated retrieves all topics for a user (owned + assigned) with pagination
func (s *TopicService) GetAllTopicsByUserIDPaginated(userID string, page, pageSize int) ([]*models.Topic, int64, error) {
	// Get assigned topic IDs
	assignedTopicIDs, err := s.topicUserRepo.GetTopicIDsAssignedToUser(userID)
	if err != nil {
		logrus.Warnf("Failed to get assigned topic IDs for user %s: %v", userID, err)
		assignedTopicIDs = []string{} // Continue with empty list
	}

	// Get topics (owned + assigned) with pagination
	return s.topicRepo.GetByUserIDIncludingAssignedPaginated(userID, assignedTopicIDs, page, pageSize)
}

// CanUserAccessTopic checks if a user can access a topic (creator, assigned, or admin)
func (s *TopicService) CanUserAccessTopic(userID string, topicID string, isAdmin bool) (bool, string, error) {
	// Admin can access all topics
	if isAdmin {
		return true, "admin", nil
	}

	// Get topic
	topic, err := s.topicRepo.GetByID(topicID)
	if err != nil {
		return false, "", fmt.Errorf("topic not found: %w", err)
	}

	// Check if user is creator
	if topic.UserProfile.UserID == userID {
		return true, "creator", nil
	}

	// Check if user is assigned
	isAssigned, err := s.topicUserRepo.IsUserAssigned(topicID, userID)
	if err != nil {
		return false, "", fmt.Errorf("failed to check assignment: %w", err)
	}
	if isAssigned {
		return true, "assigned", nil
	}

	return false, "", nil
}

// GetTopicByID retrieves a topic by ID
func (s *TopicService) GetTopicByID(id string) (*models.Topic, error) {
	return s.topicRepo.GetByID(id)
}

// GetAllTopicsPaginated retrieves all topics with pagination, search, and filters (admin only)
// search: searches in topic name, description, and creator username
// creatorID: filter by creator user ID
// syncStatus: filter by sync status
// isActive: filter by is_active status (nil = no filter)
func (s *TopicService) GetAllTopicsPaginated(page, pageSize int, search, creatorID, syncStatus string, isActive *bool) ([]*models.Topic, int64, error) {
	return s.topicRepo.GetAllPaginated(page, pageSize, search, creatorID, syncStatus, isActive)
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

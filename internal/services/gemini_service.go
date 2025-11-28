package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/sirupsen/logrus"
)

type GeminiService struct {
	userProfileRepo      *repository.UserProfileRepository
	appRepo              *repository.AppRepository
	topicRepo            *repository.TopicRepository
	chromeProfileService *ChromeProfileService
}

func NewGeminiService(userProfileRepo *repository.UserProfileRepository, appRepo *repository.AppRepository, topicRepo *repository.TopicRepository, chromeProfileService *ChromeProfileService) *GeminiService {
	return &GeminiService{
		userProfileRepo:      userProfileRepo,
		appRepo:              appRepo,
		topicRepo:            topicRepo,
		chromeProfileService: chromeProfileService,
	}
}

func (s *GeminiService) GenerateOutlineAndUpload(userID string, topicID string, req *models.GenerateOutlineRequest) (*models.GenerateOutlineResponse, error) {
	logrus.Infof("GenerateOutlineAndUpload called for user %s with topic_id: %s", userID, topicID)

	// Get topic from DB
	topic, err := s.topicRepo.GetByID(topicID)
	if err != nil {
		logrus.Errorf("Failed to get topic %s: %v", topicID, err)
		return nil, fmt.Errorf("topic not found: %w", err)
	}

	// Verify topic belongs to user
	userProfile, err := s.userProfileRepo.GetByUserID(userID)
	if err != nil {
		logrus.Errorf("Failed to get user profile for user %s: %v", userID, err)
		return nil, fmt.Errorf("user profile not found: %w", err)
	}

	if topic.UserProfileID != userProfile.ID {
		logrus.Errorf("Topic %s does not belong to user %s", topicID, userID)
		return nil, fmt.Errorf("topic not found")
	}

	profileDirName := userProfile.ProfileDirName
	logrus.Infof("Using profile: name=%s, dirName=%s", userProfile.Name, profileDirName)

	// Use gem name from topic (already stored in DB when topic was created)
	gemName := topic.GeminiGemName
	if gemName == "" {
		// Fallback: Create gem name if not stored yet (should not happen for synced topics)
		username := "user" // Default fallback
		if userProfile.User.ID != "" {
			if userProfile.User.Username != "" {
				username = s.normalizeUsername(userProfile.User.Username)
			} else {
				logrus.Warnf("User.Username is empty for userID %s, using default 'user' prefix", userID)
			}
		} else {
			logrus.Warnf("User not preloaded for userID %s, using default 'user' prefix", userID)
		}
		gemName = fmt.Sprintf("%s_%s", username, topic.Name)
		logrus.Warnf("Topic %s does not have gemini_gem_name, generated fallback: '%s'", topic.ID, gemName)
	} else {
		logrus.Infof("Using gem name from topic: '%s'", gemName)
	}

	// Get prompts from topic
	notebooklmPrompt := topic.NotebooklmPrompt
	sendPromptText := topic.SendPromptText
	logrus.Infof("Prompts from topic: notebooklmPrompt length=%d, sendPromptText length=%d", len(notebooklmPrompt), len(sendPromptText))

	launchReq := &LaunchChromeProfileRequest{
		UserProfileID: userProfile.ID,
		EnsureGmail:   true,
		EntityType:    "gemini",
		EntityID:      topic.ID,
	}

	logrus.Infof("Launching Chrome profile for user %s, profileID: %s", userID, userProfile.ID)
	launchResp, err := s.chromeProfileService.LaunchChromeProfile(userID, launchReq)
	if err != nil {
		logrus.Errorf("Failed to launch Chrome profile for user %s: %v", userID, err)
		return nil, fmt.Errorf("failed to launch Chrome profile: %w", err)
	}
	logrus.Infof("Chrome profile launched successfully, tunnelURL: %s", launchResp.TunnelURL)

	defer func() {
		if releaseErr := s.chromeProfileService.ReleaseChromeProfile(userID, &ReleaseChromeProfileRequest{
			UserProfileID: userProfile.ID,
		}); releaseErr != nil {
			logrus.Warnf("Failed to release Chrome profile lock: %v", releaseErr)
		}
	}()

	tunnelURL := launchResp.TunnelURL
	apiURL := fmt.Sprintf("%s/gemini/generate-outline-and-upload", strings.TrimSuffix(tunnelURL, "/"))

	requestBody := map[string]interface{}{
		"name":             userProfile.Name,
		"dirName":          profileDirName,
		"profileDirName":   profileDirName,
		"gem":              gemName,
		"notebooklmPrompt": notebooklmPrompt, // Always send, even if empty
		"sendPromptText":   sendPromptText,   // Always send, even if empty
	}

	if req.DebugPort > 0 {
		requestBody["debugPort"] = req.DebugPort
	}

	websiteURLs := s.parseURLArray(req.Website)
	if len(websiteURLs) > 0 {
		requestBody["website"] = websiteURLs
	}

	youtubeURLs := s.parseURLArray(req.YouTube)
	if len(youtubeURLs) > 0 {
		requestBody["youtube"] = youtubeURLs
	}
	if req.TextContent != "" {
		requestBody["textContent"] = req.TextContent
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		logrus.Errorf("Failed to marshal request body: %v", err)
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "Green-Provider-Services/1.0")
	httpReq.Header.Set("X-User-ID", userID)

	client := &http.Client{
		Timeout: 5 * time.Minute,
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		logrus.Errorf("HTTP request failed to automation backend %s: %v", apiURL, err)
		return nil, fmt.Errorf("failed to call automation backend: %w", err)
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logrus.Errorf("Failed to read response body: %v", err)
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logrus.Errorf("Automation backend returned error status %d: %s", resp.StatusCode, string(bodyBytes))
		var errorResp map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &errorResp); err == nil {
			if errorMsg, ok := errorResp["error"].(string); ok {
				return nil, fmt.Errorf("automation backend error: %s", errorMsg)
			}
			if errorMsg, ok := errorResp["message"].(string); ok {
				return nil, fmt.Errorf("automation backend error: %s", errorMsg)
			}
		}
		return nil, fmt.Errorf("automation backend returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var responseData map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &responseData); err != nil {
		logrus.Warnf("Failed to parse response as JSON, returning raw response: %v", err)
		return &models.GenerateOutlineResponse{
			Success: true,
			Message: "Outline generated and uploaded successfully",
			Data:    string(bodyBytes),
		}, nil
	}

	response := &models.GenerateOutlineResponse{
		Success: true,
		Data:    responseData,
	}

	if msg, ok := responseData["message"].(string); ok {
		response.Message = msg
	} else {
		response.Message = "Outline generated and uploaded successfully"
	}

	return response, nil
}

func (s *GeminiService) parseURLArray(urls []string) []string {
	if len(urls) == 0 {
		return []string{}
	}

	parsedURLs := make([]string, 0, len(urls))
	for _, urlStr := range urls {
		urlStr = strings.TrimSpace(urlStr)
		if urlStr == "" {
			continue
		}

		if strings.HasPrefix(urlStr, "[") && strings.HasSuffix(urlStr, "]") {
			var urlArray []string
			if err := json.Unmarshal([]byte(urlStr), &urlArray); err == nil {
				parsedURLs = append(parsedURLs, urlArray...)
			} else {
				logrus.Warnf("Failed to parse URL JSON string '%s': %v, using as-is", urlStr, err)
				parsedURLs = append(parsedURLs, urlStr)
			}
		} else {
			parsedURLs = append(parsedURLs, urlStr)
		}
	}

	return parsedURLs
}

// normalizeUsername normalizes username for use as prefix
// Converts to lowercase, replaces spaces and special chars with underscore
func (s *GeminiService) normalizeUsername(username string) string {
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

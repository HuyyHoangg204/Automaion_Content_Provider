package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
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
	topicService         *TopicService
	chromeProfileService *ChromeProfileService
}

func NewGeminiService(userProfileRepo *repository.UserProfileRepository, appRepo *repository.AppRepository, topicRepo *repository.TopicRepository, topicService *TopicService, chromeProfileService *ChromeProfileService) *GeminiService {
	return &GeminiService{
		userProfileRepo:      userProfileRepo,
		appRepo:              appRepo,
		topicRepo:            topicRepo,
		topicService:         topicService,
		chromeProfileService: chromeProfileService,
	}
}

func (s *GeminiService) GenerateOutlineAndUpload(userID string, topicID string, req *models.GenerateOutlineRequest) (*models.GenerateOutlineResponse, error) {
	logrus.Infof("GenerateOutlineAndUpload called for user %s with topic_id: %s", userID, topicID)

	// Get topic from DB with UserProfile preloaded
	topic, err := s.topicRepo.GetByID(topicID)
	if err != nil {
		logrus.Errorf("Failed to get topic %s: %v", topicID, err)
		return nil, fmt.Errorf("topic not found: %w", err)
	}

	// Check if user has permission to access this topic (owner or assigned)
	canAccess, accessType, err := s.topicService.CanUserAccessTopic(userID, topicID, false)
	if err != nil {
		logrus.Errorf("Failed to check topic access for user %s, topic %s: %v", userID, topicID, err)
		return nil, fmt.Errorf("failed to check topic access: %w", err)
	}
	if !canAccess {
		logrus.Errorf("User %s does not have permission to access topic %s", userID, topicID)
		return nil, fmt.Errorf("topic not found")
	}

	// Use topic owner's profile instead of current user's profile
	// This ensures the profile has the correct Gemini account login
	ownerProfile, err := s.userProfileRepo.GetByID(topic.UserProfileID)
	if err != nil {
		logrus.Errorf("Failed to get owner profile %s for topic %s: %v", topic.UserProfileID, topicID, err)
		return nil, fmt.Errorf("failed to get owner profile: %w", err)
	}

	logrus.Infof("User %s (access type: %s) using owner profile %s (user: %s) for topic %s",
		userID, accessType, ownerProfile.ID, ownerProfile.UserID, topicID)

	profileDirName := ownerProfile.ProfileDirName
	logrus.Infof("Using owner profile: name=%s, dirName=%s", ownerProfile.Name, profileDirName)

	// Generate gem name dynamically from owner's username + topic name (topic no longer stores a single gem name)
	username := "user" // Default fallback
	if ownerProfile.User.ID != "" {
		if ownerProfile.User.Username != "" {
			username = s.normalizeUsername(ownerProfile.User.Username)
		} else {
			logrus.Warnf("Owner User.Username is empty for userID %s, using default 'user' prefix", ownerProfile.UserID)
		}
	} else {
		logrus.Warnf("Owner User not preloaded for profileID %s, using default 'user' prefix", ownerProfile.ID)
	}
	gemName := fmt.Sprintf("%s_%s", username, topic.Name)

	// Use owner's profile to launch Chrome (ensures correct Gemini account login)
	launchReq := &LaunchChromeProfileRequest{
		UserProfileID: ownerProfile.ID, // Use owner's profile, not current user's profile
		EnsureGmail:   true,
		EntityType:    "gemini",
		EntityID:      topic.ID,
	}

	logrus.Infof("Launching Chrome profile for user %s (using owner profile %s for topic %s)", userID, ownerProfile.ID, topicID)
	launchResp, err := s.chromeProfileService.LaunchChromeProfile(userID, launchReq)
	if err != nil {
		logrus.Errorf("Failed to launch Chrome profile for user %s: %v", userID, err)
		return nil, fmt.Errorf("failed to launch Chrome profile: %w", err)
	}
	logrus.Infof("Chrome profile launched successfully, tunnelURL: %s", launchResp.TunnelURL)

	defer func() {
		if releaseErr := s.chromeProfileService.ReleaseChromeProfile(userID, &ReleaseChromeProfileRequest{
			UserProfileID: ownerProfile.ID, // Release owner's profile lock
		}); releaseErr != nil {
			logrus.Warnf("Failed to release Chrome profile lock: %v", releaseErr)
		}
	}()

	tunnelURL := launchResp.TunnelURL
	apiURL := fmt.Sprintf("%s/gemini/generate-outline-and-upload", strings.TrimSuffix(tunnelURL, "/"))

	requestBody := map[string]interface{}{
		"name":           ownerProfile.Name, // Use owner's profile name
		"dirName":        profileDirName,
		"profileDirName": profileDirName,
		"gem":            gemName,
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
		// Clean citation markers from raw response string
		cleanedResponse := s.removeCitationMarkers(string(bodyBytes))
		return &models.GenerateOutlineResponse{
			Success: true,
			Message: "Outline generated and uploaded successfully",
			Data:    cleanedResponse,
		}, nil
	}

	// Remove citation markers from response data
	s.cleanCitationMarkersFromData(responseData)

	response := &models.GenerateOutlineResponse{
		Success: true,
		Data:    responseData,
	}

	if msg, ok := responseData["message"].(string); ok {
		response.Message = s.removeCitationMarkers(msg)
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

// removeCitationMarkers removes citation markers like [cite:...] and [cite_start] from text
func (s *GeminiService) removeCitationMarkers(text string) string {
	// Remove [cite_start] markers
	text = strings.ReplaceAll(text, "[cite_start]", "")

	// Remove [cite:...] patterns using regex
	// Pattern matches [cite: followed by optional spaces, numbers, commas, spaces, and closing ]
	citeRegex := regexp.MustCompile(`\[cite:\s*[0-9,\s]*\]`)
	text = citeRegex.ReplaceAllString(text, "")

	// Clean up multiple spaces that might be left
	spaceRegex := regexp.MustCompile(`\s+`)
	text = spaceRegex.ReplaceAllString(text, " ")

	return strings.TrimSpace(text)
}

// cleanCitationMarkersFromData recursively removes citation markers from response data
func (s *GeminiService) cleanCitationMarkersFromData(data interface{}) {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, value := range v {
			switch val := value.(type) {
			case string:
				v[key] = s.removeCitationMarkers(val)
			case map[string]interface{}:
				s.cleanCitationMarkersFromData(val)
			case []interface{}:
				s.cleanCitationMarkersFromData(val)
			}
		}
	case []interface{}:
		for i, item := range v {
			switch val := item.(type) {
			case string:
				v[i] = s.removeCitationMarkers(val)
			case map[string]interface{}:
				s.cleanCitationMarkersFromData(val)
			case []interface{}:
				s.cleanCitationMarkersFromData(val)
			}
		}
	}
}

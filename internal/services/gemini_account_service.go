package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/sirupsen/logrus"
)

type GeminiAccountService struct {
	geminiAccountRepo *repository.GeminiAccountRepository
	appRepo           *repository.AppRepository
	boxRepo           *repository.BoxRepository
	topicRepo         *repository.TopicRepository
	topicUserRepo     *repository.TopicUserRepository
}

func NewGeminiAccountService(
	geminiAccountRepo *repository.GeminiAccountRepository,
	appRepo *repository.AppRepository,
	boxRepo *repository.BoxRepository,
	topicRepo *repository.TopicRepository,
	topicUserRepo *repository.TopicUserRepository,
) *GeminiAccountService {
	return &GeminiAccountService{
		geminiAccountRepo: geminiAccountRepo,
		appRepo:           appRepo,
		boxRepo:           boxRepo,
		topicRepo:         topicRepo,
		topicUserRepo:     topicUserRepo,
	}
}

// SetupGeminiAccount sets up a Gemini account on a specific machine
// If account already exists, updates password and email instead of returning error
func (s *GeminiAccountService) SetupGeminiAccount(req *models.SetupGeminiAccountRequest) (*models.GeminiAccount, error) {
	// Check if machine exists
	box, err := s.boxRepo.GetByMachineID(req.MachineID)
	if err != nil {
		return nil, fmt.Errorf("machine not found: %w", err)
	}

	// Check if account already exists for this machine
	// Logic: 1 machine = 1 account, cho phép update account (thay đổi email/password)
	// QUAN TRỌNG: Chỉ check để biết có account hay không, KHÔNG update database ở đây
	// Phải đợi automation backend response thành công rồi mới update database
	var existingAccount *models.GeminiAccount
	existingAccounts, err := s.geminiAccountRepo.GetByMachineID(req.MachineID)
	if err == nil && len(existingAccounts) > 0 {
		// Machine đã có account, sẽ update account đầu tiên (1 machine = 1 account)
		existingAccount = existingAccounts[0]
		logrus.Infof("Found existing Gemini account %s with email %s on machine %s, will update to email %s after automation backend success",
			existingAccount.ID, existingAccount.Email, req.MachineID, req.Email)
	}

	// Get automation app for this machine to get tunnel URL
	apps, err := s.appRepo.GetByBoxID(box.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get apps for machine: %w", err)
	}

	var automationApp *models.App
	for _, app := range apps {
		if app.TunnelURL != nil && *app.TunnelURL != "" && strings.ToLower(app.Name) == "automation" {
			automationApp = app
			break
		}
	}

	if automationApp == nil || automationApp.TunnelURL == nil {
		return nil, errors.New("automation app with tunnel URL not found for this machine")
	}

	// QUAN TRỌNG: Luôn gọi automation backend TRƯỚC, đợi response thành công rồi mới update database

	// Kiểm tra tunnel URL có accessible không trước khi gọi API chính
	tunnelBaseURL := strings.TrimSuffix(*automationApp.TunnelURL, "/")
	healthCheckURL := fmt.Sprintf("%s/health", tunnelBaseURL)

	logrus.Infof("Checking automation backend health at %s before setup", healthCheckURL)
	healthClient := &http.Client{
		Timeout: 5 * time.Second, // Short timeout for health check
	}

	healthResp, healthErr := healthClient.Get(healthCheckURL)
	if healthErr != nil {
		logrus.Warnf("Health check failed (may be normal if /health endpoint doesn't exist): %v", healthErr)
		// Continue anyway - some backends don't have /health endpoint
	} else {
		healthResp.Body.Close()
		if healthResp.StatusCode >= 200 && healthResp.StatusCode < 300 {
			logrus.Infof("Automation backend health check passed (status: %d)", healthResp.StatusCode)
		} else {
			logrus.Warnf("Automation backend health check returned status %d", healthResp.StatusCode)
		}
	}

	// Call automation backend API to setup/update Gemini account
	// Endpoint đúng: /gemini/accounts/setup (có "s" trong "accounts")
	apiURL := fmt.Sprintf("%s/gemini/accounts/setup", tunnelBaseURL)
	logrus.Infof("Calling automation backend at %s for machine %s (email: %s)", apiURL, req.MachineID, req.Email)

	requestBody := map[string]interface{}{
		"email":    req.Email,
		"password": req.Password,
		// Note: API chỉ cần email và password, không cần name hoặc userDataDir
	}
	if req.DebugPort != nil {
		requestBody["debugPort"] = *req.DebugPort
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "Green-Provider-Services/1.0")

	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	// Log DNS resolution và connection details
	logrus.Infof("Attempting to connect to automation backend: %s", apiURL)

	startTime := time.Now()
	resp, err := client.Do(httpReq)
	requestDuration := time.Since(startTime)

	if err != nil {
		logrus.Errorf("Automation backend request failed after %v: %v", requestDuration, err)
		logrus.Errorf("Tunnel URL: %s, Resolved IP: 158.69.59.214:8085 (from error message)", *automationApp.TunnelURL)
		logrus.Errorf("Possible causes: 1) Automation backend not running, 2) Firewall blocking port 8085, 3) Tunnel not working properly")
		return nil, fmt.Errorf("failed to call automation backend: %w", err)
	}
	defer resp.Body.Close()

	logrus.Infof("Automation backend responded with status %d after %v", resp.StatusCode, requestDuration)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check response status - nếu lỗi thì return ngay, KHÔNG update database
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logrus.Errorf("Automation backend returned error status %d after %v. Response: %s", resp.StatusCode, requestDuration, string(bodyBytes))
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

	// Parse response - chỉ parse khi status code thành công
	var responseData map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &responseData); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract gemini_accessible from response
	// Response có thể có format: {"success": true, "message": "...", "path": "..."}
	// hoặc {"geminiAccessible": true, ...}
	geminiAccessible := true // Default to true for successful setup
	if accessible, ok := responseData["geminiAccessible"].(bool); ok {
		geminiAccessible = accessible
	}

	// Check success field if present
	if success, ok := responseData["success"].(bool); ok && !success {
		logrus.Warnf("Automation backend returned success=false in response: %s", string(bodyBytes))
		// Vẫn tiếp tục vì status code đã là 200
	}

	logrus.Infof("Automation backend setup successful, gemini_accessible: %v", geminiAccessible)

	// QUAN TRỌNG: Chỉ update database SAU KHI automation backend đã thành công
	// If account already exists, update it instead of creating new
	if existingAccount != nil {
		// Update existing account
		existingAccount.Email = req.Email
		existingAccount.Password = req.Password // TODO: Encrypt password before storing
		existingAccount.IsActive = true
		existingAccount.IsLocked = false // Unlock if was locked
		existingAccount.LockedAt = nil
		existingAccount.LockedReason = ""
		existingAccount.GeminiAccessible = geminiAccessible
		existingAccount.AppID = &automationApp.ID

		if err := s.geminiAccountRepo.Update(existingAccount); err != nil {
			logrus.Errorf("Failed to update Gemini account: %v", err)
			return nil, fmt.Errorf("failed to update Gemini account in database: %w", err)
		}

		logrus.Infof("Successfully updated Gemini account %s (email: %s) on machine %s", existingAccount.ID, existingAccount.Email, existingAccount.MachineID)
		return existingAccount, nil
	}

	// Create new GeminiAccount in database
	// 1 machine = 1 account
	account := &models.GeminiAccount{
		MachineID:        req.MachineID,
		AppID:            &automationApp.ID,
		Email:            req.Email,
		Password:         req.Password, // TODO: Encrypt password before storing
		IsActive:         true,
		IsLocked:         false,
		GeminiAccessible: geminiAccessible,
		TopicsCount:      0,
	}

	if err := s.geminiAccountRepo.Create(account); err != nil {
		// Check if error is due to unique constraint violation
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "duplicate key") || strings.Contains(errStr, "unique constraint") || strings.Contains(errStr, "violates unique constraint") {
			return nil, fmt.Errorf("gemini account with email %s already exists on machine %s", req.Email, req.MachineID)
		}
		logrus.Errorf("Failed to create Gemini account: %v", err)
		return nil, fmt.Errorf("failed to create Gemini account in database: %w", err)
	}

	logrus.Infof("Successfully setup Gemini account %s (email: %s) on machine %s", account.ID, account.Email, account.MachineID)

	return account, nil
}

// GetAvailableAccountForMachine gets the Gemini account for a specific machine
// Logic: 1 machine = 1 account (returns the first available account)
// Also increments topics_count and updates last_used_at
func (s *GeminiAccountService) GetAvailableAccountForMachine(machineID string) (*models.GeminiAccount, error) {
	accounts, err := s.geminiAccountRepo.GetActiveByMachineID(machineID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active accounts for machine: %w", err)
	}

	if len(accounts) == 0 {
		return nil, errors.New("no available Gemini accounts found for this machine")
	}

	// 1 machine = 1 account: return the first account
	// If multiple accounts exist (legacy data), use the first one
	account := accounts[0]
	if len(accounts) > 1 {
		logrus.Warnf("Machine %s has %d accounts, using the first one (ID: %s). Consider cleaning up duplicate accounts.", machineID, len(accounts), account.ID)
	}

	// Update account: increment topics_count and last_used_at
	if err := s.geminiAccountRepo.IncrementTopicsCount(account.ID); err != nil {
		logrus.Warnf("Failed to increment topics_count for account %s: %v", account.ID, err)
	}
	if err := s.geminiAccountRepo.UpdateLastUsedAt(account.ID); err != nil {
		logrus.Warnf("Failed to update last_used_at for account %s: %v", account.ID, err)
	}

	// Reload account to get updated values
	reloadedAccount, err := s.geminiAccountRepo.GetByID(account.ID)
	if err != nil {
		logrus.Warnf("Failed to reload account %s: %v", account.ID, err)
		// Return original account if reload fails (still has all necessary data)
		return account, nil
	}

	return reloadedAccount, nil
}

// LockAccount locks a Gemini account and deletes all topics created with it
func (s *GeminiAccountService) LockAccount(accountID string, reason string) (int, error) {
	// Get account
	account, err := s.geminiAccountRepo.GetByID(accountID)
	if err != nil {
		return 0, fmt.Errorf("account not found: %w", err)
	}

	if account.IsLocked {
		return 0, errors.New("account is already locked")
	}

	// Get all topics created with this account
	topics, err := s.geminiAccountRepo.GetTopicsByAccountID(accountID)
	if err != nil {
		return 0, fmt.Errorf("failed to get topics for account: %w", err)
	}

	topicCount := len(topics)
	topicIDs := make([]string, 0, topicCount)
	for _, topic := range topics {
		topicIDs = append(topicIDs, topic.ID)
	}

	// Delete topic assignments (topic_users)
	if len(topicIDs) > 0 {
		for _, topicID := range topicIDs {
			// Get all assignments for this topic
			topicUsers, err := s.topicUserRepo.GetByTopicID(topicID)
			if err == nil {
				for _, topicUser := range topicUsers {
					if err := s.topicUserRepo.DeleteByID(topicUser.ID); err != nil {
						logrus.Warnf("Failed to delete topic assignment %s: %v", topicUser.ID, err)
					}
				}
			}
		}
	}

	// Delete topics
	for _, topic := range topics {
		if err := s.topicRepo.Delete(topic.ID); err != nil {
			logrus.Warnf("Failed to delete topic %s: %v", topic.ID, err)
		}
	}

	// Lock account
	if err := s.geminiAccountRepo.LockAccount(accountID, reason); err != nil {
		return 0, fmt.Errorf("failed to lock account: %w", err)
	}

	logrus.Infof("Locked Gemini account %s and deleted %d topics", accountID, topicCount)

	return topicCount, nil
}

// UnlockAccount unlocks a Gemini account
func (s *GeminiAccountService) UnlockAccount(accountID string) error {
	account, err := s.geminiAccountRepo.GetByID(accountID)
	if err != nil {
		return fmt.Errorf("account not found: %w", err)
	}

	if !account.IsLocked {
		return errors.New("account is not locked")
	}

	if err := s.geminiAccountRepo.UnlockAccount(accountID); err != nil {
		return fmt.Errorf("failed to unlock account: %w", err)
	}

	logrus.Infof("Unlocked Gemini account %s", accountID)

	return nil
}

// GetAllAccounts retrieves all Gemini accounts with optional filters
// TopicsCount is calculated from actual topics in DB, not from cached field
func (s *GeminiAccountService) GetAllAccounts(machineID string, isActive *bool, isLocked *bool) ([]*models.GeminiAccount, error) {
	accounts, err := s.geminiAccountRepo.GetAll(machineID, isActive, isLocked)
	if err != nil {
		return nil, err
	}

	// Update topics_count from actual DB count for each account
	for _, account := range accounts {
		actualCount, err := s.geminiAccountRepo.CountTopicsByAccountID(account.ID)
		if err == nil {
			account.TopicsCount = actualCount
		} else {
			logrus.Warnf("Failed to count topics for account %s: %v", account.ID, err)
		}
	}

	return accounts, nil
}

// GetAccountByID retrieves a Gemini account by ID
// TopicsCount is calculated from actual topics in DB, not from cached field
func (s *GeminiAccountService) GetAccountByID(accountID string) (*models.GeminiAccount, error) {
	account, err := s.geminiAccountRepo.GetByID(accountID)
	if err != nil {
		return nil, err
	}

	// Update topics_count from actual DB count
	actualCount, err := s.geminiAccountRepo.CountTopicsByAccountID(accountID)
	if err == nil {
		account.TopicsCount = actualCount
	} else {
		logrus.Warnf("Failed to count topics for account %s: %v", accountID, err)
	}

	return account, nil
}

// GetAccountsByMachineID retrieves all Gemini accounts for a specific machine
// TopicsCount is calculated from actual topics in DB, not from cached field
func (s *GeminiAccountService) GetAccountsByMachineID(machineID string) ([]*models.GeminiAccount, error) {
	accounts, err := s.geminiAccountRepo.GetByMachineID(machineID)
	if err != nil {
		return nil, err
	}

	// Update topics_count from actual DB count for each account
	for _, account := range accounts {
		actualCount, err := s.geminiAccountRepo.CountTopicsByAccountID(account.ID)
		if err == nil {
			account.TopicsCount = actualCount
		} else {
			logrus.Warnf("Failed to count topics for account %s: %v", account.ID, err)
		}
	}

	return accounts, nil
}

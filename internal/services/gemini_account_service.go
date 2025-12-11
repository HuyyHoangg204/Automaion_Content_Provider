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

	// Check if account already exists for this machine with the same email
	// Nếu machine đã có account với email khác, báo lỗi (1 machine không thể có 2 accounts khác nhau)
	// Nhưng cho phép setup cùng email trên nhiều machines
	existingAccount, err := s.geminiAccountRepo.GetByMachineIDAndEmail(req.MachineID, req.Email)
	if err == nil && existingAccount != nil {
		// Account với email này đã tồn tại trên machine này, sẽ update
	} else {
		// Check if machine has account with different email
		existingAccounts, err := s.geminiAccountRepo.GetByMachineID(req.MachineID)
		if err == nil && len(existingAccounts) > 0 {
			// Machine đã có account với email khác
			for _, acc := range existingAccounts {
				if acc.Email != req.Email {
					return nil, fmt.Errorf("machine %s already has a Gemini account with email %s. Cannot setup account with different email %s. Please update existing account instead", req.MachineID, acc.Email, req.Email)
				}
			}
		}
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

	// Call automation backend API to setup/update Gemini account
	apiURL := fmt.Sprintf("%s/gemini/account/setup", strings.TrimSuffix(*automationApp.TunnelURL, "/"))

	requestBody := map[string]interface{}{
		"email":    req.Email,
		"password": req.Password,
		// Note: Không gửi accountIndex vì 1 machine = 1 account, automation backend tự biết
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

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call automation backend: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
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

	// Parse response
	var responseData map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &responseData); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract gemini_accessible from response
	// If setup is successful (status 200), default to true unless explicitly set to false
	geminiAccessible := true // Default to true for successful setup
	if accessible, ok := responseData["geminiAccessible"].(bool); ok {
		geminiAccessible = accessible
	}

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
	account, err = s.geminiAccountRepo.GetByID(account.ID)
	if err != nil {
		logrus.Warnf("Failed to reload account %s: %v", account.ID, err)
	}

	return account, nil
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
func (s *GeminiAccountService) GetAllAccounts(machineID string, isActive *bool, isLocked *bool) ([]*models.GeminiAccount, error) {
	return s.geminiAccountRepo.GetAll(machineID, isActive, isLocked)
}

// GetAccountByID retrieves a Gemini account by ID
func (s *GeminiAccountService) GetAccountByID(accountID string) (*models.GeminiAccount, error) {
	return s.geminiAccountRepo.GetByID(accountID)
}

// GetAccountsByMachineID retrieves all Gemini accounts for a specific machine
func (s *GeminiAccountService) GetAccountsByMachineID(machineID string) ([]*models.GeminiAccount, error) {
	return s.geminiAccountRepo.GetByMachineID(machineID)
}

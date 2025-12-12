package repository

import (
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"gorm.io/gorm"
)

type GeminiAccountRepository struct {
	db *gorm.DB
}

func NewGeminiAccountRepository(db *gorm.DB) *GeminiAccountRepository {
	return &GeminiAccountRepository{db: db}
}

// Create creates a new Gemini account
func (r *GeminiAccountRepository) Create(account *models.GeminiAccount) error {
	return r.db.Create(account).Error
}

// GetByID retrieves a Gemini account by ID
func (r *GeminiAccountRepository) GetByID(id string) (*models.GeminiAccount, error) {
	var account models.GeminiAccount
	err := r.db.Preload("App").Preload("Topics").First(&account, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &account, nil
}

// GetByMachineID retrieves all Gemini accounts for a specific machine
func (r *GeminiAccountRepository) GetByMachineID(machineID string) ([]*models.GeminiAccount, error) {
	var accounts []*models.GeminiAccount
	err := r.db.Where("machine_id = ?", machineID).Order("created_at ASC").Find(&accounts).Error
	return accounts, err
}

// GetActiveByMachineID retrieves active (not locked) Gemini accounts for a specific machine
// Note: 1 machine = 1 account, but this method returns array for backward compatibility
func (r *GeminiAccountRepository) GetActiveByMachineID(machineID string) ([]*models.GeminiAccount, error) {
	var accounts []*models.GeminiAccount
	err := r.db.Where("machine_id = ? AND is_active = ? AND is_locked = ? AND gemini_accessible = ?",
		machineID, true, false, true).
		Order("created_at ASC"). // Order by creation time (first account created)
		Find(&accounts).Error
	return accounts, err
}

// GetByMachineIDAndEmail retrieves a Gemini account by machine ID and email
func (r *GeminiAccountRepository) GetByMachineIDAndEmail(machineID, email string) (*models.GeminiAccount, error) {
	var account models.GeminiAccount
	err := r.db.Where("machine_id = ? AND email = ?", machineID, email).First(&account).Error
	if err != nil {
		return nil, err
	}
	return &account, nil
}

// GetByEmail retrieves all Gemini accounts with the same email (có thể trên nhiều machines)
func (r *GeminiAccountRepository) GetByEmail(email string) ([]*models.GeminiAccount, error) {
	var accounts []*models.GeminiAccount
	err := r.db.Where("email = ?", email).Order("created_at ASC").Find(&accounts).Error
	return accounts, err
}

// GetActiveByEmail retrieves active (not locked) Gemini accounts with the same email
func (r *GeminiAccountRepository) GetActiveByEmail(email string) ([]*models.GeminiAccount, error) {
	var accounts []*models.GeminiAccount
	err := r.db.Where("email = ? AND is_active = ? AND is_locked = ? AND gemini_accessible = ?",
		email, true, false, true).
		Order("created_at ASC").
		Find(&accounts).Error
	return accounts, err
}

// GetByAccountID retrieves all Gemini accounts with the same account ID (for getting all machines with same account)
// Note: This is used when we have a GeminiAccountID and want to find all machines that have this account
func (r *GeminiAccountRepository) GetMachinesByAccountID(accountID string) ([]*models.GeminiAccount, error) {
	// First get the account to get its email
	account, err := r.GetByID(accountID)
	if err != nil {
		return nil, err
	}
	// Then get all accounts with the same email
	return r.GetActiveByEmail(account.Email)
}

// GetAll retrieves all Gemini accounts with optional filters
func (r *GeminiAccountRepository) GetAll(machineID string, isActive *bool, isLocked *bool) ([]*models.GeminiAccount, error) {
	query := r.db.Model(&models.GeminiAccount{})

	if machineID != "" {
		query = query.Where("machine_id = ?", machineID)
	}
	if isActive != nil {
		query = query.Where("is_active = ?", *isActive)
	}
	if isLocked != nil {
		query = query.Where("is_locked = ?", *isLocked)
	}

	var accounts []*models.GeminiAccount
	err := query.Preload("App").Order("machine_id ASC, created_at ASC").Find(&accounts).Error
	return accounts, err
}

// Update updates a Gemini account
func (r *GeminiAccountRepository) Update(account *models.GeminiAccount) error {
	return r.db.Save(account).Error
}

// LockAccount locks a Gemini account
func (r *GeminiAccountRepository) LockAccount(id string, reason string) error {
	now := r.db.NowFunc()
	return r.db.Model(&models.GeminiAccount{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_locked":     true,
			"locked_at":     now,
			"locked_reason": reason,
			"is_active":     false,
		}).Error
}

// UnlockAccount unlocks a Gemini account
func (r *GeminiAccountRepository) UnlockAccount(id string) error {
	return r.db.Model(&models.GeminiAccount{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_locked":     false,
			"locked_at":     nil,
			"locked_reason": "",
			"is_active":     true,
		}).Error
}

// IncrementTopicsCount increments the topics_count for an account
func (r *GeminiAccountRepository) IncrementTopicsCount(id string) error {
	return r.db.Model(&models.GeminiAccount{}).
		Where("id = ?", id).
		UpdateColumn("topics_count", gorm.Expr("topics_count + 1")).Error
}

// DecrementTopicsCount decrements the topics_count for an account
func (r *GeminiAccountRepository) DecrementTopicsCount(id string) error {
	return r.db.Model(&models.GeminiAccount{}).
		Where("id = ?", id).
		UpdateColumn("topics_count", gorm.Expr("GREATEST(topics_count - 1, 0)")).Error
}

// UpdateLastUsedAt updates the last_used_at timestamp for an account
func (r *GeminiAccountRepository) UpdateLastUsedAt(id string) error {
	now := r.db.NowFunc()
	return r.db.Model(&models.GeminiAccount{}).
		Where("id = ?", id).
		Update("last_used_at", now).Error
}

// GetTopicsByAccountID retrieves all topics for a specific Gemini account
func (r *GeminiAccountRepository) GetTopicsByAccountID(accountID string) ([]*models.Topic, error) {
	var topics []*models.Topic
	err := r.db.Where("gemini_account_id = ?", accountID).Find(&topics).Error
	return topics, err
}

// CountTopicsByAccountID counts the actual number of topics for a specific Gemini account from DB
func (r *GeminiAccountRepository) CountTopicsByAccountID(accountID string) (int, error) {
	var count int64
	err := r.db.Model(&models.Topic{}).
		Where("gemini_account_id = ?", accountID).
		Count(&count).Error
	return int(count), err
}

// Delete deletes a Gemini account (soft delete if using GORM soft delete)
func (r *GeminiAccountRepository) Delete(id string) error {
	return r.db.Delete(&models.GeminiAccount{}, "id = ?", id).Error
}

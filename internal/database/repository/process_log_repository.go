package repository

import (
	"time"

	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"gorm.io/gorm"
)

type ProcessLogRepository struct {
	db *gorm.DB
}

func NewProcessLogRepository(db *gorm.DB) *ProcessLogRepository {
	return &ProcessLogRepository{db: db}
}

// Create creates a new process log
func (r *ProcessLogRepository) Create(log *models.ProcessLog) error {
	return r.db.Create(log).Error
}

// GetByEntity retrieves logs for a specific entity
func (r *ProcessLogRepository) GetByEntity(entityType, entityID string, limit, offset int) ([]*models.ProcessLog, error) {
	var logs []*models.ProcessLog
	err := r.db.Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error
	return logs, err
}

// GetByUserID retrieves logs for a specific user
func (r *ProcessLogRepository) GetByUserID(userID string, limit, offset int) ([]*models.ProcessLog, error) {
	var logs []*models.ProcessLog
	err := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error
	return logs, err
}

// GetLatestByEntity retrieves the latest log for a specific entity
func (r *ProcessLogRepository) GetLatestByEntity(entityType, entityID string) (*models.ProcessLog, error) {
	var log models.ProcessLog
	err := r.db.Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		Order("created_at DESC").
		First(&log).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// CountByEntity counts logs for a specific entity
func (r *ProcessLogRepository) CountByEntity(entityType, entityID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.ProcessLog{}).
		Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		Count(&count).Error
	return count, err
}

// DeleteOldLogs deletes logs older than specified days
func (r *ProcessLogRepository) DeleteOldLogs(days int) (int64, error) {
	cutoffDate := time.Now().AddDate(0, 0, -days)
	result := r.db.Where("created_at < ?", cutoffDate).Delete(&models.ProcessLog{})
	return result.RowsAffected, result.Error
}

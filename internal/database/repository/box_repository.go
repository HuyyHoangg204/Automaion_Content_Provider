package repository

import (
	"github.com/onegreenvn/green-provider-services-backend/internal/models"

	"gorm.io/gorm"
)

type BoxRepository struct {
	db *gorm.DB
}

func NewBoxRepository(db *gorm.DB) *BoxRepository {
	return &BoxRepository{db: db}
}

// Create creates a new box
func (r *BoxRepository) Create(box *models.Box) error {
	return r.db.Create(box).Error
}

// GetByID retrieves a box by ID
func (r *BoxRepository) GetByID(id string) (*models.Box, error) {
	var box models.Box
	err := r.db.Preload("User").Preload("Apps").First(&box, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &box, nil
}

// GetByUserID retrieves paginated boxes for a specific user
func (r *BoxRepository) GetByUserID(userID string, page, pageSize int) ([]*models.Box, int, error) {
	var boxes []*models.Box
	var total int64

	// Count total records
	err := r.db.Model(&models.Box{}).Where("user_id = ?", userID).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Get paginated results
	err = r.db.Where("user_id = ?", userID).
		Preload("Apps").
		Offset(offset).
		Limit(pageSize).
		Order("created_at DESC").
		Find(&boxes).Error

	return boxes, int(total), err
}

// GetAllByUserID retrieves all boxes for a specific user (for sync operations)
func (r *BoxRepository) GetAllByUserID(userID string) ([]*models.Box, error) {
	var boxes []*models.Box
	err := r.db.Where("user_id = ?", userID).
		Preload("Apps").
		Order("created_at DESC").
		Find(&boxes).Error
	return boxes, err
}

// GetByMachineID retrieves a box by machine ID
func (r *BoxRepository) GetByMachineID(machineID string) (*models.Box, error) {
	var box models.Box
	err := r.db.Where("machine_id = ?", machineID).First(&box).Error
	if err != nil {
		return nil, err
	}
	return &box, nil
}

// GetByUserIDAndID retrieves a box by user ID and box ID
func (r *BoxRepository) GetByUserIDAndID(userID, boxID string) (*models.Box, error) {
	var box models.Box
	err := r.db.Where("user_id = ? AND id = ?", userID, boxID).Preload("Apps").First(&box).Error
	if err != nil {
		return nil, err
	}
	return &box, nil
}

// Update updates a box
func (r *BoxRepository) Update(box *models.Box) error {
	// Use raw SQL to ensure all fields are updated
	return r.db.Exec("UPDATE boxes SET user_id = ?, name = ?, updated_at = NOW() WHERE id = ?",
		box.UserID, box.Name, box.ID).Error
}

// Delete deletes a box
func (r *BoxRepository) Delete(id string) error {
	return r.db.Delete(&models.Box{}, "id = ?", id).Error
}

// DeleteByUserIDAndID deletes a box by user ID and box ID
func (r *BoxRepository) DeleteByUserIDAndID(userID, boxID string) error {
	return r.db.Where("user_id = ? AND id = ?", userID, boxID).Delete(&models.Box{}).Error
}

// GetAll retrieves all boxes (admin only)
func (r *BoxRepository) GetAll() ([]*models.Box, error) {
	var boxes []*models.Box
	err := r.db.Preload("User").Preload("Apps").Find(&boxes).Error
	return boxes, err
}

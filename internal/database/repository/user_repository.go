package repository

import (
	"time"

	"green-anti-detect-browser-backend-v1/internal/models"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(id uint) (*models.User, error) {
	var user models.User
	err := r.db.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByUsername retrieves a user by username
func (r *UserRepository) GetByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Update updates a user
func (r *UserRepository) Update(user *models.User) error {
	return r.db.Save(user).Error
}

// Delete deletes a user
func (r *UserRepository) Delete(id uint) error {
	return r.db.Delete(&models.User{}, id).Error
}

// GetAll retrieves all users
func (r *UserRepository) GetAll() ([]*models.User, error) {
	var users []*models.User
	err := r.db.Find(&users).Error
	return users, err
}

// List retrieves all users with pagination
func (r *UserRepository) List(offset, limit int) ([]models.User, error) {
	var users []models.User
	err := r.db.Offset(offset).Limit(limit).Find(&users).Error
	return users, err
}

// UpdateLastLogin updates the last login time for a user
func (r *UserRepository) UpdateLastLogin(userID uint) error {
	now := time.Now()
	return r.db.Model(&models.User{}).Where("id = ?", userID).Update("last_login_at", now).Error
}

// IncrementTokenVersion increments the token version for a user
func (r *UserRepository) IncrementTokenVersion(userID uint) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).UpdateColumn("token_version", gorm.Expr("token_version + 1")).Error
}

// CheckUsernameExists checks if a username already exists
func (r *UserRepository) CheckUsernameExists(username string) (bool, error) {
	var count int64
	err := r.db.Model(&models.User{}).Where("username = ?", username).Count(&count).Error
	return count > 0, err
}

// GetAllUsers returns all users (admin and non-admin) with search and pagination
func (r *UserRepository) GetAllUsers(page, pageSize int, search string) ([]models.User, int64, error) {
	var users []models.User
	var total int64
	query := r.db.Model(&models.User{})
	if search != "" {
		query = query.Where("username LIKE ?", "%"+search+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

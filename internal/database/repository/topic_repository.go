package repository

import (
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"gorm.io/gorm"
)

type TopicRepository struct {
	db *gorm.DB
}

func NewTopicRepository(db *gorm.DB) *TopicRepository {
	return &TopicRepository{db: db}
}

// Create creates a new topic
func (r *TopicRepository) Create(topic *models.Topic) error {
	return r.db.Create(topic).Error
}

// GetByID retrieves a topic by ID
func (r *TopicRepository) GetByID(id string) (*models.Topic, error) {
	var topic models.Topic
	err := r.db.Preload("UserProfile").Preload("UserProfile.User").First(&topic, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &topic, nil
}

// GetByUserProfileID retrieves all topics for a user profile
func (r *TopicRepository) GetByUserProfileID(userProfileID string) ([]*models.Topic, error) {
	var topics []*models.Topic
	err := r.db.Where("user_profile_id = ?", userProfileID).Order("created_at DESC").Find(&topics).Error
	return topics, err
}

// GetByUserID retrieves all topics for a user (through UserProfile)
func (r *TopicRepository) GetByUserID(userID string) ([]*models.Topic, error) {
	var topics []*models.Topic
	err := r.db.Joins("JOIN user_profiles ON topics.user_profile_id = user_profiles.id").
		Where("user_profiles.user_id = ?", userID).
		Order("topics.created_at DESC").
		Find(&topics).Error
	return topics, err
}

// GetByGeminiGemID retrieves a topic by Gemini Gem ID
func (r *TopicRepository) GetByGeminiGemID(geminiGemID string) (*models.Topic, error) {
	var topic models.Topic
	err := r.db.Preload("UserProfile").Where("gemini_gem_id = ?", geminiGemID).First(&topic).Error
	if err != nil {
		return nil, err
	}
	return &topic, nil
}

// Update updates a topic
func (r *TopicRepository) Update(topic *models.Topic) error {
	return r.db.Save(topic).Error
}

// Delete deletes a topic
func (r *TopicRepository) Delete(id string) error {
	return r.db.Delete(&models.Topic{}, "id = ?", id).Error
}

// GetAll retrieves all topics (admin only)
func (r *TopicRepository) GetAll() ([]*models.Topic, error) {
	var topics []*models.Topic
	err := r.db.Preload("UserProfile").Preload("UserProfile.User").Order("created_at DESC").Find(&topics).Error
	return topics, err
}

// GetBySyncStatus retrieves topics by sync status
func (r *TopicRepository) GetBySyncStatus(syncStatus string) ([]*models.Topic, error) {
	var topics []*models.Topic
	err := r.db.Where("sync_status = ?", syncStatus).Find(&topics).Error
	return topics, err
}


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

// GetByUserID retrieves all topics for a user (through UserProfile - owned topics)
func (r *TopicRepository) GetByUserID(userID string) ([]*models.Topic, error) {
	var topics []*models.Topic
	err := r.db.Joins("JOIN user_profiles ON topics.user_profile_id = user_profiles.id").
		Where("user_profiles.user_id = ?", userID).
		Order("topics.created_at DESC").
		Find(&topics).Error
	return topics, err
}

// GetByUserIDIncludingAssigned retrieves all topics for a user (owned + assigned)
// This method combines owned topics and assigned topics
func (r *TopicRepository) GetByUserIDIncludingAssigned(userID string, assignedTopicIDs []string) ([]*models.Topic, error) {
	var topics []*models.Topic

	// Get owned topics
	ownedTopics, err := r.GetByUserID(userID)
	if err != nil {
		return nil, err
	}

	// Get assigned topics if any
	if len(assignedTopicIDs) > 0 {
		var assignedTopics []*models.Topic
		err = r.db.Preload("UserProfile").Preload("UserProfile.User").
			Where("id IN ?", assignedTopicIDs).
			Order("created_at DESC").
			Find(&assignedTopics).Error
		if err != nil {
			return nil, err
		}

		// Combine owned and assigned topics, avoiding duplicates
		topicMap := make(map[string]*models.Topic)
		for _, topic := range ownedTopics {
			topicMap[topic.ID] = topic
		}
		for _, topic := range assignedTopics {
			if _, exists := topicMap[topic.ID]; !exists {
				topicMap[topic.ID] = topic
			}
		}

		// Convert map back to slice
		topics = make([]*models.Topic, 0, len(topicMap))
		for _, topic := range topicMap {
			topics = append(topics, topic)
		}
	} else {
		topics = ownedTopics
	}

	return topics, nil
}

// GetByUserIDIncludingAssignedPaginated retrieves all topics for a user (owned + assigned) with pagination
func (r *TopicRepository) GetByUserIDIncludingAssignedPaginated(userID string, assignedTopicIDs []string, page, pageSize int) ([]*models.Topic, int64, error) {
	// Build query for owned topics
	ownedQuery := r.db.Table("topics").
		Joins("JOIN user_profiles ON topics.user_profile_id = user_profiles.id").
		Where("user_profiles.user_id = ?", userID)

	// Build query for assigned topics
	var combinedQuery *gorm.DB
	if len(assignedTopicIDs) > 0 {
		// Union owned and assigned topics
		combinedQuery = r.db.Table("topics").
			Where("topics.user_profile_id IN (SELECT id FROM user_profiles WHERE user_id = ?) OR topics.id IN ?", userID, assignedTopicIDs)
	} else {
		// Only owned topics
		combinedQuery = ownedQuery
	}

	// Count total
	var total int64
	if err := combinedQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results - first get topic IDs
	var topicIDs []string
	offset := (page - 1) * pageSize
	err := combinedQuery.
		Select("topics.id").
		Order("topics.created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Pluck("topics.id", &topicIDs).Error

	if err != nil {
		return nil, 0, err
	}

	if len(topicIDs) == 0 {
		return []*models.Topic{}, total, nil
	}

	// Then get full topic objects with preloads
	var topics []*models.Topic
	err = r.db.Where("id IN ?", topicIDs).
		Preload("UserProfile").Preload("UserProfile.User").
		Order("created_at DESC").
		Find(&topics).Error

	if err != nil {
		return nil, 0, err
	}

	return topics, total, nil
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

func (r *TopicRepository) Update(topic *models.Topic) error {
	return r.db.Save(topic).Error
}

// UpdateGeminiInfo updates Gemini info safely (prevents re-insertion if deleted)
func (r *TopicRepository) UpdateGeminiInfo(id string, geminiGemName string, geminiAccountID *string) error {
	return r.db.Model(&models.Topic{}).Where("id = ?", id).Updates(map[string]interface{}{
		"gemini_gem_id":     nil,
		"gemini_gem_name":   geminiGemName,
		"gemini_account_id": geminiAccountID,
		"sync_error":        "",
	}).Error
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

// GetAllPaginated retrieves all topics with pagination, search, and filters (admin only)
// search: searches in topic name, description, and creator username
// creatorID: filter by creator user ID
// syncStatus: filter by sync status
// isActive: filter by is_active status (nil = no filter)
func (r *TopicRepository) GetAllPaginated(page, pageSize int, search, creatorID, syncStatus string, isActive *bool) ([]*models.Topic, int64, error) {
	// Build base query - use Table() instead of Model() to avoid GORM confusion with JOINs
	query := r.db.Table("topics").
		Joins("LEFT JOIN user_profiles ON topics.user_profile_id = user_profiles.id").
		Joins("LEFT JOIN users ON user_profiles.user_id = users.id")

	// Apply search filter
	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where(
			"topics.name ILIKE ? OR topics.description ILIKE ? OR users.username ILIKE ?",
			searchPattern, searchPattern, searchPattern,
		)
	}

	// Apply creator filter
	if creatorID != "" {
		query = query.Where("users.id = ?", creatorID)
	}

	// Apply sync status filter
	if syncStatus != "" {
		query = query.Where("topics.sync_status = ?", syncStatus)
	}

	// Apply is_active filter
	if isActive != nil {
		query = query.Where("topics.is_active = ?", *isActive)
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	// First, get topic IDs with pagination
	var topicIDs []string
	offset := (page - 1) * pageSize
	err := query.
		Select("topics.id").
		Order("topics.created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Pluck("topics.id", &topicIDs).Error

	if err != nil {
		return nil, 0, err
	}

	if len(topicIDs) == 0 {
		return []*models.Topic{}, total, nil
	}

	// Then, get full topic objects with preloads
	var topics []*models.Topic
	err = r.db.Where("id IN ?", topicIDs).
		Preload("UserProfile").Preload("UserProfile.User").
		Order("created_at DESC").
		Find(&topics).Error

	if err != nil {
		return nil, 0, err
	}

	return topics, total, nil
}

// GetBySyncStatus retrieves topics by sync status
func (r *TopicRepository) GetBySyncStatus(syncStatus string) ([]*models.Topic, error) {
	var topics []*models.Topic
	err := r.db.Where("sync_status = ?", syncStatus).Find(&topics).Error
	return topics, err
}

// GetByGeminiAccountID retrieves all topics created with a specific Gemini account
func (r *TopicRepository) GetByGeminiAccountID(geminiAccountID string) ([]*models.Topic, error) {
	var topics []*models.Topic
	err := r.db.Where("gemini_account_id = ?", geminiAccountID).
		Preload("UserProfile").Preload("UserProfile.User").
		Order("created_at DESC").
		Find(&topics).Error
	return topics, err
}

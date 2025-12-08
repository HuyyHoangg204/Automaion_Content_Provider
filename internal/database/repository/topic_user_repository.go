package repository

import (
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"gorm.io/gorm"
)

type TopicUserRepository struct {
	db *gorm.DB
}

func NewTopicUserRepository(db *gorm.DB) *TopicUserRepository {
	return &TopicUserRepository{db: db}
}

// Create assigns a topic to a user
func (r *TopicUserRepository) Create(topicUser *models.TopicUser) error {
	return r.db.Create(topicUser).Error
}

// GetByID retrieves a topic-user assignment by ID
func (r *TopicUserRepository) GetByID(id string) (*models.TopicUser, error) {
	var topicUser models.TopicUser
	err := r.db.Preload("Topic").Preload("User").First(&topicUser, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &topicUser, nil
}

// GetByTopicID retrieves all users assigned to a topic
func (r *TopicUserRepository) GetByTopicID(topicID string) ([]*models.TopicUser, error) {
	var topicUsers []*models.TopicUser
	err := r.db.Preload("User").Where("topic_id = ?", topicID).Order("assigned_at DESC").Find(&topicUsers).Error
	return topicUsers, err
}

// GetByUserID retrieves all topics assigned to a user
func (r *TopicUserRepository) GetByUserID(userID string) ([]*models.TopicUser, error) {
	var topicUsers []*models.TopicUser
	err := r.db.Preload("Topic").Preload("Topic.UserProfile").Preload("Topic.UserProfile.User").
		Where("user_id = ?", userID).Order("assigned_at DESC").Find(&topicUsers).Error
	return topicUsers, err
}

// GetByTopicAndUser checks if a user is assigned to a topic
func (r *TopicUserRepository) GetByTopicAndUser(topicID, userID string) (*models.TopicUser, error) {
	var topicUser models.TopicUser
	err := r.db.Where("topic_id = ? AND user_id = ?", topicID, userID).First(&topicUser).Error
	if err != nil {
		return nil, err
	}
	return &topicUser, nil
}

// IsUserAssigned checks if a user is assigned to a topic
func (r *TopicUserRepository) IsUserAssigned(topicID, userID string) (bool, error) {
	var count int64
	err := r.db.Model(&models.TopicUser{}).
		Where("topic_id = ? AND user_id = ?", topicID, userID).
		Count(&count).Error
	return count > 0, err
}

// Update updates a topic-user assignment
func (r *TopicUserRepository) Update(topicUser *models.TopicUser) error {
	return r.db.Save(topicUser).Error
}

// Delete removes a topic-user assignment
func (r *TopicUserRepository) Delete(topicID, userID string) error {
	return r.db.Where("topic_id = ? AND user_id = ?", topicID, userID).Delete(&models.TopicUser{}).Error
}

// DeleteByID removes a topic-user assignment by ID
func (r *TopicUserRepository) DeleteByID(id string) error {
	return r.db.Delete(&models.TopicUser{}, "id = ?", id).Error
}

// GetTopicIDsAssignedToUser returns list of topic IDs assigned to a user
func (r *TopicUserRepository) GetTopicIDsAssignedToUser(userID string) ([]string, error) {
	var topicIDs []string
	err := r.db.Model(&models.TopicUser{}).
		Where("user_id = ?", userID).
		Pluck("topic_id", &topicIDs).Error
	return topicIDs, err
}

// GetByTopicIDs retrieves all users assigned to multiple topics (batch load)
// Returns a map where key is topicID and value is list of TopicUser assignments
func (r *TopicUserRepository) GetByTopicIDs(topicIDs []string) (map[string][]*models.TopicUser, error) {
	if len(topicIDs) == 0 {
		return make(map[string][]*models.TopicUser), nil
	}

	var topicUsers []*models.TopicUser
	err := r.db.Preload("User").
		Where("topic_id IN ?", topicIDs).
		Order("assigned_at DESC").
		Find(&topicUsers).Error
	if err != nil {
		return nil, err
	}

	// Group by topic ID
	result := make(map[string][]*models.TopicUser)
	for _, topicUser := range topicUsers {
		result[topicUser.TopicID] = append(result[topicUser.TopicID], topicUser)
	}

	return result, nil
}


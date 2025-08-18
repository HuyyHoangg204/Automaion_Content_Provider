package repository

import (
	"time"

	"green-provider-services-backend/internal/models"

	"gorm.io/gorm"
)

type RefreshTokenRepository struct {
	db *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

// Create creates a new refresh token
func (r *RefreshTokenRepository) Create(refreshToken *models.RefreshToken) error {
	return r.db.Create(refreshToken).Error
}

// GetByToken retrieves a refresh token by token string
func (r *RefreshTokenRepository) GetByToken(token string) (*models.RefreshToken, error) {
	var refreshToken models.RefreshToken
	err := r.db.Where("token = ? AND is_revoked = ?", token, false).First(&refreshToken).Error
	if err != nil {
		return nil, err
	}
	return &refreshToken, nil
}

// GetByUserID retrieves all refresh tokens for a user
func (r *RefreshTokenRepository) GetByUserID(userID string) ([]models.RefreshToken, error) {
	var refreshTokens []models.RefreshToken
	err := r.db.Where("user_id = ? AND is_revoked = ?", userID, false).Find(&refreshTokens).Error
	return refreshTokens, err
}

// RevokeToken revokes a specific refresh token
func (r *RefreshTokenRepository) RevokeToken(token string) error {
	return r.db.Model(&models.RefreshToken{}).Where("token = ?", token).Update("is_revoked", true).Error
}

// RevokeAllUserTokens revokes all refresh tokens for a user
func (r *RefreshTokenRepository) RevokeAllUserTokens(userID string) error {
	return r.db.Model(&models.RefreshToken{}).Where("user_id = ?", userID).Update("is_revoked", true).Error
}

// DeleteExpiredTokens deletes expired refresh tokens
func (r *RefreshTokenRepository) DeleteExpiredTokens() error {
	return r.db.Where("expires_at < ?", time.Now()).Delete(&models.RefreshToken{}).Error
}

// DeleteRevokedTokens deletes revoked refresh tokens
func (r *RefreshTokenRepository) DeleteRevokedTokens() error {
	return r.db.Where("is_revoked = ?", true).Delete(&models.RefreshToken{}).Error
}

// CleanupTokens cleans up expired and revoked tokens
func (r *RefreshTokenRepository) CleanupTokens() error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("expires_at < ?", time.Now()).Delete(&models.RefreshToken{}).Error; err != nil {
			return err
		}
		return tx.Where("is_revoked = ?", true).Delete(&models.RefreshToken{}).Error
	})
}

// CountActiveTokensByUser counts active refresh tokens for a user
func (r *RefreshTokenRepository) CountActiveTokensByUser(userID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.RefreshToken{}).Where("user_id = ? AND is_revoked = ?", userID, false).Count(&count).Error
	return count, err
}

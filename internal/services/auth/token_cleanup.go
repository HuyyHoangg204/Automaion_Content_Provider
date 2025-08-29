package auth

import (
	"time"

	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type TokenCleanupService struct {
	refreshTokenRepo *repository.RefreshTokenRepository
	interval         time.Duration
	stopChan         chan bool
}

func NewTokenCleanupService(db *gorm.DB) *TokenCleanupService {
	return &TokenCleanupService{
		refreshTokenRepo: repository.NewRefreshTokenRepository(db),
		interval:         24 * time.Hour, // Cleanup every 24 hours
		stopChan:         make(chan bool),
	}
}

// Start starts the token cleanup service
func (s *TokenCleanupService) Start() {
	go s.run()
	logrus.Info("Token cleanup service started")
}

// Stop stops the token cleanup service
func (s *TokenCleanupService) Stop() {
	s.stopChan <- true
	logrus.Info("Token cleanup service stopped")
}

// run runs the cleanup loop
func (s *TokenCleanupService) run() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Run initial cleanup
	s.cleanup()

	for {
		select {
		case <-ticker.C:
			s.cleanup()
		case <-s.stopChan:
			return
		}
	}
}

// cleanup performs the actual cleanup of expired and revoked tokens
func (s *TokenCleanupService) cleanup() {
	logrus.Info("Starting token cleanup...")

	// Cleanup expired and revoked tokens
	err := s.refreshTokenRepo.CleanupTokens()
	if err != nil {
		logrus.Errorf("Failed to cleanup tokens: %v", err)
		return
	}

	logrus.Info("Token cleanup completed")
}

// SetInterval sets the cleanup interval
func (s *TokenCleanupService) SetInterval(interval time.Duration) {
	s.interval = interval
}

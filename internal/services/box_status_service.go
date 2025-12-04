package services

import (
	"time"

	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type BoxStatusUpdateService struct {
	boxRepo  *repository.BoxRepository
	interval time.Duration
	stopChan chan bool
}

func NewBoxStatusUpdateService(db *gorm.DB) *BoxStatusUpdateService {
	return &BoxStatusUpdateService{
		boxRepo:  repository.NewBoxRepository(db),
		interval: 1 * time.Minute, // Check every 1 minute
		stopChan: make(chan bool),
	}
}

// Start starts the box status update service
func (s *BoxStatusUpdateService) Start() {
	go s.run()
	logrus.Info("Box status update service started")
}

// Stop stops the box status update service
func (s *BoxStatusUpdateService) Stop() {
	s.stopChan <- true
	logrus.Info("Box status update service stopped")
}

// run runs the update loop
func (s *BoxStatusUpdateService) run() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Run initial update
	s.updateOfflineBoxes()

	for {
		select {
		case <-ticker.C:
			s.updateOfflineBoxes()
		case <-s.stopChan:
			return
		}
	}
}

// updateOfflineBoxes updates is_online = false for boxes that haven't sent heartbeat in > 5 minutes
func (s *BoxStatusUpdateService) updateOfflineBoxes() {
	logrus.Debug("Starting box status update...")

	// Get all boxes that are marked as online
	allBoxes, err := s.boxRepo.GetAll()
	if err != nil {
		logrus.Errorf("Failed to get boxes: %v", err)
		return
	}

	now := time.Now()
	offlineThreshold := 5 * time.Minute
	updatedCount := 0

	for _, box := range allBoxes {
		// Only update boxes that are marked as online
		if !box.IsOnline {
			continue
		}

		// Check if last heartbeat is older than threshold
		timeSinceLastSeen := now.Sub(box.UpdatedAt)
		if timeSinceLastSeen > offlineThreshold {
			// Update box to offline
			box.IsOnline = false
			if err := s.boxRepo.Update(box); err != nil {
				logrus.Errorf("Failed to update box %s to offline: %v", box.ID, err)
				continue
			}
			updatedCount++
			logrus.Debugf("Updated box %s (%s) to offline (last heartbeat: %v ago)", box.ID, box.Name, timeSinceLastSeen)
		}
	}

	if updatedCount > 0 {
		logrus.Infof("Box status update completed: marked %d box(es) as offline", updatedCount)
	} else {
		logrus.Debug("Box status update completed: no boxes to update")
	}
}

// SetInterval sets the update interval
func (s *BoxStatusUpdateService) SetInterval(interval time.Duration) {
	s.interval = interval
}

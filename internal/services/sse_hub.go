package services

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/sirupsen/logrus"
)

// SSEHub manages Server-Sent Events connections for real-time log streaming
type SSEHub struct {
	// Map of entity keys to channels
	// Key format: "entity_type:entity_id" or "user:user_id"
	clients map[string]map[chan []byte]bool
	mu      sync.RWMutex
}

// NewSSEHub creates a new SSE hub
func NewSSEHub() *SSEHub {
	return &SSEHub{
		clients: make(map[string]map[chan []byte]bool),
	}
}

// RegisterClient registers a new SSE client for an entity
func (h *SSEHub) RegisterClient(entityType, entityID string) chan []byte {
	h.mu.Lock()
	defer h.mu.Unlock()

	key := fmt.Sprintf("%s:%s", entityType, entityID)
	clientChan := make(chan []byte, 10) // Buffer size 10

	if h.clients[key] == nil {
		h.clients[key] = make(map[chan []byte]bool)
	}
	h.clients[key][clientChan] = true

	logrus.Infof("SSE client registered for %s (total clients: %d)", key, len(h.clients[key]))
	return clientChan
}

// UnregisterClient unregisters an SSE client
func (h *SSEHub) UnregisterClient(entityType, entityID string, clientChan chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	key := fmt.Sprintf("%s:%s", entityType, entityID)
	if h.clients[key] != nil {
		delete(h.clients[key], clientChan)
		close(clientChan)

		// Clean up empty maps
		if len(h.clients[key]) == 0 {
			delete(h.clients, key)
		}
	}

	logrus.Infof("SSE client unregistered for %s (remaining clients: %d)", key, len(h.clients[key]))
}

// BroadcastLog broadcasts a log to all clients subscribed to the entity
func (h *SSEHub) BroadcastLog(log *models.ProcessLog) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Broadcast to entity-specific clients
	entityKey := fmt.Sprintf("%s:%s", log.EntityType, log.EntityID)
	clientsMap, exists := h.clients[entityKey]
	if !exists {
		clientsMap = nil
	}
	h.broadcastToKeyLocked(entityKey, log, clientsMap)

	// Broadcast to user-specific clients
	userKey := fmt.Sprintf("user:%s", log.UserID)
	userClientsMap, userExists := h.clients[userKey]
	if !userExists {
		userClientsMap = nil
	}
	h.broadcastToKeyLocked(userKey, log, userClientsMap)
}

// broadcastToKeyLocked broadcasts log to clients (assumes lock is already held)
func (h *SSEHub) broadcastToKeyLocked(key string, log *models.ProcessLog, clients map[chan []byte]bool) {
	if len(clients) == 0 {
		return
	}

	// Convert log to JSON
	logJSON, err := json.Marshal(log)
	if err != nil {
		logrus.Errorf("Failed to marshal log for SSE: %v", err)
		return
	}

	// Format as SSE message with event type
	// Frontend EventSource cần event type để xử lý
	message := fmt.Sprintf("event: log\ndata: %s\n\n", string(logJSON))

	// Send to all clients (non-blocking)
	for clientChan := range clients {
		select {
		case clientChan <- []byte(message):
		default:
			// Channel is full, skip this client
			logrus.Warnf("SSE client channel full, skipping: %s", key)
		}
	}
}

// GetClientCount returns the number of clients for a specific entity
func (h *SSEHub) GetClientCount(entityType, entityID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", entityType, entityID)
	if clients, exists := h.clients[key]; exists {
		return len(clients)
	}
	return 0
}

// SendHeartbeat sends a heartbeat message to keep connection alive
func (h *SSEHub) SendHeartbeat(entityType, entityID string) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", entityType, entityID)
	clients, exists := h.clients[key]
	if !exists {
		return
	}

	heartbeat := fmt.Sprintf(": heartbeat %s\n\n", time.Now().Format(time.RFC3339))
	for clientChan := range clients {
		select {
		case clientChan <- []byte(heartbeat):
		default:
			// Skip if channel is full
		}
	}
}

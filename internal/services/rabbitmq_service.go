package services

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQService struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// GetChannel returns the RabbitMQ channel (for use by other services)
func (s *RabbitMQService) GetChannel() *amqp.Channel {
	return s.channel
}

func NewRabbitMQService() (*RabbitMQService, error) {
	// Get RabbitMQ connection details from environment
	host := getEnv("RABBITMQ_HOST", "localhost")
	port := getEnv("RABBITMQ_PORT", "5672")
	user := getEnv("RABBITMQ_USER", "guest")
	pass := getEnv("RABBITMQ_PASS", "guest")

	// Build connection URL (guest user automatically uses / vhost)
	url := fmt.Sprintf("amqp://%s:%s@%s:%s/", user, pass, host, port)

	// Connect to RabbitMQ
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	// Create channel
	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare queue
	queueName := "campaign_executor"
	_, err = channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	service := &RabbitMQService{
		conn:    conn,
		channel: channel,
	}

	log.Printf("RabbitMQ service initialized successfully")
	return service, nil
}

// PublishMessage publishes a message to the specified queue
func (s *RabbitMQService) PublishMessage(ctx interface{}, queueName string, message map[string]interface{}) error {
	// Convert message to JSON
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Publish message
	err = s.channel.Publish(
		"",        // exchange
		queueName, // routing key
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
			Timestamp:   time.Now(),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Printf("Message published to queue %s: %+v", queueName, message)
	return nil
}

// Close closes the RabbitMQ connection
func (s *RabbitMQService) Close() error {
	if s.channel != nil {
		if err := s.channel.Close(); err != nil {
			log.Printf("Error closing channel: %v", err)
		}
	}
	if s.conn != nil {
		if err := s.conn.Close(); err != nil {
			log.Printf("Error closing connection: %v", err)
		}
	}
	return nil
}

// getEnv gets environment variable with fallback default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

package config

import (
	"os"
	"strconv"
)

// FrpConfig holds FRP configuration
type FrpConfig struct {
	Domain   string
	Port     int
	Token    string
	Protocol string
}

// GetFrpConfig returns FRP configuration from environment variables
func GetFrpConfig() *FrpConfig {
	port, _ := strconv.Atoi(getEnv("FRP_SERVER_PORT", "8080"))

	return &FrpConfig{
		Domain:   getEnv("FRP_DOMAIN", "frp.onegreen.cloud"),
		Port:     port,
		Token:    getEnv("FRP_TOKEN", "HelloWorld"),
		Protocol: getEnv("FRP_PROTOCOL", "http"),
	}
}

// getEnv gets environment variable with fallback default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

package config

import (
	"os"
	"strconv"
)

// FrpConfig holds FRP configuration
type FrpConfig struct {
	Domain           string
	Port             int
	Token            string
	Protocol         string
	CustomDomainHost string
}

// GetFrpConfig returns FRP configuration from environment variables
func GetFrpConfig() *FrpConfig {
	port, _ := strconv.Atoi(getEnv("FRP_SERVER_PORT", "8700"))

	return &FrpConfig{
		Domain:           getEnv("FRP_DOMAIN", ""),
		Port:             port,
		Token:            getEnv("FRP_TOKEN", ""),
		Protocol:         getEnv("FRP_PROTOCOL", ""),
		CustomDomainHost: getEnv("FRP_CUSTOM_DOMAIN_HOST", ""),
	}
}

// getEnv gets environment variable with fallback default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

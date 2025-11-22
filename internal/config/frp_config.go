package config

import (
	"os"
	"strconv"
)

// FrpConfig holds FRP configuration
type FrpConfig struct {
	Domain       string
	Port         int
	Token        string
	Protocol     string
	CustomDomain string
}

// GetFrpConfig returns FRP configuration from environment variables
func GetFrpConfig() *FrpConfig {
	port, _ := strconv.Atoi(getEnv("FRP_SERVER_PORT", "8700"))

	return &FrpConfig{
		Domain:       getEnv("FRP_SERVER_DOMAIN", ""),
		Port:         port,
		Token:        getEnv("FRP_TOKEN", ""),
		Protocol:     getEnv("FRP_PROTOCOL", ""),
		CustomDomain: getEnv("FRP_CUSTOM_DOMAIN", ""),
	}
}

// GetAutomationProfilesPath returns the Automation Profiles base directory path
func GetAutomationProfilesPath() string {
	return getEnv("AUTOMATION_PROFILES_PATH", "C:\\Users\\tranh\\AppData\\Local\\Automation_Profiles")
}

// getEnv gets environment variable with fallback default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

package config

import (
	"fmt"
	"os"
)

// HidemiumConfig contains Hidemium platform configuration
type HidemiumConfig struct {
	Name    string            `json:"name"`
	BaseURL string            `json:"base_url"`
	Routes  map[string]string `json:"routes"`
}

// GetHidemiumConfig returns Hidemium configuration
func GetHidemiumConfig() *HidemiumConfig {
	domain := os.Getenv("HIDEMIUM_DOMAIN")
	path := os.Getenv("HIDEMIUM_PATH")

	return &HidemiumConfig{
		Name:    "Hidemium",
		BaseURL: fmt.Sprintf("http://{machine_id}.%s%s", domain, path),
		Routes: map[string]string{
			// Profile Management - Based on Hidemium v4 API docs
			"list_profiles": "/v1/browser/list?is_local=false",
		},
	}
}

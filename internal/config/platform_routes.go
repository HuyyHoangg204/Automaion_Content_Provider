package config

import (
	"fmt"
	"strings"
)

// PlatformRoute represents a route configuration for a specific platform
type PlatformRoute struct {
	Name        string `json:"name"`
	BaseURL     string `json:"base_url"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
}

// PlatformEndpoint represents a specific endpoint for a platform
type PlatformEndpoint struct {
	Platform    string `json:"platform"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	FullURL     string `json:"full_url"`
	Description string `json:"description"`
}

// PlatformRoutes contains all platform route configurations
type PlatformRoutes struct {
	Hidemium HidemiumRoutes `json:"hidemium"`
	Genlogin GenloginRoutes `json:"genlogin"`
	// Add more platforms here as needed
}

// HidemiumRoutes contains all Hidemium-specific routes
type HidemiumRoutes struct {
	BaseURL     string                      `json:"base_url"`
	Description string                      `json:"description"`
	Endpoints   map[string]HidemiumEndpoint `json:"endpoints"`
}

// HidemiumEndpoint represents a specific Hidemium API endpoint
type HidemiumEndpoint struct {
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	Description string            `json:"description"`
	Parameters  map[string]string `json:"parameters,omitempty"`
}

// GenloginRoutes contains all Genlogin-specific routes
type GenloginRoutes struct {
	BaseURL     string                      `json:"base_url"`
	Description string                      `json:"description"`
	Endpoints   map[string]GenloginEndpoint `json:"endpoints"`
}

// GenloginEndpoint represents a specific Genlogin API endpoint
type GenloginEndpoint struct {
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	Description string            `json:"description"`
	Parameters  map[string]string `json:"parameters,omitempty"`
}

// GetPlatformRoutes returns the complete platform routes configuration
func GetPlatformRoutes() *PlatformRoutes {
	return &PlatformRoutes{
		Hidemium: HidemiumRoutes{
			BaseURL:     "http://{machine_id}.agent-controller.onegreen.cloud/frps",
			Description: "Hidemium browser automation platform",
			Endpoints: map[string]HidemiumEndpoint{
				"list_profiles": {
					Method:      "POST",
					Path:        "/v1/browser/list",
					Description: "Get list of browser profiles with pagination",
					Parameters: map[string]string{
						"orderName":     "0",
						"orderLastOpen": "0",
						"page":          "1",
						"limit":         "100",
						"search":        "",
						"status":        "",
						"date_range":    "[\"\", \"\"]",
						"folder_id":     "[]",
					},
				},
				"get_profile": {
					Method:      "GET",
					Path:        "/v1/browser/{profile_id}",
					Description: "Get specific browser profile details",
				},
				"start_profile": {
					Method:      "POST",
					Path:        "/v1/browser/start",
					Description: "Start a browser profile",
				},
				"stop_profile": {
					Method:      "POST",
					Path:        "/v1/browser/stop",
					Description: "Stop a running browser profile",
				},
				"create_profile": {
					Method:      "POST",
					Path:        "/v1/browser/create",
					Description: "Create a new browser profile",
				},
				"update_profile": {
					Method:      "PUT",
					Path:        "/v1/browser/{profile_id}",
					Description: "Update an existing browser profile",
				},
				"delete_profile": {
					Method:      "DELETE",
					Path:        "/v1/browser/{profile_id}",
					Description: "Delete a browser profile",
				},
			},
		},
		Genlogin: GenloginRoutes{
			BaseURL:     "http://{machine_id}.agent-controller.onegreen.cloud/genlogin",
			Description: "Genlogin browser automation platform",
			Endpoints: map[string]GenloginEndpoint{
				"list_profiles": {
					Method:      "GET",
					Path:        "/api/profiles",
					Description: "Get list of browser profiles",
				},
				"get_profile": {
					Method:      "GET",
					Path:        "/api/profiles/{profile_id}",
					Description: "Get specific browser profile details",
				},
				"start_profile": {
					Method:      "POST",
					Path:        "/api/profiles/{profile_id}/start",
					Description: "Start a browser profile",
				},
				"stop_profile": {
					Method:      "POST",
					Path:        "/api/profiles/{profile_id}/stop",
					Description: "Stop a running browser profile",
				},
				"create_profile": {
					Method:      "POST",
					Path:        "/api/profiles",
					Description: "Create a new browser profile",
				},
				"update_profile": {
					Method:      "PUT",
					Path:        "/api/profiles/{profile_id}",
					Description: "Update an existing browser profile",
				},
				"delete_profile": {
					Method:      "DELETE",
					Path:        "/api/profiles/{profile_id}",
					Description: "Delete a browser profile",
				},
			},
		},
	}
}

// GetEndpointURL constructs the full URL for a specific platform endpoint
func GetEndpointURL(platform, machineID, endpointName string) (string, error) {
	routes := GetPlatformRoutes()

	switch platform {
	case "hidemium":
		if endpoint, exists := routes.Hidemium.Endpoints[endpointName]; exists {
			baseURL := routes.Hidemium.BaseURL
			// Replace {machine_id} placeholder with actual machine ID
			baseURL = strings.Replace(baseURL, "{machine_id}", machineID, 1)
			return baseURL + endpoint.Path, nil
		}
	case "genlogin":
		if endpoint, exists := routes.Genlogin.Endpoints[endpointName]; exists {
			baseURL := routes.Genlogin.BaseURL
			// Replace {machine_id} placeholder with actual machine ID
			baseURL = strings.Replace(baseURL, "{machine_id}", machineID, 1)
			return baseURL + endpoint.Path, nil
		}
	}

	return "", fmt.Errorf("endpoint '%s' not found for platform '%s'", endpointName, platform)
}

// GetEndpointInfo returns endpoint information for a specific platform
func GetEndpointInfo(platform, endpointName string) (interface{}, error) {
	routes := GetPlatformRoutes()

	switch platform {
	case "hidemium":
		if endpoint, exists := routes.Hidemium.Endpoints[endpointName]; exists {
			return endpoint, nil
		}
	case "genlogin":
		if endpoint, exists := routes.Genlogin.Endpoints[endpointName]; exists {
			return endpoint, nil
		}
	}

	return nil, fmt.Errorf("endpoint '%s' not found for platform '%s'", endpointName, platform)
}

// IsPlatformSupported checks if a platform is supported
func IsPlatformSupported(platform string) bool {
	switch platform {
	case "hidemium", "genlogin":
		return true
	default:
		return false
	}
}

// GetSupportedPlatforms returns list of supported platforms
func GetSupportedPlatforms() []string {
	return []string{"hidemium", "genlogin"}
}

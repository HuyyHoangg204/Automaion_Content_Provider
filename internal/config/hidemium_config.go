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
			// Existing routes
			"list_profiles": "/v1/browser/list",

			// Profile CRUD Operations
			"create_profile":    "/v1/browser/create",
			"get_profile":       "/v1/browser/{profile_id}",
			"update_profile":    "/v1/browser/{profile_id}/update",
			"delete_profile":    "/v1/browser/{profile_id}/delete",
			"duplicate_profile": "/v1/browser/{profile_id}/duplicate",

			// Profile Status Management
			"start_profile":   "/v1/browser/{profile_id}/start",
			"stop_profile":    "/v1/browser/{profile_id}/stop",
			"restart_profile": "/v1/browser/{profile_id}/restart",

			// Profile Data Management
			"export_profile":  "/v1/browser/{profile_id}/export",
			"import_profile":  "/v1/browser/import",
			"backup_profile":  "/v1/browser/{profile_id}/backup",
			"restore_profile": "/v1/browser/{profile_id}/restore",

			// Profile Settings
			"get_profile_settings":    "/v1/browser/{profile_id}/settings",
			"update_profile_settings": "/v1/browser/{profile_id}/settings/update",

			// Profile Folders
			"list_folders":           "/v1/browser/folders",
			"create_folder":          "/v1/browser/folders/create",
			"move_profile_to_folder": "/v1/browser/{profile_id}/move",

			// Profile Search & Filter
			"search_profiles": "/v1/browser/search",
			"filter_profiles": "/v1/browser/filter",

			// Profile Statistics
			"get_profile_stats":  "/v1/browser/{profile_id}/stats",
			"get_profiles_stats": "/v1/browser/stats",

			// Campaign Management
			"list_campaigns":  "/v1/campaigns",
			"create_campaign": "/v1/campaigns/create",
			"get_campaign":    "/v1/campaigns/{campaign_id}",
			"update_campaign": "/v1/campaigns/{campaign_id}/update",
			"delete_campaign": "/v1/campaigns/{campaign_id}/delete",

			// Campaign Execution
			"start_campaign":  "/v1/campaigns/{campaign_id}/start",
			"stop_campaign":   "/v1/campaigns/{campaign_id}/stop",
			"pause_campaign":  "/v1/campaigns/{campaign_id}/pause",
			"resume_campaign": "/v1/campaigns/{campaign_id}/resume",

			// Campaign Status & Monitoring
			"get_campaign_status":  "/v1/campaigns/{campaign_id}/status",
			"get_campaign_logs":    "/v1/campaigns/{campaign_id}/logs",
			"get_campaign_results": "/v1/campaigns/{campaign_id}/results",

			// Profile Assignment to Campaigns
			"assign_profiles_to_campaign":   "/v1/campaigns/{campaign_id}/profiles/assign",
			"remove_profiles_from_campaign": "/v1/campaigns/{campaign_id}/profiles/remove",
			"get_campaign_profiles":         "/v1/campaigns/{campaign_id}/profiles",

			// Script Management
			"upload_script": "/v1/scripts/upload",
			"list_scripts":  "/v1/scripts",
			"get_script":    "/v1/scripts/{script_id}",
			"update_script": "/v1/scripts/{script_id}/update",
			"delete_script": "/v1/scripts/{script_id}/delete",

			// Automation Control
			"execute_script":             "/v1/browser/{profile_id}/execute",
			"execute_script_on_profiles": "/v1/browser/execute-batch",
			"get_execution_status":       "/v1/executions/{execution_id}/status",
			"get_execution_logs":         "/v1/executions/{execution_id}/logs",

			// System & Monitoring
			"get_system_info":    "/v1/system/info",
			"get_system_status":  "/v1/system/status",
			"get_system_health":  "/v1/system/health",
			"get_memory_usage":   "/v1/system/memory",
			"get_cpu_usage":      "/v1/system/cpu",
			"get_disk_usage":     "/v1/system/disk",
			"get_network_status": "/v1/system/network",

			// Browser Management
			"get_browser_versions": "/v1/browser/versions",
			"update_browser":       "/v1/browser/update",
			"get_browser_status":   "/v1/browser/status",

			// Proxy Management
			"list_proxies": "/v1/proxies",
			"add_proxy":    "/v1/proxies/add",
			"update_proxy": "/v1/proxies/{proxy_id}/update",
			"delete_proxy": "/v1/proxies/{proxy_id}/delete",
			"test_proxy":   "/v1/proxies/{proxy_id}/test",

			// Extension Management
			"list_extensions":     "/v1/extensions",
			"install_extension":   "/v1/extensions/install",
			"uninstall_extension": "/v1/extensions/{extension_id}/uninstall",
			"update_extension":    "/v1/extensions/{extension_id}/update",
		},
	}
}

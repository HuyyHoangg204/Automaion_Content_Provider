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
			// Remote profile management - Based on Hidemium v4 API docs
			"open_profile":  "/openProfile?uuid={uuid}",  //Get Method
			"close_profile": "/closeProfile?uuid={uuid}", //Get Method
			"checking":      "/authorize?uuid={uuid}",

			// Profile Management - Based on Hidemium v4 API docs
			"list_profiles":       "/v1/browser/list?is_local=false",
			"list_config_default": "/v1/browser/list?is_local=false",
			"list_status":         "/v2/status-profile?is_local=false",
			"list_tag":            "/v2/tag?is_local=false",
			"list_version":        "/v2/browser/get-list-version",
			"get_profile_by_uuid": "/v2/browser/get-profile-by-uuid/{uuid}?is_local=false",
			"get_list_folder":     "/v1/folder/list?is_local=false&page={page}&limit={limit}",

			// Interaction profile - Based on Hidemium v4 API docs
			"create_profile_by_default": "/create-profile-by-default?is_local=false",
			"create_profile_customize":  "/create-profile-custom?is_local=false",
			"change_fingerprint":        "/v2/browser/change-fingerprint?is_local=false",
			"update_note":               "/v2/browser/update-note?is_local=false",
			"update_name":               "/v2/browser/update-once?is_local=false",
			"sync_tags":                 "/v2/tag?is_local=true",
			"change_status":             "/v2/status-profile/change-status?is_local=true",
			"delete_profile":            "/v1/browser/destroy?is_local=false",
			"add_profile_to_folder":     "/v1/folder/{folder_uuid}/add-browser?is_local=true",

			// Proxy management - Based on Hidemium v4 API docs
			"update_proxy":           "/v2/proxy/quick-edit?is_local=false",
			"update_profile's_proxy": "/v2/browser/proxy/update?is_local=false",

			// Campaign management - Based on Hidemium v4 API docs
			"get_campaign":                   "/automation/campaign?search=&page={page}&limit={limit}",
			"create_schedule":                "/automation/schedule",
			"get_schedule":                   "/automation/schedule?campaign_id={campaign_id}&page={page}&limit={limit}",
			"update_schedule_status":         "/automation/schedule?campaign_id={campaign_id}&page={page}&limit={limit}",
			"delete_schedule":                "/automation/delete-schedule",
			"create_campaign":                "/automation/campaign",
			"add_profile_to_campaign":        "/automation/campaign/save-campaign-profile",
			"update_campaign_input_variable": "/automation/campaign/save-auto-campaign",          //Post Method
			"delete_campaign":                "/automation/delete-campaign",                      //Post Method
			"delete_all_profile_in_campaign": "/automation/campaign/delete-all-campaign-profile", //Post Delete
			"get_user_uuid":                  "/user-settings/token",
		},
	}
}

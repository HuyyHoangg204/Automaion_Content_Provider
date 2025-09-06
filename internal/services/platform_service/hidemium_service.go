package platform_service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type HidemiumService struct{}

func NewHidemiumService() *HidemiumService {
	return &HidemiumService{}
}

func (s *HidemiumService) FetchAllProfilesWithPagination(tunnelURL string) ([]map[string]interface{}, error) {
	var allProfiles []map[string]interface{}
	page := 1
	limit := 100

	for {
		profiles, err := s.fetchProfilesFromPlatformWithPagination(tunnelURL, page, limit)
		if err != nil {
			return nil, err
		}

		if len(profiles) == 0 {
			break
		}

		allProfiles = append(allProfiles, profiles...)
		page++
	}

	return allProfiles, nil
}

func (s *HidemiumService) fetchProfilesFromPlatformWithPagination(tunnelURL string, page, limit int) ([]map[string]interface{}, error) {
	if tunnelURL == "" {
		return nil, fmt.Errorf("no tunnel URL configured")
	}

	profilesURL := s.buildProfilesURLWithPagination(tunnelURL, page, limit)

	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest("POST", profilesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Green-Controller/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request to platform: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("platform returned status %d", resp.StatusCode)
	}

	platformProfiles, err := s.parseHidemiumProfilesResponse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse hidemium response: %w", err)
	}

	return platformProfiles, nil
}

func (s *HidemiumService) buildProfilesURLWithPagination(baseURL string, page, limit int) string {
	baseURL = strings.TrimSuffix(baseURL, "/")
	return fmt.Sprintf("%s/v1/browser/list?page=%d&limit=%d", baseURL, page, limit)
}

func (s *HidemiumService) parseHidemiumProfilesResponse(body io.Reader) ([]map[string]interface{}, error) {
	var response map[string]interface{}
	if err := json.NewDecoder(body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode hidemium response: %w", err)
	}

	var profiles []map[string]interface{}

	if data, ok := response["data"].(map[string]interface{}); ok {
		if content, ok := data["content"].([]interface{}); ok {
			for _, item := range content {
				if profile, ok := item.(map[string]interface{}); ok {
					profiles = append(profiles, profile)
				}
			}
			return profiles, nil
		}
	}

	possibleFields := []string{"data", "profiles", "result", "list", "browsers"}

	for _, field := range possibleFields {
		if value, exists := response[field]; exists {
			switch v := value.(type) {
			case []interface{}:
				for _, item := range v {
					if profile, ok := item.(map[string]interface{}); ok {
						profiles = append(profiles, profile)
					}
				}
				return profiles, nil
			case map[string]interface{}:
				for _, nestedValue := range v {
					if nestedArray, ok := nestedValue.([]interface{}); ok {
						for _, item := range nestedArray {
							if profile, ok := item.(map[string]interface{}); ok {
								profiles = append(profiles, profile)
							}
						}
						return profiles, nil
					}
				}
			}
		}
	}

	return nil, nil
}

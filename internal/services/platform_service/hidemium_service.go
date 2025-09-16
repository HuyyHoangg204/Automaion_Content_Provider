package platform_service

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/onegreenvn/green-provider-services-backend/internal/models"
)

type HidemiumService struct{}

func NewHidemiumService() *HidemiumService {
	return &HidemiumService{}
}

func (s *HidemiumService) FetchAllProfilesWithPagination(tunnelURL string) ([]map[string]interface{}, error) {
	var allProfiles []map[string]interface{}
	page := 1
	limit := 100

	pageLimitStr := os.Getenv("SYNC_PAGE_LIMIT")
	pageLimit, err := strconv.Atoi(pageLimitStr)
	if err != nil || pageLimit <= 0 {
		pageLimit = 100
	}

	log.Printf("Start fetching all profiles from tunnel: %s", tunnelURL)

	for {
		if page > pageLimit {
			log.Printf("Reached page limit of %d, stopping sync.", pageLimit)
			break
		}
		log.Printf("Fetching page %d of profiles...", page)
		profiles, err := s.fetchProfilesFromPlatformWithPagination(tunnelURL, page, limit)
		if err != nil {
			log.Printf("Error fetching page %d: %v", page, err)
			return nil, err
		}

		if len(profiles) == 0 {
			log.Printf("No more profiles to fetch. Total pages fetched: %d", page-1)
			break
		}

		log.Printf("Fetched %d profiles from page %d", len(profiles), page)
		allProfiles = append(allProfiles, profiles...)
		page++
	}

	log.Printf("Successfully fetched a total of %d profiles.", len(allProfiles))
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
	var response models.HidemiumResponse
	if err := json.NewDecoder(body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode hidemium response: %w", err)
	}

	content, ok := response.Data["content"]
	if !ok {
		return nil, fmt.Errorf("content field not found in response data")
	}

	profiles, ok := content.([]interface{})
	if !ok {
		return nil, fmt.Errorf("content is not an array of profiles")
	}

	var result []map[string]interface{}
	for _, p := range profiles {
		profile, ok := p.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("profile is not a map")
		}
		result = append(result, profile)
	}

	return result, nil
}

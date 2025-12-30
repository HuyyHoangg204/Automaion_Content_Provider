package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/sirupsen/logrus"
)

type ScriptService struct {
	scriptRepo           *repository.ScriptRepository
	topicRepo            *repository.TopicRepository
	userProfileRepo      *repository.UserProfileRepository
	boxRepo              *repository.BoxRepository
	chromeProfileService *ChromeProfileService
	geminiAccountService *GeminiAccountService
	fileService          *FileService
	baseURL              string
	// In-memory cache để lưu file IDs vừa upload theo userID
	recentUploadedFiles sync.Map // map[string][]string
}

func NewScriptService(
	scriptRepo *repository.ScriptRepository,
	topicRepo *repository.TopicRepository,
	userProfileRepo *repository.UserProfileRepository,
	boxRepo *repository.BoxRepository,
	chromeProfileService *ChromeProfileService,
	geminiAccountService *GeminiAccountService,
	fileService *FileService,
) *ScriptService {
	// Get base URL from environment
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		baseURL = fmt.Sprintf("http://localhost:%s", port)
		logrus.Warnf("BASE_URL not set, using default: %s", baseURL)
	}

	return &ScriptService{
		scriptRepo:           scriptRepo,
		topicRepo:            topicRepo,
		userProfileRepo:      userProfileRepo,
		boxRepo:              boxRepo,
		chromeProfileService: chromeProfileService,
		geminiAccountService: geminiAccountService,
		fileService:          fileService,
		baseURL:              baseURL,
	}
}

// SaveScript saves or updates a script for a topic and user (upsert)
func (s *ScriptService) SaveScript(topicID, userID string, req *models.SaveScriptRequest) (*models.ScriptResponse, error) {
	// Check if topic exists
	_, err := s.topicRepo.GetByID(topicID)
	if err != nil {
		return nil, fmt.Errorf("topic not found: %w", err)
	}

	// Check if script exists (1-1 relationship)
	existingScript, err := s.scriptRepo.GetByTopicIDAndUserID(topicID, userID)
	if err != nil && err.Error() != "record not found" {
		return nil, fmt.Errorf("failed to check existing script: %w", err)
	}

	var script *models.Script
	if existingScript != nil {
		// Update existing script
		script = existingScript
		script.UpdatedAt = time.Now()

		// Update script
		if err := s.scriptRepo.Update(script); err != nil {
			return nil, fmt.Errorf("failed to update script: %w", err)
		}
	} else {
		// Create new script
		script = &models.Script{
			TopicID: topicID,
			UserID:  userID,
		}
		if err := s.scriptRepo.Create(script); err != nil {
			return nil, fmt.Errorf("failed to create script: %w", err)
		}
	}

	// Get existing projects for this script
	existingProjects, err := s.scriptRepo.GetProjectsByScriptID(script.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing projects: %w", err)
	}
	existingProjectMap := make(map[string]*models.ScriptProject) // Map project_id -> existing project
	for _, p := range existingProjects {
		existingProjectMap[p.ProjectID] = p
	}

	// Track which project_ids are in the new request
	requestProjectIDs := make(map[string]bool)

	// Upsert projects: create new or update existing
	projectsToCreate := make([]*models.ScriptProject, 0)
	projectsToUpdate := make([]*models.ScriptProject, 0)
	projectIDMap := make(map[string]*models.ScriptProject) // Map frontend project_id -> DB project

	for _, projectReq := range req.Projects {
		requestProjectIDs[projectReq.ID] = true

		// Parse created_at from ISO 8601 string
		createdAt, err := time.Parse(time.RFC3339, projectReq.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("invalid created_at format for project %s: %w", projectReq.ID, err)
		}

		existingProject, exists := existingProjectMap[projectReq.ID]
		if exists {
			// Update existing project
			// Note: Name không được update sau khi đã tạo (để tránh conflict với gem_name)
			// existingProject.Name = projectReq.Name
			existingProject.Filename = projectReq.OutputName
			// Note: gem_name không lưu trong DB, được generate từ name khi cần
			existingProject.Description = projectReq.Description
			existingProject.Instructions = projectReq.Instructions
			if projectReq.GeminiAccountID != "" {
				existingProject.GeminiAccountID = &projectReq.GeminiAccountID
			}
			existingProject.CreatedAt = createdAt
			existingProject.UpdatedAt = time.Now()
			projectsToUpdate = append(projectsToUpdate, existingProject)
			projectIDMap[projectReq.ID] = existingProject
		} else {
			// Create new project
			project := &models.ScriptProject{
				ScriptID:  script.ID,
				ProjectID: projectReq.ID, // Frontend ID
				Name:      projectReq.Name,
				Filename:  projectReq.OutputName,
				// Note: gem_name không lưu trong DB, được generate từ name khi cần
				Description:  projectReq.Description,
				Instructions: projectReq.Instructions,
				CreatedAt:    createdAt,
				CreatedAtDB:  time.Now(),
			}
			if projectReq.GeminiAccountID != "" {
				project.GeminiAccountID = &projectReq.GeminiAccountID
			}
			projectsToCreate = append(projectsToCreate, project)
			projectIDMap[projectReq.ID] = project
		}
	}

	// Create new projects
	if len(projectsToCreate) > 0 {
		if err := s.scriptRepo.CreateProjects(projectsToCreate); err != nil {
			return nil, fmt.Errorf("failed to create projects: %w", err)
		}

		// TODO: Trigger gem creation for new projects
		// Logic tạo gem đã được chuyển từ CreateTopic sang đây
		// Mỗi project mới (có gem_name) nên trigger tạo gem trên automation backend
		// Cần thêm dependencies: ChromeProfileService, GeminiAccountService, BoxRepository, FileService, UserProfileRepository
		// Xem logic cũ trong TopicService.triggerGemCreation để tham khảo
	}

	// Update existing projects
	for _, project := range projectsToUpdate {
		if err := s.scriptRepo.UpdateProject(project); err != nil {
			return nil, fmt.Errorf("failed to update project %s: %w", project.ProjectID, err)
		}
	}

	// Delete projects that are not in the new request
	projectsToDelete := make([]string, 0)
	for projectID := range existingProjectMap {
		if !requestProjectIDs[projectID] {
			projectsToDelete = append(projectsToDelete, projectID)
		}
	}
	if len(projectsToDelete) > 0 {
		if err := s.scriptRepo.DeleteProjectsByScriptIDAndProjectIDs(script.ID, projectsToDelete); err != nil {
			return nil, fmt.Errorf("failed to delete old projects: %w", err)
		}
	}

	// Upsert prompts: update existing or create new
	// Track which prompt IDs are in the request to delete prompts that are no longer present
	requestPromptIDs := make(map[string]bool)
	promptsToCreate := make([]*models.ScriptPrompt, 0)
	promptsToUpdate := make([]*models.ScriptPrompt, 0)
	allExistingPrompts := make([]*models.ScriptPrompt, 0) // Collect all existing prompts for deletion check

	for _, projectReq := range req.Projects {
		project := projectIDMap[projectReq.ID]
		if project == nil {
			continue
		}

		// Get existing prompts for this project
		existingPrompts, err := s.scriptRepo.GetPromptsByScriptIDAndProjectID(script.ID, project.ProjectID)
		if err != nil && err.Error() != "record not found" {
			return nil, fmt.Errorf("failed to get existing prompts: %w", err)
		}
		allExistingPrompts = append(allExistingPrompts, existingPrompts...)
		existingPromptMap := make(map[string]*models.ScriptPrompt) // Map ID -> prompt
		for _, p := range existingPrompts {
			existingPromptMap[p.ID] = p
		}

		for order, promptReq := range projectReq.Prompts {
			// Nếu có PromptID, lấy files từ cache; nếu không, dùng InputFiles từ request
			inputFiles := promptReq.InputFiles
			if promptReq.PromptID != "" {
				// Lấy files từ cache dựa trên prompt_id (KHÔNG xóa để user vẫn có thể GET files sau đó)
				fileIDs := s.GetUploadedFilesForPrompt(userID, project.ProjectID, promptReq.PromptID)
				if len(fileIDs) > 0 {
					// Convert file IDs thành file names (original_name) để gửi cho automation backend
					fileNames := make([]string, 0, len(fileIDs))
					for _, fileID := range fileIDs {
						file, err := s.fileService.GetFile(fileID, userID)
						if err != nil {
							logrus.Warnf("Failed to get file %s for prompt: %v", fileID, err)
							continue
						}
						fileNames = append(fileNames, file.OriginalName)
					}
					inputFiles = fileNames
				}
			}

			if promptReq.ID != "" {
				// Update existing prompt
				existingPrompt, exists := existingPromptMap[promptReq.ID]
				if exists {
					existingPrompt.PromptText = promptReq.Text
					existingPrompt.Filename = promptReq.Filename
					existingPrompt.InputFiles = inputFiles
					existingPrompt.Exit = promptReq.Exit
					existingPrompt.Merge = promptReq.Merge
					existingPrompt.PromptOrder = order
					existingPrompt.TempPromptID = promptReq.PromptID // Update temp_prompt_id nếu có
					existingPrompt.UpdatedAt = time.Now()
					promptsToUpdate = append(promptsToUpdate, existingPrompt)
					requestPromptIDs[promptReq.ID] = true
				} else {
					// ID provided but not found - treat as new prompt
					newPrompt := &models.ScriptPrompt{
						ID:           promptReq.ID, // Use provided ID
						ScriptID:     script.ID,
						ProjectID:    project.ProjectID,
						TempPromptID: promptReq.PromptID,
						PromptText:   promptReq.Text,
						Filename:     promptReq.Filename,
						InputFiles:   inputFiles,
						Exit:         promptReq.Exit,
						Merge:        promptReq.Merge,
						PromptOrder:  order,
					}
					promptsToCreate = append(promptsToCreate, newPrompt)
					requestPromptIDs[promptReq.ID] = true
				}
			} else {
				// Create new prompt (no ID provided)
				newPrompt := &models.ScriptPrompt{
					ScriptID:     script.ID,
					ProjectID:    project.ProjectID,
					TempPromptID: promptReq.PromptID,
					PromptText:   promptReq.Text,
					Filename:     promptReq.Filename,
					InputFiles:   inputFiles,
					Exit:         promptReq.Exit,
					Merge:        promptReq.Merge,
					PromptOrder:  order,
				}
				promptsToCreate = append(promptsToCreate, newPrompt)
			}
		}
	}

	// Delete prompts that are no longer in the request
	promptsToDelete := make([]string, 0)
	for _, existingPrompt := range allExistingPrompts {
		if !requestPromptIDs[existingPrompt.ID] {
			promptsToDelete = append(promptsToDelete, existingPrompt.ID)
		}
	}
	if len(promptsToDelete) > 0 {
		if err := s.scriptRepo.DeletePromptsByIDs(promptsToDelete); err != nil {
			logrus.Warnf("Failed to delete prompts: %v", err)
		}
	}

	// Create new prompts
	if len(promptsToCreate) > 0 {
		if err := s.scriptRepo.CreatePrompts(promptsToCreate); err != nil {
			return nil, fmt.Errorf("failed to create prompts: %w", err)
		}
	}

	// Update existing prompts
	for _, prompt := range promptsToUpdate {
		if err := s.scriptRepo.UpdatePrompt(prompt); err != nil {
			return nil, fmt.Errorf("failed to update prompt %s: %w", prompt.ID, err)
		}
	}

	// Get existing edges for this script
	existingEdges, err := s.scriptRepo.GetEdgesByScriptID(script.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing edges: %w", err)
	}
	existingEdgeMap := make(map[string]*models.ScriptEdge) // Map edge_id -> existing edge
	for _, e := range existingEdges {
		existingEdgeMap[e.EdgeID] = e
	}

	// Track which edge_ids are in the new request
	requestEdgeIDs := make(map[string]bool)

	// Upsert edges: create new or update existing
	edgesToCreate := make([]*models.ScriptEdge, 0)
	edgesToUpdate := make([]*models.ScriptEdge, 0)

	for _, edgeReq := range req.Edges {
		requestEdgeIDs[edgeReq.ID] = true

		// Validate source and target projects exist
		_, sourceExists := projectIDMap[edgeReq.Source]
		_, targetExists := projectIDMap[edgeReq.Target]

		if !sourceExists || !targetExists {
			return nil, fmt.Errorf("edge references non-existent project: source=%s, target=%s", edgeReq.Source, edgeReq.Target)
		}

		existingEdge, exists := existingEdgeMap[edgeReq.ID]
		if exists {
			// Update existing edge
			existingEdge.SourceProjectID = edgeReq.Source
			existingEdge.TargetProjectID = edgeReq.Target
			existingEdge.SourceName = edgeReq.SourceName
			existingEdge.TargetName = edgeReq.TargetName
			edgesToUpdate = append(edgesToUpdate, existingEdge)
		} else {
			// Create new edge
			edge := &models.ScriptEdge{
				ScriptID:        script.ID,
				EdgeID:          edgeReq.ID,     // Frontend ID
				SourceProjectID: edgeReq.Source, // Frontend project_id
				TargetProjectID: edgeReq.Target, // Frontend project_id
				SourceName:      edgeReq.SourceName,
				TargetName:      edgeReq.TargetName,
			}
			edgesToCreate = append(edgesToCreate, edge)
		}
	}

	// Create new edges
	if len(edgesToCreate) > 0 {
		if err := s.scriptRepo.CreateEdges(edgesToCreate); err != nil {
			return nil, fmt.Errorf("failed to create edges: %w", err)
		}
	}

	// Update existing edges
	for _, edge := range edgesToUpdate {
		if err := s.scriptRepo.UpdateEdge(edge); err != nil {
			return nil, fmt.Errorf("failed to update edge %s: %w", edge.EdgeID, err)
		}
	}

	// Delete edges that are not in the new request
	edgesToDelete := make([]string, 0)
	for edgeID := range existingEdgeMap {
		if !requestEdgeIDs[edgeID] {
			edgesToDelete = append(edgesToDelete, edgeID)
		}
	}
	if len(edgesToDelete) > 0 {
		if err := s.scriptRepo.DeleteEdgesByScriptIDAndEdgeIDs(script.ID, edgesToDelete); err != nil {
			return nil, fmt.Errorf("failed to delete old edges: %w", err)
		}
	}

	// Note: KHÔNG clear project_id và temp_prompt_id trong bảng files vì:
	// 1. Files đã được map với prompt qua prompt.InputFiles
	// 2. Cần giữ lại để GetUploadedFilesForPrompt có thể fallback từ DB khi cache rỗng
	// 3. Có thể cần để support việc edit/update prompt sau này
	// 4. Không gây conflict vì mỗi file chỉ map với 1 prompt tại 1 thời điểm

	// Reload script with all relationships
	savedScript, err := s.scriptRepo.GetByTopicIDAndUserID(topicID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to reload script: %w", err)
	}

	return s.toScriptResponse(savedScript), nil
}

// GetScript gets a script by topic_id and user_id
func (s *ScriptService) GetScript(topicID, userID string) (*models.ScriptResponse, error) {
	script, err := s.scriptRepo.GetByTopicIDAndUserID(topicID, userID)
	if err != nil {
		return nil, fmt.Errorf("script not found: %w", err)
	}

	return s.toScriptResponse(script), nil
}

// DeleteScript deletes a script by topic_id and user_id
func (s *ScriptService) DeleteScript(topicID, userID string) error {
	return s.scriptRepo.DeleteByTopicIDAndUserID(topicID, userID)
}

// CloneScript clones a script from source user to target user for a specific topic
func (s *ScriptService) CloneScript(sourceUserID, targetUserID, topicID string) error {
	// 1. Check if source script exists
	sourceScript, err := s.scriptRepo.GetByTopicIDAndUserID(topicID, sourceUserID)
	if err != nil {
		// If source script not found, nothing to clone, just return nil
		if err.Error() == "record not found" {
			return nil
		}
		return fmt.Errorf("failed to get source script: %w", err)
	}

	// 2. Check if target script already exists
	// If exists, we might want to overwrite or skip. Current requirement implies "initial assignment", so let's overwrite/ensure it exists.
	targetScript, err := s.scriptRepo.GetByTopicIDAndUserID(topicID, targetUserID)
	if err != nil && err.Error() != "record not found" {
		return fmt.Errorf("failed to check target script: %w", err)
	}

	if targetScript == nil {
		// Create new script for target user
		targetScript = &models.Script{
			TopicID: topicID,
			UserID:  targetUserID,
		}
		if err := s.scriptRepo.Create(targetScript); err != nil {
			return fmt.Errorf("failed to create target script: %w", err)
		}
	} else {
		// Start fresh: Delete existing projects/edges/prompts for target script
		// Note: Cascading delete should handle child records if we delete projects/edges, but let's be explicit and clear projects first.
		// Actually, simpler way: GetProjects and DeleteProjects. Or dependent on repository methods.
		// Assuming cascading delete is configured in GORM models (constraint:OnDelete:CASCADE),
		// but GORM soft delete or manual cleanup might be safer.
		// Let's rely on repository methods to clear current data to avoid mixing.
		// For now, let's assume we proceed to upsert/copy over. But existing data might conflict or dupe.
		// Ideally: Clear old data for target user's script to match source exactly.

		// Delete all projects for target script
		projects, _ := s.scriptRepo.GetProjectsByScriptID(targetScript.ID)
		projectIDs := make([]string, len(projects))
		for i, p := range projects {
			projectIDs[i] = p.ProjectID
		}
		if len(projectIDs) > 0 {
			if err := s.scriptRepo.DeleteProjectsByScriptIDAndProjectIDs(targetScript.ID, projectIDs); err != nil {
				return fmt.Errorf("failed to clear existing projects: %w", err)
			}
		}

		// Delete all edges for target script
		edges, _ := s.scriptRepo.GetEdgesByScriptID(targetScript.ID)
		edgeIDs := make([]string, len(edges))
		for i, e := range edges {
			edgeIDs[i] = e.EdgeID
		}
		if len(edgeIDs) > 0 {
			if err := s.scriptRepo.DeleteEdgesByScriptIDAndEdgeIDs(targetScript.ID, edgeIDs); err != nil {
				return fmt.Errorf("failed to clear existing edges: %w", err)
			}
		}
	}

	// 3. Clone Projects
	projectsToCreate := make([]*models.ScriptProject, 0, len(sourceScript.Projects))
	promptsToCreate := make([]*models.ScriptPrompt, 0)

	for _, p := range sourceScript.Projects {
		newProject := &models.ScriptProject{
			ScriptID:  targetScript.ID,
			ProjectID: p.ProjectID, // Keep same frontend ID to maintain Edge relationships
			Name:      p.Name,
			Filename:  p.Filename,
			// Note: gem_name không lưu trong DB, được generate từ name khi cần
			Description:     p.Description,
			Instructions:    p.Instructions,
			GeminiAccountID: p.GeminiAccountID,
			CreatedAt:       p.CreatedAt,
			CreatedAtDB:     time.Now(),
		}
		projectsToCreate = append(projectsToCreate, newProject)

		// Prepare prompts for this project
		for _, pr := range p.Prompts {
			newPrompt := &models.ScriptPrompt{
				ScriptID:    targetScript.ID,
				ProjectID:   p.ProjectID,
				PromptText:  pr.PromptText,
				Filename:    pr.Filename,
				InputFiles:  pr.InputFiles,
				Exit:        pr.Exit,
				Merge:       pr.Merge, // Copy Merge field
				PromptOrder: pr.PromptOrder,
			}
			promptsToCreate = append(promptsToCreate, newPrompt)
		}
	}

	if len(projectsToCreate) > 0 {
		if err := s.scriptRepo.CreateProjects(projectsToCreate); err != nil {
			return fmt.Errorf("failed to clone projects: %w", err)
		}
	}

	// 4. Clone Prompts
	if len(promptsToCreate) > 0 {
		if err := s.scriptRepo.CreatePrompts(promptsToCreate); err != nil {
			return fmt.Errorf("failed to clone prompts: %w", err)
		}
	}

	// 5. Clone Edges
	edgesToCreate := make([]*models.ScriptEdge, 0, len(sourceScript.Edges))
	for _, e := range sourceScript.Edges {
		newEdge := &models.ScriptEdge{
			ScriptID:        targetScript.ID,
			EdgeID:          e.EdgeID, // Keep same frontend ID
			SourceProjectID: e.SourceProjectID,
			TargetProjectID: e.TargetProjectID,
			SourceName:      e.SourceName,
			TargetName:      e.TargetName,
		}
		edgesToCreate = append(edgesToCreate, newEdge)
	}

	if len(edgesToCreate) > 0 {
		if err := s.scriptRepo.CreateEdges(edgesToCreate); err != nil {
			return fmt.Errorf("failed to clone edges: %w", err)
		}
	}

	return nil
}

// toScriptResponse converts Script model to ScriptResponse
func (s *ScriptService) toScriptResponse(script *models.Script) *models.ScriptResponse {
	projects := make([]models.ScriptProjectResponse, 0, len(script.Projects))
	for _, project := range script.Projects {
		prompts := make([]models.ScriptPromptResponse, 0, len(project.Prompts))
		for _, prompt := range project.Prompts {
			prompts = append(prompts, models.ScriptPromptResponse{
				ID:          prompt.ID,
				Text:        prompt.PromptText,
				Filename:    prompt.Filename,
				InputFiles:  prompt.InputFiles,
				Exit:        prompt.Exit,
				Merge:       prompt.Merge, // Map Merge to response
				PromptOrder: prompt.PromptOrder,
			})
		}

		resp := models.ScriptProjectResponse{
			ProjectID: project.ProjectID, // Chỉ trả project_id (frontend ID), không cần DB UUID
			Name:      project.Name,
			Filename:  project.Filename,
			// Note: gem_name không trả về, được generate từ name khi cần
			Description:  project.Description,
			Instructions: project.Instructions,
			CreatedAt:    project.CreatedAt.Format(time.RFC3339),
			Prompts:      prompts,
		}
		if project.GeminiAccountID != nil {
			resp.GeminiAccountID = project.GeminiAccountID
		}
		projects = append(projects, resp)
	}

	edges := make([]models.ScriptEdgeResponse, 0, len(script.Edges))
	for _, edge := range script.Edges {
		edges = append(edges, models.ScriptEdgeResponse{
			ID:         edge.ID,
			EdgeID:     edge.EdgeID,
			Source:     edge.SourceProjectID,
			Target:     edge.TargetProjectID,
			SourceName: edge.SourceName,
			TargetName: edge.TargetName,
		})
	}

	return &models.ScriptResponse{
		ID:        script.ID,
		TopicID:   script.TopicID,
		UserID:    script.UserID,
		Projects:  projects,
		Edges:     edges,
		CreatedAt: script.CreatedAt.Format(time.RFC3339),
		UpdatedAt: script.UpdatedAt.Format(time.RFC3339),
	}
}

// CreateProject creates a new project and triggers gem creation on automation backend
func (s *ScriptService) CreateProject(topicID, userID string, req *models.CreateProjectRequest) (*models.CreateProjectResponse, error) {
	// Check if topic exists
	_, err := s.topicRepo.GetByID(topicID)
	if err != nil {
		return nil, fmt.Errorf("topic not found: %w", err)
	}

	// Get user profile
	userProfile, err := s.userProfileRepo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("user profile not found: %w", err)
	}

	// Get or create script (1-1 relationship)
	script, err := s.scriptRepo.GetByTopicIDAndUserID(topicID, userID)
	if err != nil && err.Error() != "record not found" {
		return nil, fmt.Errorf("failed to check existing script: %w", err)
	}

	if script == nil {
		// Create new script
		script = &models.Script{
			TopicID: topicID,
			UserID:  userID,
		}
		if err := s.scriptRepo.Create(script); err != nil {
			return nil, fmt.Errorf("failed to create script: %w", err)
		}
	}

	// Generate project ID (frontend ID - timestamp)
	projectID := fmt.Sprintf("%d", time.Now().UnixMilli())

	// Generate gem name từ projectID và name để đảm bảo unique cho mỗi project
	// Format: {projectID}_{name}
	gemName := fmt.Sprintf("%s_%s", projectID, req.Name)

	// Create project in database
	project := &models.ScriptProject{
		ScriptID:  script.ID,
		ProjectID: projectID,
		Name:      req.Name,
		// Note: gem_name không lưu trong DB, được generate từ name khi cần
		Description:  req.Description,
		Instructions: req.Instructions,
		CreatedAt:    time.Now(),
		CreatedAtDB:  time.Now(),
	}
	// Note: gemini_account_id sẽ được tự động set khi system chọn account trong background

	if err := s.scriptRepo.CreateProjects([]*models.ScriptProject{project}); err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	// Trigger gem creation on automation backend (in background)
	go func() {
		// Step 1: Launch Chrome with lock
		// Note: Dùng topicID làm EntityID để ProcessLogService có thể xử lý logs (giống logic cũ)
		// projectID có thể thêm vào metadata nếu cần
		// Debug port được generate theo user để tránh conflict và để server quản lý
		debugPort := s.generateDebugPort(userID)
		launchReq := &LaunchChromeProfileRequest{
			UserProfileID: userProfile.ID,
			EnsureGmail:   true,
			EntityType:    "topic", // Dùng "topic" để ProcessLogService.handleTopicLog có thể xử lý
			EntityID:      topicID, // Dùng topicID (UUID) thay vì script.ID
			DebugPort:     debugPort,
		}

		launchResp, err := s.chromeProfileService.LaunchChromeProfile(userID, launchReq)
		if err != nil {
			logrus.Errorf("Failed to launch Chrome for project %s: %v", projectID, err)
			// Xóa project khi automation fail
			if deleteErr := s.scriptRepo.DeleteProjectsByScriptIDAndProjectIDs(script.ID, []string{projectID}); deleteErr != nil {
				logrus.Errorf("Failed to delete project %s after Chrome launch failure: %v", projectID, deleteErr)
			} else {
				logrus.Infof("Deleted project %s due to Chrome launch failure", projectID)
			}
			return
		}

		// Step 1.5: Get Gemini account for this machine (if available)
		var geminiAccount *models.GeminiAccount
		if s.geminiAccountService != nil && launchResp.MachineID != "" {
			// Get Box to get machine_id (string) from BoxID (UUID)
			box, err := s.boxRepo.GetByID(launchResp.MachineID)
			if err == nil && box != nil {
				// Get available Gemini account for this machine
				account, err := s.geminiAccountService.GetAvailableAccountForMachine(box.MachineID)
				if err == nil && account != nil {
					geminiAccount = account
					// Update project with account ID
					project.GeminiAccountID = &account.ID
					if updateErr := s.scriptRepo.UpdateProject(project); updateErr != nil {
						logrus.Warnf("Failed to update project with Gemini account: %v", updateErr)
					}
					logrus.Infof("Using Gemini account %s (email: %s) for project %s on machine %s", account.ID, account.Email, projectID, box.MachineID)
				} else {
					logrus.Warnf("No available Gemini account found for machine %s, project will be created without account association", box.MachineID)
				}
			}
		}

		// Step 2: Trigger Gem creation on Gemini (fire-and-forget)
		err = s.triggerGemCreationForProject(userProfile, req, gemName, launchResp.TunnelURL, userID, geminiAccount)
		if err != nil {
			// Chỉ xóa project nếu không gửi được request (network error, không phải timeout)
			if isNetworkError(err) {
				logrus.Errorf("Failed to trigger Gem creation for project %s (network error): %v", projectID, err)
				// Release lock on error
				s.chromeProfileService.ReleaseChromeProfile(userID, &ReleaseChromeProfileRequest{
					UserProfileID: userProfile.ID,
				})
				// Xóa project khi không gửi được request
				if deleteErr := s.scriptRepo.DeleteProjectsByScriptIDAndProjectIDs(script.ID, []string{projectID}); deleteErr != nil {
					logrus.Errorf("Failed to delete project %s after trigger failure: %v", projectID, deleteErr)
				} else {
					logrus.Infof("Deleted project %s due to trigger failure (network error)", projectID)
				}
				return
			}
			// Timeout hoặc lỗi khác → automation backend có thể vẫn đang chạy
			logrus.Warnf("Gem creation trigger returned error for project %s (may be timeout, automation backend still running): %v", projectID, err)
		}

		// Step 3: Release lock
		if err := s.chromeProfileService.ReleaseChromeProfile(userID, &ReleaseChromeProfileRequest{
			UserProfileID: userProfile.ID,
		}); err != nil {
			logrus.Warnf("Failed to release lock for project %s: %v", projectID, err)
		}
	}()

	// Return response immediately
	response := &models.CreateProjectResponse{
		ProjectID: projectID,
		Name:      project.Name,
		// Note: gem_name không trả về, được generate từ name khi cần
		CreatedAt: project.CreatedAt.Format(time.RFC3339),
	}
	if project.Description != "" {
		response.Description = project.Description
	}
	if project.Instructions != "" {
		response.Instructions = project.Instructions
	}
	// Note: gemini_account_id không trả về trong response, được quản lý nội bộ

	return response, nil
}

// triggerGemCreationForProject triggers Gem creation on automation backend for a project
func (s *ScriptService) triggerGemCreationForProject(userProfile *models.UserProfile, req *models.CreateProjectRequest, gemName string, tunnelURL string, userID string, geminiAccount *models.GeminiAccount) error {
	// Build API URL: POST /gemini/gems
	apiURL := fmt.Sprintf("%s/gemini/gems", strings.TrimSuffix(tunnelURL, "/"))

	// Tự động lấy files mới nhất của user từ cache (files vừa upload)
	// User upload files trước, sau đó tạo project → files tự động được gửi lên automation backend
	knowledgeFiles := s.getRecentUserFilesAsURLs(userID)

	// Đảm bảo knowledgeFiles luôn là array (không phải null)
	if knowledgeFiles == nil {
		knowledgeFiles = []string{}
	}

	// Prepare request body
	requestBody := map[string]interface{}{
		"name":           req.Name,
		"profileDirName": userProfile.ProfileDirName,
		"gemName":        gemName,
		"description":    req.Description,
		"instructions":   req.Instructions,
		"knowledgeFiles": knowledgeFiles,
	}

	if geminiAccount != nil {
		logrus.Infof("Using Gemini account %s (email: %s) for project on machine", geminiAccount.ID, geminiAccount.Email)
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "Green-Provider-Services/1.0")

	// Make request with short timeout (chỉ để trigger, không đợi response)
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to trigger Gem creation: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		logrus.Warnf("Automation backend returned status %d for Gem creation trigger: %s", resp.StatusCode, string(bodyBytes))
	}

	logrus.Infof("Gem creation triggered for project, waiting for logs from automation backend")
	return nil
}

// Helper functions (copied from TopicService)

// AddUploadedFiles lưu file IDs vừa upload vào cache
func (s *ScriptService) AddUploadedFiles(userID string, fileIDs []string) {
	if len(fileIDs) == 0 {
		return
	}

	existing, _ := s.recentUploadedFiles.LoadOrStore(userID, []string{})
	existingIDs := existing.([]string)

	existingMap := make(map[string]bool)
	for _, id := range existingIDs {
		existingMap[id] = true
	}

	newIDs := make([]string, 0, len(existingIDs)+len(fileIDs))
	newIDs = append(newIDs, existingIDs...)

	for _, id := range fileIDs {
		if !existingMap[id] {
			newIDs = append(newIDs, id)
		}
	}

	s.recentUploadedFiles.Store(userID, newIDs)
}

// GetAndClearUploadedFiles lấy file IDs từ cache và xóa sau khi lấy
func (s *ScriptService) GetAndClearUploadedFiles(userID string) []string {
	value, ok := s.recentUploadedFiles.LoadAndDelete(userID)
	if !ok {
		return []string{}
	}

	fileIDs := value.([]string)
	return fileIDs
}

// AddUploadedFilesForPrompt lưu file IDs vừa upload vào cache và DB cho prompt cụ thể
// Nếu đã có files trong cache, sẽ append thêm (không replace)
func (s *ScriptService) AddUploadedFilesForPrompt(userID, projectID, promptID string, fileIDs []string) {
	if len(fileIDs) == 0 {
		return
	}

	// 1. Lưu vào cache (in-memory)
	cacheKey := fmt.Sprintf("%s_%s_%s", userID, projectID, promptID)

	existing, _ := s.recentUploadedFiles.LoadOrStore(cacheKey, []string{})
	existingIDs := existing.([]string)

	existingMap := make(map[string]bool)
	for _, id := range existingIDs {
		existingMap[id] = true
	}

	newIDs := make([]string, 0, len(existingIDs)+len(fileIDs))
	newIDs = append(newIDs, existingIDs...)

	for _, id := range fileIDs {
		if !existingMap[id] {
			newIDs = append(newIDs, id)
		}
	}

	s.recentUploadedFiles.Store(cacheKey, newIDs)

	// 2. Update files trong DB với project_id và temp_prompt_id (chỉ update những file mới)
	// Note: Files đã được lưu với project_id và temp_prompt_id khi upload, không cần update lại
}

// GetUploadedFilesForPrompt lấy file IDs từ cache cho prompt (không xóa) - để user xem danh sách
// Nếu cache rỗng, sẽ fallback lấy từ DB (bảng files với project_id và temp_prompt_id)
func (s *ScriptService) GetUploadedFilesForPrompt(userID, projectID, promptID string) []string {
	cacheKey := fmt.Sprintf("%s_%s_%s", userID, projectID, promptID)

	value, ok := s.recentUploadedFiles.Load(cacheKey)
	if ok {
		fileIDs := value.([]string)
		return fileIDs
	}

	// Cache rỗng → Fallback: Lấy từ DB (bảng files)
	if s.fileService != nil {
		files, err := s.fileService.GetFilesByProjectAndPrompt(userID, projectID, promptID)
		if err != nil {
			logrus.Warnf("Failed to get prompt files from DB: %v", err)
			return []string{}
		}

		fileIDs := make([]string, 0, len(files))
		for _, file := range files {
			fileIDs = append(fileIDs, file.ID)
		}

		// Nếu tìm thấy trong DB, restore vào cache để lần sau nhanh hơn
		if len(fileIDs) > 0 {
			s.recentUploadedFiles.Store(cacheKey, fileIDs)
		}

		return fileIDs
	}

	return []string{}
}

// GetAndClearUploadedFilesForPrompt lấy file IDs từ cache cho prompt và xóa sau khi lấy (dùng khi save script)
func (s *ScriptService) GetAndClearUploadedFilesForPrompt(userID, projectID, promptID string) []string {
	cacheKey := fmt.Sprintf("%s_%s_%s", userID, projectID, promptID)

	value, ok := s.recentUploadedFiles.LoadAndDelete(cacheKey)
	if !ok {
		return []string{}
	}

	fileIDs := value.([]string)
	return fileIDs
}

// getRecentUserFilesAsURLs lấy files vừa upload từ cache và convert thành download URLs
func (s *ScriptService) getRecentUserFilesAsURLs(userID string) []string {
	fileIDs := s.GetAndClearUploadedFiles(userID)

	if len(fileIDs) == 0 {
		return []string{}
	}

	urls := make([]string, 0, len(fileIDs))
	for _, fileID := range fileIDs {
		downloadURL := fmt.Sprintf("%s/api/v1/files/%s/download", strings.TrimSuffix(s.baseURL, "/"), fileID)
		urls = append(urls, downloadURL)
	}
	return urls
}

// Note: isNetworkError and normalizeUsername are already defined in topic_service.go
// and can be used here since they're in the same package

// generateDebugPort generates a deterministic debug port per user (same logic as ScriptExecutionService)
func (s *ScriptService) generateDebugPort(userID string) int {
	const basePort = 9222
	const portRange = 778 // 9222 - 9999

	h := fnv.New32a()
	h.Write([]byte(userID))
	offset := int(h.Sum32() % uint32(portRange))

	return basePort + offset
}

package services

import (
	"fmt"
	"time"

	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
)

type ScriptService struct {
	scriptRepo *repository.ScriptRepository
	topicRepo  *repository.TopicRepository
}

func NewScriptService(scriptRepo *repository.ScriptRepository, topicRepo *repository.TopicRepository) *ScriptService {
	return &ScriptService{
		scriptRepo: scriptRepo,
		topicRepo:  topicRepo,
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
			existingProject.Name = projectReq.Name
			existingProject.CreatedAt = createdAt
			existingProject.UpdatedAt = time.Now()
			projectsToUpdate = append(projectsToUpdate, existingProject)
			projectIDMap[projectReq.ID] = existingProject
		} else {
			// Create new project
			project := &models.ScriptProject{
				ScriptID:    script.ID,
				ProjectID:   projectReq.ID, // Frontend ID
				Name:        projectReq.Name,
				CreatedAt:   createdAt,
				CreatedAtDB: time.Now(),
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

	// Delete and recreate prompts for all projects (simpler than tracking individual prompts)
	// Get all project_ids that need prompts updated (both new and updated projects)
	projectIDsForPrompts := make([]string, 0, len(req.Projects))
	for _, projectReq := range req.Projects {
		projectIDsForPrompts = append(projectIDsForPrompts, projectReq.ID)
	}
	if len(projectIDsForPrompts) > 0 {
		// Delete old prompts for these projects
		if err := s.scriptRepo.DeletePromptsByScriptIDAndProjectIDs(script.ID, projectIDsForPrompts); err != nil {
			return nil, fmt.Errorf("failed to delete old prompts: %w", err)
		}
	}

	// Create prompts for each project
	prompts := make([]*models.ScriptPrompt, 0)
	for _, projectReq := range req.Projects {
		project := projectIDMap[projectReq.ID]
		if project == nil {
			continue
		}

		for order, promptReq := range projectReq.Prompts {
			prompt := &models.ScriptPrompt{
				ScriptID:    script.ID,         // Cần để reference đến ScriptProject composite key
				ProjectID:   project.ProjectID, // Frontend project_id (varchar), part of composite key
				PromptText:  promptReq.Text,
				Exit:        promptReq.Exit,
				PromptOrder: order,
			}
			prompts = append(prompts, prompt)
		}
	}

	// Create prompts in batch
	if len(prompts) > 0 {
		if err := s.scriptRepo.CreatePrompts(prompts); err != nil {
			return nil, fmt.Errorf("failed to create prompts: %w", err)
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

// toScriptResponse converts Script model to ScriptResponse
func (s *ScriptService) toScriptResponse(script *models.Script) *models.ScriptResponse {
	projects := make([]models.ScriptProjectResponse, 0, len(script.Projects))
	for _, project := range script.Projects {
		prompts := make([]models.ScriptPromptResponse, 0, len(project.Prompts))
		for _, prompt := range project.Prompts {
			prompts = append(prompts, models.ScriptPromptResponse{
				ID:          prompt.ID,
				Text:        prompt.PromptText,
				Exit:        prompt.Exit,
				PromptOrder: prompt.PromptOrder,
			})
		}

		projects = append(projects, models.ScriptProjectResponse{
			ProjectID: project.ProjectID, // Chỉ trả project_id (frontend ID), không cần DB UUID
			Name:      project.Name,
			CreatedAt: project.CreatedAt.Format(time.RFC3339),
			Prompts:   prompts,
		})
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

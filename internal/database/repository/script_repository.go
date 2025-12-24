package repository

import (
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"gorm.io/gorm"
)

type ScriptRepository struct {
	db *gorm.DB
}

func NewScriptRepository(db *gorm.DB) *ScriptRepository {
	return &ScriptRepository{db: db}
}

// GetByTopicIDAndUserID gets script by topic_id and user_id (1-1 relationship)
func (r *ScriptRepository) GetByTopicIDAndUserID(topicID, userID string) (*models.Script, error) {
	var script models.Script
	err := r.db.Where("topic_id = ? AND user_id = ?", topicID, userID).
		Preload("Projects.Prompts").
		Preload("Edges").
		First(&script).Error
	if err != nil {
		return nil, err
	}
	return &script, nil
}

// Create creates a new script
func (r *ScriptRepository) Create(script *models.Script) error {
	return r.db.Create(script).Error
}

// Update updates an existing script
func (r *ScriptRepository) Update(script *models.Script) error {
	return r.db.Save(script).Error
}

// Delete deletes a script (cascade deletes projects, prompts, edges)
func (r *ScriptRepository) Delete(scriptID string) error {
	return r.db.Delete(&models.Script{}, "id = ?", scriptID).Error
}

// DeleteByTopicIDAndUserID deletes script by topic_id and user_id
func (r *ScriptRepository) DeleteByTopicIDAndUserID(topicID, userID string) error {
	return r.db.Where("topic_id = ? AND user_id = ?", topicID, userID).
		Delete(&models.Script{}).Error
}

// CreateProject creates a script project
func (r *ScriptRepository) CreateProject(project *models.ScriptProject) error {
	return r.db.Create(project).Error
}

// CreateProjects creates multiple script projects in batch
func (r *ScriptRepository) CreateProjects(projects []*models.ScriptProject) error {
	if len(projects) == 0 {
		return nil
	}
	return r.db.CreateInBatches(projects, 100).Error
}

// CreatePrompt creates a script prompt
func (r *ScriptRepository) CreatePrompt(prompt *models.ScriptPrompt) error {
	return r.db.Create(prompt).Error
}

// CreatePrompts creates multiple script prompts in batch
func (r *ScriptRepository) CreatePrompts(prompts []*models.ScriptPrompt) error {
	if len(prompts) == 0 {
		return nil
	}
	return r.db.CreateInBatches(prompts, 100).Error
}

// CreateEdge creates a script edge
func (r *ScriptRepository) CreateEdge(edge *models.ScriptEdge) error {
	return r.db.Create(edge).Error
}

// CreateEdges creates multiple script edges in batch
func (r *ScriptRepository) CreateEdges(edges []*models.ScriptEdge) error {
	if len(edges) == 0 {
		return nil
	}
	return r.db.CreateInBatches(edges, 100).Error
}

// DeleteProjectsByScriptID deletes all projects for a script (cascade deletes prompts)
func (r *ScriptRepository) DeleteProjectsByScriptID(scriptID string) error {
	return r.db.Where("script_id = ?", scriptID).Delete(&models.ScriptProject{}).Error
}

// DeleteEdgesByScriptID deletes all edges for a script
func (r *ScriptRepository) DeleteEdgesByScriptID(scriptID string) error {
	return r.db.Where("script_id = ?", scriptID).Delete(&models.ScriptEdge{}).Error
}

// GetProjectsByScriptID gets all projects for a script
func (r *ScriptRepository) GetProjectsByScriptID(scriptID string) ([]*models.ScriptProject, error) {
	var projects []*models.ScriptProject
	err := r.db.Where("script_id = ?", scriptID).Find(&projects).Error
	if err != nil {
		return nil, err
	}
	return projects, nil
}

// GetProjectByScriptIDAndProjectID gets a project by script_id and project_id (frontend ID)
func (r *ScriptRepository) GetProjectByScriptIDAndProjectID(scriptID, projectID string) (*models.ScriptProject, error) {
	var project models.ScriptProject
	err := r.db.Where("script_id = ? AND project_id = ?", scriptID, projectID).
		First(&project).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

// UpdateProject updates a script project
func (r *ScriptRepository) UpdateProject(project *models.ScriptProject) error {
	return r.db.Save(project).Error
}

// DeleteProjectsByScriptIDAndProjectIDs deletes specific projects by script_id and list of project_ids
func (r *ScriptRepository) DeleteProjectsByScriptIDAndProjectIDs(scriptID string, projectIDs []string) error {
	if len(projectIDs) == 0 {
		return nil
	}
	return r.db.Where("script_id = ? AND project_id IN ?", scriptID, projectIDs).
		Delete(&models.ScriptProject{}).Error
}

// DeletePromptsByScriptIDAndProjectIDs deletes prompts by script_id and list of project_ids
func (r *ScriptRepository) DeletePromptsByScriptIDAndProjectIDs(scriptID string, projectIDs []string) error {
	if len(projectIDs) == 0 {
		return nil
	}
	return r.db.Where("script_id = ? AND project_id IN ?", scriptID, projectIDs).
		Delete(&models.ScriptPrompt{}).Error
}

// GetEdgesByScriptID gets all edges for a script
func (r *ScriptRepository) GetEdgesByScriptID(scriptID string) ([]*models.ScriptEdge, error) {
	var edges []*models.ScriptEdge
	err := r.db.Where("script_id = ?", scriptID).Find(&edges).Error
	if err != nil {
		return nil, err
	}
	return edges, nil
}

// UpdateEdge updates a script edge
func (r *ScriptRepository) UpdateEdge(edge *models.ScriptEdge) error {
	return r.db.Save(edge).Error
}

// DeleteEdgesByScriptIDAndEdgeIDs deletes specific edges by script_id and list of edge_ids
func (r *ScriptRepository) DeleteEdgesByScriptIDAndEdgeIDs(scriptID string, edgeIDs []string) error {
	if len(edgeIDs) == 0 {
		return nil
	}
	return r.db.Where("script_id = ? AND edge_id IN ?", scriptID, edgeIDs).
		Delete(&models.ScriptEdge{}).Error
}

// ScriptExecution methods

// CreateExecution creates a new script execution
func (r *ScriptRepository) CreateExecution(execution *models.ScriptExecution) error {
	return r.db.Create(execution).Error
}

// GetExecutionByID gets an execution by ID
func (r *ScriptRepository) GetExecutionByID(executionID string) (*models.ScriptExecution, error) {
	var execution models.ScriptExecution
	err := r.db.Where("id = ?", executionID).First(&execution).Error
	if err != nil {
		return nil, err
	}
	return &execution, nil
}

// UpdateExecution updates an execution
func (r *ScriptRepository) UpdateExecution(execution *models.ScriptExecution) error {
	return r.db.Save(execution).Error
}

// GetRunningExecutionsByUserID gets running executions for a user (for rate limiting)
func (r *ScriptRepository) GetRunningExecutionsByUserID(userID string) ([]*models.ScriptExecution, error) {
	var executions []*models.ScriptExecution
	err := r.db.Where("user_id = ? AND status IN ?", userID, []string{"pending", "running"}).
		Find(&executions).Error
	if err != nil {
		return nil, err
	}
	return executions, nil
}

// GetRunningExecutionsByTopicID gets running executions for a topic (for rate limiting)
func (r *ScriptRepository) GetRunningExecutionsByTopicID(topicID string) ([]*models.ScriptExecution, error) {
	var executions []*models.ScriptExecution
	err := r.db.Where("topic_id = ? AND status IN ?", topicID, []string{"pending", "running"}).
		Find(&executions).Error
	if err != nil {
		return nil, err
	}
	return executions, nil
}

// CreateProjectExecution creates a new project execution record
func (r *ScriptRepository) CreateProjectExecution(projectExec *models.ScriptProjectExecution) error {
	return r.db.Create(projectExec).Error
}

// GetProjectExecutionByID gets a project execution by ID
func (r *ScriptRepository) GetProjectExecutionByID(id string) (*models.ScriptProjectExecution, error) {
	var projectExec models.ScriptProjectExecution
	err := r.db.Where("id = ?", id).First(&projectExec).Error
	if err != nil {
		return nil, err
	}
	return &projectExec, nil
}

// GetProjectExecutionsByExecutionID gets all project executions for an execution
func (r *ScriptRepository) GetProjectExecutionsByExecutionID(executionID string) ([]*models.ScriptProjectExecution, error) {
	var projectExecs []*models.ScriptProjectExecution
	err := r.db.Where("execution_id = ?", executionID).
		Order("project_order ASC").
		Find(&projectExecs).Error
	if err != nil {
		return nil, err
	}
	return projectExecs, nil
}

// GetProjectExecutionByExecutionIDAndProjectID gets a project execution by execution_id and project_id
func (r *ScriptRepository) GetProjectExecutionByExecutionIDAndProjectID(executionID, projectID string) (*models.ScriptProjectExecution, error) {
	var projectExec models.ScriptProjectExecution
	err := r.db.Where("execution_id = ? AND project_id = ?", executionID, projectID).First(&projectExec).Error
	if err != nil {
		return nil, err
	}
	return &projectExec, nil
}

// UpdateProjectExecution updates a project execution
func (r *ScriptRepository) UpdateProjectExecution(projectExec *models.ScriptProjectExecution) error {
	return r.db.Save(projectExec).Error
}

// GetCompletedProjectExecutionsByExecutionID gets all completed project executions for an execution
func (r *ScriptRepository) GetCompletedProjectExecutionsByExecutionID(executionID string) ([]*models.ScriptProjectExecution, error) {
	var projectExecs []*models.ScriptProjectExecution
	err := r.db.Where("execution_id = ? AND status = ?", executionID, "completed").
		Order("project_order ASC").
		Find(&projectExecs).Error
	if err != nil {
		return nil, err
	}
	return projectExecs, nil
}


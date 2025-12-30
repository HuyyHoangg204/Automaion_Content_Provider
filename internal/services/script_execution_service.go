package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

type ScriptExecutionService struct {
	scriptRepo           *repository.ScriptRepository
	topicRepo            *repository.TopicRepository
	userProfileRepo      *repository.UserProfileRepository
	chromeProfileService *ChromeProfileService
	rabbitMQ             *RabbitMQService
	fileService          *FileService
	baseURL              string
	executionStopChan    chan bool // Stop channel cho execution worker (cũ)
	projectStopChan      chan bool // Stop channel cho project worker (mới)
	maxConcurrentPerUser int       // Limit executions per user (không limit per topic vì topic shared)
}

func NewScriptExecutionService(
	scriptRepo *repository.ScriptRepository,
	topicRepo *repository.TopicRepository,
	userProfileRepo *repository.UserProfileRepository,
	chromeProfileService *ChromeProfileService,
	rabbitMQ *RabbitMQService,
	fileService *FileService,
	baseURL string,
) *ScriptExecutionService {
	logrus.Info("[ScriptExecutionService] Initializing service...")
	return &ScriptExecutionService{
		scriptRepo:           scriptRepo,
		topicRepo:            topicRepo,
		userProfileRepo:      userProfileRepo,
		chromeProfileService: chromeProfileService,
		rabbitMQ:             rabbitMQ,
		fileService:          fileService,
		baseURL:              baseURL,
		executionStopChan:    make(chan bool),
		projectStopChan:      make(chan bool),
		maxConcurrentPerUser: 1, // Mỗi user chỉ được execute 1 lần cùng lúc
	}
}

// ExecuteScript triggers script execution by publishing to queue
func (s *ScriptExecutionService) ExecuteScript(topicID, userID string) (*models.ExecuteScriptResponse, error) {
	// Get script
	script, err := s.scriptRepo.GetByTopicIDAndUserID(topicID, userID)
	if err != nil {
		return nil, fmt.Errorf("script not found: %w", err)
	}

	// Rate limiting: Check concurrent executions per user
	// Mỗi user có giới hạn số executions đang chạy cùng lúc
	runningExecutions, err := s.scriptRepo.GetRunningExecutionsByUserID(userID)
	if err != nil {
		logrus.Warnf("Failed to check running executions for user %s: %v", userID, err)
	} else if len(runningExecutions) >= s.maxConcurrentPerUser {
		return nil, fmt.Errorf("maximum concurrent executions reached for user (%d/%d)", len(runningExecutions), s.maxConcurrentPerUser)
	}

	// NOTE: Không limit per topic vì 1 topic có thể được nhiều users share
	// Mỗi user đã có limit riêng ở trên

	// Validate script has projects
	if len(script.Projects) == 0 {
		return nil, fmt.Errorf("script has no projects")
	}

	// Validate script has no cycles
	if err := s.validateScriptNoCycles(script); err != nil {
		return nil, fmt.Errorf("script validation failed: %w", err)
	}

	// Topological sort projects
	executionOrder, err := s.topologicalSort(script.Projects, script.Edges)
	if err != nil {
		return nil, fmt.Errorf("failed to sort projects: %w", err)
	}

	// Create execution record
	execution := &models.ScriptExecution{
		ScriptID: script.ID,
		TopicID:  topicID,
		UserID:   userID,
		Status:   "pending",
	}
	if err := s.scriptRepo.CreateExecution(execution); err != nil {
		return nil, fmt.Errorf("failed to create execution record: %w", err)
	}

	// Create project execution records and publish to queue
	for order, projectID := range executionOrder {
		project := s.findProjectByID(script.Projects, projectID)
		if project == nil {
			return nil, fmt.Errorf("project %s not found", projectID)
		}

		// Create project execution record
		projectExec := &models.ScriptProjectExecution{
			ExecutionID:  execution.ID,
			ProjectID:    project.ProjectID,
			ProjectOrder: order,
			Status:       "pending",
		}
		if err := s.scriptRepo.CreateProjectExecution(projectExec); err != nil {
			return nil, fmt.Errorf("failed to create project execution record: %w", err)
		}

		// Publish first project to queue
		if order == 0 {
			message := map[string]interface{}{
				"execution_id":    execution.ID,
				"project_exec_id": projectExec.ID,
				"project_id":      project.ProjectID,
				"project_order":   order,
				"script_id":       script.ID,
				"topic_id":        topicID,
				"user_id":         userID,
			}

			if err := s.rabbitMQ.PublishMessage(nil, "script_projects", message); err != nil {
				execution.Status = "failed"
				execution.ErrorMessage = fmt.Sprintf("Failed to publish project to queue: %v", err)
				s.scriptRepo.UpdateExecution(execution)
				return nil, fmt.Errorf("failed to publish project to queue: %w", err)
			}
		}
	}

	logrus.Infof("[Execute] Started execution %s with %d projects", execution.ID, len(executionOrder))

	return &models.ExecuteScriptResponse{
		ExecutionID: execution.ID,
		ScriptID:    script.ID,
		TopicID:     topicID,
		Status:      "pending",
		Message:     "Script execution queued successfully",
	}, nil
}

// StartWorker starts consuming from queue and processing executions
func (s *ScriptExecutionService) StartWorker() error {
	queueName := "script_executions"

	// Declare queue with DLQ
	_, err := s.rabbitMQ.channel.QueueDeclare(
		queueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		amqp.Table{
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": "script_executions_dlq",
		},
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Declare DLQ
	_, err = s.rabbitMQ.channel.QueueDeclare(
		"script_executions_dlq",
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare DLQ: %w", err)
	}

	// Set prefetch count (mỗi worker xử lý tối đa 1 job cùng lúc để tránh quá tải)
	err = s.rabbitMQ.channel.Qos(1, 0, false)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	// Consume messages
	msgs, err := s.rabbitMQ.channel.Consume(
		queueName,
		"",    // consumer tag
		false, // auto-ack (manual ack để retry khi fail)
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	logrus.Info("[ExecutionWorker] Started, consuming from queue: script_executions")

	// Process messages in goroutine
	go func() {
		for {
			select {
			case <-s.executionStopChan:
				logrus.Info("Script execution worker stopped")
				return
			case msg, ok := <-msgs:
				if !ok {
					logrus.Warn("RabbitMQ channel closed")
					return
				}

				// Process message
				if err := s.processExecutionMessage(msg); err != nil {
					logrus.Errorf("Failed to process execution message: %v", err)

					// Check retry count từ message headers hoặc execution record
					retryCount := 0
					if retryHeader, ok := msg.Headers["x-retry-count"]; ok {
						if count, ok := retryHeader.(int); ok {
							retryCount = count
						}
					}

					maxRetries := 3
					if retryCount >= maxRetries {
						// Đã retry quá nhiều → move to DLQ
						logrus.Errorf("Execution failed after %d retries, moving to DLQ", retryCount)
						msg.Nack(false, false) // requeue=false → move to DLQ
					} else {
						// Retry với delay
						retryCount++
						logrus.Warnf("Execution failed, retry %d/%d after delay", retryCount, maxRetries)

						// Nack và republish với delay (sử dụng delay queue hoặc sleep)
						// Tạm thời: nack với requeue=false và republish với delay
						msg.Nack(false, false)

						// Republish với retry count header và delay
						go func() {
							time.Sleep(time.Duration(retryCount) * 10 * time.Second) // Exponential backoff: 10s, 20s, 30s

							// Republish message với retry count
							var message map[string]interface{}
							if err := json.Unmarshal(msg.Body, &message); err == nil {
								// Publish lại với headers
								body, _ := json.Marshal(message)
								s.rabbitMQ.channel.Publish(
									"",
									"script_executions",
									false,
									false,
									amqp.Publishing{
										ContentType: "application/json",
										Body:        body,
										Headers: amqp.Table{
											"x-retry-count": retryCount,
										},
										Timestamp: time.Now(),
									},
								)
							}
						}()
					}
				} else {
					// Ack message
					msg.Ack(false)
				}
			}
		}
	}()

	return nil
}

// StopWorker stops the old execution worker
func (s *ScriptExecutionService) StopWorker() {
	logrus.Info("[ExecutionWorker] Stopping...")
	close(s.executionStopChan)
}

// StopProjectWorker stops the project worker
func (s *ScriptExecutionService) StopProjectWorker() {
	logrus.Info("[ProjectWorker] Stopping...")
	close(s.projectStopChan)
}

// StartProjectWorker starts consuming project messages from queue with auto-restart
func (s *ScriptExecutionService) StartProjectWorker() error {
	go s.runProjectWorker()
	return nil
}

// runProjectWorker runs the project worker with auto-restart on failure
func (s *ScriptExecutionService) runProjectWorker() {
	for {
		select {
		case <-s.projectStopChan:
			return
		default:
		}

		if err := s.consumeProjectMessages(); err != nil {
			logrus.Errorf("[ProjectWorker] Error: %v, restarting in 5s...", err)
			time.Sleep(5 * time.Second)
			continue
		}
	}
}

// consumeProjectMessages sets up consumer and processes messages
func (s *ScriptExecutionService) consumeProjectMessages() error {
	queueName := "script_projects"

	_, err := s.rabbitMQ.channel.QueueDeclare(
		queueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		amqp.Table{
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": "script_projects_dlq",
		},
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	_, err = s.rabbitMQ.channel.QueueDeclare(
		"script_projects_dlq",
		true, false, false, false, nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare DLQ: %w", err)
	}

	err = s.rabbitMQ.channel.Qos(1, 0, false)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	msgs, err := s.rabbitMQ.channel.Consume(
		queueName,
		"project-worker", // consumer tag
		false,            // auto-ack
		false,            // exclusive
		false,            // no-local
		false,            // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	logrus.Info("[ProjectWorker] Started listening on queue: script_projects")

	for {
		select {
		case <-s.projectStopChan:
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("RabbitMQ channel closed")
			}

			if err := s.processProjectMessage(msg); err != nil {
				logrus.Errorf("[ProjectWorker] Failed: %v", err)
				msg.Nack(false, false)
			} else {
				msg.Ack(false)
			}
		}
	}
}

// processProjectMessage processes a project message from queue
func (s *ScriptExecutionService) processProjectMessage(msg amqp.Delivery) error {
	var message map[string]interface{}
	if err := json.Unmarshal(msg.Body, &message); err != nil {
		return nil // Skip invalid messages
	}

	projectExecID, ok := message["project_exec_id"].(string)
	if !ok {
		return nil // Skip invalid messages
	}

	// Get project execution record
	projectExec, err := s.scriptRepo.GetProjectExecutionByID(projectExecID)
	if err != nil {
		return nil // Stale message, skip
	}

	// Get execution record
	execution, err := s.scriptRepo.GetExecutionByID(projectExec.ExecutionID)
	if err != nil {
		return nil // Stale message, skip
	}

	// Update execution status to running (nếu chưa)
	if execution.Status == "pending" {
		now := time.Now()
		execution.Status = "running"
		execution.StartedAt = &now
		execution.CurrentProjectID = &projectExec.ProjectID
		if err := s.scriptRepo.UpdateExecution(execution); err != nil {
			return fmt.Errorf("failed to update execution status: %w", err)
		}
	}

	// Update project execution status to running
	now := time.Now()
	projectExec.Status = "running"
	projectExec.StartedAt = &now
	if err := s.scriptRepo.UpdateProjectExecution(projectExec); err != nil {
		return fmt.Errorf("failed to update project execution status: %w", err)
	}

	// Get script
	script, err := s.scriptRepo.GetByTopicIDAndUserID(execution.TopicID, execution.UserID)
	if err != nil {
		return fmt.Errorf("failed to get script: %w", err)
	}

	// Get project
	project := s.findProjectByID(script.Projects, projectExec.ProjectID)
	if project == nil {
		return fmt.Errorf("project %s not found", projectExec.ProjectID)
	}

	// Get topic
	topic, err := s.topicRepo.GetByID(execution.TopicID)
	if err != nil {
		return fmt.Errorf("failed to get topic: %w", err)
	}

	// Get user profile
	ownerProfile, err := s.userProfileRepo.GetByID(topic.UserProfileID)
	if err != nil {
		return fmt.Errorf("failed to get owner profile: %w", err)
	}

	// Generate debugPort từ userID (deterministic - cùng userID luôn ra cùng port)
	debugPort := s.generateDebugPort(execution.UserID)

	var tunnelURL string

	// Chỉ launch Chrome cho project đầu tiên (order = 0)
	if projectExec.ProjectOrder == 0 {
		launchReq := &LaunchChromeProfileRequest{
			UserProfileID: ownerProfile.ID,
			EnsureGmail:   true,
			EntityType:    "script_execution",
			EntityID:      topic.ID,
			DebugPort:     debugPort,
		}

		launchResp, err := s.chromeProfileService.LaunchChromeProfile(execution.UserID, launchReq)
		if err != nil {
			return fmt.Errorf("failed to launch Chrome profile: %w", err)
		}

		tunnelURL = launchResp.TunnelURL
		execution.TunnelURL = tunnelURL
		s.scriptRepo.UpdateExecution(execution)
	} else {
		tunnelURL = execution.TunnelURL
		if tunnelURL == "" {
			return fmt.Errorf("tunnelURL not found in execution")
		}
	}

	profileDirName := ownerProfile.ProfileDirName
	// Generate gemName từ projectID và name để đảm bảo unique cho mỗi project (không lưu trong DB nữa)
	// Format: {projectID}_{name}
	gemName := fmt.Sprintf("%s_%s", project.ProjectID, project.Name)

	// Get prompts for this project
	prompts := s.getPromptsForProject(script.Projects, project.ProjectID)

	logrus.Infof("[ProjectWorker] Executing project %s (order %d) debugPort=%d", project.ProjectID, projectExec.ProjectOrder, debugPort)

	// Call automation backend API với debugPort
	if err := s.callAutomationBackendProjectAsync(tunnelURL, profileDirName, gemName, project, prompts, execution, topic, debugPort); err != nil {
		projectExec.Status = "failed"
		projectExec.ErrorMessage = err.Error()
		completedAt := time.Now()
		projectExec.CompletedAt = &completedAt
		s.scriptRepo.UpdateProjectExecution(projectExec)
		return fmt.Errorf("failed to call automation backend: %w", err)
	}

	return nil
}

// callAutomationBackendProjectAsync calls automation backend API - fire and forget
// Không đợi response vì automation backend sẽ gửi log project_completed khi xong
func (s *ScriptExecutionService) callAutomationBackendProjectAsync(
	tunnelURL, profileDirName, gemName string,
	project *models.ScriptProject,
	prompts []*models.ScriptPrompt,
	execution *models.ScriptExecution,
	topic *models.Topic,
	debugPort int,
) error {
	apiURL := fmt.Sprintf("%s/gemini/projects", strings.TrimSuffix(tunnelURL, "/"))

	promptList := make([]map[string]interface{}, 0, len(prompts))
	for _, prompt := range prompts {
		// Convert file names thành download URLs
		inputFilesURLs := s.convertFileNamesToURLs(execution.UserID, prompt.InputFiles)

		promptMap := map[string]interface{}{
			"prompt":           prompt.PromptText,
			"output":           prompt.Filename,   // Chỉ file name, không có extension
			"input_files":      prompt.InputFiles, // File names từ previous executions
			"input_files_urls": inputFilesURLs,    // URLs để download từ backend cloud
			"prompt_id":        prompt.ID,
		}
		// Chỉ thêm merge nếu true
		if prompt.Merge {
			promptMap["merge"] = true
		}
		// Chỉ thêm exit nếu true
		if prompt.Exit {
			promptMap["exit"] = true
		}
		promptList = append(promptList, promptMap)
	}

	requestBody := map[string]interface{}{
		"execution_id": execution.ID,
		"project":      project.ProjectID,
		"gemName":      gemName,
		"prompts":      promptList,
		"debugPort":    debugPort, // Chrome debug port để automation backend connect đúng Chrome
	}
	// Thêm output_merge nếu project có filename (project level, không có extension)
	if project.Filename != "" {
		requestBody["output_merge"] = project.Filename
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "Green-Provider-Services/1.0")
	httpReq.Header.Set("X-User-ID", execution.UserID)
	httpReq.Header.Set("X-Entity-Type", "script_execution")
	httpReq.Header.Set("X-Entity-ID", topic.ID)

	// Fire and forget - gửi request trong goroutine, không đợi response
	// Automation backend sẽ gửi log project_completed khi xong
	go func() {
		client := &http.Client{
			Timeout: 60 * time.Minute, // Long timeout cho việc chạy prompts
		}

		resp, err := client.Do(httpReq)
		if err != nil {
			logrus.Errorf("Automation backend request failed for project %s: %v", project.ProjectID, err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			bodyBytes, _ := io.ReadAll(resp.Body)
			logrus.Errorf("Automation backend returned error status %d for project %s: %s", resp.StatusCode, project.ProjectID, string(bodyBytes))
			return
		}

		logrus.Infof("Automation backend finished processing project %s", project.ProjectID)
	}()

	logrus.Infof("Project %s execution request sent to automation backend (fire-and-forget)", project.ProjectID)
	return nil
}

// TriggerNextProject triggers the next project in execution when a project completes
func (s *ScriptExecutionService) TriggerNextProject(executionID, completedProjectID string) error {
	// Get execution
	execution, err := s.scriptRepo.GetExecutionByID(executionID)
	if err != nil {
		return fmt.Errorf("failed to get execution %s: %w", executionID, err)
	}

	// Get all project executions
	projectExecs, err := s.scriptRepo.GetProjectExecutionsByExecutionID(executionID)
	if err != nil {
		return fmt.Errorf("failed to get project executions: %w", err)
	}

	// Find completed project order
	completedOrder := -1
	for _, pe := range projectExecs {
		if pe.ProjectID == completedProjectID && pe.Status == "completed" {
			completedOrder = pe.ProjectOrder
			break
		}
	}

	if completedOrder == -1 {
		return fmt.Errorf("completed project %s not found or not completed", completedProjectID)
	}

	// Find next project (order + 1)
	var nextProjectExec *models.ScriptProjectExecution
	for _, pe := range projectExecs {
		if pe.ProjectOrder == completedOrder+1 && pe.Status == "pending" {
			nextProjectExec = pe
			break
		}
	}

	if nextProjectExec == nil {
		// No more projects → check if all projects are completed
		allCompleted := true
		for _, pe := range projectExecs {
			if pe.Status != "completed" {
				allCompleted = false
				break
			}
		}

		if allCompleted {
			// Mark execution as completed
			execution.Status = "completed"
			completedAt := time.Now()
			execution.CompletedAt = &completedAt
			if err := s.scriptRepo.UpdateExecution(execution); err != nil {
				return fmt.Errorf("failed to update execution status: %w", err)
			}
			logrus.Infof("Execution %s completed - all projects finished", executionID)
		} else {
			// Check dependencies: các projects phụ thuộc đã xong chưa?
			// Tạm thời: chỉ trigger project tiếp theo theo thứ tự
			logrus.Infof("No more projects to trigger for execution %s", executionID)
		}
		return nil
	}

	// Get script to find project details
	script, err := s.scriptRepo.GetByTopicIDAndUserID(execution.TopicID, execution.UserID)
	if err != nil {
		return fmt.Errorf("failed to get script: %w", err)
	}

	project := s.findProjectByID(script.Projects, nextProjectExec.ProjectID)
	if project == nil {
		return fmt.Errorf("project %s not found", nextProjectExec.ProjectID)
	}

	// Publish next project to queue
	message := map[string]interface{}{
		"execution_id":    execution.ID,
		"project_exec_id": nextProjectExec.ID,
		"project_id":      project.ProjectID,
		"project_order":   nextProjectExec.ProjectOrder,
		"script_id":       script.ID,
		"topic_id":        execution.TopicID,
		"user_id":         execution.UserID,
	}

	if err := s.rabbitMQ.PublishMessage(nil, "script_projects", message); err != nil {
		return fmt.Errorf("failed to publish next project to queue: %w", err)
	}

	return nil
}

// MarkProjectCompletedByTopicID marks a project as completed using topic ID
func (s *ScriptExecutionService) MarkProjectCompletedByTopicID(topicID, projectID string) error {
	runningExecutions, err := s.scriptRepo.GetRunningExecutionsByTopicID(topicID)
	if err != nil {
		return fmt.Errorf("failed to get running executions for topic %s: %w", topicID, err)
	}

	if len(runningExecutions) == 0 {
		return fmt.Errorf("no running execution found for topic %s", topicID)
	}

	return s.MarkProjectCompleted(runningExecutions[0].ID, projectID)
}

// MarkProjectCompleted marks a project as completed when receiving project_completed log
func (s *ScriptExecutionService) MarkProjectCompleted(executionID, projectID string) error {
	projectExec, err := s.scriptRepo.GetProjectExecutionByExecutionIDAndProjectID(executionID, projectID)
	if err != nil {
		return fmt.Errorf("failed to get project execution: %w", err)
	}

	projectExec.Status = "completed"
	completedAt := time.Now()
	projectExec.CompletedAt = &completedAt
	if err := s.scriptRepo.UpdateProjectExecution(projectExec); err != nil {
		return fmt.Errorf("failed to update project execution status: %w", err)
	}

	logrus.Infof("[Completed] Project %s execution %s", projectID, executionID)
	return s.TriggerNextProject(executionID, projectID)
}

// processExecutionMessage processes a message from queue
func (s *ScriptExecutionService) processExecutionMessage(msg amqp.Delivery) error {
	var message map[string]interface{}
	if err := json.Unmarshal(msg.Body, &message); err != nil {
		logrus.Warnf("[ExecutionWorker] Invalid message format, skipping: %v", err)
		return nil // Skip invalid messages
	}

	executionID, ok := message["execution_id"].(string)
	if !ok {
		logrus.Warn("[ExecutionWorker] Missing execution_id in message, skipping")
		return nil // Skip invalid messages
	}

	logrus.Infof("[ExecutionWorker] Processing script execution %s", executionID)

	// Get execution record
	execution, err := s.scriptRepo.GetExecutionByID(executionID)
	if err != nil {
		// Record not found = message cũ, DB đã reset → skip, không retry
		logrus.Warnf("[ExecutionWorker] Execution %s not found in DB (stale message?), skipping", executionID)
		return nil
	}

	// Update status to running
	now := time.Now()
	execution.Status = "running"
	execution.StartedAt = &now
	if err := s.scriptRepo.UpdateExecution(execution); err != nil {
		return fmt.Errorf("failed to update execution status: %w", err)
	}

	// Execute script
	if err := s.executeScript(execution); err != nil {
		// Check if error is permanent (should not retry)
		isPermanentError := s.isPermanentError(err)

		if isPermanentError {
			// Permanent error → mark as failed, don't retry
			execution.Status = "failed"
			execution.ErrorMessage = err.Error()
			completedAt := time.Now()
			execution.CompletedAt = &completedAt
			s.scriptRepo.UpdateExecution(execution)
			logrus.Errorf("Execution %s failed with permanent error: %v", executionID, err)
			return nil // Return nil để ack message, không retry
		}

		// Transient error → increment retry count và return error để retry
		execution.RetryCount++
		execution.ErrorMessage = err.Error()
		s.scriptRepo.UpdateExecution(execution)
		return fmt.Errorf("script execution failed: %w", err)
	}

	// Update status to completed
	execution.Status = "completed"
	completedAt := time.Now()
	execution.CompletedAt = &completedAt
	if err := s.scriptRepo.UpdateExecution(execution); err != nil {
		logrus.Errorf("Failed to update execution status to completed: %v", err)
	}

	logrus.Infof("Script execution %s completed successfully", executionID)
	return nil
}

// executeScript executes a script by running projects in topological order
func (s *ScriptExecutionService) executeScript(execution *models.ScriptExecution) error {
	// Get script with all relations
	script, err := s.scriptRepo.GetByTopicIDAndUserID(execution.TopicID, execution.UserID)
	if err != nil {
		return fmt.Errorf("failed to get script: %w", err)
	}

	// Get topic
	topic, err := s.topicRepo.GetByID(execution.TopicID)
	if err != nil {
		return fmt.Errorf("failed to get topic: %w", err)
	}

	// Get user profile (owner's profile)
	ownerProfile, err := s.userProfileRepo.GetByID(topic.UserProfileID)
	if err != nil {
		return fmt.Errorf("failed to get owner profile: %w", err)
	}

	// Topological sort projects
	executionOrder, err := s.topologicalSort(script.Projects, script.Edges)
	if err != nil {
		return fmt.Errorf("failed to sort projects: %w", err)
	}

	logrus.Infof("Executing script %s with %d projects in order", script.ID, len(executionOrder))

	// Generate debugPort từ userID (deterministic)
	debugPort := s.generateDebugPort(execution.UserID)

	// Launch Chrome profile với debugPort
	launchReq := &LaunchChromeProfileRequest{
		UserProfileID: ownerProfile.ID,
		EnsureGmail:   true,
		EntityType:    "script_execution",
		EntityID:      topic.ID,
		DebugPort:     debugPort, // Truyền debugPort để automation backend mở Chrome với port này
	}

	launchResp, err := s.chromeProfileService.LaunchChromeProfile(execution.UserID, launchReq)
	if err != nil {
		return fmt.Errorf("failed to launch Chrome profile: %w", err)
	}
	defer func() {
		if releaseErr := s.chromeProfileService.ReleaseChromeProfile(execution.UserID, &ReleaseChromeProfileRequest{
			UserProfileID: ownerProfile.ID,
		}); releaseErr != nil {
			logrus.Warnf("Failed to release Chrome profile lock: %v", releaseErr)
		}
	}()

	tunnelURL := launchResp.TunnelURL
	profileDirName := ownerProfile.ProfileDirName

	// Execute each project in order
	for i, projectID := range executionOrder {
		project := s.findProjectByID(script.Projects, projectID)
		if project == nil {
			return fmt.Errorf("project %s not found", projectID)
		}

		// Generate gemName từ projectID và name để đảm bảo unique cho mỗi project (không lưu trong DB nữa)
		// Format: {projectID}_{name}
		gemName := fmt.Sprintf("%s_%s", project.ProjectID, project.Name)

		// Update current project
		execution.CurrentProjectID = &project.ProjectID
		s.scriptRepo.UpdateExecution(execution)

		logrus.Infof("Executing project %d/%d: %s (%s) with debugPort=%d", i+1, len(executionOrder), project.Name, project.ProjectID, debugPort)

		// Get prompts for this project (sorted by prompt_order)
		prompts := s.getPromptsForProject(script.Projects, project.ProjectID)

		// Call automation backend API với debugPort
		if err := s.callAutomationBackendProject(tunnelURL, profileDirName, gemName, project, prompts, execution, topic, debugPort); err != nil {
			return fmt.Errorf("failed to execute project %s: %w", project.ProjectID, err)
		}

		logrus.Infof("Project %s completed successfully", project.ProjectID)
	}

	return nil
}

// callAutomationBackendProject calls automation backend API /gemini/projects
func (s *ScriptExecutionService) callAutomationBackendProject(
	tunnelURL, profileDirName, gemName string,
	project *models.ScriptProject,
	prompts []*models.ScriptPrompt,
	execution *models.ScriptExecution,
	topic *models.Topic,
	debugPort int,
) error {
	apiURL := fmt.Sprintf("%s/gemini/projects", strings.TrimSuffix(tunnelURL, "/"))

	// Convert prompts to API format
	promptList := make([]map[string]interface{}, 0, len(prompts))
	for _, prompt := range prompts {
		// Convert file names thành download URLs
		inputFilesURLs := s.convertFileNamesToURLs(execution.UserID, prompt.InputFiles)

		promptMap := map[string]interface{}{
			"prompt":           prompt.PromptText,
			"output":           prompt.Filename,   // Chỉ file name, không có extension
			"input_files":      prompt.InputFiles, // File names từ previous executions
			"input_files_urls": inputFilesURLs,    // URLs để download từ backend cloud
			"prompt_id":        prompt.ID,
		}
		// Chỉ thêm merge nếu true
		if prompt.Merge {
			promptMap["merge"] = true
		}
		// Chỉ thêm exit nếu true
		if prompt.Exit {
			promptMap["exit"] = true
		}
		promptList = append(promptList, promptMap)
	}

	// Build request body theo format của API /gemini/projects
	requestBody := map[string]interface{}{
		"execution_id": execution.ID,
		"project":      project.ProjectID,
		"gemName":      gemName,
		"prompts":      promptList,
		"debugPort":    debugPort, // Chrome debug port để automation backend connect đúng Chrome
	}
	// Thêm output_merge nếu project có filename (project level, không có extension)
	if project.Filename != "" {
		requestBody["output_merge"] = project.Filename
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "Green-Provider-Services/1.0")
	httpReq.Header.Set("X-User-ID", execution.UserID)
	httpReq.Header.Set("X-Entity-Type", "script_execution")
	httpReq.Header.Set("X-Entity-ID", topic.ID) // ✅ Dùng topic.ID, không phải execution.ID

	client := &http.Client{
		Timeout: 30 * time.Minute, // Long timeout cho project execution
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to call automation backend: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logrus.Errorf("Automation backend returned error status %d: %s", resp.StatusCode, string(bodyBytes))
		var errorResp map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &errorResp); err == nil {
			if errorMsg, ok := errorResp["error"].(string); ok {
				return fmt.Errorf("automation backend error: %s", errorMsg)
			}
			if errorMsg, ok := errorResp["message"].(string); ok {
				return fmt.Errorf("automation backend error: %s", errorMsg)
			}
		}
		return fmt.Errorf("automation backend returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	logrus.Infof("Project %s executed successfully", project.ProjectID)
	return nil
}

// validateScriptNoCycles validates that script has no cycles
func (s *ScriptExecutionService) validateScriptNoCycles(script *models.Script) error {
	// Build adjacency map
	adjMap := make(map[string][]string)
	inDegree := make(map[string]int)

	// Initialize in-degree for all projects
	for _, project := range script.Projects {
		inDegree[project.ProjectID] = 0
	}

	// Build graph
	for _, edge := range script.Edges {
		source := edge.SourceProjectID
		target := edge.TargetProjectID

		if adjMap[source] == nil {
			adjMap[source] = make([]string, 0)
		}
		adjMap[source] = append(adjMap[source], target)
		inDegree[target]++
	}

	// Kahn's algorithm for cycle detection
	queue := make([]string, 0)
	for projectID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, projectID)
		}
	}

	visitedCount := 0
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		visitedCount++

		for _, neighbor := range adjMap[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// If visitedCount != total projects, there's a cycle
	if visitedCount != len(script.Projects) {
		return fmt.Errorf("script contains cycles")
	}

	return nil
}

// topologicalSort sorts projects in topological order
func (s *ScriptExecutionService) topologicalSort(projects []models.ScriptProject, edges []models.ScriptEdge) ([]string, error) {
	// Build adjacency map and in-degree
	adjMap := make(map[string][]string)
	inDegree := make(map[string]int)

	// Initialize in-degree for all projects
	for _, project := range projects {
		inDegree[project.ProjectID] = 0
	}

	// Build graph
	for _, edge := range edges {
		source := edge.SourceProjectID
		target := edge.TargetProjectID

		if adjMap[source] == nil {
			adjMap[source] = make([]string, 0)
		}
		adjMap[source] = append(adjMap[source], target)
		inDegree[target]++
	}

	// Kahn's algorithm
	queue := make([]string, 0)
	result := make([]string, 0)

	// Add all nodes with in-degree 0
	for projectID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, projectID)
		}
	}

	// If no entry nodes, use created_at order
	if len(queue) == 0 {
		logrus.Warn("No entry nodes found, using created_at order")
		for _, project := range projects {
			result = append(result, project.ProjectID)
		}
		return result, nil
	}

	// Process queue
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		for _, neighbor := range adjMap[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// Check if all projects are included
	if len(result) != len(projects) {
		return nil, fmt.Errorf("topological sort incomplete: %d/%d projects", len(result), len(projects))
	}

	return result, nil
}

// findProjectByID finds a project by project_id
func (s *ScriptExecutionService) findProjectByID(projects []models.ScriptProject, projectID string) *models.ScriptProject {
	for i := range projects {
		if projects[i].ProjectID == projectID {
			return &projects[i]
		}
	}
	return nil
}

// getPromptsForProject gets prompts for a project, sorted by prompt_order
func (s *ScriptExecutionService) getPromptsForProject(projects []models.ScriptProject, projectID string) []*models.ScriptPrompt {
	for i := range projects {
		if projects[i].ProjectID == projectID {
			prompts := make([]*models.ScriptPrompt, len(projects[i].Prompts))
			for j := range projects[i].Prompts {
				prompts[j] = &projects[i].Prompts[j]
			}
			return prompts
		}
	}
	return []*models.ScriptPrompt{}
}

// isPermanentError checks if error is permanent (should not retry)
func (s *ScriptExecutionService) isPermanentError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()

	// Permanent errors: profile in use, invalid script, missing data
	permanentErrors := []string{
		"profile is currently in use",
		"script not found",
		"topic not found",
		"script has no projects",
		"script contains cycles",
		"invalid execution_id",
	}

	for _, permanentErr := range permanentErrors {
		if strings.Contains(errMsg, permanentErr) {
			return true
		}
	}

	return false
}

// convertFileNamesToURLs converts file names (original_name) to download URLs
// Query files from user and match by original_name, then create download URLs
func (s *ScriptExecutionService) convertFileNamesToURLs(userID string, fileNames []string) []string {
	if len(fileNames) == 0 || s.fileService == nil {
		return []string{}
	}

	// Get all user files
	userFiles, err := s.fileService.GetUserFiles(userID)
	if err != nil {
		logrus.Warnf("Failed to get user files for converting to URLs: %v", err)
		return []string{}
	}

	// Create map: original_name -> file (lấy file mới nhất nếu có nhiều cùng tên)
	fileMap := make(map[string]*models.File)
	for _, file := range userFiles {
		existing, exists := fileMap[file.OriginalName]
		if !exists || file.CreatedAt.After(existing.CreatedAt) {
			fileMap[file.OriginalName] = file
		}
	}

	// Convert file names to URLs
	urls := make([]string, 0, len(fileNames))
	for _, fileName := range fileNames {
		file, found := fileMap[fileName]
		if !found {
			continue
		}
		downloadURL := fmt.Sprintf("%s/api/v1/files/%s/download", strings.TrimSuffix(s.baseURL, "/"), file.ID)
		urls = append(urls, downloadURL)
	}

	return urls
}

// generateDebugPort generates a unique debug port for a user
// Base port 9222 + offset based on userID hash (range 0-777)
// Mỗi user có 1 port riêng, tránh conflict khi nhiều user dùng chung profile
func (s *ScriptExecutionService) generateDebugPort(userID string) int {
	const basePort = 9222
	const portRange = 778 // 9222 - 9999

	h := fnv.New32a()
	h.Write([]byte(userID))
	offset := int(h.Sum32() % uint32(portRange))

	return basePort + offset
}

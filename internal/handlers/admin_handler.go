package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/onegreenvn/green-provider-services-backend/internal/services"
	"github.com/onegreenvn/green-provider-services-backend/internal/services/auth"
	"github.com/onegreenvn/green-provider-services-backend/internal/utils"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type AdminHandler struct {
	authService   *auth.AuthService
	boxService    *services.BoxService
	appService    *services.AppService
	roleService   *services.RoleService
	topicService  *services.TopicService
	scriptService *services.ScriptService // Injected ScriptService
	topicUserRepo *repository.TopicUserRepository
	userRepo      *repository.UserRepository
}

func NewAdminHandler(authService *auth.AuthService, db *gorm.DB, topicService *services.TopicService, scriptService *services.ScriptService) *AdminHandler {
	// Create repositories
	userRepo := repository.NewUserRepository(db)
	boxRepo := repository.NewBoxRepository(db)
	appRepo := repository.NewAppRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	topicUserRepo := repository.NewTopicUserRepository(db)
	userProfileRepo := repository.NewUserProfileRepository(db)
	geminiAccountRepo := repository.NewGeminiAccountRepository(db)

	// Create services
	boxService := services.NewBoxService(boxRepo, userRepo)
	appService := services.NewAppService(appRepo, boxRepo, userRepo)
	userProfileService := services.NewUserProfileService(userProfileRepo, appRepo, geminiAccountRepo, boxRepo)
	roleService := services.NewRoleService(roleRepo, userRepo, userProfileService)

	// Set app service and repo for box service (for status checking)
	boxService.SetAppService(appService)
	boxService.SetAppRepo(appRepo)

	return &AdminHandler{
		authService:   authService,
		boxService:    boxService,
		appService:    appService,
		roleService:   roleService,
		topicService:  topicService,
		scriptService: scriptService,
		topicUserRepo: topicUserRepo,
		userRepo:      userRepo,
	}
}

// Register godoc
// @Summary Register a new user (Admin only)
// @Description Register a new user account with username and password (Admin privileges required). Role can be specified via role_id field. If not provided, defaults to "topic_user".
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.RegisterRequest true "Registration request (username, password, first_name, last_name, role_id)"
// @Success 201 {object} models.AuthResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/admin/register [post]
func (h *AdminHandler) Register(c *gin.Context) {
	// Check if user is admin
	user := c.MustGet("user").(*models.User)
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}

	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	response, err := h.authService.Register(&req)
	if err != nil {
		if strings.Contains(err.Error(), "username already exists") {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// GetAllUsers godoc
// @Summary Get all users (Admin only)
// @Description Get list of all users in the system with pagination and search (Admin privileges required)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)" minimum(1)
// @Param limit query int false "Number of items per page (default: 10, max: 100)" minimum(1) maximum(100)
// @Param search query string false "Search term for username"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/admin/users [get]
func (h *AdminHandler) GetAllUsers(c *gin.Context) {
	// Check if user is admin
	user := c.MustGet("user").(*models.User)
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}

	// Parse query parameters
	page, pageSize := utils.ParsePaginationFromQuery(c.Query("page"), c.Query("limit"))
	search := c.DefaultQuery("search", "")

	users, total, err := h.authService.GetAllUsers(page, pageSize, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get users", "details": err.Error()})
		return
	}

	// Load roles for each user and create response with roles
	usersWithRoles := make([]gin.H, len(users))
	for i, user := range users {
		// Get user roles
		roles, err := h.roleService.GetUserRoles(user.ID)
		if err != nil {
			// If failed to get roles, set empty array
			roles = []models.Role{}
		}

		// Convert roles to role names
		roleNames := make([]string, len(roles))
		for j, role := range roles {
			roleNames[j] = role.Name
		}

		usersWithRoles[i] = gin.H{
			"id":            user.ID,
			"username":      user.Username,
			"first_name":    user.FirstName,
			"last_name":     user.LastName,
			"is_active":     user.IsActive,
			"is_admin":      user.IsAdmin,
			"token_version": user.TokenVersion,
			"created_at":    user.CreatedAt,
			"updated_at":    user.UpdatedAt,
			"last_login_at": user.LastLoginAt,
			"roles":         roleNames,
		}
	}

	// Calculate pagination info
	paginationInfo := utils.CalculatePaginationInfo(int(total), page, pageSize)

	response := gin.H{
		"data":         usersWithRoles,
		"total":        total,
		"page":         paginationInfo.Page,
		"limit":        paginationInfo.PageSize,
		"total_pages":  paginationInfo.TotalPages,
		"has_next":     paginationInfo.HasNext,
		"has_previous": paginationInfo.HasPrevious,
	}
	c.JSON(http.StatusOK, response)
}

// SetUserStatus godoc
// @Summary Set user active status (Admin only)
// @Description Set the active status of a user account (Admin privileges required)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param request body map[string]bool true "Status request {\"is_active\": true/false}"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/admin/users/{id}/status [put]
func (h *AdminHandler) SetUserStatus(c *gin.Context) {
	// Check if user is admin
	user := c.MustGet("user").(*models.User)
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}

	// Get user ID from URL
	userID := c.Param("id")

	// Parse request body
	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Set user status
	err := h.authService.SetUserActive(userID, req.IsActive)
	if err != nil {
		if strings.Contains(err.Error(), "user not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user status", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User status updated successfully"})
}

// AdminGetAllBoxes godoc
// @Summary Get all boxes (Admin only)
// @Description Get all boxes in the system (Admin privileges required)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.BoxResponse
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/admin/boxes [get]
func (h *AdminHandler) AdminGetAllBoxes(c *gin.Context) {
	// Check if user is admin
	user := c.MustGet("user").(*models.User)
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}

	boxes, err := h.boxService.GetAllBoxes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get boxes", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, boxes)
}

// AdminGetAllApps godoc
// @Summary Get all apps (Admin only)
// @Description Get all apps in the system (Admin privileges required)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.App
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/admin/apps [get]
func (h *AdminHandler) AdminGetAllApps(c *gin.Context) {
	// Check if user is admin
	user := c.MustGet("user").(*models.User)
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}

	apps, err := h.appService.GetAllApps()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get apps", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, apps)
}

// AdminGetAllBoxesWithStatus godoc
// @Summary Get all boxes with online/offline status (Admin only)
// @Description Get all boxes in the system with their online/offline status checked via health check (Admin privileges required)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.BoxWithStatusResponse
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/admin/boxes/status [get]
func (h *AdminHandler) AdminGetAllBoxesWithStatus(c *gin.Context) {
	// Check if user is admin
	user := c.MustGet("user").(*models.User)
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}

	boxes, err := h.boxService.GetAllBoxesWithStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get boxes with status", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, boxes)
}

// ResetPassword godoc
// @Summary Reset user password (Admin only)
// @Description Reset a user's password to a new password specified by admin (Admin privileges required)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param request body models.ResetPasswordRequest true "New password request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/admin/users/{id}/reset-password [post]
func (h *AdminHandler) ResetPassword(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}
	var req models.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	err := h.authService.ResetPassword(userID, req.NewPassword)
	if err != nil {
		if strings.Contains(err.Error(), "user not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset password", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Password reset successfully",
		"user_id": userID,
	})
}

// GetAllRoles godoc
// @Summary Get all roles (Admin only)
// @Description Get list of all roles in the system (Admin privileges required)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.Role
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/admin/roles [get]
func (h *AdminHandler) GetAllRoles(c *gin.Context) {
	// Check if user is admin
	user := c.MustGet("user").(*models.User)
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}

	roles, err := h.roleService.GetAllRoles()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get roles", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, roles)
}

// GetUserRoles godoc
// @Summary Get user roles (Admin only)
// @Description Get all roles assigned to a specific user (Admin privileges required)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} models.UserRoleResponse
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/admin/users/{id}/roles [get]
func (h *AdminHandler) GetUserRoles(c *gin.Context) {
	// Check if user is admin
	user := c.MustGet("user").(*models.User)
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}

	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	userRoleResponse, err := h.roleService.GetUserRoleResponse(userID)
	if err != nil {
		if strings.Contains(err.Error(), "user not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user roles", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, userRoleResponse)
}

// AssignRoleToUser godoc
// @Summary Assign role to user (Admin only)
// @Description Assign a role to a user (Admin privileges required)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param request body models.AssignRoleRequest true "Role assignment request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/admin/users/{id}/roles [post]
func (h *AdminHandler) AssignRoleToUser(c *gin.Context) {
	// Check if user is admin
	user := c.MustGet("user").(*models.User)
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}

	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	var req models.AssignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	err := h.roleService.AssignRoleToUser(userID, req.RoleID)
	if err != nil {
		if strings.Contains(err.Error(), "user not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		if strings.Contains(err.Error(), "role") && strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign role", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Role assigned successfully",
		"user_id": userID,
		"role_id": req.RoleID,
	})
}

// RemoveRoleFromUser godoc
// @Summary Remove role from user (Admin only)
// @Description Remove a role from a user (Admin privileges required)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param request body models.RemoveRoleRequest true "Role removal request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/admin/users/{id}/roles [delete]
func (h *AdminHandler) RemoveRoleFromUser(c *gin.Context) {
	// Check if user is admin
	user := c.MustGet("user").(*models.User)
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}

	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	var req models.RemoveRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	err := h.roleService.RemoveRoleFromUser(userID, req.RoleID)
	if err != nil {
		if strings.Contains(err.Error(), "user not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		if strings.Contains(err.Error(), "role") && strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove role", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Role removed successfully",
		"user_id": userID,
		"role_id": req.RoleID,
	})
}

// GetAllTopics godoc
// @Summary Get all topics (Admin only)
// @Description Get all topics in the system with pagination, search, and filters (Admin privileges required)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)" minimum(1)
// @Param limit query int false "Number of items per page (default: 20, max: 100)" minimum(1) maximum(100)
// @Param search query string false "Search term for topic name, description, or creator username"
// @Param creator_id query string false "Filter by creator user ID"
// @Param sync_status query string false "Filter by sync status"
// @Param is_active query bool false "Filter by is_active status"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/admin/topics [get]
func (h *AdminHandler) GetAllTopics(c *gin.Context) {
	// Check if user is admin
	user := c.MustGet("user").(*models.User)
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}

	// Parse query parameters
	page, pageSize := utils.ParsePaginationFromQuery(c.Query("page"), c.Query("limit"))
	search := c.DefaultQuery("search", "")
	creatorID := c.DefaultQuery("creator_id", "")
	syncStatus := c.DefaultQuery("sync_status", "")

	// Parse is_active filter
	var isActive *bool
	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		if isActiveStr == "true" {
			val := true
			isActive = &val
		} else if isActiveStr == "false" {
			val := false
			isActive = &val
		}
	}

	// Get topics with pagination and filters
	topics, total, err := h.topicService.GetAllTopicsPaginated(page, pageSize, search, creatorID, syncStatus, isActive)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get topics", "details": err.Error()})
		return
	}

	// Collect topic IDs for batch loading assigned users
	topicIDs := make([]string, len(topics))
	for i, topic := range topics {
		topicIDs[i] = topic.ID
	}

	// Batch load assigned users for all topics
	assignedUsersMap, err := h.topicUserRepo.GetByTopicIDs(topicIDs)
	if err != nil {
		// Log error but continue - assigned users will be empty
		assignedUsersMap = make(map[string][]*models.TopicUser)
	}

	// Convert to responses
	responses := make([]gin.H, len(topics))
	for i, topic := range topics {
		// Build creator info
		creatorInfo := gin.H{}
		if topic.UserProfile.ID != "" && topic.UserProfile.User.ID != "" {
			creatorInfo = gin.H{
				"id":       topic.UserProfile.User.ID,
				"username": topic.UserProfile.User.Username,
			}
		}

		// Build assigned users list
		assignedUsers := make([]gin.H, 0)
		if topicAssignments, exists := assignedUsersMap[topic.ID]; exists {
			for _, assignment := range topicAssignments {
				if assignment.User.ID != "" {
					assignedUsers = append(assignedUsers, gin.H{
						"id":              assignment.User.ID,
						"username":        assignment.User.Username,
						"first_name":      assignment.User.FirstName,
						"last_name":       assignment.User.LastName,
						"assigned_at":     assignment.AssignedAt.Format("2006-01-02T15:04:05Z07:00"),
						"permission_type": assignment.PermissionType,
					})
				}
			}
		}

		// Format dates
		lastSyncedAt := ""
		if topic.LastSyncedAt != nil {
			lastSyncedAt = topic.LastSyncedAt.Format("2006-01-02T15:04:05Z07:00")
		}

		responses[i] = gin.H{
			"id":              topic.ID,
			"user_profile_id": topic.UserProfileID,
			"name":            topic.Name,
			"description":     topic.Description,
			"is_active":       topic.IsActive,
			"sync_status":     topic.SyncStatus,
			"sync_error":      topic.SyncError,
			"created_at":      topic.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			"updated_at":      topic.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			"last_synced_at":  lastSyncedAt,
			"creator":         creatorInfo,
			"assigned_users":  assignedUsers,
		}
	}

	// Calculate pagination info
	paginationInfo := utils.CalculatePaginationInfo(int(total), page, pageSize)

	response := gin.H{
		"data":         responses,
		"total":        total,
		"page":         paginationInfo.Page,
		"limit":        paginationInfo.PageSize,
		"total_pages":  paginationInfo.TotalPages,
		"has_next":     paginationInfo.HasNext,
		"has_previous": paginationInfo.HasPrevious,
	}

	c.JSON(http.StatusOK, response)
}

// AssignTopicToUser godoc
// @Summary Assign topic to user (Admin only)
// @Description Assign a topic to a user so they can access it (Admin privileges required)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Topic ID"
// @Param request body models.AssignTopicRequest true "Topic assignment request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{} "Conflict: User already assigned to this topic"
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/admin/topics/{id}/assign [post]
func (h *AdminHandler) AssignTopicToUser(c *gin.Context) {
	user := c.MustGet("user").(*models.User)
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}

	topicID := c.Param("id")
	if topicID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Topic ID is required"})
		return
	}

	var req models.AssignTopicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Validate permission_type
	permissionType := req.PermissionType
	if permissionType == "" {
		permissionType = "read" // Default
	}
	if permissionType != "read" && permissionType != "write" && permissionType != "full" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid permission_type. Must be 'read', 'write', or 'full'"})
		return
	}

	// Check if topic exists
	topic, err := h.topicService.GetTopicByID(topicID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Topic not found"})
		return
	}

	// Check if user is trying to assign to creator (creator already has access)
	if topic.UserProfile.UserID == req.UserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User is already the creator of this topic"})
		return
	}

	// Check if already assigned
	isAssigned, err := h.topicUserRepo.IsUserAssigned(topicID, req.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check assignment", "details": err.Error()})
		return
	}
	if isAssigned {
		c.JSON(http.StatusConflict, gin.H{"error": "User is already assigned to this topic"})
		return
	}

	// Create assignment
	topicUser := &models.TopicUser{
		TopicID:        topicID,
		UserID:         req.UserID,
		AssignedBy:     &user.ID,
		PermissionType: permissionType,
	}

	if err := h.topicUserRepo.Create(topicUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign topic", "details": err.Error()})
		return
	}

	// NEW: Clone creator's script to assigned user
	// Get creator's user ID from topic.UserProfile (need to be sure it's loaded)
	// topic.UserProfile is loaded in GetTopicByID above
	creatorUserID := topic.UserProfile.UserID
	if creatorUserID != "" {
		if err := h.scriptService.CloneScript(creatorUserID, req.UserID, topicID); err != nil {
			logrus.Errorf("Failed to clone script from creator %s to user %s for topic %s: %v", creatorUserID, req.UserID, topicID, err)
			// Non-critical error, so we just log it and don't fail the request?
			// Or should we return warning? Let's just log for now to avoid breaking assignment if script fails.
		} else {
			logrus.Infof("Successfully cloned script from creator %s to user %s for topic %s", creatorUserID, req.UserID, topicID)
		}
	} else {
		logrus.Warnf("Creator user ID not found for topic %s, skipping script clone", topicID)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Topic assigned successfully",
		"topic_id":        topicID,
		"user_id":         req.UserID,
		"permission_type": permissionType,
	})
}

// GetTopicAssignedUsers godoc
// @Summary Get users assigned to a topic (Admin only)
// @Description Get list of all users assigned to a specific topic (Admin privileges required)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Topic ID"
// @Success 200 {array} models.TopicUserResponse
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/admin/topics/{id}/users [get]
func (h *AdminHandler) GetTopicAssignedUsers(c *gin.Context) {
	user := c.MustGet("user").(*models.User)
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}

	topicID := c.Param("id")
	if topicID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Topic ID is required"})
		return
	}

	// Check if topic exists
	_, err := h.topicService.GetTopicByID(topicID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Topic not found"})
		return
	}

	// Get assigned users
	topicUsers, err := h.topicUserRepo.GetByTopicID(topicID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get assigned users", "details": err.Error()})
		return
	}

	// Convert to response format
	responses := make([]models.TopicUserResponse, len(topicUsers))
	for i, tu := range topicUsers {
		assignedByUser := ""
		if tu.AssignedBy != nil {
			assignedBy, err := h.userRepo.GetByID(*tu.AssignedBy)
			if err == nil {
				assignedByUser = assignedBy.Username
			}
		}

		responses[i] = models.TopicUserResponse{
			ID:             tu.ID,
			TopicID:        tu.TopicID,
			UserID:         tu.UserID,
			Username:       tu.User.Username,
			FirstName:      tu.User.FirstName,
			LastName:       tu.User.LastName,
			AssignedBy:     tu.AssignedBy,
			AssignedByUser: &assignedByUser,
			AssignedAt:     tu.AssignedAt,
			PermissionType: tu.PermissionType,
		}
	}

	c.JSON(http.StatusOK, responses)
}

// RemoveTopicAssignment godoc
// @Summary Remove topic assignment from user (Admin only)
// @Description Remove a topic assignment from a user (Admin privileges required)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Topic ID"
// @Param user_id path string true "User ID"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/admin/topics/{id}/users/{user_id} [delete]
func (h *AdminHandler) RemoveTopicAssignment(c *gin.Context) {
	user := c.MustGet("user").(*models.User)
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}

	topicID := c.Param("id")
	userID := c.Param("user_id")
	if topicID == "" || userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Topic ID and User ID are required"})
		return
	}

	// Check if topic exists
	_, err := h.topicService.GetTopicByID(topicID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Topic not found"})
		return
	}

	// Check if assignment exists
	isAssigned, err := h.topicUserRepo.IsUserAssigned(topicID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check assignment", "details": err.Error()})
		return
	}
	if !isAssigned {
		c.JSON(http.StatusNotFound, gin.H{"error": "User is not assigned to this topic"})
		return
	}

	// Remove assignment
	if err := h.topicUserRepo.Delete(topicID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove assignment", "details": err.Error()})
		return
	}

	// NEW: Delete user's script for this topic
	if err := h.scriptService.DeleteScript(topicID, userID); err != nil {
		logrus.Errorf("Failed to delete script for user %s topic %s after unassignment: %v", userID, topicID, err)
		// Non-critical error, just log it.
	} else {
		logrus.Infof("Successfully deleted script for user %s topic %s after unassignment", userID, topicID)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Topic assignment removed successfully",
		"topic_id": topicID,
		"user_id":  userID,
	})
}

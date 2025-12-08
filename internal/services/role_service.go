package services

import (
	"fmt"

	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/sirupsen/logrus"
)

type RoleService struct {
	roleRepo *repository.RoleRepository
	userRepo *repository.UserRepository
}

func NewRoleService(roleRepo *repository.RoleRepository, userRepo *repository.UserRepository) *RoleService {
	return &RoleService{
		roleRepo: roleRepo,
		userRepo: userRepo,
	}
}

// GetAllRoles returns all roles in the system
func (s *RoleService) GetAllRoles() ([]models.Role, error) {
	return s.roleRepo.GetAll()
}

// GetRoleByName retrieves a role by name
func (s *RoleService) GetRoleByName(name string) (*models.Role, error) {
	return s.roleRepo.GetByName(name)
}

// CreateRole creates a new role
func (s *RoleService) CreateRole(name, description string) (*models.Role, error) {
	// Check if role already exists
	exists, err := s.roleRepo.CheckNameExists(name)
	if err != nil {
		return nil, fmt.Errorf("failed to check role existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("role with name '%s' already exists", name)
	}

	role := &models.Role{
		Name:        name,
		Description: description,
	}

	if err := s.roleRepo.Create(role); err != nil {
		return nil, fmt.Errorf("failed to create role: %w", err)
	}

	return role, nil
}

// AssignRoleToUser assigns a role to a user by role ID
func (s *RoleService) AssignRoleToUser(userID string, roleID string) error {
	// Check if user exists
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Get role by ID
	role, err := s.roleRepo.GetByID(roleID)
	if err != nil {
		return fmt.Errorf("role with ID '%s' not found: %w", roleID, err)
	}

	// Assign role
	if err := s.roleRepo.AssignRoleToUser(user.ID, role.ID); err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}

	logrus.Infof("Assigned role '%s' (ID: %s) to user '%s'", role.Name, role.ID, user.Username)
	return nil
}

// AssignRoleToUserByName assigns a role to a user by role name (internal helper)
func (s *RoleService) AssignRoleToUserByName(userID string, roleName string) error {
	// Get role by name first
	role, err := s.roleRepo.GetByName(roleName)
	if err != nil {
		return fmt.Errorf("role '%s' not found: %w", roleName, err)
	}
	// Then assign by ID
	return s.AssignRoleToUser(userID, role.ID)
}

// RemoveRoleFromUser removes a role from a user by role ID
func (s *RoleService) RemoveRoleFromUser(userID string, roleID string) error {
	// Check if user exists
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Get role by ID
	role, err := s.roleRepo.GetByID(roleID)
	if err != nil {
		return fmt.Errorf("role with ID '%s' not found: %w", roleID, err)
	}

	// Remove role
	if err := s.roleRepo.RemoveRoleFromUser(user.ID, role.ID); err != nil {
		return fmt.Errorf("failed to remove role: %w", err)
	}

	logrus.Infof("Removed role '%s' (ID: %s) from user '%s'", role.Name, role.ID, user.Username)
	return nil
}

// GetUserRoles retrieves all roles for a user
func (s *RoleService) GetUserRoles(userID string) ([]models.Role, error) {
	return s.roleRepo.GetUserRoles(userID)
}

// UserHasRole checks if a user has a specific role
func (s *RoleService) UserHasRole(userID string, roleName string) (bool, error) {
	return s.roleRepo.UserHasRole(userID, roleName)
}

// GetUsersByRole retrieves all users with a specific role
func (s *RoleService) GetUsersByRole(roleName string) ([]models.User, error) {
	return s.roleRepo.GetUsersByRole(roleName)
}

// GetUserRoleResponse returns user with their role names
func (s *RoleService) GetUserRoleResponse(userID string) (*models.UserRoleResponse, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	roles, err := s.roleRepo.GetUserRoles(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	roleNames := make([]string, len(roles))
	for i, role := range roles {
		roleNames[i] = role.Name
	}

	return &models.UserRoleResponse{
		UserID:   user.ID,
		Username: user.Username,
		Roles:    roleNames,
	}, nil
}

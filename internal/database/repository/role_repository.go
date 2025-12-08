package repository

import (
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"gorm.io/gorm"
)

type RoleRepository struct {
	db *gorm.DB
}

func NewRoleRepository(db *gorm.DB) *RoleRepository {
	return &RoleRepository{db: db}
}

// Create creates a new role
func (r *RoleRepository) Create(role *models.Role) error {
	return r.db.Create(role).Error
}

// GetByID retrieves a role by ID
func (r *RoleRepository) GetByID(id string) (*models.Role, error) {
	var role models.Role
	err := r.db.First(&role, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// GetByName retrieves a role by name
func (r *RoleRepository) GetByName(name string) (*models.Role, error) {
	var role models.Role
	err := r.db.Where("name = ?", name).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// GetAll returns all roles
func (r *RoleRepository) GetAll() ([]models.Role, error) {
	var roles []models.Role
	err := r.db.Find(&roles).Error
	if err != nil {
		return nil, err
	}
	return roles, nil
}

// Update updates a role
func (r *RoleRepository) Update(role *models.Role) error {
	return r.db.Save(role).Error
}

// Delete deletes a role
func (r *RoleRepository) Delete(id string) error {
	return r.db.Delete(&models.Role{}, "id = ?", id).Error
}

// CheckNameExists checks if a role name already exists
func (r *RoleRepository) CheckNameExists(name string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Role{}).Where("name = ?", name).Count(&count).Error
	return count > 0, err
}

// AssignRoleToUser assigns a role to a user
func (r *RoleRepository) AssignRoleToUser(userID string, roleID string) error {
	// Check if assignment already exists
	var count int64
	err := r.db.Table("user_roles").
		Where("user_id = ? AND role_id = ?", userID, roleID).
		Count(&count).Error
	if err != nil {
		return err
	}
	if count > 0 {
		// Already assigned, return nil (idempotent)
		return nil
	}

	// Assign role
	return r.db.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)", userID, roleID).Error
}

// RemoveRoleFromUser removes a role from a user
func (r *RoleRepository) RemoveRoleFromUser(userID string, roleID string) error {
	return r.db.Exec("DELETE FROM user_roles WHERE user_id = ? AND role_id = ?", userID, roleID).Error
}

// GetUserRoles retrieves all roles for a user
func (r *RoleRepository) GetUserRoles(userID string) ([]models.Role, error) {
	var user models.User
	err := r.db.Preload("Roles").First(&user, "id = ?", userID).Error
	if err != nil {
		return nil, err
	}
	return user.Roles, nil
}

// UserHasRole checks if a user has a specific role
func (r *RoleRepository) UserHasRole(userID string, roleName string) (bool, error) {
	var count int64
	err := r.db.Table("user_roles").
		Joins("JOIN roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ? AND roles.name = ?", userID, roleName).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetUsersByRole retrieves all users with a specific role
func (r *RoleRepository) GetUsersByRole(roleName string) ([]models.User, error) {
	var users []models.User
	err := r.db.Table("users").
		Joins("JOIN user_roles ON users.id = user_roles.user_id").
		Joins("JOIN roles ON user_roles.role_id = roles.id").
		Where("roles.name = ?", roleName).
		Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}


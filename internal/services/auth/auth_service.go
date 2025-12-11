package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/onegreenvn/green-provider-services-backend/internal/services"
	"github.com/onegreenvn/green-provider-services-backend/internal/utils"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	userRepo           *repository.UserRepository
	refreshTokenRepo   *repository.RefreshTokenRepository
	userProfileService *services.UserProfileService
	roleService        *services.RoleService
	jwtSecret          []byte
	accessTokenTTL     time.Duration
	refreshTokenTTL    time.Duration
}

func NewAuthService(db *gorm.DB, roleService *services.RoleService) *AuthService {
	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	if len(jwtSecret) == 0 {
		jwtSecret = []byte("default-secret-key-change-in-production")
	}

	accessTokenTTL := 15 * time.Minute
	if ttl := os.Getenv("ACCESS_TOKEN_TTL"); ttl != "" {
		if parsed, err := time.ParseDuration(ttl); err == nil {
			accessTokenTTL = parsed
		}
	}

	refreshTokenTTL := 7 * 24 * time.Hour // 7 days
	if ttl := os.Getenv("REFRESH_TOKEN_TTL"); ttl != "" {
		if parsed, err := time.ParseDuration(ttl); err == nil {
			refreshTokenTTL = parsed
		}
	}

	logrus.Infof("Access token TTL: %f", accessTokenTTL.Hours())
	logrus.Infof("Refresh token TTL: %f", refreshTokenTTL.Hours())

	// Initialize UserProfileService
	userProfileRepo := repository.NewUserProfileRepository(db)
	appRepo := repository.NewAppRepository(db)
	geminiAccountRepo := repository.NewGeminiAccountRepository(db)
	boxRepo := repository.NewBoxRepository(db)
	userProfileService := services.NewUserProfileService(userProfileRepo, appRepo, geminiAccountRepo, boxRepo)

	return &AuthService{
		userRepo:           repository.NewUserRepository(db),
		refreshTokenRepo:   repository.NewRefreshTokenRepository(db),
		userProfileService: userProfileService,
		roleService:        roleService,
		jwtSecret:          jwtSecret,
		accessTokenTTL:     accessTokenTTL,
		refreshTokenTTL:    refreshTokenTTL,
	}
}

// Register registers a new user
func (s *AuthService) Register(req *models.RegisterRequest) (*models.AuthResponse, error) {
	// Check if username already exists
	exists, err := s.userRepo.CheckUsernameExists(req.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to check username: %w", err)
	}
	if exists {
		return nil, errors.New("username already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &models.User{
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		IsActive:     true,
		TokenVersion: 0,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Assign role to new user
	// If RoleID is provided, assign that role; otherwise, assign default "topic_user"
	// Profile creation is handled by RoleService when assigning topic_creator role
	if s.roleService != nil {
		if req.RoleID != "" {
			// Assign specified role
			if err := s.roleService.AssignRoleToUser(user.ID, req.RoleID); err != nil {
				logrus.Warnf("Failed to assign role %s to user %s: %v", req.RoleID, user.ID, err)
				// Don't fail registration if role assignment fails, but try to assign default role
				if err := s.roleService.AssignRoleToUserByName(user.ID, "topic_user"); err != nil {
					logrus.Warnf("Failed to assign default role 'topic_user' to user %s: %v", user.ID, err)
				} else {
					logrus.Infof("Assigned default role 'topic_user' to user %s after specified role assignment failed", user.ID)
				}
			} else {
				logrus.Infof("Successfully assigned role %s to user %s", req.RoleID, user.ID)
			}
		} else {
			// Assign default role "topic_user"
			if err := s.roleService.AssignRoleToUserByName(user.ID, "topic_user"); err != nil {
				logrus.Warnf("Failed to assign default role 'topic_user' to user %s: %v", user.ID, err)
				// Don't fail registration if role assignment fails
			} else {
				logrus.Infof("Successfully assigned default role 'topic_user' to user %s", user.ID)
			}
		}
	}

	// Generate tokens
	return s.generateAuthResponse(user)
}

// Login authenticates a user
func (s *AuthService) Login(req *models.LoginRequest) (*models.AuthResponse, error) {
	// Get user by username
	user, err := s.userRepo.GetByUsername(req.Username)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Check if user is active
	if !user.IsActive {
		return nil, errors.New("account is deactivated")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Update last login
	if err := s.userRepo.UpdateLastLogin(user.ID); err != nil {
		// Log error but don't fail login
		fmt.Printf("Failed to update last login: %v\n", err)
	}

	// Generate tokens
	return s.generateAuthResponse(user)
}

// RefreshToken refreshes an access token using a refresh token
func (s *AuthService) RefreshToken(refreshTokenStr string) (*models.AuthResponse, error) {
	// Get refresh token from database
	refreshToken, err := s.refreshTokenRepo.GetByToken(refreshTokenStr)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	// Check if token is expired
	if refreshToken.ExpiresAt.Before(time.Now()) {
		// Delete expired token
		s.refreshTokenRepo.RevokeToken(refreshTokenStr)
		return nil, errors.New("refresh token expired")
	}

	// Get user
	user, err := s.userRepo.GetByID(refreshToken.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Check if user is active
	if !user.IsActive {
		return nil, errors.New("account is deactivated")
	}

	// Revoke the used refresh token
	if err := s.refreshTokenRepo.RevokeToken(refreshTokenStr); err != nil {
		return nil, fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	// Generate new tokens
	return s.generateAuthResponse(user)
}

// Logout logs out a user
func (s *AuthService) Logout(refreshTokenStr string, userID string) error {
	if refreshTokenStr != "" {
		return s.refreshTokenRepo.RevokeToken(refreshTokenStr)
	} else {
		if err := s.userRepo.IncrementTokenVersion(userID); err != nil {
			return fmt.Errorf("failed to increment token version: %w", err)
		}
		if err := s.refreshTokenRepo.RevokeAllUserTokens(userID); err != nil {
			return fmt.Errorf("failed to revoke all refresh tokens: %w", err)
		}
		return nil
	}
}

// ValidateToken validates and parses a JWT token
func (s *AuthService) ValidateToken(tokenString string) (*models.TokenInfo, error) {
	token, err := jwt.ParseWithClaims(tokenString, &models.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if claims, ok := token.Claims.(*models.JWTClaims); ok && token.Valid {
		// Get user to check token version
		user, err := s.userRepo.GetByID(claims.UserID)
		if err != nil {
			return nil, errors.New("user not found")
		}

		// Check if user is active
		if !user.IsActive {
			return nil, errors.New("account is deactivated")
		}

		// Check token version
		if claims.TokenVersion != user.TokenVersion {
			return nil, errors.New("token version mismatch")
		}

		return &models.TokenInfo{
			UserID:       claims.UserID,
			Username:     claims.Username,
			TokenVersion: claims.TokenVersion,
			ExpiresAt:    claims.ExpiresAt.Time,
		}, nil
	}

	return nil, errors.New("invalid token claims")
}

// generateAuthResponse generates access and refresh tokens for a user
func (s *AuthService) generateAuthResponse(user *models.User) (*models.AuthResponse, error) {
	// Load user roles if not already loaded
	if len(user.Roles) == 0 && s.roleService != nil {
		roles, err := s.roleService.GetUserRoles(user.ID)
		if err != nil {
			logrus.Warnf("Failed to load roles for user %s: %v", user.ID, err)
			// Continue without roles if loading fails
			user.Roles = []models.Role{}
		} else {
			user.Roles = roles
		}
	}

	// Generate access token
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.accessTokenTTL.Seconds()),
		User:         *user,
	}, nil
}

// generateAccessToken generates a JWT access token
func (s *AuthService) generateAccessToken(user *models.User) (string, error) {
	claims := &models.JWTClaims{
		UserID:       user.ID,
		Username:     user.Username,
		TokenVersion: user.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "green-anti-detect-browser-backend",
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// generateRefreshToken generates a refresh token and stores it in the database
func (s *AuthService) generateRefreshToken(user *models.User) (string, error) {
	// Generate random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	// Create refresh token record
	refreshToken := &models.RefreshToken{
		Token:     token,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(s.refreshTokenTTL),
		IsRevoked: false,
	}

	if err := s.refreshTokenRepo.Create(refreshToken); err != nil {
		return "", fmt.Errorf("failed to store refresh token: %w", err)
	}

	return token, nil
}

// CreateAdminUser creates an admin user if it doesn't exist
func (s *AuthService) CreateAdminUser() error {
	// Check if admin user already exists
	existingUser, err := s.userRepo.GetByUsername("admin")
	if err == nil && existingUser != nil {
		return nil // Admin user already exists
	}

	// Hash password
	password := "Helloworld@@123"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Create admin user
	adminUser := &models.User{
		Username:     "admin",
		PasswordHash: string(hashedPassword),
		FirstName:    "Admin",
		LastName:     "User",
		IsActive:     true,
		IsAdmin:      true,
		TokenVersion: 0,
	}

	if err := s.userRepo.Create(adminUser); err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	return nil
}

// SetUserActive sets the active status of a user
func (s *AuthService) SetUserActive(userID string, isActive bool) error {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	user.IsActive = isActive
	return s.userRepo.Update(user)
}

// ChangePassword changes a user's password
func (s *AuthService) ChangePassword(userID string, currentPassword, newPassword string) error {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return errors.New("current password is incorrect")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password and increment token version
	user.PasswordHash = string(hashedPassword)
	user.TokenVersion++

	return s.userRepo.Update(user)
}

// ResetPassword resets a user's password (admin only)
func (s *AuthService) ResetPassword(userID string, newPassword string) error {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password and increment token version
	user.PasswordHash = string(hashedPassword)
	user.TokenVersion++

	return s.userRepo.Update(user)
}

// GetAllUsers returns all users (admin only) with pagination and search
func (s *AuthService) GetAllUsers(page, pageSize int, search string) ([]*models.User, int64, error) {
	// Validate and normalize pagination parameters
	page, pageSize = utils.ValidateAndNormalizePagination(page, pageSize)

	users, total, err := s.userRepo.GetAllUsers(page, pageSize, search)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get users: %w", err)
	}

	// Convert to pointers for consistency
	userPointers := make([]*models.User, len(users))
	for i := range users {
		userPointers[i] = &users[i]
	}

	return userPointers, total, nil
}

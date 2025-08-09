package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"green-anti-detect-browser-backend-v1/internal/database/repository"
	"green-anti-detect-browser-backend-v1/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo         *repository.UserRepository
	refreshTokenRepo *repository.RefreshTokenRepository
	jwtSecret        []byte
	accessTokenTTL   time.Duration
	refreshTokenTTL  time.Duration
}

func NewAuthService(userRepo *repository.UserRepository, refreshTokenRepo *repository.RefreshTokenRepository) *AuthService {
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

	return &AuthService{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		jwtSecret:        jwtSecret,
		accessTokenTTL:   accessTokenTTL,
		refreshTokenTTL:  refreshTokenTTL,
	}
}

// Register registers a new user
func (s *AuthService) Register(req *models.RegisterRequest, userAgent, ipAddress string) (*models.AuthResponse, error) {
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

	// Generate tokens
	return s.generateAuthResponse(user, userAgent, ipAddress)
}

// Login authenticates a user
func (s *AuthService) Login(req *models.LoginRequest, userAgent, ipAddress string) (*models.AuthResponse, error) {
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
	return s.generateAuthResponse(user, userAgent, ipAddress)
}

// RefreshToken refreshes an access token using a refresh token
func (s *AuthService) RefreshToken(refreshTokenStr string, userAgent, ipAddress string) (*models.AuthResponse, error) {
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
	return s.generateAuthResponse(user, userAgent, ipAddress)
}

// Logout logs out a user
func (s *AuthService) Logout(refreshTokenStr string, userID uint) error {
	if refreshTokenStr != "" {
		// Revoke specific refresh token
		return s.refreshTokenRepo.RevokeToken(refreshTokenStr)
	} else {
		// Revoke all refresh tokens for the user
		return s.refreshTokenRepo.RevokeAllUserTokens(userID)
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
func (s *AuthService) generateAuthResponse(user *models.User, userAgent, ipAddress string) (*models.AuthResponse, error) {
	// Generate access token
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err := s.generateRefreshToken(user, userAgent, ipAddress)
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
			Subject:   strconv.FormatUint(uint64(user.ID), 10),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// generateRefreshToken generates a refresh token and stores it in the database
func (s *AuthService) generateRefreshToken(user *models.User, userAgent, ipAddress string) (string, error) {
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
		UserAgent: userAgent,
		IPAddress: ipAddress,
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
	password := "admin123"
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

// SetUserActive sets the active status of a user (admin only)
func (s *AuthService) SetUserActive(userID uint, isActive bool) error {
	// Get user to check if exists
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Update user active status
	user.IsActive = isActive
	if err := s.userRepo.Update(user); err != nil {
		return fmt.Errorf("failed to update user status: %w", err)
	}

	return nil
}

// ChangePassword changes the current user's password
func (s *AuthService) ChangePassword(userID uint, currentPassword, newPassword string) error {
	// Get user
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

	// Update password
	user.PasswordHash = string(hashedPassword)
	if err := s.userRepo.Update(user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// ResetPassword resets a user's password (admin only)
func (s *AuthService) ResetPassword(userID uint, newPassword string) error {
	// Get user
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	user.PasswordHash = string(hashedPassword)
	if err := s.userRepo.Update(user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// GetAllUsers returns all users (admin only)
func (s *AuthService) GetAllUsers() ([]*models.User, error) {
	users, err := s.userRepo.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}
	return users, nil
}

// DeleteUser deletes a user (admin only)
func (s *AuthService) DeleteUser(userID uint) error {
	// Check if user exists
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Delete user
	if err := s.userRepo.Delete(user.ID); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

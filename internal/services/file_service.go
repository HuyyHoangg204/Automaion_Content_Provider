package services

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/sirupsen/logrus"
)

type FileService struct {
	fileRepo   *repository.FileRepository
	baseURL    string
	storageDir string
	jwtSecret  []byte
}

// FileDownloadClaims represents JWT claims for file download token
type FileDownloadClaims struct {
	FileID string `json:"file_id"`
	jwt.RegisteredClaims
}

func NewFileService(fileRepo *repository.FileRepository, baseURL string) *FileService {
	// Default storage directory
	storageDir := os.Getenv("FILE_STORAGE_DIR")
	if storageDir == "" {
		storageDir = "./storage/files"
	}

	// Create storage directory if it doesn't exist
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		logrus.Warnf("Failed to create storage directory %s: %v", storageDir, err)
	}

	// Get JWT secret for signing download tokens
	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	if len(jwtSecret) == 0 {
		jwtSecret = []byte("default-secret-key-change-in-production")
		logrus.Warn("JWT_SECRET not set, using default secret for file download tokens")
	}

	return &FileService{
		fileRepo:   fileRepo,
		baseURL:    baseURL,
		storageDir: storageDir,
		jwtSecret:  jwtSecret,
	}
}

// UploadFile handles file upload and saves to storage
func (s *FileService) UploadFile(userID string, fileHeader *multipart.FileHeader, req *models.FileUploadRequest) (*models.File, error) {
	// Open uploaded file
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer file.Close()

	// Generate unique filename
	fileID := uuid.New().String()
	ext := filepath.Ext(fileHeader.Filename)
	fileName := fileID + ext
	originalName := fileHeader.Filename

	// Create user-specific directory
	userDir := filepath.Join(s.storageDir, userID)
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create user directory: %w", err)
	}

	// Full file path
	filePath := filepath.Join(userDir, fileName)

	// Create destination file
	dst, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	// Copy file content
	fileSize, err := io.Copy(dst, file)
	if err != nil {
		os.Remove(filePath) // Clean up on error
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// Get MIME type
	mimeType := fileHeader.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Create file record in database
	fileModel := &models.File{
		UserID:       userID,
		FileName:     fileName,
		OriginalName: originalName,
		MimeType:     mimeType,
		FileSize:     fileSize,
		FilePath:     filePath,
	}

	if err := s.fileRepo.Create(fileModel); err != nil {
		os.Remove(filePath) // Clean up on error
		return nil, fmt.Errorf("failed to save file record: %w", err)
	}

	logrus.Infof("File uploaded successfully: %s (ID: %s, Size: %d bytes)", originalName, fileModel.ID, fileSize)

	return fileModel, nil
}

// GetFile retrieves a file by ID
func (s *FileService) GetFile(fileID string, userID string) (*models.File, error) {
	file, err := s.fileRepo.GetByID(fileID)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}

	// Check access permission - user can only access their own files
	if file.UserID != userID {
		return nil, fmt.Errorf("access denied: file does not belong to user")
	}

	return file, nil
}

// DownloadFile returns the file content for download (requires user authentication)
func (s *FileService) DownloadFile(fileID string, userID string) (*models.File, *os.File, error) {
	file, err := s.GetFile(fileID, userID)
	if err != nil {
		return nil, nil, err
	}

	// Open file from storage
	f, err := os.Open(file.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, f, nil
}

// DownloadFileByToken returns the file content for download using token (no user check)
func (s *FileService) DownloadFileByToken(fileID string) (*models.File, *os.File, error) {
	// Get file without user check (token already validated)
	file, err := s.fileRepo.GetByID(fileID)
	if err != nil {
		return nil, nil, fmt.Errorf("file not found: %w", err)
	}

	// Open file from storage
	f, err := os.Open(file.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, f, nil
}

// GetDownloadURL generates download URL for a file (requires authentication)
func (s *FileService) GetDownloadURL(fileID string) string {
	return fmt.Sprintf("%s/api/v1/files/%s/download", strings.TrimSuffix(s.baseURL, "/"), fileID)
}

// GenerateSignedDownloadURL generates a signed download URL with token for automation backend
// Token expires in 1 hour by default
func (s *FileService) GenerateSignedDownloadURL(fileID string) (string, error) {
	// Verify file exists
	file, err := s.fileRepo.GetByID(fileID)
	if err != nil {
		return "", fmt.Errorf("file not found: %w", err)
	}

	// Create JWT token with file ID
	claims := &FileDownloadClaims{
		FileID: file.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)), // Token expires in 1 hour
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "green-provider-services-backend",
			Subject:   file.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	// Generate signed URL with token in query param
	downloadURL := fmt.Sprintf("%s/api/v1/files/%s/download?token=%s", strings.TrimSuffix(s.baseURL, "/"), fileID, tokenString)
	return downloadURL, nil
}

// ValidateDownloadToken validates a download token and returns the file ID
func (s *FileService) ValidateDownloadToken(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &FileDownloadClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}

	if claims, ok := token.Claims.(*FileDownloadClaims); ok && token.Valid {
		return claims.FileID, nil
	}

	return "", fmt.Errorf("invalid token claims")
}

// GetUserFiles retrieves all files for a user
func (s *FileService) GetUserFiles(userID string) ([]*models.File, error) {
	return s.fileRepo.GetByUserID(userID)
}

// FileToResponse converts File model to FileResponse
func (s *FileService) FileToResponse(file *models.File) models.FileResponse {
	return models.FileResponse{
		ID:           file.ID,
		UserID:       file.UserID,
		FileName:     file.FileName,
		OriginalName: file.OriginalName,
		MimeType:     file.MimeType,
		FileSize:     file.FileSize,
		DownloadURL:  s.GetDownloadURL(file.ID),
		CreatedAt:    file.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    file.UpdatedAt.Format(time.RFC3339),
	}
}

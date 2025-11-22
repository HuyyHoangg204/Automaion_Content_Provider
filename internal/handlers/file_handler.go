package handlers

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/onegreenvn/green-provider-services-backend/internal/services"
	"gorm.io/gorm"
)

type FileHandler struct {
	fileService *services.FileService
}

func NewFileHandler(db *gorm.DB, baseURL string) *FileHandler {
	fileRepo := repository.NewFileRepository(db)
	fileService := services.NewFileService(fileRepo, baseURL)

	return &FileHandler{
		fileService: fileService,
	}
}

// UploadFile godoc
// @Summary Upload file(s)
// @Description Upload one or multiple files to the server. Returns file IDs that can be used in knowledge_files when creating topics.
// @Description Supports both single file (form field: "file") and multiple files (form field: "files[]")
// @Tags files
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file false "Single file to upload"
// @Param files formData file false "Multiple files to upload (files[])"
// @Success 201 {object} map[string]interface{} "Single file: {file: FileResponse}, Multiple files: {files: []FileResponse}"
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/files/upload [post]
func (h *FileHandler) UploadFile(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	// Parse form data (category, etc.) - use PostForm to avoid consuming multipart reader
	var req models.FileUploadRequest
	if category := c.PostForm("category"); category != "" {
		req.Category = category
	}

	// Check for multiple files first - parse multipart form
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse multipart form", "details": err.Error()})
		return
	}

	// Try to get files with different possible keys
	var files []*multipart.FileHeader

	// Check all possible keys in form files
	if formFiles := form.File["files[]"]; len(formFiles) > 0 {
		files = formFiles
	} else if formFiles := form.File["files"]; len(formFiles) > 0 {
		files = formFiles
	} else {
		// Debug: check what keys are available
		allKeys := make([]string, 0)
		for key := range form.File {
			allKeys = append(allKeys, key)
		}
		// If no files found with expected keys, try to get from form directly
		if len(allKeys) > 0 {
			// Use first available file key
			files = form.File[allKeys[0]]
		}
	}

	// If no multiple files found, check for single file
	if len(files) == 0 {
		fileHeader, err := c.FormFile("file")
		if err != nil {
			// Debug info
			debugKeys := make([]string, 0)
			if form != nil {
				for key := range form.File {
					debugKeys = append(debugKeys, key)
				}
			}
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "No file provided. Use 'file' for single file or 'files[]' for multiple files.",
				"details": err.Error(),
				"debug":   fmt.Sprintf("Available file keys in form: %v", debugKeys),
			})
			return
		}

		// Single file upload
		file, err := h.fileService.UploadFile(userID, fileHeader, &req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file", "details": err.Error()})
			return
		}

		response := h.fileService.FileToResponse(file)
		c.JSON(http.StatusCreated, gin.H{"file": response})
		return
	}

	// Multiple files upload
	uploadedFiles := make([]models.FileResponse, 0, len(files))
	var uploadErrors []string

	for _, fileHeader := range files {
		file, err := h.fileService.UploadFile(userID, fileHeader, &req)
		if err != nil {
			uploadErrors = append(uploadErrors, fmt.Sprintf("%s: %v", fileHeader.Filename, err))
			continue
		}

		response := h.fileService.FileToResponse(file)
		uploadedFiles = append(uploadedFiles, response)
	}

	if len(uploadedFiles) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to upload all files",
			"details": uploadErrors,
		})
		return
	}

	// Return results - include errors if some files failed
	result := gin.H{
		"files": uploadedFiles,
		"count": len(uploadedFiles),
	}

	if len(uploadErrors) > 0 {
		result["errors"] = uploadErrors
		result["message"] = fmt.Sprintf("Uploaded %d/%d files. Some files failed to upload.", len(uploadedFiles), len(files))
		c.JSON(http.StatusPartialContent, result)
	} else {
		c.JSON(http.StatusCreated, result)
	}
}

// DownloadFile godoc
// @Summary Download a file
// @Description Download a file by ID. Files can only be downloaded by the owner.
// @Tags files
// @Produce application/octet-stream
// @Security BearerAuth
// @Param id path string true "File ID"
// @Success 200 {file} file
// @Failure 404 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/files/{id}/download [get]
func (h *FileHandler) DownloadFile(c *gin.Context) {
	userID := c.GetString("user_id")
	fileID := c.Param("id")

	// Get file and open it
	file, f, err := h.fileService.DownloadFile(fileID, userID)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		} else {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		}
		return
	}
	defer f.Close()

	// Set headers for download
	c.Header("Content-Disposition", `attachment; filename="`+file.OriginalName+`"`)
	c.Header("Content-Type", file.MimeType)
	c.Header("Content-Length", string(rune(file.FileSize)))

	// Stream file to response
	c.DataFromReader(http.StatusOK, file.FileSize, file.MimeType, f, nil)
}

// GetMyFiles godoc
// @Summary Get user's files
// @Description Get all files uploaded by the current user
// @Tags files
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.FileResponse
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/files [get]
func (h *FileHandler) GetMyFiles(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	files, err := h.fileService.GetUserFiles(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get files", "details": err.Error()})
		return
	}

	responses := make([]models.FileResponse, len(files))
	for i, file := range files {
		responses[i] = h.fileService.FileToResponse(file)
	}

	c.JSON(http.StatusOK, responses)
}

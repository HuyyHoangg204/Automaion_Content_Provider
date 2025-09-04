package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/services/excel"
	"gorm.io/gorm"
)

// ExcelHandler handles HTTP requests related to Excel operations
type ExcelHandler struct {
	excelService *excel.Service
	exportsDir   string
	tempDir      string
	basePath     string
}

// NewExcelHandler creates a new ExcelHandler instance
func NewExcelHandler(db *gorm.DB, exportsDir, tempDir, basePath string) *ExcelHandler {
	flowGroupRepo := repository.NewFlowGroupRepository(db)
	campaignRepo := repository.NewCampaignRepository(db)

	return &ExcelHandler{
		excelService: excel.NewExcelService(
			flowGroupRepo,
			campaignRepo,
			exportsDir,
			tempDir,
		),
		exportsDir: exportsDir,
		tempDir:    tempDir,
		basePath:   basePath,
	}
}

// ExportFlowGroups handles GET /api/v1/excel/export/flow-groups/:flowgroupid
// @Summary Export flow group to Excel
// @Description Export a specific flow group and its flows to an Excel file
// @Tags excel
// @Accept json
// @Produce json
// @Param flowgroupid path string true "Flow Group ID"
// @Success 302 {string} string "Redirect to download URL"
// @Failure 404 {object} map[string]interface{} "success: false, error: error message"
// @Failure 500 {object} map[string]interface{} "success: false, error: error message"
// @Router /api/v1/excel/export/flow-groups/{flowgroupid} [get]
func (h *ExcelHandler) ExportFlowGroups(c *gin.Context) {
	// Get flow group ID from URL parameter
	flowGroupID := c.Param("flowgroupid")

	// Validate flow group ID
	if flowGroupID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Flow group ID is required",
		})
		return
	}

	result, err := h.excelService.ExportFlowGroupToExcel(flowGroupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	if !result.Success {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   result.Message,
		})
		return
	}

	// Redirect to the download URL
	downloadURL := fmt.Sprintf("%s/api/v1/excel/download/%s", h.basePath, result.Filename)
	c.Redirect(http.StatusFound, downloadURL)
}

// DownloadExcelFile handles GET /api/v1/excel/download/:filename
// @Summary Download Excel file
// @Description Download a previously exported Excel file
// @Tags excel
// @Accept json
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Security BearerAuth
// @Param filename path string true "Excel filename"
// @Success 200 {file} binary "Excel file"
// @Failure 404 {object} map[string]interface{} "success: false, error: error message"
// @Failure 500 {object} map[string]interface{} "success: false, error: error message"
// @Router /api/v1/excel/download/{filename} [get]
func (h *ExcelHandler) DownloadExcelFile(c *gin.Context) {
	filename := c.Param("filename")
	filePath := filepath.Join(h.exportsDir, filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "File not found",
		})
		return
	}

	// Set headers for file download
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Expires", "0")
	c.Header("Cache-Control", "must-revalidate")
	c.Header("Pragma", "public")

	// Serve the file
	c.File(filePath)
}

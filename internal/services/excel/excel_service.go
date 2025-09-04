package excel

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/xuri/excelize/v2"
)

// Service handles Excel operations for phone data
type Service struct {
	flowGroupRepo *repository.FlowGroupRepository
	campaignRepo  *repository.CampaignRepository
	exportsDir    string
	tempDir       string
}

// NewExcelService creates a new Excel service instance
func NewExcelService(
	flowGroupRepo *repository.FlowGroupRepository,
	campaignRepo *repository.CampaignRepository,
	exportsDir, tempDir string) *Service {
	// Create exports directory if it doesn't exist
	if _, err := os.Stat(exportsDir); os.IsNotExist(err) {
		os.MkdirAll(exportsDir, 0755)
	}

	// Create temp directory if it doesn't exist
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		os.MkdirAll(tempDir, 0755)
	}

	return &Service{
		flowGroupRepo: flowGroupRepo,
		campaignRepo:  campaignRepo,
		exportsDir:    exportsDir,
		tempDir:       tempDir,
	}
}

// ExportResult contains the result of an export operation
type ExportResult struct {
	Success  bool
	Message  string
	Filename string
}

// ImportResult contains the result of an import operation
type ImportResult struct {
	Success         bool
	Message         string
	RecordsCount    int
	NewColumnsCount int
	NewColumns      []string
}

// ExportFlowGroupToExcel exports a specific flow group and its flows to an Excel file
func (s *Service) ExportFlowGroupToExcel(flowGroupID string) (*ExportResult, error) {
	// Generate file path
	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("flow_group_%s_%d.xlsx", flowGroupID, timestamp)
	filePath := filepath.Join(s.exportsDir, filename)

	// Get the flow group with its flows for the user
	flowGroup, err := s.flowGroupRepo.GetByID(flowGroupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get flow group: %w", err)
	}

	if flowGroup == nil {
		return nil, fmt.Errorf("flow group with id %s not found", flowGroupID)
	}

	// Fetch campaign and script information
	var campaignName string
	var scriptName string
	if flowGroup.CampaignID != "" {
		campaign, err := s.campaignRepo.GetByID(flowGroup.CampaignID)
		if err == nil && campaign != nil {
			campaignName = campaign.Name
			scriptName = campaign.ScriptName
		}
	}

	// Create a new Excel file
	f := excelize.NewFile()

	// Create styles for different statuses
	errorStyle, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"D9D9D9"}, // Gray
			Pattern: 1,
		},
	})

	stoppedStyle, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"B4C6E7"}, // Light blue
			Pattern: 1,
		},
	})

	runningStyle, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"FFC000"}, // Orange
			Pattern: 1,
		},
	})

	scheduledStyle, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"FFFF00"}, // Yellow
			Pattern: 1,
		},
	})

	// Create Flows sheet
	flowSheetName := "Flows"
	defaultSheetName := f.GetSheetName(0)
	err = f.SetSheetName(defaultSheetName, flowSheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to rename sheet: %w", err)
	}
	f.SetActiveSheet(0)

	// Define columns for flows
	flowColumns := []string{
		"id", "name", "description", "flow_group_id", "flow_group_name",
		"campaign_id", "campaign_name", "script_name",
		"profile_id", "status", "message",
		"created_at", "updated_at",
	}

	// Write headers for flows
	for i, col := range flowColumns {
		cell := fmt.Sprintf("%s1", columnToLetter(i+1))
		f.SetCellValue(flowSheetName, cell, col)
	}

	// Apply header styling
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"FFFF00"},
			Pattern: 1,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
		},
	})
	if err == nil {
		// Apply style to header row
		f.SetCellStyle(flowSheetName, "A1", columnToLetter(len(flowColumns))+strconv.Itoa(1), headerStyle)
	}

	// Set column widths for flow sheet
	for i, col := range flowColumns {
		colLetter := columnToLetter(i + 1)
		width := 20.0 // Default width

		switch col {
		case "id", "flow_group_id", "profile_id", "campaign_id":
			width = 15.0
		case "name", "script_name", "campaign_name", "flow_group_name":
			width = 25.0
		case "description":
			width = 40.0
		case "status":
			width = 15.0
		case "message":
			width = 50.0
		case "created_at", "updated_at":
			width = 20.0
		}

		f.SetColWidth(flowSheetName, colLetter, colLetter, width)
	}

	// Write flow data
	if len(flowGroup.Flows) > 0 {
		for j, flow := range flowGroup.Flows {
			rowNum := j + 2 // Start from row 2 (after headers)

			// Extract execution data from flow results
			var resultMessage string

			if flow.Result != nil {
				// Extract message
				if msg, ok := flow.Result["message"].(string); ok {
					resultMessage = msg
				}

			}

			f.SetCellValue(flowSheetName, fmt.Sprintf("A%d", rowNum), flow.ID)
			f.SetCellValue(flowSheetName, fmt.Sprintf("B%d", rowNum), flow.Name)
			f.SetCellValue(flowSheetName, fmt.Sprintf("C%d", rowNum), flow.Description)
			f.SetCellValue(flowSheetName, fmt.Sprintf("D%d", rowNum), flow.FlowGroupID)
			f.SetCellValue(flowSheetName, fmt.Sprintf("E%d", rowNum), flowGroup.Name)
			f.SetCellValue(flowSheetName, fmt.Sprintf("F%d", rowNum), flowGroup.CampaignID)
			f.SetCellValue(flowSheetName, fmt.Sprintf("G%d", rowNum), campaignName)
			f.SetCellValue(flowSheetName, fmt.Sprintf("H%d", rowNum), scriptName)
			f.SetCellValue(flowSheetName, fmt.Sprintf("I%d", rowNum), flow.ProfileID)
			f.SetCellValue(flowSheetName, fmt.Sprintf("J%d", rowNum), flow.Status)
			f.SetCellValue(flowSheetName, fmt.Sprintf("K%d", rowNum), resultMessage)

			f.SetCellValue(flowSheetName, fmt.Sprintf("T%d", rowNum), flow.CreatedAt.Format(time.RFC3339))
			f.SetCellValue(flowSheetName, fmt.Sprintf("U%d", rowNum), flow.UpdatedAt.Format(time.RFC3339))

			// Apply row styling based on status
			switch strings.ToLower(flow.Status) {
			case "error":
				f.SetCellStyle(flowSheetName, fmt.Sprintf("A%d", rowNum), fmt.Sprintf("%s%d", columnToLetter(len(flowColumns)), rowNum), errorStyle)
			case "stopped":
				f.SetCellStyle(flowSheetName, fmt.Sprintf("A%d", rowNum), fmt.Sprintf("%s%d", columnToLetter(len(flowColumns)), rowNum), stoppedStyle)
			case "running":
				f.SetCellStyle(flowSheetName, fmt.Sprintf("A%d", rowNum), fmt.Sprintf("%s%d", columnToLetter(len(flowColumns)), rowNum), runningStyle)
			case "scheduled":
				f.SetCellStyle(flowSheetName, fmt.Sprintf("A%d", rowNum), fmt.Sprintf("%s%d", columnToLetter(len(flowColumns)), rowNum), scheduledStyle)
			}
		}
	} else {
		// No flows in the group
		f.SetCellValue(flowSheetName, "A2", "no flows found in this group")
	}

	// Save the file
	if err := f.SaveAs(filePath); err != nil {
		return nil, fmt.Errorf("failed to save Excel file: %w", err)
	}

	return &ExportResult{
		Success:  true,
		Message:  fmt.Sprintf("Successfully exported flow group %s with %d flows", flowGroup.ID, len(flowGroup.Flows)),
		Filename: filename,
	}, nil
}

// Helper function to convert column number to Excel column letter
func columnToLetter(col int) string {
	var result string
	for col > 0 {
		col--
		result = string(rune('A'+col%26)) + result
		col /= 26
	}
	return result
}

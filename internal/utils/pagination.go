package utils

import (
	"math"
)

// PaginationParams represents pagination parameters
type PaginationParams struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// PaginationResponse represents pagination response metadata
type PaginationResponse struct {
	Total       int  `json:"total"`
	Page        int  `json:"page"`
	PageSize    int  `json:"page_size"`
	TotalPages  int  `json:"total_pages"`
	HasNext     bool `json:"has_next"`
	HasPrevious bool `json:"has_previous"`
}

// ValidateAndNormalizePagination validates and normalizes pagination parameters
func ValidateAndNormalizePagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

// CalculatePaginationInfo calculates pagination metadata
func CalculatePaginationInfo(total, page, pageSize int) PaginationResponse {
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if totalPages == 0 {
		totalPages = 1
	}

	return PaginationResponse{
		Total:       total,
		Page:        page,
		PageSize:    pageSize,
		TotalPages:  totalPages,
		HasNext:     page < totalPages,
		HasPrevious: page > 1,
	}
}

// CalculateOffset calculates the offset for database queries
func CalculateOffset(page, pageSize int) int {
	return (page - 1) * pageSize
}

// ShouldGetAll checks if we should get all records (when pageSize is very large)
func ShouldGetAll(pageSize int) bool {
	return pageSize >= 1000
}

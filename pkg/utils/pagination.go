package utils

import "math"

// PaginationParams holds pagination request parameters
type PaginationParams struct {
	Page  int `form:"page"`
	Limit int `form:"limit"`
}

// PaginationMeta holds pagination response metadata
type PaginationMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalCount int64 `json:"totalCount"`
	TotalPages int   `json:"totalPages"`
}

// GetPaginationParams extracts page and limit with defaults
// Default: page=1, limit=0 (0 means no limit/all items)
func GetPaginationParams(page, limit int) PaginationParams {
	if page < 1 {
		page = 1
	}
	if limit < 0 {
		limit = 0
	}
	return PaginationParams{
		Page:  page,
		Limit: limit,
	}
}

// CalculateOffset returns the SQL offset
func (p PaginationParams) CalculateOffset() int {
	if p.Page < 1 {
		return 0
	}
	return (p.Page - 1) * p.Limit
}

// CalculateMeta generates pagination metadata
func CalculateMeta(totalCount int64, page, limit int) PaginationMeta {
	if limit <= 0 {
		return PaginationMeta{
			Page:       1,
			Limit:      int(totalCount),
			TotalCount: totalCount,
			TotalPages: 1,
		}
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(limit)))
	if totalPages < 0 {
		totalPages = 0
	}

	return PaginationMeta{
		Page:       page,
		Limit:      limit,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}
}

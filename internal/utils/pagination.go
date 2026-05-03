package utils

import (
	"net/http"
	"strconv"
)

const (
	DefaultPage    = 1
	DefaultPerPage = 20
	MaxPerPage     = 100
)

type Pagination struct {
	Page    int   `json:"page"`
	PerPage int   `json:"per_page"`
	Total   int64 `json:"total"`
	Limit   int64 `json:"-"`
	Offset  int64 `json:"-"`
}

// ParsePagination extracts page and per_page from query params with sane defaults.
func ParsePagination(r *http.Request) Pagination {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))

	if page < 1 {
		page = DefaultPage
	}
	if perPage < 1 {
		perPage = DefaultPerPage
	}
	if perPage > MaxPerPage {
		perPage = MaxPerPage
	}

	return Pagination{
		Page:    page,
		PerPage: perPage,
		Limit:   int64(perPage),
		Offset:  int64((page - 1) * perPage),
	}
}

// PaginatedResponse builds a standard paginated response envelope.
func PaginatedResponse(key string, data interface{}, p Pagination) map[string]interface{} {
	totalPages := int64(0)
	if p.Total > 0 {
		totalPages = (p.Total + int64(p.PerPage) - 1) / int64(p.PerPage)
	}

	return map[string]interface{}{
		key: data,
		"pagination": map[string]interface{}{
			"page":        p.Page,
			"per_page":    p.PerPage,
			"total":       p.Total,
			"total_pages": totalPages,
		},
	}
}

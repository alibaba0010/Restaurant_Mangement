package dto

// PaginationMeta provides pagination details
type PaginationMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// PaginatedResponse is a generic wrapper for paginated data
type PaginatedResponse struct {
	Title string         `json:"title,omitempty"`
	Data  interface{}    `json:"data"`
	Meta  PaginationMeta `json:"meta"`
}

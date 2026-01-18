package dto

// PaginationMeta provides pagination details
type PaginationMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// PaginatedResponse is a generic wrapper for paginated data
type PaginatedResponse[T any] struct {
	Title string         `json:"title,omitempty"`
	Data  []T            `json:"data"`
	Meta  PaginationMeta `json:"meta"`
}

// CursorMeta provides cursor-based pagination details
type CursorMeta struct {
	NextCursor string `json:"next_cursor"`
	HasMore    bool   `json:"has_more"`
	Total      int64  `json:"total,omitempty"` // Optional: keeping track of total count can be expensive, but sometimes requested
}

// CursorPaginatedResponse is a generic wrapper for cursor paginated data
type CursorPaginatedResponse[T any] struct {
	Title string     `json:"title,omitempty"`
	Data  []T        `json:"data"`
	Meta  CursorMeta `json:"meta"`
}

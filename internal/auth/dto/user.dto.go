package dto

// CurrentUserResponse represents the response structure for the current user endpoint
type CurrentUserResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Address   string `json:"address,omitempty"`
	Role      string `json:"role"`
	AvatarURL string `json:"avatar_url,omitempty"`
	PhoneNumber string  `json:"phone_number,omitempty"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// PaginationQuery holds query params for paging, filtering and sorting
type PaginationQuery struct {
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"page_size" form:"page_size"`
	Q        string `json:"q" form:"q"`           // search query (name/email)
	Role     string `json:"role" form:"role"`     // filter by role
	SortBy   string `json:"sort_by" form:"sort_by"`
	Order    string `json:"order" form:"order"`   // asc or desc
}

// PaginationMeta provides pagination details in responses
type PaginationMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// UsersListResponse is the response for listing users
type UsersListResponse struct {
	Title string                 `json:"title"`
	Data  []CurrentUserResponse  `json:"data"`
	Meta  PaginationMeta         `json:"meta"`
}
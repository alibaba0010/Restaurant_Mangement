package dto

import "github.com/alibaba0010/postgres-api/internal/common/types"

// CurrentUserResponse represents the response structure for the current user endpoint
type CurrentUserResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Role      types.UserRole `json:"role"`
	Address   string `json:"address,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
	PhoneNumber string  `json:"phone_number,omitempty"`
	Status    types.UserStatus `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// PaginationQuery holds query params for paging, filtering and sorting
type PaginationQuery struct {
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"page_size" form:"page_size"`
	Q        string `json:"q" form:"q"`           // search query (name/email)
	Role     types.UserRole `json:"role" form:"role"`     // filter by role

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

// UpdateUserRoleInput is used for updating a user's role and status
type UpdateUserRoleInput struct {
	Role   types.UserRole   `json:"role" validate:"omitempty,oneof=user admin management"`
	Status types.UserStatus `json:"status" validate:"omitempty,oneof=active inactive suspended pending"`
}

// UpdateUserResponse is a generic response for user updates
type UpdateUserResponse struct {
	Title string              `json:"title"`
	Data  CurrentUserResponse `json:"data"`
}
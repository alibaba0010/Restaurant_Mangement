package dto

import (
	commondto "github.com/alibaba0010/postgres-api/internal/common/dto"
	"github.com/alibaba0010/postgres-api/internal/common/types"
)

// UserData represents the response structure for user data
type UserData struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Email       string           `json:"email"`
	Role        types.UserRole   `json:"role"`
	Address     string           `json:"address,omitempty"`
	AvatarURL   string           `json:"avatar_url,omitempty"`
	PhoneNumber string           `json:"phone_number,omitempty"`
	Status      types.UserStatus `json:"status"`
	CreatedAt   string           `json:"created_at"`
	UpdatedAt   string           `json:"updated_at"`
}

// UsersListResponse is the response for listing users
type UsersListResponse struct {
	Title string               `json:"title"`
	Data  []UserData           `json:"data"`
	Meta  commondto.PaginationMeta `json:"meta"`
}

// UpdateUserRoleInput is used for updating a user's role and status
type UpdateUserRoleInput struct {
	Role   types.UserRole   `json:"role" validate:"omitempty,oneof=user admin management"`
	Status types.UserStatus `json:"status" validate:"omitempty,oneof=active inactive suspended pending"`
}

// UpdateUserResponse is a generic response for user updates
type UpdateUserResponse struct {
	Title string              `json:"title"`
	Data  UserData					  `json:"data"`
}
// UpdateUserInput is used for updating a user's address or phone number
type UpdateUserInput struct {
	Address     string `json:"address" validate:"omitempty,min=5,max=255"`
	PhoneNumber string `json:"phone_number" validate:"omitempty,min=10,max=15"`
}

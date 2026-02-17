package dto

import (
	"github.com/alibaba0010/postgres-api/internal/common/types"
)

// MessageResponse is a common single-message response used across endpoints
type MessageResponse struct {
	Title       string `json:"title"`
	Message     string `json:"message"`
	AccessToken string `json:"access_token,omitempty"`
}

// SingleDataResponse is a generic response for a single data object
type SingleDataResponse[T any] struct {
	Title string `json:"title,omitempty"`
	Data  T      `json:"data"`
}
// AuthenticatedUser is stored in request context for downstream handlers
type AuthenticatedUser struct {
	UserID string
	Role   types.UserRole
}
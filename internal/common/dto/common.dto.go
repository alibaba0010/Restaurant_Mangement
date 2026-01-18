package dto

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

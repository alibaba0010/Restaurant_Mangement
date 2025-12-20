package dto

// MessageResponse is a common single-message response used across endpoints
type MessageResponse struct {
    Title   string `json:"title"`
    Message string `json:"message"`
}

// ResendVerificationInput is the payload for requesting a resend of verification email
type ResendVerificationInput struct {
    Email string `json:"email" validate:"required,email"`
}

// AccessTokenResponse returns a fresh access token
type AccessTokenResponse struct {
    AccessToken string `json:"access_token"`
}

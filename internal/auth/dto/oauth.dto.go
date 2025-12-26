package dto

type VerifyOAuthInput struct {
	Code  string `json:"code" validate:"required"`
	State string `json:"state" validate:"required"`
}

type OAuthLoginResponse struct {
	URL string `json:"url"`
}

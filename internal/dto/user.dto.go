package dto

// CurrentUserResponse represents the response structure for the current user endpoint
type CurrentUserResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Address   string `json:"address,omitempty"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}
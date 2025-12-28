package dto

import (
	"regexp"

	"github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/go-playground/validator/v10"
)

type SignupInput struct {
	Name            string `json:"name" validate:"required,min=3,max=50"`
	Email           string `json:"email" validate:"required,email"`
	Password        string `json:"password" validate:"required,min=6,max=18,password_special"`
	ConfirmPassword string `json:"confirmPassword" validate:"required,eqfield=Password"`
	Role            types.UserRole `json:"role" validate:"omitempty,oneof=user admin management"`

}

type SigninInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

// SigninResponse contains user info and tokens after successful login
type SigninResponse struct {
	Title string     `json:"title"`
	Data  SigninData `json:"data"`
}

type SigninData struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	Email     string           `json:"email"`
	Role      types.UserRole   `json:"role"`
	Address   string `json:"address,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
	PhoneNumber string  `json:"phone_number,omitempty"`
	Status    types.UserStatus `json:"status"`
	CreatedAt string           `json:"created_at"`
	UpdatedAt string           `json:"updated_at"`
}

// Response is a generic API response envelope used across handlers.
type SignUpResponse struct {
	Title string      `json:"title"`
	Data  SigninData `json:"data"`
}


// UpdateUserInput is used for updating a user's address or phone number
type UpdateUserInput struct {
	Address     string `json:"address" validate:"omitempty,min=5,max=255"`
	PhoneNumber string `json:"phone_number" validate:"omitempty,min=10,max=15"`
}


// LogoutResponse is returned after a successful logout
type LogoutResponse struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

// RegisterValidators registers custom validators on the provided validator instance.
func RegisterValidators(v *validator.Validate) {
	_ = v.RegisterValidation("password_special", validatePasswordSpecial)
}

// validatePasswordSpecial ensures password contains at least one uppercase,
// one lowercase, one digit, and one special character.
func validatePasswordSpecial(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	// Check for at least one uppercase letter
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	// Check for at least one lowercase letter
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	// Check for at least one digit
	hasDigit := regexp.MustCompile(`[0-9]`).MatchString(password)
	// Check for at least one special character
	hasSpecial := regexp.MustCompile(`[!@#$%^&*()_+\-=[\]{};':"\\|,.<>\/?]`).MatchString(password)

	return hasUpper && hasLower && hasDigit && hasSpecial
}
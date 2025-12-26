package dto

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

type SignupInput struct {
	Name            string `json:"name" validate:"required,min=3,max=50"`
	Email           string `json:"email" validate:"required,email"`
	Password        string `json:"password" validate:"required,min=6,max=18,password_special"`
	ConfirmPassword string `json:"confirmPassword" validate:"required,eqfield=Password"`
	Role            string `json:"role" validate:"omitempty,oneof=user admin management"`
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
	ID           string `json:"id"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	Role         string `json:"role"`
}

// Response is a generic API response envelope used across handlers.
type SignUpResponse struct {
	Title string      `json:"title"`
	Data  SignUpData `json:"data"`
}
type SignUpData struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// UpdateAddressInput is used for updating a user's address
type UpdateAddressInput struct {
	Address string `json:"address" validate:"required,min=5,max=255"`
}

// UpdateAddressResponse contains the updated user data after address update
type UpdateAddressResponse struct {
	Title string `json:"title"`
	Data  struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		Address   string `json:"address"`
		Role      string `json:"role"`
		UpdatedAt string `json:"updated_at"`
	} `json:"data"`
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
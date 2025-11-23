package services

import (
	"context"
	"strings"

	"github.com/go-playground/validator/v10"
	"golang.org/x/crypto/bcrypt"

	"github.com/alibaba0010/postgres-api/internal/database"
	"github.com/alibaba0010/postgres-api/internal/dto"
	"github.com/alibaba0010/postgres-api/internal/errors"
	"github.com/alibaba0010/postgres-api/internal/models"
	"github.com/alibaba0010/postgres-api/internal/utils"
)

// RegisterUser handles the DB work for signing up a new user.
// It checks for an existing email, hashes the password and inserts the user.
// Returns the created user (with ID populated) or an AppError for controller to return.
func RegisterUser(ctx context.Context, input dto.SignupInput) (*models.User, *errors.AppError) {
	// Validate input using same validation rules as controllers previously used
	validate := validator.New()
	dto.RegisterValidators(validate)
	if err := validate.Struct(input); err != nil {
		var messages []string
		for _, e := range err.(validator.ValidationErrors) {
			var msg string
			switch e.Tag() {
			case "required":
				msg = e.Field() + " is required"
			case "min":
				msg = e.Field() + " must be at least " + e.Param() + " characters"
			case "max":
				msg = e.Field() + " must be at most " + e.Param() + " characters"
			case "email":
				msg = e.Field() + " must be a valid email address"
			case "password_special":
				msg = e.Field() + " must contain at least one uppercase letter, one lowercase letter, one digit, and one special character"
			case "eqfield":
				msg = e.Field() + " must match " + e.Param()
			default:
				msg = e.Field() + " failed validation: " + e.Tag()
			}
			messages = append(messages, msg)
		}
		return nil, errors.ValidationError(strings.Join(messages, "; "))
	}

	// Check if user already exists
	exists, err := database.DB.NewSelect().Model((*models.User)(nil)).
		Where("email = ?", input.Email).
		Exists(ctx)
	if err != nil {
		return nil, errors.InternalError(err)
	}
	if exists {
		return nil, errors.DuplicateError("email")
	}
// --------------------------------- -----------------------------
// generate token for email verification
// send email with the token
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.InternalError(err)
	}

       user := &models.User{
	       Name:     input.Name,
	       Email:    input.Email,
	       Password: string(hashedPassword),
       }

	// Ensure ID is set to a UUIDv7 before insert (DB has no default)
	newUUID, err := utils.GenerateUUIDv7()
	if err != nil {
		return nil, errors.InternalError(err)
	}
	user.ID = newUUID.String()

       // Insert and request returning id so bun populates user.ID
       _, err = database.DB.NewInsert().Model(user).
	       Returning("id").
	       Exec(ctx)
	if err != nil {
		return nil, errors.InternalError(err)
	}

	return user, nil
}

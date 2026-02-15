package utils

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/go-playground/validator/v10"
)

var (
	validate *validator.Validate
	once     sync.Once
)

// getValidator returns the singleton validator instance, initializing it if necessary.
func getValidator() *validator.Validate {
	once.Do(func() {
		validate = validator.New()
		_ = validate.RegisterValidation("password_special", validatePasswordSpecial)
	})
	return validate
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

// ValidateStruct validates a struct using the validator package and returns formatted validation errors.
// Returns a slice of error messages suitable for the AppError.Messages field.
func ValidateStruct(data interface{}) []string {
	v := getValidator()
	err := v.Struct(data)
	if err == nil {
		return nil
	}

	var messages []string
	if ves, ok := err.(validator.ValidationErrors); ok {
		for _, fe := range ves {
			var msg string
			field := fe.Field()
			switch fe.Tag() {
			case "required":
				msg = fmt.Sprintf("%s is required", field)
			case "min":
				msg = fmt.Sprintf("%s is too short (minimum %s characters)", field, fe.Param())
			case "max":
				msg = fmt.Sprintf("%s is too long (maximum %s characters)", field, fe.Param())
			case "email":
				msg = fmt.Sprintf("%s must be a valid email address", field)
			case "url":
				msg = fmt.Sprintf("%s must be a valid URL", field)
			case "uuid":
				msg = fmt.Sprintf("%s must be a valid UUID", field)
			case "gt":
				msg = fmt.Sprintf("%s must be greater than %s", field, fe.Param())
			case "gte":
				msg = fmt.Sprintf("%s must be greater than or equal to %s", field, fe.Param())
			case "lt":
				msg = fmt.Sprintf("%s must be less than %s", field, fe.Param())
			case "lte":
				msg = fmt.Sprintf("%s must be less than or equal to %s", field, fe.Param())
			case "oneof":
				msg = fmt.Sprintf("%s must be one of: %s", field, fe.Param())
			case "password_special":
				msg = "password must contain at least one uppercase letter, one lowercase letter, one digit, and one special character"
			case "eqfield":
				msg = fmt.Sprintf("%s must match %s", field, fe.Param())
			default:
				msg = fmt.Sprintf("%s is invalid", field)
			}
			messages = append(messages, msg)
		}
	}
	return messages
}

// ValidateInput checks the struct and returns a wrapped AppError with all messages if any.
func ValidateInput(data interface{}) *errors.AppError {
	msgs := ValidateStruct(data)
	if len(msgs) > 0 {
		return errors.ValidationErrors(msgs)
	}
	return nil
}

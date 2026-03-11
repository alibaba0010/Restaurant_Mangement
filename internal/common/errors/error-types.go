package errors

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

func ValidationError(message string) *AppError {
	return New("Validation Error", message, http.StatusBadRequest, nil)
}

// ValidationErrors returns an AppError that contains multiple validation messages.
func ValidationErrors(messages []string) *AppError {
	return &AppError{
		Title:    "Validation Error",
		Message:  strings.Join(messages, "; "),
		Messages: messages,
		Status:   http.StatusBadRequest,
		Err:      nil,
	}
}
// bad request error
func BadRequestError(message string) *AppError {
	return New("Bad Request", message, http.StatusBadRequest, nil)
}

func DuplicateError(field string) *AppError {
	return New("Duplicate Value", ""+field+" already exists", http.StatusBadRequest, nil)
}

func NotFoundError(message string) *AppError {
	return New("Not Found", message, http.StatusNotFound, nil)
}

func UnauthorizedError(message string) *AppError {
	return New("Unauthorized", message, http.StatusUnauthorized, nil)
}

func ForbiddenError(message string) *AppError {
	return New("Forbidden", message, http.StatusForbidden, nil)
}

func InternalError(err error) *AppError {
	if err == nil {
		return New("Internal Server Error", "Something went wrong, try again later", http.StatusInternalServerError, nil)
	}

	// Detect "no rows" cases coming from database/sql or pgx
	if errors.Is(err, sql.ErrNoRows) || err == pgx.ErrNoRows || strings.Contains(strings.ToLower(err.Error()), "no rows") {
		return New("Not Found", "Requested resource not found", http.StatusNotFound, err)
	}

	msg := "An unexpected error occurred. Please try again later."
	return New("Internal Server Error", msg, http.StatusInternalServerError, err)
}

func RouteNotExist() *AppError {
	return New("Route Error", "Route does not exist", http.StatusNotFound, nil)
}

// RateLimitError returns an AppError for rate limit exceeded
func RateLimitError() *AppError {
	return New("Rate Limit Exceeded", "Too many requests. Please try again later", http.StatusTooManyRequests, nil)
}

// CORSError returns an AppError for CORS policy violations
func CORSError(origin string) *AppError {
	message := fmt.Sprintf("CORS policy violation: origin '%s' is not allowed", origin)
	return New("CORS Error", message, http.StatusForbidden, nil)
}

// TransactionError returns an AppError for transaction-related failures
func TransactionError(operation string, err error) *AppError {
	message := fmt.Sprintf("Database transaction failed while %s. Please try again later", operation)
	return New("Transaction Error", message, http.StatusInternalServerError, err)
}

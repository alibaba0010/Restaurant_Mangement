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

func DuplicateError(field string) *AppError {
	return New("Duplicate Value", "Duplicate value entered for "+field+" field, please choose another value", http.StatusBadRequest, nil)
}

func NotFoundError(message string) *AppError {
	return New("Not Found", message, http.StatusNotFound, nil)
}


func InternalError(err error) *AppError {
	if err == nil {
		return New("Internal Server Error", "Something went wrong, try again later", http.StatusInternalServerError, nil)
	}

	// Detect "no rows" cases coming from database/sql or pgx
	if errors.Is(err, sql.ErrNoRows) || err == pgx.ErrNoRows || strings.Contains(strings.ToLower(err.Error()), "no rows") {
		return New("Not Found", "Requested resource not found", http.StatusNotFound, err)
	}


	short := err.Error()
	if len(short) > 200 {
		short = short[:200]
	}
	msg := fmt.Sprintf("Something went wrong, try again later: %s", short)
	return New("Internal Server Error", msg, http.StatusInternalServerError, err)
}

func RouteNotExist() *AppError {
	return New("Route Error", "Route does not exist", http.StatusNotFound, nil)
}

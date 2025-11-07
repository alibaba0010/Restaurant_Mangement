package errors

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/alibaba0010/postgres-api/logger"
	"github.com/jackc/pgconn"
)

// HandleError converts arbitrary errors into AppError and writes the JSON response.
// Usage in handlers:
//   if err != nil {
//       errors.HandleError(w, r, err)
//       return
//   }
func HandleError(w http.ResponseWriter, r *http.Request, err error) {
    // Map to AppError (may return nil). ErrorResponse handles nil by producing a 500.
    ErrorResponse(w, r, mapToAppError(err))
}

// mapToAppError contains heuristics to map common error types to your AppError helpers.
func mapToAppError(err error) *AppError {
    if err == nil {
        return nil
    }

    // If already an AppError, return as-is.
    if ae, ok := err.(*AppError); ok {
        return ae
    }

    // sql.ErrNoRows => not found
    if err == sql.ErrNoRows {
        return NotFoundError("resource not found")
    }

    // Postgres unique violation => duplicate value (handle pgx/pgconn errors)
    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) {
        // unique_violation is 23505
        if pgErr.Code == "23505" {
            // try to extract field name from Detail like: "Key (email)=(x) already exists."
            field := "value"
            if d := pgErr.Detail; d != "" {
                if parts := strings.Split(d, "("); len(parts) >= 2 {
                    if p2 := strings.SplitN(parts[1], ")", 2); len(p2) >= 1 {
                        candidate := strings.TrimSpace(p2[0])
                        if candidate != "" {
                            field = candidate
                        }
                    }
                }
            }
            // fallback to constraint name if available
            if field == "value" && pgErr.ConstraintName != "" {
                field = pgErr.ConstraintName
            }
            return DuplicateError(field)
        }
    }

    // simple validation detection from error text (fallback)
    lower := strings.ToLower(err.Error())
    if strings.Contains(lower, "validation") || strings.Contains(lower, "validate") {
        return ValidationError(err.Error())
    }

    // fallback to internal error
    return InternalError(err)
}

// RecoverMiddleware recovers panics in handlers and returns an internal server error JSON.
func RecoverMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if rec := recover(); rec != nil {
                // Log minimal panic info (do not print the panic value or stack)
                logger.Log.Error("panic recovered in request", zap.String("path", r.URL.Path))
                // send generic 500 response without exposing internal details
                ErrorResponse(w, r, InternalError(fmt.Errorf("panic")))
            }
        }()
        next.ServeHTTP(w, r)
    })
}
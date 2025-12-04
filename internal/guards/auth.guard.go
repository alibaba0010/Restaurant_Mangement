package guards

import (
	"context"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/alibaba0010/postgres-api/internal/errors"
	"github.com/alibaba0010/postgres-api/internal/logger"
	"github.com/alibaba0010/postgres-api/internal/services"
)

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

const (
	// UserClaimsKey stores the user claims in request context
	UserClaimsKey ContextKey = "user_claims"
)

// AuthenticatedUser is stored in request context for downstream handlers
type AuthenticatedUser struct {
	UserID string
	Role   string
}

// AuthMiddleware validates the access token from Authorization header (Bearer scheme).
// If expired, returns 401 with "access token is expired" message.
// If invalid, returns 401 with "access token is invalid" message.
// Sets context "user_claims" for downstream handlers.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// Extract access token from Authorization header
		authHeader := request.Header.Get("Authorization")
		if authHeader == "" {
			errors.ErrorResponse(writer, request, errors.UnauthorizedError("authorization header required"))
			return
		}

		// Expect "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			errors.ErrorResponse(writer, request, errors.UnauthorizedError("invalid authorization header format"))
			return
		}

		accessToken := parts[1]

		// Try to verify access token
		claims, appErr := services.VerifyAccessToken(accessToken)
		if appErr == nil {
			// Access token is valid, proceed
			ctx := context.WithValue(request.Context(), UserClaimsKey, claims)
			next.ServeHTTP(writer, request.WithContext(ctx))
			return
		}

		// Access token is invalid or expired
		// Client should call POST /auth/refresh to get a new access token
		errors.ErrorResponse(writer, request, appErr)
	})
}

// ExtractAuthenticatedUser extracts authenticated user info from request headers
// (set by AuthMiddleware).
func ExtractAuthenticatedUser(request *http.Request) *AuthenticatedUser {
	userID := request.Header.Get("X-User-Id")
	role := request.Header.Get("X-User-Role")
	if userID == "" {
		return nil
	}
	return &AuthenticatedUser{
		UserID: userID,
		Role:   role,
	}
}



func RequireRole(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			user := ExtractAuthenticatedUser(request)
			if user == nil {
				errors.ErrorResponse(writer, request, errors.UnauthorizedError("user not authenticated"))
				return
			}

			// Check if user's role has permission for any of the allowed roles
			if !services.CheckRolePermission(user.Role, allowedRoles...) {
				logger.Log.Warn("unauthorized access attempt", 
					zap.String("user_id", user.UserID), 
					zap.String("user_role", user.Role), 
					zap.Strings("required_roles", allowedRoles))
				errors.ErrorResponse(writer, request, errors.ForbiddenError("insufficient permissions for this resource"))
				return
			}

			logger.Log.Debug("user authorized", zap.String("user_id", user.UserID), zap.String("role", user.Role))
			next.ServeHTTP(writer, request)
		})
	}
}



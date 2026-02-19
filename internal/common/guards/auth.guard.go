package guards

import (
	"context"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/alibaba0010/postgres-api/internal/auth/services"
	"github.com/alibaba0010/postgres-api/internal/common/dto"
	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/common/types"
)

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

const (
	// UserClaimsKey stores the user claims in request context
	UserClaimsKey ContextKey = "user_claims"
)



// AuthMiddleware validates access token from Authorization header or request cookie.
// On success: stores claims in context (UserClaimsKey)
// On fail: returns 401 with specific error (expired vs invalid)
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		tokenString := extractToken(request)

		if tokenString == "" {
			errors.ErrorResponse(writer, request, errors.UnauthorizedError("authentication required"))
			return
		}

		claims, appErr := services.VerifyAccessToken(tokenString)
		if appErr == nil {
			ctx := context.WithValue(request.Context(), UserClaimsKey, claims)
			next.ServeHTTP(writer, request.WithContext(ctx))
			return
		}

		errors.ErrorResponse(writer, request, appErr)
	})
}

func extractToken(request *http.Request) string {
	tokenString := ""
	authHeader := request.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			tokenString = parts[1]
		}
	}

	// Fallback to cookie
	if tokenString == "" {
		cookie, err := request.Cookie("access_token")
		if err == nil {
			tokenString = cookie.Value
		}
	}
	return tokenString
}

// ExtractAuthenticatedUser extracts user claims from context (set by AuthMiddleware)
func ExtractAuthenticatedUser(request *http.Request) *dto.AuthenticatedUser {
	if v := request.Context().Value(UserClaimsKey); v != nil {
		if claims, ok := v.(*services.AccessTokenClaims); ok && claims != nil {
			return &dto.AuthenticatedUser{
				UserID: claims.UserID,
				Role:   claims.Role,
			}
		}
	}
	return nil
}

// GetUserEmail retrieves the user's email from the database given their userID.
func GetUserEmail(ctx context.Context, userID string) (string, error) {
	userData, appErr := services.GetUserByID(ctx, userID)
	if appErr != nil {
		return "", appErr
	}
	return userData.Email, nil
}

// RequireRole enforces role-based access control using role hierarchy (admin > management > user)
func RequireRole(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			user := ExtractAuthenticatedUser(request)
			if user == nil {
				errors.ErrorResponse(writer, request, errors.UnauthorizedError("user not authenticated"))
				return
			}

			if !CheckRolePermission(user.Role, allowedRoles...) {
				logger.Log.Warn("unauthorized access attempt",
					zap.String("user_id", user.UserID),
					zap.String("user_role", string(user.Role)),
					zap.Strings("required_roles", allowedRoles))
				errors.ErrorResponse(writer, request, errors.ForbiddenError("insufficient permissions for this resource"))
				return
			}

			logger.Log.Debug("user authorized", zap.String("user_id", user.UserID), zap.String("role", string(user.Role)))
			next.ServeHTTP(writer, request)
		})
	}
}

// CheckRolePermission checks if userRole has permission for any of requiredRoles
// Uses role hierarchy: admin > management > user
func CheckRolePermission(userRole types.UserRole, requiredRoles ...string) bool {
	if !userRole.IsValid() {
		logger.Log.Warn("invalid user role", zap.String("role", string(userRole)))
		return false
	}

	for _, requiredRole := range requiredRoles {
		if requiredRoleEnum, ok := types.ToUserRole(requiredRole); ok && userRole.HasPermission(requiredRoleEnum) {
			return true
		}
	}
	return false
}

package guards

import (
	"net/http"
	"strings"
	"time"

	"github.com/alibaba0010/postgres-api/internal/config"
	"github.com/alibaba0010/postgres-api/internal/errors"
	"github.com/alibaba0010/postgres-api/internal/logger"
	"github.com/alibaba0010/postgres-api/internal/services"
	"github.com/alibaba0010/postgres-api/internal/utils"
	"go.uber.org/zap"
)

// AuthenticatedUser is stored in request context for downstream handlers
type AuthenticatedUser struct {
	UserID string
	Role   string
}

// AuthMiddleware validates the access token from Authorization header (Bearer scheme).
// If expired, attempts to refresh using the refresh_token cookie.
// Sets ctx.Request.Header["X-User-Id"] and ["X-User-Role"] for downstream handlers.
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
			request.Header.Set("X-User-Id", claims.UserID)
			request.Header.Set("X-User-Role", claims.Role)
			next.ServeHTTP(writer, request)
			return
		}

		// Access token is invalid or expired, try to refresh
		logger.Log.Debug("access token invalid or expired, attempting refresh")

		// Get refresh token from cookie
		refreshCookie, err := request.Cookie("refresh_token")
		if err != nil {
			// No refresh token cookie, user must login again
			errors.ErrorResponse(writer, request, errors.UnauthorizedError("refresh token missing; please login again"))
			return
		}
		
		refreshToken := refreshCookie.Value
		logger.Log.Info("Refresh token value in cookie ", zap.String("refresh_token",refreshToken))

		// Extract IP and User-Agent for refresh validation
		ip := utils.ExtractClientIP(request)
		userAgent := request.Header.Get("User-Agent")

		// Extract userID from access token claims even if expired (to know which user to refresh)
		expiredClaims, _ := services.VerifyAccessToken(accessToken)
		userID := ""
		if expiredClaims != nil {
			userID = expiredClaims.UserID
		}

		// If we can't get userID from claims, try parsing the refresh token
		if userID == "" {
			refreshClaims, err := services.ValidateRefreshToken(refreshToken)
			if err != nil {
				errors.ErrorResponse(writer, request, errors.UnauthorizedError("invalid refresh token; please login again"))
				return
			}
			userID = refreshClaims.UserID
		}

		// Attempt to refresh the access token
		newTokenPair, appErr := services.RefreshAccessToken(request.Context(), refreshToken, userID, ip, userAgent)
		if appErr != nil {
			// Refresh failed, user must login again
			errors.ErrorResponse(writer, request, errors.UnauthorizedError("refresh token invalid or revoked; please login again"))
			return
		}

		// Successfully refreshed, set new tokens
		// Update Authorization header for this request
		request.Header.Set("Authorization", "Bearer "+newTokenPair.AccessToken)

		// Set new refresh token cookie
		newCookie := &http.Cookie{
			Name:     "refresh_token",
			Value:    newTokenPair.RefreshToken,
			HttpOnly: true,
			Path:     "/",
			Expires:  time.Now().Add(services.RefreshTokenDuration),
			Secure:   false,
			SameSite: http.SameSiteLaxMode,
		}
		if strings.HasPrefix(config.LoadConfig().FRONTEND_URL, "https") {
			newCookie.Secure = true
		}
		http.SetCookie(writer, newCookie)

		// Send new access token in response header for client to update
		writer.Header().Set("X-New-Access-Token", newTokenPair.AccessToken)

		// Extract user info from refresh token claims
		refreshClaims, _ := services.ValidateRefreshToken(refreshToken)
		if refreshClaims != nil {
			request.Header.Set("X-User-Id", refreshClaims.UserID)
			request.Header.Set("X-User-Role", refreshClaims.Role)
			logger.Log.Info("access token refreshed successfully", zap.String("user_id", refreshClaims.UserID))
		}

		next.ServeHTTP(writer, request)
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



package utils

import (
	"crypto/rand"
	// "encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/config"

	"github.com/google/uuid"
	"go.uber.org/zap"
)
func GenerateUUIDv7() (uuid.UUID, error) {
	// uuid.NewV7 requires a source of randomness and the current time.
	// We'll use time.Now() and the default crypto/rand source.
	return uuid.NewV7()
}

func ParseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

// GenerateToken generates a random 32-byte hex string.
func GenerateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}


// extractClientIP extracts the client IP from request headers with fallbacks.
func ExtractClientIP(request *http.Request) string {
	// Check X-Forwarded-For header (may be comma-separated)
	if xf := request.Header.Get("X-Forwarded-For"); xf != "" {
		parts := strings.Split(xf, ",")
		return strings.TrimSpace(parts[0])
	}

	// Check X-Real-IP header
	if xr := request.Header.Get("X-Real-Ip"); xr != "" {
		return xr
	}

	// Fall back to RemoteAddr
	remote := request.RemoteAddr
	if i := strings.LastIndex(remote, ":"); i != -1 {
		return remote[:i]
	}
	return remote
}

// SetAuthCookies sets both access and refresh token cookies.
func SetAuthCookies(writer http.ResponseWriter, accessToken, refreshToken string, accessDuration, refreshDuration time.Duration) {
	SetAccessTokenCookie(writer, accessToken, accessDuration)
	SetRefreshTokenCookie(writer, refreshToken, refreshDuration)
}

// SetRefreshTokenCookie sets a refresh token HTTP-only cookie with proper security settings.
// The Secure flag is automatically set to true if the frontend URL is HTTPS.
func SetRefreshTokenCookie(writer http.ResponseWriter, refreshToken string, tokenDuration time.Duration) {
	logger.Log.Info("Refresh token value: ", zap.String("Refresh Token......",refreshToken))
	cfg := config.LoadConfig()
	isSecure := strings.HasPrefix(cfg.FRONTEND_URL, "https")

	cookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		HttpOnly: true,
		Path:     "/",
		Expires:  time.Now().Add(tokenDuration),
		MaxAge:   int(tokenDuration.Seconds()),
		Secure:   isSecure,
		SameSite: http.SameSiteLaxMode,
	}

	// If it's a cross-origin request in dev, some browsers might need None/Secure
	// but Lax should work on localhost. If we want to support cross-domain, we'd need None/Secure.
	if isSecure {
		cookie.SameSite = http.SameSiteNoneMode
	}

	http.SetCookie(writer, cookie)
}


// ClearRefreshTokenCookie clears the refresh token cookie by setting MaxAge to -1.
func ClearRefreshTokenCookie(writer http.ResponseWriter, isSecure bool) {
	cookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(writer, cookie)
}

// SetAccessTokenCookie sets an access token HTTP-only cookie.
func SetAccessTokenCookie(writer http.ResponseWriter, accessToken string, tokenDuration time.Duration) {
	cfg := config.LoadConfig()
	isSecure := strings.HasPrefix(cfg.FRONTEND_URL, "https")

	cookie := &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		HttpOnly: true,
		Path:     "/",
		Expires:  time.Now().Add(tokenDuration),
		MaxAge:   int(tokenDuration.Seconds()),
		Secure:   isSecure,
		SameSite: http.SameSiteLaxMode,
	}

	if isSecure {
		cookie.SameSite = http.SameSiteNoneMode
	}

	http.SetCookie(writer, cookie)
	}


// ClearAccessTokenCookie clears the access token cookie.
func ClearAccessTokenCookie(writer http.ResponseWriter, isSecure bool) {
	cookie := &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(writer, cookie)
}
// writeJSON writes a JSON response with the provided status code.
// Keeps handlers compact and consistent.
func WriteJSON(writer http.ResponseWriter, status int, v any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(v)
}
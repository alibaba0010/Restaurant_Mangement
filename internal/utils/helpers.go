package utils

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/alibaba0010/postgres-api/internal/config"
	"github.com/google/uuid"
)
func GenerateUUIDv7() (uuid.UUID, error) {
	// uuid.NewV7 requires a source of randomness and the current time.
	// We'll use time.Now() and the default crypto/rand source.
	return uuid.NewV7()
}

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

// SetRefreshTokenCookie sets a refresh token HTTP-only cookie with proper security settings.
// The Secure flag is automatically set to true if the frontend URL is HTTPS.
func SetRefreshTokenCookie(writer http.ResponseWriter, refreshToken string, tokenDuration time.Duration) {
	cfg := config.LoadConfig()
	cookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		HttpOnly: true,
		Path:     "/",
		Expires:  time.Now().Add(tokenDuration),
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	}
	if strings.HasPrefix(cfg.FRONTEND_URL, "https") {
		cookie.Secure = true
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
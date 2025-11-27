package utils

import (
	"net/http"
	"strings"
	"crypto/rand"
	"encoding/hex"
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
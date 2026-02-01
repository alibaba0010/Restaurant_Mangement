package utils

import (
	"crypto/rand"
	"strconv"

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
// ParseUUID parses a string to uuid.UUID type.
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

// WriteJSON writes a JSON response with the provided status code.
func WriteJSON[T any](writer http.ResponseWriter, status int, v T) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	if err := json.NewEncoder(writer).Encode(v); err != nil {
		logger.Log.Error("failed to encode JSON response", zap.Error(err))
	}
}

func CalculateTotalPages(total int64, pageSize int) int {
	if pageSize <= 0 {
		return 0
	}
	return int((total + int64(pageSize) - 1) / int64(pageSize))
}

func ParseFloat(value string, defaultValue float64) float64 {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return defaultValue
	}
	return parsed
}

// ParseInt parses a string to an integer with a default fallback.
func ParseInt(value string, defaultValue int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

// SanitizeSort validates and cleans sort parameters to prevent SQL injection and ensure valid defaults.
func SanitizeSort(sortBy, order string, allowedFields []string, defaultField string) (string, string) {
	sortBy = strings.ToLower(sortBy)
	order = strings.ToUpper(order)

	validField := false
	for _, field := range allowedFields {
		if sortBy == strings.ToLower(field) {
			sortBy = field // Use the exact field name from whitelist
			validField = true
			break
		}
	}

	if !validField {
		sortBy = defaultField
	}

	if order != "ASC" && order != "DESC" {
		order = "DESC"
	}

	return sortBy, order
}

// ListParams holds common query parameters for listing endpoints
type ListParams struct {
	Page     int    // DB Offset = (Page-1)*PageSize
	PageSize int    // Legacy Limit
	Cursor   string // Cursor for cursor-based pagination
	Limit    int    // Limit for cursor-based pagination
	Query    string
	SortBy   string
	Order    string
}

// ParseListParams extracts common query parameters from the request
func ParseListParams(r *http.Request) ListParams {
	q := r.URL.Query()

	limit := ParseInt(q.Get("limit"), 20)
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	pageSize := ParseInt(q.Get("page_size"), 20)
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	return ListParams{
		Page:     ParseInt(q.Get("page"), 1),
		PageSize: pageSize,
		Cursor:   q.Get("cursor"),
		Limit:    limit,
		Query:    q.Get("q"),
		SortBy:   q.Get("sort_by"),
		Order:    q.Get("order"),
	}
}




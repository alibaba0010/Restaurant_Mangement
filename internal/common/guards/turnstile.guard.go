package guards

import (
	"net/http"
	"strings"
	"go.uber.org/zap"

	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/config"
	"github.com/alibaba0010/postgres-api/internal/utils"

)

// TurnstileMiddleware validates the Cloudflare Turnstile token from request headers
func TurnstileMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg := config.LoadConfig()
		// Skip verification if disabled or in development (if explicitly set)
		// For now, we only skip if the secret key is missing.
		if cfg.TURNSTILE_SECRET_KEY == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Look for token in headers
		// Common headers: X-Turnstile-Token or cf-turnstile-response
		token := r.Header.Get("X-Turnstile-Token")
		if token == "" {
			token = r.Header.Get("cf-turnstile-response")
		}

		// If token is missing, reject the request
		if token == "" {
			logger.Log.Warn("Turnstile token missing", zap.String("path", r.URL.Path))
			errors.ErrorResponse(w, r, errors.BadRequestError("Complete The Security Challenge before proceeding"))
			return
		}

		logger.Log.Info("Turnstile token received", zap.String("path", r.URL.Path))
		// Get remote IP safely
		remoteIP := r.Header.Get("X-Forwarded-For")
		if remoteIP != "" {
			remoteIP = strings.Split(remoteIP, ",")[0]
		} else {
			remoteIP = strings.Split(r.RemoteAddr, ":")[0]
		}

		// Verify token with Cloudflare
		success, err := utils.VerifyTurnstileToken(token, remoteIP)
		if err != nil {
			// If it's a server error (networking), we might decide to let it pass or fail.
			// Recommending "Fail-Closed" for security.
			errors.ErrorResponse(w, r, errors.InternalError(err))
			return
		}

		if !success {
			logger.Log.Warn("Turnstile challenge failed", zap.String("path", r.URL.Path), zap.String("remote_ip", remoteIP))
			errors.ErrorResponse(w, r, errors.ForbiddenError("security challenge failed: possibly a bot"))
			return
		}

		next.ServeHTTP(w, r)
	})
}

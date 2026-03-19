package guards

import (
	"net"
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
	cfg := config.LoadConfig()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip verification if disabled
		if cfg.TURNSTILE_SECRET_KEY == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Bypass Turnstile if the user is authenticated (has Authorization header)
		if authHeader := r.Header.Get("Authorization"); authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			// (Optional) We could structurally decode/verify it here, but typically API handlers 
			// themselves run independent Auth checks if they care. Here we just bypass the captcha.
			next.ServeHTTP(w, r)
			return
		}

		// Look for token in headers
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

		// Get remote IP safely
		var remoteIP string
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			remoteIP = strings.Split(xff, ",")[0]
		} else {
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				remoteIP = r.RemoteAddr
			} else {
				remoteIP = host
			}
		}

		// Verify token with Cloudflare
		success, err := utils.VerifyTurnstileToken(token, remoteIP)
		if err != nil {
			// Fail-Closed for security.
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

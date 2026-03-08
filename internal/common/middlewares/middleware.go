package middlewares

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/alibaba0010/postgres-api/internal/common/dto"
	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/utils"
	"go.uber.org/zap"
)

// Recover returns the existing recover middleware from errors package
func Recover() func(http.Handler) http.Handler {
	return errors.RecoverMiddleware
}

// RequestLogger returns the existing request logger middleware
func RequestLogger() func(http.Handler) http.Handler {
	return logger.Logger
}

// CORS returns a middleware that sets common CORS headers and handles preflight requests.
// It supports a specific allowed origin or allows all if set to "*".
// For development convenience, if the allowedOrigin is set to a localhost address,
// it will also allow other localhost ports to facilitate frontend/backend distinct ports.
func CORS(allowedOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Determine if we should allow this origin
			shouldAllow := false
			if allowedOrigin == "*" || allowedOrigin == "" {
				shouldAllow = true
			} else if origin == allowedOrigin {
				shouldAllow = true
			} else {
				// Dev helper: both must be some form of localhost
				if isLocalhost(allowedOrigin) && isLocalhost(origin) {
					shouldAllow = true
				}
			}

			// Handle CORS rejection for non-allowed origins
			if !shouldAllow && origin != "" && allowedOrigin != "*" && allowedOrigin != "" {
				logger.Log.Warn("CORS policy violation", zap.String("origin", origin), zap.String("allowed", allowedOrigin), zap.String("path", r.URL.Path))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				appErr := errors.CORSError(origin)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"title":   appErr.Title,
					"message": appErr.Message,
				})
				return
			}

			if shouldAllow && origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			} else if allowedOrigin == "*" {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if allowedOrigin != "" {
				// Fallback to strict configured origin if we didn't match dynamic rules but must set something
				w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-Turnstile-Token, cf-turnstile-response")
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isLocalhost checks if the url string contains localhost or 127.0.0.1
func isLocalhost(s string) bool {
	return len(s) > 0 && (strings.Contains(s, "localhost") || strings.Contains(s, "127.0.0.1"))
}

// TokenBucket is a simple token bucket rate limiter
type TokenBucket struct {
	rate           float64
	burst          float64
	tokens         float64
	lastRefillTime time.Time
	mu             sync.Mutex
}



// LastSeen returns the last refill/request time
func (tb *TokenBucket) LastSeen() time.Time {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	return tb.lastRefillTime
}

// Allow tries to take a token and returns true if allowed
func (tb *TokenBucket) Allow(now time.Time) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	elapsed := now.Sub(tb.lastRefillTime).Seconds()
	tb.tokens += elapsed * tb.rate
	if tb.tokens > tb.burst {
		tb.tokens = tb.burst
	}
	tb.lastRefillTime = now

	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

// NewTokenBucket creates a new token bucket that refills `rate` tokens per second up to `burst` capacity.
func NewTokenBucket(rate int, burst int) *TokenBucket {
	if rate <= 0 {
		rate = 1
	}
	if burst <= 0 {
		burst = rate
	}
	return &TokenBucket{
		rate:           float64(rate),
		burst:          float64(burst),
		tokens:         float64(burst),
		lastRefillTime: time.Now(),
	}
}
// RateLimit returns a middleware that enforces a per-IP token-bucket rate limit.
// rate = tokens added per second, burst = maximum bucket capacity.
func RateLimit(rate int, burst int) func(http.Handler) http.Handler {
	type userBucket struct {
		tb *TokenBucket
	}

	var (
		mu      sync.RWMutex
		buckets = make(map[string]*userBucket)
	)

	// Clean up old buckets after every 5 minute to prevent memory leak
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			// Phase 1: Identify buckets to delete using RLock
			var toDelete []string
			mu.RLock()
			for ip, ub := range buckets {
				if time.Since(ub.tb.LastSeen()) > 5*time.Minute {
					toDelete = append(toDelete, ip)
				}
			}
			mu.RUnlock()

			// Phase 2: Delete identified buckets using Lock
			if len(toDelete) > 0 {
				mu.Lock()
				for _, ip := range toDelete {
					// Double check under write lock
					if ub, exists := buckets[ip]; exists {
						if time.Since(ub.tb.LastSeen()) > 5*time.Minute {
							delete(buckets, ip)
						}
					}
				}
				mu.Unlock()
			}
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			now := time.Now()
			ip := utils.ExtractClientIP(r)

			// Try to get existing bucket with read lock
			mu.RLock()
			ub, exists := buckets[ip]
			mu.RUnlock()

			if !exists {
				// Create new bucket with write lock
				mu.Lock()
				// Double check after acquiring write lock
				ub, exists = buckets[ip]
				if !exists {
					ub = &userBucket{
						tb: NewTokenBucket(rate, burst),
					}
					buckets[ip] = ub
				}
				mu.Unlock()
			}

			if !ub.tb.Allow(now) {
				logger.Log.Warn("rate limit exceeded",
					zap.String("ip", ip),
					zap.String("path", r.URL.Path),
					zap.String("method", r.Method))

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				appErr := errors.RateLimitError()
				utils.WriteJSON(w, http.StatusTooManyRequests, dto.MessageResponse{
					Title:   appErr.Title,
					Message: appErr.Message,
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

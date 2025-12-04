package middlewares

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/alibaba0010/postgres-api/internal/errors"
	"github.com/alibaba0010/postgres-api/internal/logger"
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

// CORS returns a middleware that sets common CORS headers and handles preflight.
// allowedOrigin can be '*' or a specific origin. If empty, defaults to '*'.
func CORS(allowedOrigin string) func(http.Handler) http.Handler {
    if allowedOrigin == "" {
        allowedOrigin = "*"
    }
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
            w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
            w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
            w.Header().Set("Access-Control-Allow-Credentials", "true")

            if r.Method == http.MethodOptions {
                // Short-circuit preflight
                w.WriteHeader(http.StatusNoContent)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}

// TokenBucket is a simple token bucket rate limiter
type TokenBucket struct {
    tokens chan struct{}
}

// NewTokenBucket creates a new token bucket that refills `rate` tokens per second up to `burst` capacity.
func NewTokenBucket(rate int, burst int) *TokenBucket {
    if rate <= 0 {
        rate = 1
    }
    if burst <= 0 {
        burst = rate
    }
    tb := &TokenBucket{tokens: make(chan struct{}, burst)}
    // fill initial burst
    for i := 0; i < burst; i++ {
        select {
        case tb.tokens <- struct{}{}:
        default:
        }
    }

    // start refill goroutine
    go func() {
        ticker := time.NewTicker(time.Second)
        defer ticker.Stop()
        for range ticker.C {
            for i := 0; i < rate; i++ {
                select {
                case tb.tokens <- struct{}{}:
                default:
                    // channel full
                }
            }
        }
    }()

    return tb
}

// Allow tries to take a token and returns true if allowed
func (tb *TokenBucket) Allow() bool {
    select {
    case <-tb.tokens:
        return true
    default:
        return false
    }
}

// RateLimit returns a middleware that enforces a global token-bucket rate limit.
// rate = tokens added per second, burst = maximum burst capacity.
func RateLimit(rate int, burst int) func(http.Handler) http.Handler {
    tb := NewTokenBucket(rate, burst)
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if !tb.Allow() {
                logger.Log.Warn("rate limit exceeded", zap.String("path", r.URL.Path))
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusTooManyRequests)
                _ = json.NewEncoder(w).Encode(map[string]string{"title": "Too Many Requests", "message": "rate limit exceeded"})
                return
            }
            // additional production checks could go here (if needed)
            next.ServeHTTP(w, r)
        })
    }
}

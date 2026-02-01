package logger

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	// "sync"
	"go.uber.org/zap"
)

type contextKey string

const RequestIDKey contextKey = "request_id"

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
	size   int64
}

func (lrw *loggingResponseWriter) WriteHeader(status int) {
	lrw.status = status
	lrw.ResponseWriter.WriteHeader(status)
}

// Write overrides the Write method to track the response size.
func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	n, err := lrw.ResponseWriter.Write(b)
	lrw.size += int64(n)
	return n, err
}

// GetRequestID retrieves the request ID from the context
func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// Logger returns a middleware that logs incoming requests with Request ID and response size.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		start := time.Now()

		// 1. Get or Generate Request ID
		requestID := request.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = "55555555"
		}

		// 2. Add Request ID to context and response header
		ctx := context.WithValue(request.Context(), RequestIDKey, requestID)
		request = request.WithContext(ctx)
		writer.Header().Set("X-Request-ID", requestID)

		// 3. Wrap ResponseWriter to track status and size
		lrw := &loggingResponseWriter{ResponseWriter: writer, status: http.StatusOK}

		// 4. Serve Request
		next.ServeHTTP(lrw, request)

		// 5. Calculate Metrics
		duration := time.Since(start)
		ip := ExtractClientIP(request)

		// 6. Log completion
		Log.Info("Incoming request",
			zap.String("request_id", requestID),
			zap.String("method", request.Method),
			zap.String("path", request.URL.Path),
			zap.Int("status", lrw.status),
			zap.Duration("duration", duration),
			zap.Int64("bytes_written", lrw.size),
			zap.String("ip", ip),
			zap.String("user-agent", request.UserAgent()),
		)
	})
}

// ExtractClientIP extracts the client IP from request headers with fallbacks.
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

	// Fallback to RemoteAddr
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		return request.RemoteAddr
	}
	return host
}
	// // Fall back to RemoteAddr
	// remote := request.RemoteAddr
	// if i := strings.LastIndex(remote, ":"); i != -1 {
	// 	return remote[:i]
	// }
	// return remote
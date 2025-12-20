package logger

import (
	"net/http"
	"strings"
	"time"

	// "sync"
	"go.uber.org/zap"
)
type loggingResponseWriter struct {
    http.ResponseWriter
    status    int  
    // once      sync.Once
}
func (lrw *loggingResponseWriter) WriteHeader(status int) {
    lrw.status = status
    lrw.ResponseWriter.WriteHeader(status)
}
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		start := time.Now()
		// Wrap the original ResponseWriter so we can capture the status code
		lrw := &loggingResponseWriter{ResponseWriter: writer, status: http.StatusOK}
		// Pass the wrapped writer into the next handler so WriteHeader calls
		// set lrw.status and write to the original underlying writer.
		next.ServeHTTP(lrw, request)
		duration := time.Since(start)

		// extract client IP (respect X-Forwarded-For, X-Real-Ip) similar to other handlers
		var ip string
		if xf := request.Header.Get("X-Forwarded-For"); xf != "" {
			parts := strings.Split(xf, ",")
			ip = strings.TrimSpace(parts[0])
		} else if xr := request.Header.Get("X-Real-Ip"); xr != "" {
			ip = xr
		} else {
			remote := request.RemoteAddr
			if i := strings.LastIndex(remote, ":"); i != -1 {
				ip = remote[:i]
			} else {
				ip = remote
			}
		}

		// use the package-level Log variable directly (same package)
		Log.Info("Incoming request",
			zap.String("method", request.Method),
			zap.String("path", request.URL.Path),
			zap.Int("status", lrw.status),
			zap.Duration("duration", duration),
			zap.String("ip", ip),
			zap.String("user-agent", request.UserAgent()),
		)
	})
}
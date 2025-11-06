package logger

import (
	"net/http"
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
		lrw := &loggingResponseWriter{status: http.StatusOK}
		next.ServeHTTP(writer, request)
		duration := time.Since(start)

		// use the package-level Log variable directly (same package)
		Log.Info("Incoming request",
			zap.String("method", request.Method),
			zap.String("path", request.URL.Path),
			zap.Int("status", lrw.status),
			zap.Duration("duration", duration),
			zap.String("user-agent", request.UserAgent()),
		)
	})
}
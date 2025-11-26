package errors

import (
	"encoding/json"
	"net/http"

	"github.com/alibaba0010/postgres-api/internal/logger"
	"go.uber.org/zap"
)

// ErrorResponse defines what gets sent to the client
type ErrorResponseStruct struct {
    Title    string   `json:"title"`
    Message  string   `json:"message,omitempty"`
    Messages []string `json:"messages,omitempty"`
}

// AppError wraps any error with a title and HTTP status
type AppError struct {
    Title     string
    Message   string
    Messages  []string
    Status    int
    Err       error
}

func (err *AppError)Error() string{
return err.Message
}
// Helper or constructor to create new AppError easily
func New(title, message string, status int, err error) *AppError {
    return &AppError{
        Title:   title,
        Message: message,
        Status:  status,
        Err:     err,
    }
}
func ErrorResponse(writer http.ResponseWriter, request *http.Request, appErr *AppError){
    // Ensure we have a valid AppError
    if appErr == nil {
        appErr = &AppError{
            Title:   "Internal Server Error",
            Message: "an internal error occurred",
            Status:  http.StatusInternalServerError,
        }
    }

    writer.Header().Set("Content-Type", "application/json")
    writer.WriteHeader(appErr.Status)
requestPath := request.URL.Path
    // Log minimal info only. Do NOT print internal error details or stack traces to console.
    // For client-side/non-critical errors (4xx) log as Info; for server errors (5xx) log as Error
    if appErr.Status >= 500 {
        if appErr.Err != nil {
            logger.Log.Error(appErr.Title, zap.Int("status", appErr.Status), zap.String("path", requestPath), zap.Error(appErr.Err))
        } else {
            logger.Log.Error(appErr.Title, zap.Int("status", appErr.Status), zap.String("path", requestPath))
        }
    } else {
        logger.Log.Info(appErr.Title, zap.Int("status", appErr.Status), zap.String("path", requestPath))
    }

    // Respond to client (only public info)
    // If JSON encoding fails, don't attempt to write another body (avoids recursive logging)
    resp := ErrorResponseStruct{
        Title:   appErr.Title,
        Message: appErr.Message,
    }
    if len(appErr.Messages) > 0 {
        resp.Messages = appErr.Messages
    }
    _ = json.NewEncoder(writer).Encode(resp)
}

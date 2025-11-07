package errors

import (
	"encoding/json"
	"net/http"

	"github.com/alibaba0010/postgres-api/logger"
	"go.uber.org/zap"
)

// ErrorResponse defines what gets sent to the client
type ErrorResponseStruct struct {
    Title   string `json:"title"`
    Message string `json:"message"`
}

// AppError wraps any error with a title and HTTP status
type AppError struct {
    Title     string
    Message   string
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

    // Log minimal info only. Do NOT print internal error details or stack traces to console.
    // For client-side/non-critical errors (4xx) log as Info; for server errors (5xx) log as Error
    if appErr.Status >= 500 {
        logger.Log.Error(appErr.Title, zap.Int("status", appErr.Status))
    } else {
        logger.Log.Error(appErr.Title, zap.Int("status", appErr.Status))
    }

    // Respond to client (only public info)
    // If JSON encoding fails, don't attempt to write another body (avoids recursive logging)
    _ = json.NewEncoder(writer).Encode(ErrorResponseStruct{
        Title:   appErr.Title,
        Message: appErr.Message,
    })
}
// func (e *AppError) Error() string {
//     if e == nil {
//         return ""
//     }
//     if e.Message != "" {
//         return e.Message
//     }
//     if e.Err != nil {
//         return e.Err.Error()
//     }
//     return http.StatusText(e.Status)
// }

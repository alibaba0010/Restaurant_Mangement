package routes
import (
	"net/http"
	"encoding/json"

	"github.com/gorilla/mux"
	"github.com/alibaba0010/postgres-api/internal/errors"
	"github.com/alibaba0010/postgres-api/internal/logger"
	httpSwagger "github.com/swaggo/http-swagger"



)

func ApiRouter() *mux.Router {
	route:= mux.NewRouter()
	// user := route.PathPrefix("/api/v1").Subrouter()
		// Add recovery middleware early so panics are caught and do not print stack traces.	
	route.Use(errors.RecoverMiddleware)
	route.Use(logger.Logger)
	
	// Serve Swagger UI at /swagger/
	route.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)
	route.HandleFunc("/api/v1/healthcheck", HealthCheckHandler).Methods("GET")

	// route.HandleFunc("/getUser", getUserHandler).Methods("GET")
	// route.HandleFunc("/getBook", GetBookHandler).Methods("GET")
	// route.HandleFunc("/", httpHandler).Methods("GET")
	route.NotFoundHandler = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		errors.ErrorResponse(writer, request, errors.RouteNotExist())
	})

	return route
}

// HealthCheckHandler returns a simple health status for the API
func HealthCheckHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	resp := map[string]string{
		"title":   "Success",
		"message": "API is healthy and running",
	}
	if err := json.NewEncoder(writer).Encode(resp); err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
}
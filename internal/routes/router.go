package routes

import (
	"encoding/json"
	"net/http"

	"github.com/alibaba0010/postgres-api/internal/errors"
	"github.com/alibaba0010/postgres-api/internal/logger"
	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
)

func ApiRouter() *mux.Router {
	route := mux.NewRouter()
	// Add recovery middleware early so panics are caught and do not print stack traces.	
	route.Use(errors.RecoverMiddleware)
	route.Use(logger.Logger)
	
	// Serve Swagger UI at /swagger/
	route.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)
	
	// Create v1 subrouter with /api/v1 prefix
	v1 := route.PathPrefix("/api/v1").Subrouter()
	
// Routes
v1.HandleFunc("/healthcheck", HealthCheckHandler).Methods("GET")
	AuthRoutes(v1.PathPrefix("/auth").Subrouter())
	UserRoutes(v1)


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
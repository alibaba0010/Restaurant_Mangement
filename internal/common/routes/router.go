package routes

import (
	"encoding/json"
	"net/http"

	authRoutes "github.com/alibaba0010/postgres-api/internal/auth/routes"
	"github.com/alibaba0010/postgres-api/internal/common/dto"
	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/middlewares"
	restaurantRoutes "github.com/alibaba0010/postgres-api/internal/restaurants/routes"
	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
)

func ApiRouter() *mux.Router {
	route := mux.NewRouter()
	// Add recovery, logging, CORS and rate-limit middlewares

	route.Use(middlewares.Recover())
	route.Use(middlewares.RequestLogger())

	// global rate limit: 100 req/sec, burst 200 (tune as needed)
	route.Use(middlewares.RateLimit(100, 200))

	// Serve Swagger UI at /swagger/
	route.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	// Create v1 subrouter with /api/v1 prefix
	v1 := route.PathPrefix("/api/v1").Subrouter()

	// Routes
	v1.HandleFunc("/healthcheck", HealthCheckHandler).Methods("GET")
	authRoutes.AuthRoutes(v1.PathPrefix("/auth").Subrouter())
	authRoutes.UserRoutes(v1)
	restaurantRoutes.RestaurantRoutes(v1)
	restaurantRoutes.MenuRoutes(v1)

	route.NotFoundHandler = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		errors.ErrorResponse(writer, request, errors.RouteNotExist())
	})

	return route
}

// HealthCheckHandler returns a simple health status for the API
func HealthCheckHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	resp := dto.MessageResponse{Title: "Success", Message: "API is healthy and running"}
	if err := json.NewEncoder(writer).Encode(resp); err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
}

package routes

import (
	"net/http"

	"github.com/alibaba0010/postgres-api/internal/common/guards"
	"github.com/alibaba0010/postgres-api/internal/restaurants/controllers"
	"github.com/gorilla/mux"
)

// RestaurantRoutes registers restaurant-related handlers
func RestaurantRoutes(route *mux.Router) {
	// Restaurants endpoint
	restaurants := route.PathPrefix("/restaurants").Subrouter()

	// Public Routes
	restaurants.HandleFunc("", controllers.ListRestaurantsHandler).Methods("GET")
	restaurants.HandleFunc("/{id}", controllers.GetRestaurantHandler).Methods("GET")

	// Protected Routes
	// Create Restaurant (POST /restaurants)
	restaurants.Handle("", guards.AuthMiddleware(http.HandlerFunc(controllers.CreateRestaurantHandler))).Methods("POST")

	// Update Restaurant (PUT/PATCH /restaurants/{id})
	restaurants.Handle("/{id}", guards.AuthMiddleware(http.HandlerFunc(controllers.UpdateRestaurantHandler))).Methods("PUT", "PATCH")
}

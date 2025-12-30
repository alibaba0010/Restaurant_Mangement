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
	// Publicly accessible but optionally authenticated for filtering
	// Actually, we must use AuthMiddleware here to ensure guards.ExtractAuthenticatedUser(r) works
	restaurants.Handle("", guards.AuthMiddleware(http.HandlerFunc(controllers.ListRestaurantsHandler))).Methods("GET")
	restaurants.Handle("/{id}", guards.AuthMiddleware(http.HandlerFunc(controllers.GetRestaurantHandler))).Methods("GET")

	// Protected Routes
	// Create Restaurant (POST /restaurants)
	restaurants.Handle("", guards.AuthMiddleware(guards.RequireRole("management")(http.HandlerFunc(controllers.CreateRestaurantHandler)))).Methods("POST")

	// Update Restaurant (PUT/PATCH /restaurants/{id})
	restaurants.Handle("/{id}", guards.AuthMiddleware(http.HandlerFunc(controllers.UpdateRestaurantHandler))).Methods("PUT", "PATCH")
}

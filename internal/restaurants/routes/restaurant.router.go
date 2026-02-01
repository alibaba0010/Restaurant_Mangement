package routes

import (
	"net/http"

	"github.com/alibaba0010/postgres-api/internal/common/address"
	"github.com/alibaba0010/postgres-api/internal/common/guards"
	"github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/alibaba0010/postgres-api/internal/restaurants/controllers"
	"github.com/alibaba0010/postgres-api/internal/restaurants/services"
	"github.com/gorilla/mux"
)

// RestaurantRoutes registers restaurant-related handlers
func RestaurantRoutes(route *mux.Router) {	
	// Initialize service and controller
	addressService := address.NewService()
	restaurantService := services.NewRestaurantService(addressService)
	restaurantController := controllers.NewRestaurantController(restaurantService)

	// Restaurants endpoint
	restaurants := route.PathPrefix("/restaurants").Subrouter()
	managementRole := types.RoleManagement.String()

	// Routes accessible to all authenticated users
	restaurants.Handle("", guards.AuthMiddleware(http.HandlerFunc(restaurantController.ListRestaurantsHandler))).Methods("GET")
	restaurants.Handle("/{id}", guards.AuthMiddleware(http.HandlerFunc(restaurantController.GetRestaurantHandler))).Methods("GET")
	// Update Restaurant (PUT/PATCH /restaurants/{id})
	restaurants.Handle("/{id}", guards.AuthMiddleware(http.HandlerFunc(restaurantController.UpdateRestaurantHandler))).Methods("PUT", "PATCH")

	// Protected Routes
	// Create Restaurant managed by management role
	restaurants.Handle("", guards.AuthMiddleware(guards.RequireRole(managementRole)(http.HandlerFunc(restaurantController.CreateRestaurantHandler)))).Methods("POST")
}

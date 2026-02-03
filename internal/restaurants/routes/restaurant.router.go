package routes

import (
	"github.com/alibaba0010/postgres-api/internal/common/address"
	"github.com/alibaba0010/postgres-api/internal/common/guards"
	"github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/alibaba0010/postgres-api/internal/database"
	"github.com/alibaba0010/postgres-api/internal/restaurants/controllers"
	"github.com/alibaba0010/postgres-api/internal/restaurants/repositories"
	"github.com/alibaba0010/postgres-api/internal/restaurants/services"
	"github.com/gorilla/mux"
)

// RestaurantRoutes registers restaurant-related handlers
func RestaurantRoutes(route *mux.Router) {	
	// Initialize repositories
	restaurantRepo := repositories.NewRestaurantRepository(database.DB)

	// Initialize service and controller
	addressService := address.NewService()
	restaurantService := services.NewRestaurantService(restaurantRepo, addressService, database.DB)
	restaurantController := controllers.NewRestaurantController(restaurantService)

	// Protected Restaurant Routes
	restaurants := route.PathPrefix("/restaurants").Subrouter()
	restaurants.Use(guards.AuthMiddleware)

	// General restaurant operations (Authenticated)
	restaurants.HandleFunc("", restaurantController.ListRestaurantsHandler).Methods("GET")
	restaurants.HandleFunc("/{id}", restaurantController.GetRestaurantHandler).Methods("GET")
	restaurants.HandleFunc("/{id}", restaurantController.UpdateRestaurantHandler).Methods("PUT", "PATCH")

	// Management strictly restricted operations
	managementOnly := restaurants.PathPrefix("").Subrouter()
	managementOnly.Use(guards.RequireRole(types.RoleManagement.String()))

	// Create Restaurant (POST /restaurants)
	managementOnly.HandleFunc("", restaurantController.CreateRestaurantHandler).Methods("POST")
}
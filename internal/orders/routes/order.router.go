package routes

import (
	"github.com/alibaba0010/postgres-api/internal/common/guards"
	"github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/alibaba0010/postgres-api/internal/database"
	"github.com/alibaba0010/postgres-api/internal/orders/controllers"
	"github.com/alibaba0010/postgres-api/internal/orders/repositories"
	"github.com/alibaba0010/postgres-api/internal/orders/services"
	restRepo "github.com/alibaba0010/postgres-api/internal/restaurants/repositories"
	"github.com/gorilla/mux"
)

func RegisterOrderRoutes(r *mux.Router) {
	// Initialize repositories
	orderRepo := repositories.NewOrderRepository(database.DB)
	menuRepo := restRepo.NewMenuRepository(database.DB)
	restaurantRepo := restRepo.NewRestaurantRepository(database.DB)

	// Initialize service and controller
	orderService := services.NewOrderService(orderRepo, menuRepo, restaurantRepo)
	orderController := controllers.NewOrderController(orderService)

	orderRouter := r.PathPrefix("/orders").Subrouter()
	adminRole:= types.RoleAdmin.String()
	managementRole:= types.RoleManagement.String()

	// All order routes require authentication
	orderRouter.Use(guards.AuthMiddleware)

	orderRouter.HandleFunc("", orderController.CreateOrderHandler).Methods("POST")
	orderRouter.HandleFunc("", orderController.GetUserOrdersHandler).Methods("GET")
	orderRouter.HandleFunc("/{id}", orderController.GetOrderByIDHandler).Methods("GET")

	// Management/Admin routes for status updates
	managementRouter := orderRouter.PathPrefix("/{id}/status").Subrouter()
	managementRouter.Use(guards.RequireRole(adminRole, managementRole))
	managementRouter.HandleFunc("", orderController.UpdateOrderStatusHandler).Methods("PATCH")
}

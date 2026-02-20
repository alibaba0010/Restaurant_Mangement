package routes

import (
	"net/http"

	"github.com/alibaba0010/postgres-api/internal/common/guards"
	"github.com/alibaba0010/postgres-api/internal/common/middlewares"
	"github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/alibaba0010/postgres-api/internal/database"
	"github.com/alibaba0010/postgres-api/internal/restaurants/controllers"
	"github.com/alibaba0010/postgres-api/internal/restaurants/repositories"
	"github.com/alibaba0010/postgres-api/internal/restaurants/services"
	"github.com/gorilla/mux"
)

// CategoryRoutes registers category-related handlers
func CategoryRoutes(route *mux.Router) {
	// Initialize repositories
	categoryRepo := repositories.NewCategoryRepository(database.DB)
	menuRepo := repositories.NewMenuRepository(database.DB)
	restaurantRepo := repositories.NewRestaurantRepository(database.DB)
	
	// Initialize services
	categoryService := services.NewCategoryService(categoryRepo, restaurantRepo)
	menuService := services.NewMenuService(menuRepo, categoryRepo, restaurantRepo, nil)

	// Initialize controller
	categoryController := controllers.NewCategoryController(categoryService, menuService, restaurantRepo)

	// --- Categories Endpoint ---
	categories := route.PathPrefix("/categories").Subrouter()

	// 1. Public Routes (No Auth)
	// List Categories with filters - Public with strict rate limiting
	categories.Handle("", middlewares.RateLimit(5, 10)(http.HandlerFunc(categoryController.ListCategoriesByRestaurantHandler))).Methods("GET")

	// Protected Management Routes with stricter Rate Limiting for Uploads
	management := categories.PathPrefix("").Subrouter()
	management.Use(guards.AuthMiddleware, guards.RequireRole(types.RoleManagement.String()))

	management.HandleFunc("", categoryController.CreateCategoryHandler).Methods("POST")
	management.HandleFunc("/{id}", categoryController.UpdateCategoryHandler).Methods("PUT", "PATCH")
	management.HandleFunc("/{id}", categoryController.DeleteCategoryHandler).Methods("DELETE")
}
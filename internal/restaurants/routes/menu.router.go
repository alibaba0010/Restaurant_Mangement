package routes

import (
	"context"
	"net/http"

	"github.com/alibaba0010/postgres-api/internal/common/guards"
	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/common/middlewares"
	"github.com/alibaba0010/postgres-api/internal/common/s3"
	"github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/alibaba0010/postgres-api/internal/database"
	"github.com/alibaba0010/postgres-api/internal/restaurants/controllers"
	"github.com/alibaba0010/postgres-api/internal/restaurants/repositories"
	"github.com/alibaba0010/postgres-api/internal/restaurants/services"
	"github.com/gorilla/mux"
)

// MenuRoutes registers menu-related handlers
func MenuRoutes(route *mux.Router) {
	// Initialize repositories
	menuRepo := repositories.NewMenuRepository(database.DB)
	restaurantRepo := repositories.NewRestaurantRepository(database.DB)

	// Initialize S3 service
	s3Service, err := s3.NewS3Service(context.Background())
	if err != nil {
		logger.Log.Fatal("failed to initialize S3 service: " + err.Error())
	}

	// Initialize service and controller
	categoryRepo := repositories.NewCategoryRepository(database.DB)
	menuService := services.NewMenuService(menuRepo, categoryRepo, restaurantRepo, s3Service)
	menuController := controllers.NewMenuController(menuService, restaurantRepo)

	// --- Menus Endpoint ---
	menus := route.PathPrefix("/menus").Subrouter()

	// 1. Public Routes (No Auth)
	// List Menus with filters - Public with strict rate limiting
	menus.Handle("", middlewares.RateLimit(5, 10)(http.HandlerFunc(menuController.ListMenusHandler))).Methods("GET")
	menus.HandleFunc("/{id}", menuController.GetMenuHandler).Methods("GET")

	// Protected Management Routes with stricter Rate Limiting for Uploads
	management := menus.PathPrefix("").Subrouter()
	management.Use(guards.AuthMiddleware, guards.RequireRole(types.RoleManagement.String()))

	management.HandleFunc("", menuController.CreateMenuHandler).Methods("POST")
	management.HandleFunc("/{id}", menuController.UpdateMenuHandler).Methods("PUT", "PATCH")
	management.HandleFunc("/{id}", menuController.DeleteMenuHandler).Methods("DELETE")

	// Upload Group with strict rate limiting (Recommendation: Rate limit uploads)
	uploadLimit := middlewares.RateLimit(1, 3) // 1 request per second, 3 burst
	
	management.Handle("/upload", uploadLimit(http.HandlerFunc(menuController.UploadMenuMediaHandler))).Methods("POST")
	management.Handle("/upload-url", uploadLimit(http.HandlerFunc(menuController.GetMenuUploadURLHandler))).Methods("GET")

	// Multipart Upload Group
	multipart := management.PathPrefix("/multipart").Subrouter()
	multipart.Use(uploadLimit)
	multipart.HandleFunc("/initiate", menuController.InitiateMultipartUploadHandler).Methods("POST")
	multipart.HandleFunc("/part-url", menuController.GetMultipartPartURLHandler).Methods("GET")
	multipart.HandleFunc("/complete", menuController.CompleteMultipartUploadHandler).Methods("POST")


}
	
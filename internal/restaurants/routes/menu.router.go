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

	// 1. Protected Management Routes (Specific routes FIRST)
	management := menus.PathPrefix("").Subrouter()
	management.Use(guards.AuthMiddleware, guards.RequireRole(types.RoleManagement.String()))

	uploadLimit := middlewares.RateLimit(2, 4)

	// Specific Management Routes
	management.Handle("/upload", uploadLimit(http.HandlerFunc(menuController.UploadMenuMediaHandler))).Methods("POST")
	management.Handle("/upload-url", uploadLimit(http.HandlerFunc(menuController.GetMenuUploadURLHandler))).Methods("GET")

	// Multipart Group
	multipart := management.PathPrefix("/multipart").Subrouter()
	multipart.Use(uploadLimit)
	multipart.HandleFunc("/initiate", menuController.InitiateMultipartUploadHandler).Methods("POST")
	multipart.HandleFunc("/part-url", menuController.GetMultipartPartURLHandler).Methods("GET")
	// Server-proxied chunk upload — avoids browser→S3 CORS issues
	multipart.HandleFunc("/upload-part", menuController.UploadMultipartPartHandler).Methods("POST")
	multipart.HandleFunc("/complete", menuController.CompleteMultipartUploadHandler).Methods("POST")
	multipart.HandleFunc("/abort", menuController.AbortMultipartUploadHandler).Methods("POST")

	management.HandleFunc("", menuController.CreateMenuHandler).Methods("POST")

	// Generic Management Routes (ID-based) - defined AFTER specific ones
	management.HandleFunc("/{id}", menuController.UpdateMenuHandler).Methods("PUT", "PATCH")
	management.HandleFunc("/{id}", menuController.DeleteMenuHandler).Methods("DELETE")

	// 2. Public Routes
	menus.Handle("", guards.TurnstileMiddleware(middlewares.RateLimit(5, 10)(http.HandlerFunc(menuController.ListMenusHandler)))).Methods("GET")
	
	// Generic Public Route (ID-based) - defined LAST
	menus.HandleFunc("/{id}", menuController.GetMenuHandler).Methods("GET")

}

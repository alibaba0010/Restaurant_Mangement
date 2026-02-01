package routes

import (
	"net/http"

	"github.com/alibaba0010/postgres-api/internal/common/address"
	"github.com/alibaba0010/postgres-api/internal/common/guards"
	"github.com/alibaba0010/postgres-api/internal/common/middlewares"
	"github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/alibaba0010/postgres-api/internal/restaurants/controllers"
	"github.com/alibaba0010/postgres-api/internal/restaurants/services"
	"github.com/gorilla/mux"
)

// MenuRoutes registers menu-related handlers
func MenuRoutes(route *mux.Router) {
	// Initialize services and controllers
	addressService := address.NewService()
	restaurantService := services.NewRestaurantService(addressService)
	menuService, err := services.NewMenuService()
	if err != nil {
		// Log error but continue - service will handle gracefully
		panic("failed to initialize menu service: " + err.Error())
	}
	menuController := controllers.NewMenuController(menuService, restaurantService )

	// Menus endpoint
	menus := route.PathPrefix("/menus").Subrouter()
	managementRole := guards.RequireRole(types.RoleManagement.String())

	// Protected Routes - Only Management can CREATE, UPDATE menus and UPLOAD media
	// Multipart Upload Routes
	multipart := menus.PathPrefix("/multipart").Subrouter()
	multipart.Use(guards.AuthMiddleware, managementRole)
	multipart.HandleFunc("/initiate", menuController.InitiateMultipartUploadHandler).Methods("POST")
	multipart.HandleFunc("/part-url", menuController.GetMultipartPartURLHandler).Methods("GET")
	multipart.HandleFunc("/complete", menuController.CompleteMultipartUploadHandler).Methods("POST")
	
	// Direct Upload (POST /menus/upload)
	menus.Handle("/upload", guards.AuthMiddleware(managementRole(http.HandlerFunc(menuController.UploadMenuMediaHandler)))).Methods("POST")
	
	// Get Upload URL (GET /menus/upload-url)
	menus.Handle("/upload-url", guards.AuthMiddleware(managementRole(http.HandlerFunc(menuController.GetMenuUploadURLHandler)))).Methods("GET")

	// Public Routes
	// List Menus with filters - Public with strict rate limiting (5 req/sec, burst 10)
	menus.Handle("", middlewares.RateLimit(5, 10)(http.HandlerFunc(menuController.ListMenusHandler))).Methods("GET")

	// Get Single Menu item
	menus.Handle("/{id}", http.HandlerFunc(menuController.GetMenuHandler)).Methods("GET")

	// Create Menu (POST /menus)
	menus.Handle("", guards.AuthMiddleware(managementRole(http.HandlerFunc(menuController.CreateMenuHandler)))).Methods("POST")

	// Update Menu (PUT/PATCH /menus/{id})
	menus.Handle("/{id}", guards.AuthMiddleware(managementRole(http.HandlerFunc(menuController.UpdateMenuHandler)))).Methods("PUT", "PATCH")

	// Categories routes
	categories := route.PathPrefix("/categories").Subrouter()
	categories.HandleFunc("", controllers.ListCategoriesHandler).Methods("GET")
	categories.Handle("", guards.AuthMiddleware(managementRole(http.HandlerFunc(controllers.CreateCategoryHandler)))).Methods("POST")
}

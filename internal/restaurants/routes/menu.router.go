package routes

import (
	"net/http"

	"github.com/alibaba0010/postgres-api/internal/common/guards"
	"github.com/alibaba0010/postgres-api/internal/common/middlewares"
	"github.com/alibaba0010/postgres-api/internal/restaurants/controllers"
	"github.com/alibaba0010/postgres-api/internal/common/types"

	"github.com/gorilla/mux"
)

// MenuRoutes registers menu-related handlers
func MenuRoutes(route *mux.Router) {
	// Menus endpoint
	menus := route.PathPrefix("/menus").Subrouter()
	managementRole := guards.RequireRole(types.RoleManagement.String())

	// Protected Routes - Only Management can CREATE, UPDATE menus and UPLOAD media
	// Multipart Upload Routes
	multipart := menus.PathPrefix("/multipart").Subrouter()
	multipart.Use(guards.AuthMiddleware, managementRole)
	multipart.HandleFunc("/initiate", controllers.InitiateMultipartUploadHandler).Methods("POST")
	multipart.HandleFunc("/part-url", controllers.GetMultipartPartURLHandler).Methods("GET")
	multipart.HandleFunc("/complete", controllers.CompleteMultipartUploadHandler).Methods("POST")
	
	// Direct Upload (POST /menus/upload)
	menus.Handle("/upload", guards.AuthMiddleware(managementRole(http.HandlerFunc(controllers.UploadMenuMediaHandler)))).Methods("POST")
	
	// Get Upload URL (GET /menus/upload-url)
	menus.Handle("/upload-url", guards.AuthMiddleware(managementRole(http.HandlerFunc(controllers.GetMenuUploadURLHandler)))).Methods("GET")


	// Public Routes
	// List Menus with filters - Public with strict rate limiting (5 req/sec, burst 10)
	menus.Handle("", middlewares.RateLimit(5, 10)(http.HandlerFunc(controllers.ListMenusHandler))).Methods("GET")

	// Get Single Menu item
	menus.Handle("/{id}", http.HandlerFunc(controllers.GetMenuHandler)).Methods("GET")

	// Create Menu (POST /menus)
	menus.Handle("", guards.AuthMiddleware(managementRole(http.HandlerFunc(controllers.CreateMenuHandler)))).Methods("POST")

	// Update Menu (PUT/PATCH /menus/{id})
	menus.Handle("/{id}", guards.AuthMiddleware(managementRole(http.HandlerFunc(controllers.UpdateMenuHandler)))).Methods("PUT", "PATCH")
}

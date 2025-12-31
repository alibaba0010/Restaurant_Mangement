package routes

import (
	"net/http"

	"github.com/alibaba0010/postgres-api/internal/common/guards"
	"github.com/alibaba0010/postgres-api/internal/common/middlewares"
	"github.com/alibaba0010/postgres-api/internal/restaurants/controllers"
	"github.com/gorilla/mux"
)

// MenuRoutes registers menu-related handlers
func MenuRoutes(route *mux.Router) {
	// Menus endpoint
	menus := route.PathPrefix("/menus").Subrouter()

	// Public Routes
	// List Menus with filters - Public with strict rate limiting (5 req/sec, burst 10)
	menus.Handle("", middlewares.RateLimit(5, 10)(http.HandlerFunc(controllers.ListMenusHandler))).Methods("GET")

	// Get Single Menu item
	menus.Handle("/{id}", http.HandlerFunc(controllers.GetMenuHandler)).Methods("GET")

	// Protected Routes - Only Management can CREATE, UPDATE menus and UPLOAD media
	// Create Menu (POST /menus)
	menus.Handle("", guards.AuthMiddleware(guards.RequireRole("management")(http.HandlerFunc(controllers.CreateMenuHandler)))).Methods("POST")

	// Update Menu (PUT/PATCH /menus/{id})
	menus.Handle("/{id}", guards.AuthMiddleware(guards.RequireRole("management")(http.HandlerFunc(controllers.UpdateMenuHandler)))).Methods("PUT", "PATCH")

	// Upload Media (POST /menus/upload)
	menus.Handle("/upload", guards.AuthMiddleware(guards.RequireRole("management")(http.HandlerFunc(controllers.UploadMenuMediaHandler)))).Methods("POST")
}

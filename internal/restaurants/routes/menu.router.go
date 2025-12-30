package routes

import (
	"net/http"

	"github.com/alibaba0010/postgres-api/internal/common/guards"
	"github.com/alibaba0010/postgres-api/internal/restaurants/controllers"
	"github.com/gorilla/mux"
)

// MenuRoutes registers menu-related handlers
func MenuRoutes(route *mux.Router) {
	// Menus endpoint
	menus := route.PathPrefix("/menus").Subrouter()

	// Protected Routes - Only Management can CREATE menus and UPLOAD media
	// Create Menu (POST /menus)
	menus.Handle("", guards.AuthMiddleware(guards.RequireRole("management")(http.HandlerFunc(controllers.CreateMenuHandler)))).Methods("POST")

	// Upload Media (POST /menus/upload)
	menus.Handle("/upload", guards.AuthMiddleware(guards.RequireRole("management")(http.HandlerFunc(controllers.UploadMenuMediaHandler)))).Methods("POST")
}

package routes

import (
	"github.com/alibaba0010/postgres-api/internal/controllers"
	"github.com/alibaba0010/postgres-api/internal/guards"
	"github.com/gorilla/mux"
)

// UserRoutes defines user-related routes with appropriate middleware and role-based access control
func UserRoutes(route *mux.Router) {
	userRouter := route.PathPrefix("/user").Subrouter()
	userRouter.Use(guards.AuthMiddleware)

	// All authenticated users
	userRouter.HandleFunc("", controllers.CurrentUserHandler).Methods("GET")
	userRouter.HandleFunc("", controllers.UpdateUserHandler).Methods("PATCH")// work on this later
	userRouter.HandleFunc("/logout", controllers.LogoutHandler).Methods("POST")

	// Admin only
	userRouter.Use(guards.RequireRole("admin"))
	userRouter.HandleFunc("/{id}", controllers.GetUserByIDHandler).Methods("GET")
}
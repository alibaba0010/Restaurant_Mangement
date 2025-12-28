package routes

import (
	"github.com/alibaba0010/postgres-api/internal/auth/controllers"
	"github.com/alibaba0010/postgres-api/internal/common/guards"
	"github.com/gorilla/mux"
)

// UserRoutes defines user-related routes with appropriate middleware and role-based access control
func UserRoutes(route *mux.Router) {
	// 1. Base /user subrouter (requires authentication)
	userRouter := route.PathPrefix("/user").Subrouter()
	userRouter.Use(guards.AuthMiddleware)

	userRouter.HandleFunc("", controllers.CurrentUserHandler).Methods("GET")
	userRouter.HandleFunc("", controllers.UpdateUserHandler).Methods("PATCH")
	userRouter.HandleFunc("/logout", controllers.LogoutHandler).Methods("POST")

	// 2. Admin-only user listing (/users)
	adminListRouter := userRouter.PathPrefix("/users").Subrouter()
	adminListRouter.Use(guards.RequireRole("admin"))
	adminListRouter.HandleFunc("", controllers.GetAllUsersHandler).Methods("GET")

	// 3. Admin action on specific users (/user/{id})
	adminUserRouter := userRouter.PathPrefix("/{id}").Subrouter()
	adminUserRouter.Use(guards.RequireRole("admin"))
	adminUserRouter.HandleFunc("", controllers.GetUserByIDHandler).Methods("GET")
	adminUserRouter.HandleFunc("/role", controllers.UpdateUserRoleHandler).Methods("PATCH")
}

//	userRouter.HandleFunc("/role/{id}", controllers.UpdateUserRoleHandler).Methods("PATCH")
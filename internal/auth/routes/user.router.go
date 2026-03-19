package routes

import (
"github.com/alibaba0010/postgres-api/internal/auth/controllers"
"github.com/alibaba0010/postgres-api/internal/common/guards"
"github.com/alibaba0010/postgres-api/internal/common/types"

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

// Address management routes (authenticated user only)
userRouter.HandleFunc("/addresses/{addressId}", controllers.DeleteUserAddressHandler).Methods("DELETE")
userRouter.HandleFunc("/addresses/{addressId}/default", controllers.SetDefaultUserAddressHandler).Methods("PATCH")

// 2. Admin and management user listing (/users)
adminListRouter := userRouter.PathPrefix("/users").Subrouter()
adminListRouter.Use(guards.RequireRole(types.RoleAdmin.String(), types.RoleManagement.String()))
adminListRouter.HandleFunc("", controllers.GetAllUsersHandler).Methods("GET")

// 3. Admin and management actions on specific users (/user/{id})
adminUserRouter := userRouter.PathPrefix("/{id}").Subrouter()
adminUserRouter.Use(guards.RequireRole(types.RoleAdmin.String(), types.RoleManagement.String()))
adminUserRouter.HandleFunc("", controllers.GetUserByIDHandler).Methods("GET")
adminUserRouter.HandleFunc("/role", controllers.UpdateUserRoleStatusHandler).Methods("PATCH")
}

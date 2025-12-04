package routes

import (
	"github.com/alibaba0010/postgres-api/internal/controllers"
	"github.com/alibaba0010/postgres-api/internal/guards"
	"github.com/gorilla/mux"
)

// UserRoutes defines user-related routes with appropriate middleware and role-based access control
func UserRoutes(route *mux.Router) {
	// All user routes require authentication
	userRouter := route.PathPrefix("/user").Subrouter()
	userRouter.Use(guards.AuthMiddleware)

	// GET /user - Get current authenticated user (accessible to all authenticated users)
	userRouter.HandleFunc("", controllers.CurrentUserHandler).Methods("GET")

	// GET /user/:id - Get specific user by ID (owner, admin, or management can view)
	userRouter.HandleFunc("/{id}", controllers.GetUserByIDHandler).Methods("GET")

	// PATCH /user/address - Update current user's address
	userRouter.HandleFunc("", controllers.UpdateUserAddressHandler).Methods("PATCH")

	// POST /user/logout - Logout current user (revoke all refresh tokens)
	userRouter.HandleFunc("/logout", controllers.LogoutHandler).Methods("POST")

	// Additional role-based endpoints can be added here:
	// Example: GET /user/admin - only for admin users
	// adminRouter := userRouter.PathPrefix("/admin").Subrouter()
	// adminRouter.Use(guards.RequireRole("admin"))
	// adminRouter.HandleFunc("", controllers.ListAllUsersHandler).Methods("GET")

	// Example: GET /user/profile - only for users and admins (can be extended)
	// profileRouter := userRouter.PathPrefix("/profile").Subrouter()
	// profileRouter.Use(guards.RequireRole("user", "admin"))
	// profileRouter.HandleFunc("", controllers.GetProfileHandler).Methods("GET")
}
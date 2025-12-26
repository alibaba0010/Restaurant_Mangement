package routes

import (
	"github.com/alibaba0010/postgres-api/internal/auth/controllers"
	"github.com/gorilla/mux"
)

// AuthRoutes registers auth-related handlers onto the provided subrouter.
func AuthRoutes(route *mux.Router) {
	route.HandleFunc("/signup", controllers.SignupHandler).Methods("POST")
	route.HandleFunc("/verify", controllers.ActivateUserHandler).Methods("GET")
	route.HandleFunc("/resend", controllers.ResendVerificationHandler).Methods("POST")
	route.HandleFunc("/signin", controllers.SigninHandler).Methods("POST")
	route.HandleFunc("/refresh", controllers.RefreshTokenHandler).Methods("POST")
	route.HandleFunc("/{provider}/login", controllers.InitiateOAuthHandler).Methods("GET")
	route.HandleFunc("/{provider}/verify", controllers.VerifyOAuthHandler).Methods("POST")
}
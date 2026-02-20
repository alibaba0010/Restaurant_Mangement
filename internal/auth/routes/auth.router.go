package routes

import (
	"github.com/alibaba0010/postgres-api/internal/auth/controllers"
	"github.com/alibaba0010/postgres-api/internal/common/guards"
	"github.com/gorilla/mux"
)

// AuthRoutes registers auth-related handlers onto the provided subrouter.
func AuthRoutes(route *mux.Router) {
	// Public routes with Turnstile protection
	protected := route.PathPrefix("").Subrouter()
	protected.Use(guards.TurnstileMiddleware)

	protected.HandleFunc("/signup", controllers.SignupHandler).Methods("POST")
	protected.HandleFunc("/signin", controllers.SigninHandler).Methods("POST")
	protected.HandleFunc("/forgot-password", controllers.ForgotPasswordHandler).Methods("POST")
	protected.HandleFunc("/resend", controllers.ResendVerificationHandler).Methods("POST")

	// Other routes
	route.HandleFunc("/verify", controllers.ActivateUserHandler).Methods("GET")
	route.HandleFunc("/refresh", controllers.RefreshTokenHandler).Methods("POST")
	route.HandleFunc("/reset-password", controllers.ResetPasswordHandler).Methods("POST")
	route.HandleFunc("/{provider}/login", controllers.InitiateOAuthHandler).Methods("GET")
	route.HandleFunc("/{provider}/verify", controllers.VerifyOAuthHandler).Methods("POST")
}

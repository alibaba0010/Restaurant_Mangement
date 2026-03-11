package routes

import (
	"github.com/alibaba0010/postgres-api/internal/common/events"
	"github.com/alibaba0010/postgres-api/internal/common/guards"
	"github.com/alibaba0010/postgres-api/internal/database"
	orderRepo "github.com/alibaba0010/postgres-api/internal/orders/repositories"
	"github.com/alibaba0010/postgres-api/internal/payments/controllers"
	"github.com/alibaba0010/postgres-api/internal/payments/services"
	restRepo "github.com/alibaba0010/postgres-api/internal/restaurants/repositories"
	"github.com/gorilla/mux"
)

func RegisterPaymentRoutes(r *mux.Router) {
	producer := events.GetGlobalProducer()
	orderRepoInstance := orderRepo.NewOrderRepository(database.DB)
	menuRepoInstance := restRepo.NewMenuRepository(database.DB)
	svc := services.NewPaymentService(producer, orderRepoInstance, menuRepoInstance)
	ctrl := controllers.NewPaymentController(svc)
	
	paymentRouter := r.PathPrefix("/payments").Subrouter()
	
	// Public Routes (Webhooks)
	paymentRouter.HandleFunc("/webhook/{provider}", ctrl.WebhookHandler).Methods("POST")

	// Auth Routes
	authRouter := paymentRouter.PathPrefix("").Subrouter()
	authRouter.Use(guards.AuthMiddleware)
	authRouter.HandleFunc("/initiate", ctrl.InitiatePayment).Methods("POST")
	authRouter.HandleFunc("/verify", ctrl.VerifyPayment).Methods("GET")
}

package routes

import (
	"net/http"

	authRoutes "github.com/alibaba0010/postgres-api/internal/auth/routes"
	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/middlewares"
	orderRoutes "github.com/alibaba0010/postgres-api/internal/orders/routes"
	paymentRoutes "github.com/alibaba0010/postgres-api/internal/payments/routes"
	restaurantRoutes "github.com/alibaba0010/postgres-api/internal/restaurants/routes"
	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
)

func ApiRouter(allowedOrigin string) http.Handler {
	route := mux.NewRouter()

	// Serve Swagger UI at /swagger/
	route.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	// Create v1 subrouter with /api/v1 prefix
	v1 := route.PathPrefix("/api/v1").Subrouter()

	// Routes
	v1.HandleFunc("/healthcheck", HealthCheckHandler).Methods("GET")
	authRoutes.AuthRoutes(v1.PathPrefix("/auth").Subrouter())
	authRoutes.UserRoutes(v1)
	restaurantRoutes.RestaurantRoutes(v1)
	restaurantRoutes.MenuRoutes(v1)
	restaurantRoutes.CategoryRoutes(v1)
	orderRoutes.RegisterOrderRoutes(v1)
	paymentRoutes.RegisterPaymentRoutes(v1)



	route.NotFoundHandler = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		errors.ErrorResponse(writer, request, errors.RouteNotExist())
	})

	var handler http.Handler = route

	// global rate limit: 100 req/sec, burst 200 (tune as needed)
	handler = middlewares.RateLimit(100, 200)(handler)
	
	// Issue 5.1: Request ID for tracing
	handler = middlewares.RequestID(handler)
	
	// Priority 3: Logging
	handler = middlewares.RequestLogger()(handler)
	
	// Priority 2: Recovery should be next to handle any panics in downstream loggers or handlers
	handler = middlewares.Recover()(handler)
	
	// Priority 1: CORS should be the absolute first middleware to handle OPTIONS preflight
	handler = middlewares.CORS(allowedOrigin)(handler)

	return handler
}

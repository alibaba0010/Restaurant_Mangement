package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/database"
	"go.uber.org/zap"
)

type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	services := make(map[string]string)
	overallStatus := "healthy"

	// Check Postgres
	if err := database.CheckHealth(ctx); err != nil {
		services["postgres"] = "unhealthy: " + err.Error()
		overallStatus = "degraded"
		logger.Log.Error("Health check failed for Postgres", zap.Error(err))
	} else {
		services["postgres"] = "healthy"
	}

	// Check Redis
	if err := database.CheckRedisHealth(ctx); err != nil {
		services["redis"] = "unhealthy: " + err.Error()
		overallStatus = "degraded"
		logger.Log.Error("Health check failed for Redis", zap.Error(err))
	} else {
		services["redis"] = "healthy"
	}

	// You could also check Redpanda here if you have a way to ping it easily
	// For now, these are the core dependencies.

	resp := HealthResponse{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Services:  services,
	}

	w.Header().Set("Content-Type", "application/json")
	if overallStatus == "unhealthy" {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	json.NewEncoder(w).Encode(resp)
}

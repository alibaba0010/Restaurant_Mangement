package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"go.uber.org/zap"

	_ "github.com/alibaba0010/postgres-api/docs" // swagger docs
	"github.com/alibaba0010/postgres-api/internal/common/events"
	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/common/middlewares"
	"github.com/alibaba0010/postgres-api/internal/common/routes"
	"github.com/alibaba0010/postgres-api/internal/config"
	"github.com/alibaba0010/postgres-api/internal/database"
)

func main() {
	fmt.Print("\033[H\033[2J")

	logger.InitLogger()
	// defer sync to flush logs on program exit
	defer logger.Sync()

	database.ConnectDB()
	defer database.CloseDB()

	database.ConnectRedis()
	defer database.CloseRedis()

	cfg := config.LoadConfig()

	// Initialize Redpanda Producer
	producer, err := events.NewRedpandaProducer([]string{cfg.REDPANDA_BROKERS})
	if err != nil {
		logger.Log.Warn("⚠️ Failed to initialize Redpanda Producer (Events will be disabled)", zap.Error(err))
	} else {
		events.SetGlobalProducer(producer)
		defer producer.Close()
		logger.Log.Info("✅ Connected to Redpanda")
	}

	port := cfg.Port
	frontendUrl := cfg.FRONTEND_URL
	route := routes.ApiRouter()

	// Apply CORS middleware globally
	handler := middlewares.CORS(frontendUrl)(route)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  5 * time.Minute,
		WriteTimeout: 5 * time.Minute,
		IdleTimeout:  10 * time.Minute,
	}

	logger.Log.Info("🚀 Server Listening on ", zap.String("url", "http://localhost:"+port+"/swagger/index.html"))

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	_ "github.com/alibaba0010/postgres-api/docs"
	"github.com/alibaba0010/postgres-api/internal/common/events"
	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/common/middlewares"
	"github.com/alibaba0010/postgres-api/internal/config"
	"github.com/alibaba0010/postgres-api/internal/database"
	"github.com/alibaba0010/postgres-api/internal/orders"
	"github.com/alibaba0010/postgres-api/internal/routes"
)

func main() {
    fmt.Print("\033[H\033[2J")

    logger.InitLogger()
	// defer sync to flush logs on program exit
    defer logger.Sync()

    // Database connections
    database.ConnectDB()
    defer database.CloseDB()

    database.ConnectRedis()
    defer database.CloseRedis()

    cfg := config.LoadConfig()

    // Create shutdown context
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Initialize Redpanda
    producer, consumer, err := events.InitializeRedpanda(ctx, cfg)
    if err != nil {
        logger.Log.Fatal("Failed to initialize Redpanda Consumer after retries", zap.Error(err))
    }
    if producer != nil {
        defer producer.Close()
    }
    defer consumer.Close()

    // Register Module Subscribers
    orders.RegisterSubscribers(consumer)

    // Start Consumer with graceful shutdown
    consumerDone := make(chan error, 1)
    go func() {
        consumerDone <- consumer.Start(ctx)
    }()
    logger.Log.Info("✅ Redpanda Consumer Started")

    // Setup HTTP server
    port := cfg.Port
    frontendUrl := cfg.FRONTEND_URL
    route := routes.ApiRouter()
    handler := middlewares.CORS(frontendUrl)(route)

    server := &http.Server{
        Addr:         ":" + port,
        Handler:      handler,
        ReadTimeout:  5 * time.Minute,
        WriteTimeout: 5 * time.Minute,
        IdleTimeout:  10 * time.Minute,
    }

    // Start server in goroutine
    serverErrors := make(chan error, 1)
    go func() {
        logger.Log.Info("🚀 Server Listening", 
            zap.String("url", "http://localhost:"+port+"/swagger/index.html"))
        serverErrors <- server.ListenAndServe()
    }()

    // Handle graceful shutdown
    shutdown := make(chan os.Signal, 1)
    signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

    select {
    case err := <-serverErrors:
        logger.Log.Fatal("Server error", zap.Error(err))

    case err := <-consumerDone:
        logger.Log.Error("Consumer stopped", zap.Error(err))

    case sig := <-shutdown:
        logger.Log.Info("Shutdown signal received", zap.String("signal", sig.String()))

        // Cancel consumer context
        cancel()

        // Shutdown HTTP server gracefully
        shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer shutdownCancel()

        if err := server.Shutdown(shutdownCtx); err != nil {
            logger.Log.Error("Server shutdown error", zap.Error(err))
            if err := server.Close(); err != nil {
                logger.Log.Fatal("Server close error", zap.Error(err))
            }
        }

        logger.Log.Info("Shutdown complete")
    }
}

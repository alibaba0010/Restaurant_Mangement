package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alibaba0010/postgres-api/internal/common/events"
	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/config"
	"github.com/alibaba0010/postgres-api/internal/database"
	"github.com/alibaba0010/postgres-api/internal/orders"
	"go.uber.org/zap"
)

func main() {
	logger.InitLogger()
	defer logger.Sync()

	database.ConnectDB()
	defer database.CloseDB()

	database.ConnectRedis()
	defer database.CloseRedis()

	cfg := config.LoadConfig()

	// Initialize Redpanda Consumer for the standalone Service with retry
	var consumer *events.RedpandaConsumer
	var err error
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		consumer, err = events.NewRedpandaConsumer([]string{cfg.REDPANDA_BROKERS}, "orders-service-group")
		if err == nil {
			pingCtx, cancelPing := context.WithTimeout(context.Background(), 5*time.Second)
			err = consumer.Ping(pingCtx)
			cancelPing()
			
			if err == nil {
				break
			}
		}
		
		logger.Log.Warn("Failed to connect to Redpanda Consumer for Orders, retrying...", 
			zap.Int("attempt", i+1), 
			zap.Error(err))
		time.Sleep(time.Second * time.Duration(i+1))
	}

	if err != nil {
		logger.Log.Fatal("Failed to initialize Redpanda Consumer for Orders after retries", zap.Error(err))
	}
	defer consumer.Close()

	// Use the internal/orders module initializer
	orders.RegisterSubscribers(consumer)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := consumer.Start(ctx); err != nil {
			logger.Log.Error("Orders consumer stopped with error", zap.Error(err))
		}
	}()

	logger.Log.Info("🚀 Orders Microservice (Subscribers) Started")

	<-shutdown
	cancel()
	logger.Log.Info("Shutdown signal received, closing Orders Service...")
}

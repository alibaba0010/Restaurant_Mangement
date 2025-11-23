package database

import (
	"context"
    "time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/alibaba0010/postgres-api/internal/logger"
	"github.com/alibaba0010/postgres-api/internal/config"
)

var RedisClient *redis.Client

func ConnectRedis() *redis.Client {
	cfg := config.LoadConfig()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

	RedisClient = redis.NewClient(&redis.Options{
		Addr:     cfg.REDIS_HOST + ":" + cfg.REDIS_PORT,
		Password: cfg.REDIS_PASSWORD, // no password set
		DB:       0,  // use default DB
	})

	// Ping to verify connection
	err := RedisClient.Ping(ctx).Err()
	if err != nil {
		logger.Log.Fatal("❌ Redis connection failed", zap.Error(err))
	}

	logger.Log.Info("✅ Connected to Redis")
	return RedisClient
}

// Close connection when shutting down
func CloseRedis() {
	if RedisClient != nil {
		err := RedisClient.Close()
		if err != nil {
			logger.Log.Error("Error closing Redis connection", zap.Error(err))
		}
		logger.Log.Info("🔌 Redis connection closed")
	}
}
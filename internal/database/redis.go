package database

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/config"
)

var RedisClient *redis.Client

func ConnectRedis() *redis.Client {
	cfg := config.LoadConfig()

	// short-lived context for initial ping
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

	// Start a background janitor to clean up any verify keys that were stored
	// without TTL (orphaned). Redis normally expires keys automatically, but
	// if a key was stored without an expiry due to a bug, this janitor will
	// remove `verify:*` keys that have no TTL set (TTL == -1).
	go startRedisJanitor()
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

// startRedisJanitor launches a goroutine that periodically scans Redis for
// verification keys (prefix "verify:") and deletes keys that have no TTL set.
// This is defensive: Redis normally expires keys, but this removes orphaned
// verify keys that would otherwise persist indefinitely.
func startRedisJanitor() {
	ctx := context.Background()
	ticker := time.NewTicker(5 * time.Minute)
	// Run an initial cleanup quickly
	doCleanup := func() {
		var cursor uint64
		for {
			keys, cur, err := RedisClient.Scan(ctx, cursor, "verify:*", 100).Result()
			if err != nil {
				logger.Log.Error("redis janitor scan failed", zap.Error(err))
				break
			}
			for _, k := range keys {
				ttl, err := RedisClient.TTL(ctx, k).Result()
				if err != nil {
					logger.Log.Error("redis janitor ttl failed", zap.Error(err), zap.String("key", k))
					continue
				}
				// TTL == -1 means key exists but has no expiry; remove it.
				if ttl == -1 {
					if err := RedisClient.Del(ctx, k).Err(); err != nil {
						logger.Log.Error("redis janitor del failed", zap.Error(err), zap.String("key", k))
					} else {
						logger.Log.Info("redis janitor deleted orphaned key", zap.String("key", k))
					}
				}
			}
			cursor = cur
			if cursor == 0 {
				break
			}
		}
	}

	// initial run
	doCleanup()

	go func() {
		for range ticker.C {
			doCleanup()
		}
	}()
}
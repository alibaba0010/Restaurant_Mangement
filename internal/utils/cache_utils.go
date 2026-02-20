package utils

import (
	"context"

	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/database"
	"go.uber.org/zap"
)

// InvalidateCacheByPrefix scans and deletes all Redis keys starting with the given prefix.
func InvalidateCacheByPrefix(ctx context.Context, prefix string) {
	if database.RedisClient == nil {
		return
	}
	var cursor uint64
	for {
		keys, cur, err := database.RedisClient.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			logger.Log.Error("failed to scan cache keys for invalidation", zap.String("prefix", prefix), zap.Error(err))
			break
		}
		if len(keys) > 0 {
			if err := database.RedisClient.Del(ctx, keys...).Err(); err != nil {
				logger.Log.Error("failed to delete cache keys", zap.String("prefix", prefix), zap.Error(err))
			}
		}
		cursor = cur
		if cursor == 0 {
			break
		}
	}
	logger.Log.Info("Cache invalidated", zap.String("prefix", prefix))
}

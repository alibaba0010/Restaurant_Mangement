package events

import (
	"context"
	"time"

	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/config"
	"go.uber.org/zap"
)

// InitializeRedpanda initializes the Redpanda producer and consumer with retry logic
func InitializeRedpanda(ctx context.Context, cfg config.Config) (*RedpandaProducer, *RedpandaConsumer, error) {
	// Initialize Redpanda Producer with retry
	var producer *RedpandaProducer
	var err error

	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		producer, err = NewRedpandaProducer([]string{cfg.REDPANDA_BROKERS})
		if err == nil {
			// Verify connection with Ping
			pingCtx, cancelPing := context.WithTimeout(context.Background(), 5*time.Second)
			err = producer.Ping(pingCtx)
			cancelPing()

			if err == nil {
				break
			}
		}

		logger.Log.Warn("Failed to connect to Redpanda Producer, retrying...",
			zap.Int("attempt", i+1),
			zap.Error(err))
		time.Sleep(time.Second * time.Duration(i+1))
	}

	if err != nil {
		logger.Log.Error("Could not connect to Redpanda Producer. Events will not be published.", zap.Error(err))
	} else {
		SetGlobalProducer(producer)
		logger.Log.Info("✅ Connected to Redpanda Producer")
	}

	// Initialize Redpanda Consumer
	var consumer *RedpandaConsumer
	for i := 0; i < maxRetries; i++ {
		consumer, err = NewRedpandaConsumer([]string{cfg.REDPANDA_BROKERS}, "postgres-api-group")
		if err == nil {
			pingCtx, cancelPing := context.WithTimeout(context.Background(), 5*time.Second)
			err = consumer.Ping(pingCtx)
			cancelPing()

			if err == nil {
				break
			}
		}

		logger.Log.Warn("Failed to connect to Redpanda Consumer, retrying...",
			zap.Int("attempt", i+1),
			zap.Error(err))
		time.Sleep(time.Second * time.Duration(i+1))
	}

	if err != nil {
		return producer, nil, err
	}

	return producer, consumer, nil
}

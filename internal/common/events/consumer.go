package events

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"
)

// RedpandaConsumer implements the Consumer interface using Redpanda
type RedpandaConsumer struct {
	client   *kgo.Client
	handlers map[string]EventHandler
	mu       sync.RWMutex
	started  bool
}

// NewRedpandaConsumer creates a new RedpandaConsumer using kgo
func NewRedpandaConsumer(brokers []string, groupID string) (*RedpandaConsumer, error) {
	if len(brokers) == 0 {
		return nil, fmt.Errorf("at least one broker required")
	}

	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(groupID),
		kgo.DisableAutoCommit(), // Manage commits explicitly
		kgo.FetchMaxWait(5*time.Second),
		kgo.FetchMinBytes(1),
		kgo.RequestRetries(3),
		kgo.RetryBackoffFn(func(tries int) time.Duration {
			return time.Duration(math.Pow(2, float64(tries))) * time.Second
		}),
		kgo.OnPartitionsAssigned(func(ctx context.Context, c *kgo.Client, assigned map[string][]int32) {
			logger.Log.Info("Partitions assigned", zap.Any("partitions", assigned))
		}),
		kgo.OnPartitionsRevoked(func(ctx context.Context, c *kgo.Client, revoked map[string][]int32) {
			logger.Log.Info("Partitions revoked", zap.Any("partitions", revoked))
			c.CommitUncommittedOffsets(ctx)
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	return &RedpandaConsumer{
		client:   client,
		handlers: make(map[string]EventHandler),
	}, nil
}

// Subscribe registers a handler for a topic. Cannot be called after Start().
func (c *RedpandaConsumer) Subscribe(topic string, handler EventHandler) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return fmt.Errorf("cannot subscribe after consumer has started")
	}

	c.handlers[topic] = handler
	c.client.AddConsumeTopics(topic)

	return nil
}

// Ping checks if the Redpanda cluster is reachable with a timeout
func (c *RedpandaConsumer) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := c.client.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping redpanda: %w", err)
	}
	return nil
}

// Start begins polling for messages with graceful shutdown support
func (c *RedpandaConsumer) Start(ctx context.Context) error {
	c.mu.Lock()
	c.started = true
	c.mu.Unlock()

	for {
		// Check for context cancellation before polling
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		fetches := c.client.PollFetches(ctx)
		if err := fetches.Err(); err != nil {
			if err == context.Canceled {
				return nil
			}
			// Log error and continue or return
			logger.Log.Error("Poll error", zap.Error(err))
			continue
		}

		iter := fetches.RecordIter()
		for !iter.Done() {
			record := iter.Next()
			c.mu.RLock()
			handler, ok := c.handlers[record.Topic]
			c.mu.RUnlock()

			if ok {
				event := BaseEvent{
					EventTopic:   record.Topic,
					EventKey:     string(record.Key),
					EventPayload: record.Value,
					Partition:    record.Partition,
					Offset:       record.Offset,
					Timestamp:    record.Timestamp,
				}

				if err := handler(ctx, event); err != nil {
					logger.Log.Error("Error handling event",
						zap.String("topic", record.Topic),
						zap.Int32("partition", record.Partition),
						zap.Int64("offset", record.Offset),
						zap.Error(err),
					)
					// Don't commit on error to allow for potential retry (depends on policy)
					continue
				}

				// Manual commit after successful processing
				c.client.CommitRecords(ctx, record)
			}
		}
	}
}

// Close closes the consumer connection
func (c *RedpandaConsumer) Close() {
	c.client.Close()
}

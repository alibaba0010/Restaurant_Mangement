package events

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"
)

// RedpandaProducer implements the Producer interface using Redpanda
type RedpandaProducer struct {
	client *kgo.Client
}

// NewRedpandaProducer creates a new RedpandaProducer
func NewRedpandaProducer(brokers []string) (*RedpandaProducer, error) {
	if len(brokers) == 0 {
		return nil, fmt.Errorf("at least one broker required")
	}

	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.AllowAutoTopicCreation(),
		kgo.RequiredAcks(kgo.AllISRAcks()), // Wait for all replicas
		kgo.ProducerLinger(10*time.Millisecond),
		kgo.ProducerBatchCompression(kgo.SnappyCompression()), // Enable compression
		kgo.RequestRetries(3),
		kgo.RetryBackoffFn(func(tries int) time.Duration {
			return time.Duration(math.Pow(2, float64(tries))) * time.Second
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create redpanda client: %w", err)
	}

	return &RedpandaProducer{client: client}, nil
}

// Ping checks if the Redpanda cluster is reachable with a timeout
func (p *RedpandaProducer) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := p.client.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping redpanda: %w", err)
	}
	return nil
}

// Publish publishes an event to Redpanda synchronously
func (p *RedpandaProducer) Publish(ctx context.Context, event Event) error {
	record := &kgo.Record{
		Topic: event.Topic(),
		Key:   []byte(event.Key()),
		Value: event.Payload(),
	}

	if err := p.client.ProduceSync(ctx, record).FirstErr(); err != nil {
		logger.Log.Error("Failed to produce record",
			zap.String("topic", event.Topic()),
			zap.String("key", event.Key()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to produce record: %w", err)
	}

	return nil
}

// PublishAsync publishes multiple events to Redpanda asynchronously
func (p *RedpandaProducer) PublishAsync(ctx context.Context, events []Event, callback func(error)) {
	if len(events) == 0 {
		return
	}

	for _, event := range events {
		record := &kgo.Record{
			Topic: event.Topic(),
			Key:   []byte(event.Key()),
			Value: event.Payload(),
		}

		p.client.Produce(ctx, record, func(r *kgo.Record, err error) {
			if err != nil {
				logger.Log.Error("Async produce failed",
					zap.String("topic", r.Topic),
					zap.Error(err),
				)
			}
			if callback != nil {
				callback(err)
			}
		})
	}
}

// Close closes the producer connection
func (p *RedpandaProducer) Close() {
	p.client.Close()
}

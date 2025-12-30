package events

import (
	"context"
	"fmt"

	"github.com/twmb/franz-go/pkg/kgo"
)

// RedpandaProducer implements the Producer interface using Redpanda
type RedpandaProducer struct {
	client *kgo.Client
}

// NewRedpandaProducer creates a new RedpandaProducer
func NewRedpandaProducer(brokers []string) (*RedpandaProducer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.AllowAutoTopicCreation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create redpanda client: %w", err)
	}

	return &RedpandaProducer{client: client}, nil
}

// Publish publishes an event to Redpanda
func (p *RedpandaProducer) Publish(ctx context.Context, event Event) error {
	record := &kgo.Record{
		Topic: event.Topic(),
		Key:   []byte(event.Key()),
		Value: event.Payload(),
	}

	if err := p.client.ProduceSync(ctx, record).FirstErr(); err != nil {
		return fmt.Errorf("failed to produce record: %w", err)
	}

	return nil
}

// Close closes the producer connection
func (p *RedpandaProducer) Close() {
	p.client.Close()
}

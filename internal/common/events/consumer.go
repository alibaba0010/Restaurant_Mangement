package events

import (
	"context"
	"fmt"
	"sync"

	"github.com/twmb/franz-go/pkg/kgo"
)

// RedpandaConsumer implements the Consumer interface using Redpanda
type RedpandaConsumer struct {
	client   *kgo.Client
	handlers map[string]EventHandler
	mu       sync.RWMutex
}

// NewRedpandaConsumer creates a new RedpandaConsumer using kgo
func NewRedpandaConsumer(brokers []string, groupID string) (*RedpandaConsumer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(groupID),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	return &RedpandaConsumer{
		client:   client,
		handlers: make(map[string]EventHandler),
	}, nil
}

// Subscribe registers a handler for a topic
func (c *RedpandaConsumer) Subscribe(topic string, handler EventHandler) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.handlers[topic] = handler
	c.client.AddConsumeTopics(topic)

	return nil
}

// Ping checks if the Redpanda cluster is reachable
func (c *RedpandaConsumer) Ping(ctx context.Context) error {
	if err := c.client.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping redpanda: %w", err)
	}
	return nil
}

// Start begins polling for messages
func (c *RedpandaConsumer) Start(ctx context.Context) error {
	for {
		fetches := c.client.PollFetches(ctx)
		if err := fetches.Err(); err != nil {
			if err == context.Canceled {
				return nil
			}
			return fmt.Errorf("poll error: %w", err)
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
				}

				// TODO: Implement proper error handling, retry policy, or DLQ
				if err := handler(ctx, event); err != nil {
					// For now, just log the error or ignore it to continue processing
					// In a real app, you might want to stop or nack
					fmt.Printf("Error handling event on topic %s: %v\n", record.Topic, err)
				}
			}
		}
	}
}

// Close closes the consumer connection
func (c *RedpandaConsumer) Close() {
	c.client.Close()
}

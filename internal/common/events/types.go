package events

import "context"

// Event represents a generic event in the system
type Event interface {
	Topic() string
	Key() string
	Payload() []byte
}

// BaseEvent is a simple implementation of Event
type BaseEvent struct {
	EventTopic   string
	EventKey     string
	EventPayload []byte
}

func (e BaseEvent) Topic() string {
	return e.EventTopic
}

func (e BaseEvent) Key() string {
	return e.EventKey
}

func (e BaseEvent) Payload() []byte {
	return e.EventPayload
}

// Producer defines the interface for publishing events
type Producer interface {
	Publish(ctx context.Context, event Event) error
	Close()
}

// EventHandler defines the function signature for handling events
type EventHandler func(ctx context.Context, event Event) error

// Consumer defines the interface for consuming events
type Consumer interface {
	Subscribe(topic string, handler EventHandler) error
	Start(ctx context.Context) error
	Close()
}

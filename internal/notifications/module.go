package notifications

import (
	"context"

	"github.com/alibaba0010/postgres-api/internal/common/events"
	"github.com/alibaba0010/postgres-api/internal/common/logger"
	authservices "github.com/alibaba0010/postgres-api/internal/auth/services"
	"go.uber.org/zap"
)

// RegisterSubscribers registers notification-related event handlers.
func RegisterSubscribers(consumer events.Consumer) {
	if consumer == nil {
		logger.Log.Warn("Notification Module: events consumer is nil, skipping subscriber registration")
		return
	}

	handlers := map[string]events.EventHandler{
		"order.created":        HandleOrderCreatedEvent,
		"order.status_updated": HandleOrderStatusUpdatedEvent,
		"user.registered":      HandleUserRegisteredEvent,
	}

	for topic, handler := range handlers {
		if err := consumer.Subscribe(topic, handler); err != nil {
			logger.Log.Error("Failed to subscribe to topic", zap.String("topic", topic), zap.Error(err))
		} else {
			logger.Log.Info("Notification Module subscribed to topic", zap.String("topic", topic))
		}
	}
}

// OrderEventPayload represents the expected payload for order-related events
type OrderEventPayload struct {
	OrderID   string `json:"order_id"`
	OrderType string `json:"order_type"`
	Status    string `json:"status"`
	// Keep flexible enough for variations
}

// HandleOrderCreatedEvent handles order.created events to send notifications
func HandleOrderCreatedEvent(ctx context.Context, event events.Event) error {
	logger.Log.Info("Received order.created event for notification", zap.String("key", event.Key()))
	
	// Ideally, fetch order details and user email here via service/repo,
	// For demonstration, logging the action and defining a dummy email sending process if payload contains it.

	// Placeholder for sending email to restaurant admins and users.
	subject := "New Order Created: " + event.Key()
	body := "<h1>New Order</h1><p>Order " + event.Key() + " has been created successfully.</p>"
	
	// Example to a defined mail (would normally fetch user email using order ID)
	// err := authservices.SendEmail("admin@example.com", subject, body)
	_ = subject
	_ = body
	_ = authservices.SendEmail

	return nil
}

// HandleOrderStatusUpdatedEvent handles order.status_updated events
func HandleOrderStatusUpdatedEvent(ctx context.Context, event events.Event) error {
	logger.Log.Info("Received order.status_updated event for notification", zap.String("key", event.Key()))
	
	// In a real implementation: Parse payload, extract new status, fetch user email, and send.
	return nil
}

// HandleUserRegisteredEvent handles user.registered events
func HandleUserRegisteredEvent(ctx context.Context, event events.Event) error {
	logger.Log.Info("Received user.registered event for notification", zap.String("key", event.Key()))
	return nil
}

package subscribers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alibaba0010/postgres-api/internal/common/events"
	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/alibaba0010/postgres-api/internal/orders/repositories"

	"go.uber.org/zap"
)

// OrderSubscriber listens for external events (like payments) and updates order states accordingly.
type OrderSubscriber struct {
	orderRepo *repositories.OrderRepository
}
func NewOrderSubscriber(orderRepo *repositories.OrderRepository) *OrderSubscriber {
	return &OrderSubscriber{
		orderRepo: orderRepo,
	}
}
// PaymentEventPayload represents the data received from payment-related events.
type PaymentEventPayload struct {
	PaymentID string `json:"payment_id"`
	OrderID   string `json:"order_id"`
	Status    string `json:"status"`
}

// RegisterOrderSubscribers attaches handlers to event bus topics.
func (os *OrderSubscriber)RegisterOrderSubscribers(consumer events.Consumer) {
	if consumer == nil {
		return
	}

	// Listens for successful payments to move orders to 'confirmed' status.
	if err := consumer.Subscribe("payment_successful", os.handlePaymentSuccessful); err != nil {
		logger.Log.Error("Failed to subscribe to payment_successful", zap.Error(err))
	}
	// Listens for failed payments to move orders to 'cancelled' status.
	if err := consumer.Subscribe("payment_failed", os.handlePaymentFailed); err != nil {
		logger.Log.Error("Failed to subscribe to payment_failed", zap.Error(err))
	}
}

func (os *OrderSubscriber)handlePaymentSuccessful(ctx context.Context, event events.Event) error {
	var payload PaymentEventPayload
	if err := json.Unmarshal(event.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payment event: %w", err)
	}

	logger.Log.Info("Processing payment_successful event", zap.String("order_id", payload.OrderID), zap.String("status", payload.Status))

	// Update Order Status to Confirmed
	if err := os.orderRepo.UpdateStatus(ctx, nil, payload.OrderID, types.OrderStatusConfirmed); err != nil {
		logger.Log.Error("Failed to update order status", zap.Error(err))
		return err
	}
	return nil
}

func (os *OrderSubscriber) handlePaymentFailed(ctx context.Context, event events.Event) error {
	var payload PaymentEventPayload
	if err := json.Unmarshal(event.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payment event: %w", err)
	}

	logger.Log.Info("Processing payment_failed event", zap.String("order_id", payload.OrderID))

	// Update Order Status to Cancelled? Or just log.
	// Let's cancel it for now.
	if err := os.orderRepo.UpdateStatus(ctx, nil, payload.OrderID, types.OrderStatusCancelled); err != nil {
		logger.Log.Error("Failed to update order status", zap.Error(err))
		return err
	}
	return nil
}

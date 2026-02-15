package subscribers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alibaba0010/postgres-api/internal/common/events"
	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/orders/models"
	"github.com/alibaba0010/postgres-api/internal/orders/repositories"
	"go.uber.org/zap"
)
type OrderSubscriber struct {
	orderRepo *repositories.OrderRepository
}
func NewOrderSubscriber(orderRepo *repositories.OrderRepository) *OrderSubscriber {
	return &OrderSubscriber{
		orderRepo: orderRepo,
	}
}
type PaymentEventPayload struct {
	PaymentID string `json:"payment_id"`
	OrderID   string `json:"order_id"`
	Status    string `json:"status"`
}

func (os *OrderSubscriber)RegisterOrderSubscribers(consumer events.Consumer) {
	if consumer == nil {
		return
	}

	if err := consumer.Subscribe("payment_successful", os.handlePaymentSuccessful); err != nil {
		logger.Log.Error("Failed to subscribe to payment_successful", zap.Error(err))
	}
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

	// Update Order Status to Processing
	if err := os.orderRepo.UpdateStatus(ctx, payload.OrderID, models.OrderStatusProcessing); err != nil {
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
	if err := os.orderRepo.UpdateStatus(ctx, payload.OrderID, models.OrderStatusCancelled); err != nil {
		logger.Log.Error("Failed to update order status", zap.Error(err))
		return err
	}
	return nil
}

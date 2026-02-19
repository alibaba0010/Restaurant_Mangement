package events

import (
	"encoding/json"
	"time"

	"github.com/alibaba0010/postgres-api/internal/common/events"
	"github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/alibaba0010/postgres-api/internal/payments/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	TopicPaymentEvents = "payment-events"
)

type PaymentEventPayload struct {
	PaymentID uuid.UUID              `json:"payment_id"`
	OrderID   uuid.UUID              `json:"order_id"`
	UserID    uuid.UUID              `json:"user_id"`
	Amount    decimal.Decimal        `json:"amount"`
	Status    types.PaymentStatus   `json:"status"`
	Reference string                 `json:"reference"`
	Provider  types.PaymentProvider `json:"provider"`
	Timestamp time.Time              `json:"timestamp"`
}

type PaymentBaseEvent struct {
	events.BaseEvent
}

func NewPaymentEvent(eventType string, payment *models.Payment) events.Event {
	payload := PaymentEventPayload{
		PaymentID: payment.ID,
		OrderID:   payment.OrderID,
		UserID:    payment.UserID,
		Amount:    payment.Amount,
		Status:    payment.Status,
		Reference: payment.Reference,
		Provider:  payment.Provider,
		Timestamp: time.Now(),
	}

	data, _ := json.Marshal(payload)

	return events.BaseEvent{
		EventTopic:   TopicPaymentEvents,
		EventKey:     payment.ID.String(), // Key by PaymentID or OrderID? PaymentID is safer for strict ordering of payment logs.
		EventPayload: data,
	}
}

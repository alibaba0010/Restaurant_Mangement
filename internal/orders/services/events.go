package services

import (
	"encoding/json"

	"github.com/alibaba0010/postgres-api/internal/common/events"
)

type OrderEvent struct {
	events.BaseEvent
	OrderID string `json:"order_id"`
}

func NewOrderEvent(orderID string, topic string) *OrderEvent {
	payload, _ := json.Marshal(map[string]string{"order_id": orderID})
	return &OrderEvent{
		BaseEvent: events.BaseEvent{
			EventTopic:   topic,
			EventKey:     orderID,
			EventPayload: payload,
		},
		OrderID: orderID,
	}
}

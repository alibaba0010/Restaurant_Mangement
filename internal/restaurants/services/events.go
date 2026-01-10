package services

import (
	"encoding/json"

	"github.com/alibaba0010/postgres-api/internal/common/events"
)

type MenuUpdatedEvent struct {
	events.BaseEvent
	MenuID string `json:"menu_id"`
}

func NewMenuUpdatedEvent(menuID string) *MenuUpdatedEvent {
	payload, _ := json.Marshal(map[string]string{"menu_id": menuID})
	return &MenuUpdatedEvent{
		BaseEvent: events.BaseEvent{
			EventTopic:   "menu.updated",
			EventKey:     menuID,
			EventPayload: payload,
		},
		MenuID: menuID,
	}
}

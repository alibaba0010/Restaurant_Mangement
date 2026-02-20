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
type CategoryEvent struct {
	events.BaseEvent
	CategoryID string `json:"category_id"`
}

func NewCategoryEvent(categoryID string, topic string) *CategoryEvent {
	payload, _ := json.Marshal(map[string]string{"category_id": categoryID})
	return &CategoryEvent{
		BaseEvent: events.BaseEvent{
			EventTopic:   topic,
			EventKey:     categoryID,
			EventPayload: payload,
		},
		CategoryID: categoryID,
	}
}

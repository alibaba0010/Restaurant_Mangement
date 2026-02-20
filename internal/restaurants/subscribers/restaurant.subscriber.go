package subscribers

import (
	"context"

	"github.com/alibaba0010/postgres-api/internal/common/events"
	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/utils"
	"go.uber.org/zap"
)

// RestaurantSubscriber listens for restaurant-related events and performs background tasks
// like cache invalidation.
type RestaurantSubscriber struct {
}

func NewRestaurantSubscriber() *RestaurantSubscriber {
	return &RestaurantSubscriber{}
}

// RegisterRestaurantSubscribers attaches handlers to event bus topics.
func (rs *RestaurantSubscriber) RegisterRestaurantSubscribers(consumer events.Consumer) {
	if consumer == nil {
		return
	}

	// Listens for menu changes to invalidate list cache.
	consumer.Subscribe("menu.created", rs.handleMenuChange)
	consumer.Subscribe("menu.updated", rs.handleMenuChange)
	consumer.Subscribe("menu.deleted", rs.handleMenuChange)
	
	// Listens for category changes as they affect menu structure.
	consumer.Subscribe("category.created", rs.handleMenuChange)
	consumer.Subscribe("category.updated", rs.handleMenuChange)
	consumer.Subscribe("category.deleted", rs.handleMenuChange)
}

func (rs *RestaurantSubscriber) handleMenuChange(ctx context.Context, event events.Event) error {
	logger.Log.Info("Processing menu/category change event for cache invalidation", zap.String("topic", event.Topic()))
	
	// Invalidate the menu list cache.
	utils.InvalidateCacheByPrefix(ctx, "menus:list:")
	
	return nil
}

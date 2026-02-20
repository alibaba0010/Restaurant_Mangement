package restaurants

import (
	"github.com/alibaba0010/postgres-api/internal/common/events"
	"github.com/alibaba0010/postgres-api/internal/restaurants/subscribers"
)

// RegisterSubscribers initializes and registers all restaurant-related event subscribers.
func RegisterSubscribers(consumer events.Consumer) {
	restaurantSubscriber := subscribers.NewRestaurantSubscriber()
	restaurantSubscriber.RegisterRestaurantSubscribers(consumer)
}

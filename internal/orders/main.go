package orders

import (
	"github.com/alibaba0010/postgres-api/internal/common/events"
	"github.com/alibaba0010/postgres-api/internal/database"
	"github.com/alibaba0010/postgres-api/internal/orders/repositories"
	"github.com/alibaba0010/postgres-api/internal/orders/subscribers"
)

// RegisterSubscribers initializes and registers all order-related event subscribers.
func RegisterSubscribers(consumer events.Consumer) {
	orderRepo := repositories.NewOrderRepository(database.DB)
	orderSubscriber := subscribers.NewOrderSubscriber(orderRepo)
	orderSubscriber.RegisterOrderSubscribers(consumer)
}



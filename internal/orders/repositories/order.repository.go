package repositories

import (
	"context"
	"database/sql"

	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/alibaba0010/postgres-api/internal/orders/models"

	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// OrderRepository manages database persistence for Order and OrderItem models.
type OrderRepository struct {
	db *bun.DB
}

// NewOrderRepository creates a new order repository instance
func NewOrderRepository(db *bun.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// Create inserts a new order and its items within a transaction
func (r *OrderRepository) Create(ctx context.Context, order *models.Order) error {
	err := r.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().Model(order).Exec(ctx)
		if err != nil {
			return err
		}

		for _, item := range order.OrderItems {
			item.OrderID = order.ID
		}

		_, err = tx.NewInsert().Model(&order.OrderItems).Exec(ctx)
		return err
	})

	if err != nil {
		logger.Log.Error("failed to create order", zap.Error(err))
		return err
	}
	return nil
}

// FindByID retrieves an order by ID with relations
func (r *OrderRepository) FindByID(ctx context.Context, id string) (*models.Order, error) {
	order := new(models.Order)
	err := r.db.NewSelect().
		Model(order).
		Relation("OrderItems").
		Relation("Restaurant").
		Relation("User").
		Where("o.id = ?", id).
		Scan(ctx)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		logger.Log.Error("failed to find order by id", zap.String("id", id), zap.Error(err))
		return nil, err
	}
	return order, nil
}

// FindByIDWithLock retrieves an order by ID with a row-level lock (FOR UPDATE).
func (r *OrderRepository) FindByIDWithLock(ctx context.Context, db bun.IDB, id string) (*models.Order, error) {
	if db == nil {
		db = r.db
	}
	order := new(models.Order)
	err := db.NewSelect().
		Model(order).
		Relation("OrderItems").
		Relation("Restaurant").
		Relation("User").
		Where("o.id = ?", id).
		For("UPDATE").
		Scan(ctx)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		logger.Log.Error("failed to find and lock order by id", zap.String("id", id), zap.Error(err))
		return nil, err
	}
	return order, nil
}

// FindByUserID retrieves orders for a specific user with pagination
func (r *OrderRepository) FindByUserID(ctx context.Context, userID string, limit int, cursor string) ([]models.Order, string, bool, error) {
	var orders []models.Order
	query := r.db.NewSelect().
		Model(&orders).
		Relation("OrderItems").
		Relation("Restaurant").
		Relation("User").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit + 1)

	if cursor != "" {
		query.Where("created_at < ?", cursor)
	}

	err := query.Scan(ctx)
	if err != nil {
		logger.Log.Error("failed to find orders by user id", zap.String("user_id", userID), zap.Error(err))
		return nil, "", false, err
	}

	hasMore := len(orders) > limit
	nextCursor := ""
	if hasMore {
		orders = orders[:limit]
		nextCursor = orders[limit-1].CreatedAt.Format("2006-01-02T15:04:05.999999Z07:00")
	}

	return orders, nextCursor, hasMore, nil
}

// FindByRestaurantID retrieves orders for a specific restaurant with pagination
func (r *OrderRepository) FindByRestaurantID(ctx context.Context, restaurantID string, limit int, cursor string) ([]models.Order, string, bool, error) {
	var orders []models.Order
	query := r.db.NewSelect().
		Model(&orders).
		Relation("OrderItems").
		Relation("User").
		Where("restaurant_id = ?", restaurantID).
		Order("created_at DESC").
		Limit(limit + 1)

	if cursor != "" {
		query.Where("created_at < ?", cursor)
	}

	err := query.Scan(ctx)
	if err != nil {
		logger.Log.Error("failed to find orders by restaurant id", zap.String("restaurant_id", restaurantID), zap.Error(err))
		return nil, "", false, err
	}

	hasMore := len(orders) > limit
	nextCursor := ""
	if hasMore {
		orders = orders[:limit]
		nextCursor = orders[limit-1].CreatedAt.Format("2006-01-02T15:04:05.999999Z07:00")
	}

	return orders, nextCursor, hasMore, nil
}

// UpdateStatus updates the status of an existing order and sets appropriate lifecycle timestamps.
// It accepts a bun.IDB which can be a transaction or the main DB connection.
func (r *OrderRepository) UpdateStatus(ctx context.Context, db bun.IDB, id string, status types.OrderStatus) error {
	if db == nil {
		db = r.db
	}

	q := db.NewUpdate().
		Model((*models.Order)(nil)).
		Set("status = ?", status).
		Set("updated_at = NOW()")

	switch status {
	case types.OrderStatusConfirmed:
		q = q.Set("confirmed_at = NOW()")
	case types.OrderStatusPreparing:
		q = q.Set("preparing_at = NOW()")
	case types.OrderStatusReady:
		q = q.Set("ready_at = NOW()")
	case types.OrderStatusCompleted:
		q = q.Set("completed_at = NOW()")
	case types.OrderStatusCancelled:
		q = q.Set("cancelled_at = NOW()")
	}

	_, err := q.Where("id = ?", id).Exec(ctx)
	if err != nil {
		logger.Log.Error("failed to update order status", zap.String("id", id), zap.Error(err))
		return err
	}
	return nil
}

// RunInTx executes the provided function within a database transaction.
func (r *OrderRepository) RunInTx(ctx context.Context, fn func(ctx context.Context, tx bun.Tx) error) error {
	return r.db.RunInTx(ctx, &sql.TxOptions{}, fn)
}

// GetDB returns the underlying bun.DB instance.
func (r *OrderRepository) GetDB() *bun.DB {
	return r.db
}

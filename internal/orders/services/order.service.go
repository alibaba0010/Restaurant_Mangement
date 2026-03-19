package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	commonDto "github.com/alibaba0010/postgres-api/internal/common/dto"
	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/alibaba0010/postgres-api/internal/orders/dto"
	"github.com/alibaba0010/postgres-api/internal/orders/models"
	"github.com/alibaba0010/postgres-api/internal/orders/repositories"
	restModels "github.com/alibaba0010/postgres-api/internal/restaurants/models"
	restRepo "github.com/alibaba0010/postgres-api/internal/restaurants/repositories"
	authRepo "github.com/alibaba0010/postgres-api/internal/auth/repositories"
	"github.com/uptrace/bun"

	"github.com/alibaba0010/postgres-api/internal/utils"

	"github.com/alibaba0010/postgres-api/internal/common/events"
	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"go.uber.org/zap"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func (s *OrderService) publishOrderEvent(ctx context.Context, orderID string, topic string) {
	producer := events.GetGlobalProducer()
	if producer == nil {
		return
	}

	event := NewOrderEvent(orderID, topic)

	if err := producer.Publish(ctx, event); err != nil {
		logger.Log.Error("Failed to publish order event", zap.String("topic", topic), zap.Error(err))
	}
}

// OrderService implements the business logic for customer orders,
// managing the lifecycle from creation to status updates.
type OrderService struct {
	orderRepo      *repositories.OrderRepository
	menuRepo       *restRepo.MenuRepository
	restaurantRepo *restRepo.RestaurantRepository
}

// NewOrderService creates a new instance of OrderService with the required repositories.
func NewOrderService(orderRepo *repositories.OrderRepository, menuRepo *restRepo.MenuRepository, restaurantRepo *restRepo.RestaurantRepository) *OrderService {
	return &OrderService{
		orderRepo:      orderRepo,
		menuRepo:       menuRepo,
		restaurantRepo: restaurantRepo,
	}
}

// OrderServiceInterface defines the business logic operations for orders.
type OrderServiceInterface interface {
	CreateOrder(ctx context.Context, userID string, input dto.CreateOrderInput) (*dto.OrderResponse, *errors.AppError)
	GetOrderByID(ctx context.Context, id string, user commonDto.AuthenticatedUser) (*dto.OrderResponse, *errors.AppError)
	GetUserOrders(ctx context.Context, userID string, limit int, cursor string) ([]dto.OrderResponse, string, bool, *errors.AppError)
	UpdateOrderStatus(ctx context.Context, id string, status string, user commonDto.AuthenticatedUser) *errors.AppError
}

var _ OrderServiceInterface = (*OrderService)(nil)
// CalculateServiceCharge computes the service charge based on the subtotal.
// Business Rule: 10% on orders less than 100, 5% on orders 100 or more.
func CalculateServiceCharge(subtotal decimal.Decimal) (decimal.Decimal, string) {
	threshold := decimal.NewFromInt(100)
	var rate decimal.Decimal
	var percentLabel string

	if subtotal.LessThan(threshold) {
		rate = decimal.NewFromFloat(0.10)
		percentLabel = "10%"
	} else {
		rate = decimal.NewFromFloat(0.05)
		percentLabel = "5%"
	}

	charge := subtotal.Mul(rate).Round(2)
	return charge, percentLabel
}

// MapOrderToResponse transforms a database Order model into a DTO response.
// This ensures internal database structures (like UUIDs or timestamps) are correctly 
// formatted for the API consumers.
func (s *OrderService) MapOrderToResponse(o *models.Order) dto.OrderResponse {
	items := make([]dto.OrderItemResponse, len(o.OrderItems))
	for i, item := range o.OrderItems {
		items[i] = dto.OrderItemResponse{
			ID:       item.ID.String(),
			MenuID:   item.MenuID.String(),
			Name:     item.Name,
			Quantity: item.Quantity,
			Price:    item.Price,
		}
	}

	// Determine service charge percent label for display
	_, percentLabel := CalculateServiceCharge(o.Subtotal)

	resp := dto.OrderResponse{
		ID:                   o.ID.String(),
		UserID:               o.UserID.String(),
		RestaurantID:         o.RestaurantID.String(),
		OrderType:            string(o.OrderType),
		Subtotal:             o.Subtotal,
		ServiceCharge:        o.ServiceCharge,
		ServiceChargePercent: percentLabel,
		TotalAmount:          o.TotalAmount,
		Currency:             o.Currency,
		Status:               string(o.Status),
		PaymentStatus:        string(o.PaymentStatus),
		DeliveryAddress:      o.DeliveryAddress,
		CreatedAt:            o.CreatedAt.Format(time.RFC3339),
		UpdatedAt:            o.UpdatedAt.Format(time.RFC3339),
		Items:                items,
	}

	if o.ConfirmedAt != nil {
		t := o.ConfirmedAt.Format(time.RFC3339)
		resp.ConfirmedAt = &t
	}
	if o.CompletedAt != nil {
		t := o.CompletedAt.Format(time.RFC3339)
		resp.CompletedAt = &t
	}

	return resp
}

// NormalizeStatus cleans up the status string by trimming whitespace and converting to lowercase.
func NormalizeStatus(status string) string {
	return strings.ToLower(strings.TrimSpace(status))
}

// CreateOrder handles the complex business logic for creating a new customer order.
// The workflow includes:
// 1. Input validation and ID parsing.
// 2. Ensuring the target restaurant exists and is active.
// 3. Batch-fetching all requested menu items to calculate prices (preventing N+1 queries).
// 4. Verifying item availability and restaurant ownership for each item.
// 5. Calculating the total amount and persisting within a transaction.
func (s *OrderService) CreateOrder(ctx context.Context, userID string, input dto.CreateOrderInput) (*dto.OrderResponse, *errors.AppError) {
	// Step 1: Validate input DTO.
	if err := utils.ValidateInput(input); err != nil {
		return nil, err
	}

	uID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.ValidationError("Invalid user ID")
	}

	rID, err := uuid.Parse(input.RestaurantID)
	if err != nil {
		return nil, errors.ValidationError("Invalid restaurant ID")
	}

	// Step 2: Verify restaurant existence.
	_, err = s.restaurantRepo.FindByID(ctx, input.RestaurantID)
	if err != nil {
		return nil, errors.NotFoundError("Restaurant not found")
	}

	// Step 3: Optimization - Batch fetch menu items.
	// Instead of querying the DB for each item in the order, we fetch them all at once.
	menuIDs := make([]string, len(input.Items))
	itemMap := make(map[string]dto.CreateOrderItemInput)
	seen := make(map[string]bool)

	for i, item := range input.Items {
		if seen[item.MenuID] {
			return nil, errors.ValidationError("Duplicate menu item: " + item.MenuID)
		}
		seen[item.MenuID] = true
		menuIDs[i] = item.MenuID
		itemMap[item.MenuID] = item
	}

	menuItems, err := s.menuRepo.FindByIDs(ctx, menuIDs)
	if err != nil {
		return nil, errors.InternalError(err)
	}

	if len(menuItems) != len(menuIDs) {
		return nil, errors.ValidationError("One or more menu items were not found")
	}
	// Step 4: Aggregate order details and validate items.
	orderID, err := utils.GenerateUUIDv7()
	if err != nil {
		return nil, errors.InternalError(err)
	}

	order := &models.Order{
		ID:              orderID, // Pre-generate ID to avoid null constraints and use in items
		UserID:          uID,
		RestaurantID:    rID,
		OrderType:       types.OrderType(input.OrderType), // Issue 1.1: Set order_type
		Status:          types.OrderStatusPending,
		Currency:        "NGN", // Correct for local payment context or derive from config
		DeliveryAddress: input.DeliveryAddress,
	}

	// Senior Dev Logic: If delivery address is not provided in the request, try to fetch the 
	// user's default address from the database to populate the "cart sheet" (order).
	if order.OrderType == types.OrderTypeDelivery && order.DeliveryAddress == "" {
		user, _ := authRepo.UserRepo.FindByIDWithAddresses(ctx, userID)
		if user != nil && len(user.Addresses) > 0 {
			// Find the default address
			for _, addr := range user.Addresses {
				if addr.IsDefault {
					order.DeliveryAddress = addr.RawAddress
					break
				}
			}
			// Fallback to first address if no default is explicitly marked
			if order.DeliveryAddress == "" {
				order.DeliveryAddress = user.Addresses[0].RawAddress
			}
		}
	}

	var totalAmount decimal.Decimal = decimal.Zero
	orderItems := make([]*models.OrderItem, len(menuItems))

	for i, menuItem := range menuItems {
		// Business Rule: Cannot order unavailable items.
		if !menuItem.IsAvailable {
			return nil, errors.ValidationError("Menu item not available: " + menuItem.Name)
		}

		// Security Check: Ensure the item actually belongs to the restaurant being ordered from.
		if menuItem.RestaurantID != rID {
			return nil, errors.ValidationError("Menu item " + menuItem.Name + " does not belong to the selected restaurant")
		}

		itemInput := itemMap[menuItem.ID.String()]
		orderItems[i] = &models.OrderItem{
			ID:       uuid.New(), // Pre-generate Item ID
			OrderID:  order.ID,    // Set correctly from generated Order ID
			MenuID:   menuItem.ID,
			Name:     menuItem.Name,
			Quantity: itemInput.Quantity,
			Price:    menuItem.Price,
		}
		// Accrue total cost using decimal for financial precision.
		itemTotal := menuItem.Price.Mul(decimal.NewFromInt(int64(itemInput.Quantity)))
		totalAmount = totalAmount.Add(itemTotal)
	}

	// Calculate service charge: 10% for subtotal < 100, 5% for subtotal >= 100
	serviceCharge, _ := CalculateServiceCharge(totalAmount)

	order.Subtotal = totalAmount
	order.ServiceCharge = serviceCharge
	order.TotalAmount = totalAmount.Add(serviceCharge)
	order.OrderItems = orderItems

	// Step 5: Persist order and update inventory within a database transaction.
	// This ensures atomicity: either the entire order is created and stock is deducted,
	// or nothing happens (preventing inconsistent states if one step fails).
	err = s.orderRepo.RunInTx(ctx, func(ctx context.Context, tx bun.Tx) error {
		// 5.1: Critical Section - Lock menu items to prevent race conditions.
		// We re-fetch items within the transaction using FOR UPDATE to ensure we have
		// the absolute latest stock counts and no other transaction can modify them until we finish.
		lockedItems, err := s.menuRepo.FindByIDsWithLock(ctx, tx, menuIDs)
		if err != nil {
			return err
		}

		// Create a map for quick lookup of locked items.
		lockedMap := make(map[uuid.UUID]restModels.Menu)
		for _, item := range lockedItems {
			lockedMap[item.ID] = item
		}

		// 5.2: Validate stock and calculate stock changes for batch update.
		stockChanges := make(map[uuid.UUID]int)
		for _, item := range orderItems {
			menuItem, ok := lockedMap[item.MenuID]
			if !ok {
				return fmt.Errorf("menu item %s not found during locking", item.MenuID)
			}

			// Business Rule: Ensure sufficient stock exists.
			if menuItem.StockQuantity < item.Quantity {
				if menuItem.StockQuantity == 0 {
					return fmt.Errorf("outofstock: %s is currently out of stock", menuItem.Name)
				}
				return fmt.Errorf("outofstock: %s has insufficient stock (only %d left)", menuItem.Name, menuItem.StockQuantity)
			}

			// Add to batch changes
			stockChanges[item.MenuID] = -item.Quantity
		}

		// 5.3: Update all stock in batch (Issue 6.2)
		if err := s.menuRepo.BatchUpdateStock(ctx, tx, stockChanges); err != nil {
			return err
		}

		// 5.4: Insert the Order record.
		_, err = tx.NewInsert().Model(order).Exec(ctx)
		if err != nil {
			return err
		}

		// 5.5: Insert the OrderItem records.
		_, err = tx.NewInsert().Model(&orderItems).Exec(ctx)
		return err
	})

	if err != nil {
		logger.Log.Error("Order creation failed during transaction", zap.Error(err))
		// Surface out-of-stock errors as a client-facing 400 ValidationError
		if strings.HasPrefix(err.Error(), "outofstock:") {
			msg := strings.TrimPrefix(err.Error(), "outofstock: ")
			return nil, errors.ValidationError(msg)
		}
		return nil, errors.TransactionError("creating order", err)
	}

	// Publish Event with error logging (Issue 5.13)
	producer := events.GetGlobalProducer()
	if producer != nil {
		event := NewOrderEvent(order.ID.String(), "order.created")
		if pubErr := producer.Publish(ctx, event); pubErr != nil {
			logger.Log.Error("Failed to publish order.created event", 
				zap.String("order_id", order.ID.String()), 
				zap.Error(pubErr))
		}
	}

	resp := s.MapOrderToResponse(order)
	return &resp, nil
}

// GetOrderByID retrieves an order and checks if the requesting user has permission to view it.
// This ensures data privacy between different customers.
func (s *OrderService) GetOrderByID(ctx context.Context, id string, user commonDto.AuthenticatedUser) (*dto.OrderResponse, *errors.AppError) {
	order, err := s.orderRepo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.NotFoundError("Order not found")
	}

	// Authorization check: Only Admin or the order owner can view the order.
	if user.Role != types.RoleAdmin && order.UserID.String() != user.UserID {
		return nil, errors.ForbiddenError("You are not authorized to view this order")
	}

	resp := s.MapOrderToResponse(order)
	return &resp, nil
}

// GetUserOrders retrieves orders for a specific user with support for cursor-based pagination.
// This is used for the "My Orders" history page.
func (s *OrderService) GetUserOrders(ctx context.Context, userID string, limit int, cursor string) ([]dto.OrderResponse, string, bool, *errors.AppError) {
	orders, nextCursor, hasMore, err := s.orderRepo.FindByUserID(ctx, userID, limit, cursor)
	if err != nil {
		return nil, "", false, errors.InternalError(err)
	}

	responses := make([]dto.OrderResponse, len(orders))
	for i, o := range orders {
		responses[i] = s.MapOrderToResponse(&o)
	}

	return responses, nextCursor, hasMore, nil
}

// Order status state machine validation (Issue 1.7)
var validTransitions = map[types.OrderStatus][]types.OrderStatus{
	types.OrderStatusPending:   {types.OrderStatusConfirmed, types.OrderStatusCancelled},
	types.OrderStatusConfirmed: {types.OrderStatusPreparing, types.OrderStatusCancelled},
	types.OrderStatusPreparing: {types.OrderStatusReady},
	types.OrderStatusReady:     {types.OrderStatusCompleted},
}

// UpdateOrderStatus changes the status of an existing order (e.g., Pending -> Preparing).
// It supports updates from both management (restaurant owners) and the orderer (cancellation/payment).
func (s *OrderService) UpdateOrderStatus(ctx context.Context, id string, status string, user commonDto.AuthenticatedUser) *errors.AppError {
	newStatus := types.OrderStatus(status)

	// Persist status change within a transaction for data integrity and lifecycle consistency.
	err := s.orderRepo.RunInTx(ctx, func(ctx context.Context, tx bun.Tx) error {
		// Step 1: Fetch current order state within the transaction and lock the row to prevent race conditions.
		order, err := s.orderRepo.FindByIDWithLock(ctx, tx, id)
		if err != nil {
			return err
		}

		isAdmin := user.Role == types.RoleAdmin

		// Step 2: Validate state transition (Issue 1.7)
		allowed, ok := validTransitions[order.Status]
		isValid := false
		if ok {
			for _, status := range allowed {
				if status == newStatus {
					isValid = true
					break
				}
			}
		}

		// Admins can bypass state machine, but others cannot.
		if !isAdmin && !isValid {
			return fmt.Errorf("validation: invalid status transition from %s to %s", order.Status, newStatus)
		}

		// Step 3: Authorization & Business Rules.
		isOwner := order.UserID.String() == user.UserID

		if user.Role == types.RoleManagement {
			// Management: Must own the restaurant associated with the order.
			restaurant, err := s.restaurantRepo.FindByID(ctx, order.RestaurantID.String())
			if err != nil || restaurant.UserID == nil || restaurant.UserID.String() != user.UserID {
				return fmt.Errorf("forbidden: not authorized to manage orders for this restaurant")
			}
		} else if isOwner {
			// Orderers can only cancel orders.
			if newStatus != types.OrderStatusCancelled {
				return fmt.Errorf("forbidden: orderers can only cancel orders")
			}
		} else if !isAdmin {
			return fmt.Errorf("forbidden: not authorized to update order status")
		}

		// Step 4: Persist status change.
		if err := s.orderRepo.UpdateStatus(ctx, tx, id, newStatus); err != nil {
			return err
		}

		// Step 5: Restore stock if cancelled (Senior Dev Fix)
		if newStatus == types.OrderStatusCancelled {
			stockChanges := make(map[uuid.UUID]int)
			for _, item := range order.OrderItems {
				stockChanges[item.MenuID] = item.Quantity // Positive value restores stock
			}
			if err := s.menuRepo.BatchUpdateStock(ctx, tx, stockChanges); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		// Proper typed error handling without string matching (Issue 3.1)
		// For now, still using some string check but it's more structured.
		if strings.HasPrefix(err.Error(), "forbidden:") {
			return errors.ForbiddenError(strings.TrimPrefix(err.Error(), "forbidden: "))
		}
		if strings.HasPrefix(err.Error(), "validation:") {
			return errors.ValidationError(strings.TrimPrefix(err.Error(), "validation: "))
		}
		logger.Log.Error("Order status update failed", zap.String("order_id", id), zap.Error(err))
		return errors.TransactionError("updating order status", err)
	}

	// Publish Event
	s.publishOrderEvent(ctx, id, "order.status_updated")

	return nil
}

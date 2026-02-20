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
	"github.com/uptrace/bun"

	"github.com/alibaba0010/postgres-api/internal/utils"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

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

	return dto.OrderResponse{
		ID:              o.ID.String(),
		UserID:          o.UserID.String(),
		RestaurantID:    o.RestaurantID.String(),
		TotalAmount:     o.TotalAmount,
		Status:          string(o.Status),
		DeliveryAddress: o.DeliveryAddress,
		CreatedAt:       o.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       o.UpdatedAt.Format(time.RFC3339),
		Items:           items,
	}
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

	uID, _ := uuid.Parse(userID)
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
	for i, item := range input.Items {
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
	order := &models.Order{
		UserID:          uID,
		RestaurantID:    rID,
		Status:          types.OrderStatusPending,
		DeliveryAddress: input.DeliveryAddress,
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
			MenuID:   menuItem.ID,
			Name:     menuItem.Name,
			Quantity: itemInput.Quantity,
			Price:    menuItem.Price,
		}
		// Accrue total cost using decimal for financial precision.
		itemTotal := menuItem.Price.Mul(decimal.NewFromInt(int64(itemInput.Quantity)))
		totalAmount = totalAmount.Add(itemTotal)
	}

	order.TotalAmount = totalAmount
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

		// 5.2: Validate stock and calculate final details.
		for _, item := range orderItems {
			menuItem, ok := lockedMap[item.MenuID]
			if !ok {
				return fmt.Errorf("menu item %s not found during locking", item.MenuID)
			}

			// Business Rule: Ensure sufficient stock exists.
			if menuItem.StockQuantity < item.Quantity {
				return fmt.Errorf("insufficient stock for %s (available: %d, requested: %d)", menuItem.Name, menuItem.StockQuantity, item.Quantity)
			}

			// 5.3: Deduct stock atomically.
			if err := s.menuRepo.UpdateStock(ctx, tx, item.MenuID, -item.Quantity); err != nil {
				return err
			}
		}

		// 5.4: Insert the Order record.
		_, err = tx.NewInsert().Model(order).Exec(ctx)
		if err != nil {
			return err
		}

		// 5.5: Insert the OrderItem records.
		for _, item := range orderItems {
			item.OrderID = order.ID
		}
		_, err = tx.NewInsert().Model(&orderItems).Exec(ctx)
		return err
	})

	if err != nil {
		// Differentiate between business logic errors (like stock) and system errors.
		return nil, errors.InternalError(err)
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

// UpdateOrderStatus changes the status of an existing order (e.g., Pending -> Preparing).
// It supports updates from both management (restaurant owners) and the orderer (cancellation/payment).
func (s *OrderService) UpdateOrderStatus(ctx context.Context, id string, status string, user commonDto.AuthenticatedUser) *errors.AppError {
	newStatus := types.OrderStatus(status)

	// Persist status change within a transaction for data integrity and lifecycle consistency.
	err := s.orderRepo.RunInTx(ctx, func(ctx context.Context, tx bun.Tx) error {
		// Step 1: Fetch current order state within the transaction.
		order, err := s.orderRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}

		// Step 2: Authorization & Business Rules.
		isOwner := order.UserID.String() == user.UserID
		isAdmin := user.Role == types.RoleAdmin

		if user.Role == types.RoleManagement {
			// Management: Must own the restaurant associated with the order.
			restaurant, err := s.restaurantRepo.FindByID(ctx, order.RestaurantID.String())
			if err != nil || restaurant.UserID == nil || restaurant.UserID.String() != user.UserID {
				return fmt.Errorf("forbidden: not authorized to manage orders for this restaurant")
			}
		} else if isOwner {
			// Orderer: Can only perform specific transitions (e.g., Cancel if still Pending).
			if newStatus == types.OrderStatusCancelled && order.Status != types.OrderStatusPending {
				return fmt.Errorf("validation: cannot cancel an order that is already %s", order.Status)
			}
			// Add other valid transitions for orderers here (e.g., marking as payment confirmed if applicable).
			if newStatus != types.OrderStatusCancelled && newStatus != types.OrderStatusConfirmed {
				return fmt.Errorf("forbidden: orderers can only cancel or confirm orders")
			}
		} else if !isAdmin {
			return fmt.Errorf("forbidden: not authorized to update order status")
		}

		// Step 3: Persist status change.
		return s.orderRepo.UpdateStatus(ctx, tx, id, newStatus)
	})

	if err != nil {
		// Log and return appropriately.
		if strings.Contains(err.Error(), "forbidden") {
			return errors.ForbiddenError(err.Error())
		}
		if strings.Contains(err.Error(), "validation") {
			return errors.ValidationError(err.Error())
		}
		return errors.InternalError(err)
	}
	return nil
}

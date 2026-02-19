package services

import (
	"context"
	"time"

	commonDto "github.com/alibaba0010/postgres-api/internal/common/dto"
	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/alibaba0010/postgres-api/internal/orders/dto"
	"github.com/alibaba0010/postgres-api/internal/orders/models"
	"github.com/alibaba0010/postgres-api/internal/orders/repositories"
	restRepo "github.com/alibaba0010/postgres-api/internal/restaurants/repositories"

	"github.com/alibaba0010/postgres-api/internal/utils"
	
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

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

// OrderServiceInterface defines the business logic for managing orders.
type OrderServiceInterface interface {
	CreateOrder(ctx context.Context, userID string, input dto.CreateOrderInput) (*dto.OrderResponse, *errors.AppError)
	GetOrderByID(ctx context.Context, id string, user commonDto.AuthenticatedUser) (*dto.OrderResponse, *errors.AppError)
	GetUserOrders(ctx context.Context, userID string, limit int, cursor string) ([]dto.OrderResponse, string, bool, *errors.AppError)
	UpdateOrderStatus(ctx context.Context, id string, status string, user commonDto.AuthenticatedUser) *errors.AppError
}
// MapOrderToResponse transforms a database Order model into a DTO response.
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

// CreateOrder handles the business logic for creating a new order.
func (s *OrderService) CreateOrder(ctx context.Context, userID string, input dto.CreateOrderInput) (*dto.OrderResponse, *errors.AppError) {
	if err := utils.ValidateInput(input); err != nil {
		return nil, err
	}

	uID, _ := uuid.Parse(userID)
	rID, err := uuid.Parse(input.RestaurantID)
	if err != nil {
		return nil, errors.ValidationError("Invalid restaurant ID")
	}

	// Verify restaurant existence
	_, err = s.restaurantRepo.FindByID(ctx, input.RestaurantID)
	if err != nil {
		return nil, errors.NotFoundError("Restaurant not found")
	}

	// Batch fetch menu items for optimization (Avoid N+1)
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

	order := &models.Order{
		UserID:          uID,
		RestaurantID:    rID,
		Status:          types.OrderStatusPending,
		DeliveryAddress: input.DeliveryAddress,
	}

	var totalAmount decimal.Decimal = decimal.Zero
	orderItems := make([]*models.OrderItem, len(menuItems))

	for i, menuItem := range menuItems {
		if !menuItem.IsAvailable {
			return nil, errors.ValidationError("Menu item not available: " + menuItem.Name)
		}

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
		itemTotal := menuItem.Price.Mul(decimal.NewFromInt(int64(itemInput.Quantity)))
		totalAmount = totalAmount.Add(itemTotal)
	}

	order.TotalAmount = totalAmount
	order.OrderItems = orderItems

	err = s.orderRepo.Create(ctx, order)
	if err != nil {
		return nil, errors.InternalError(err)
	}

	resp := s.MapOrderToResponse(order)
	return &resp, nil
}

// GetOrderByID retrieves an order and checks if the requesting user has permission to view it.
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

// UpdateOrderStatus changes the status of an existing order after verifying authorization.
func (s *OrderService) UpdateOrderStatus(ctx context.Context, id string, status string, user commonDto.AuthenticatedUser) *errors.AppError {
	order, err := s.orderRepo.FindByID(ctx, id)
	if err != nil {
		return errors.NotFoundError("Order not found")
	}

	// If the user has a management role, check if they own the restaurant associated with the order.
	if user.Role == types.RoleManagement {
		restaurant, err := s.restaurantRepo.FindByID(ctx, order.RestaurantID.String())
		if err != nil || restaurant.UserID == nil || restaurant.UserID.String() != user.UserID {
			return errors.ForbiddenError("You are not authorized to manage orders for this restaurant")
		}
	} else if user.Role != types.RoleAdmin {
		// Non-admin/non-manager users cannot change order status.
		return errors.ForbiddenError("You are not authorized to update order status")
	}

	err = s.orderRepo.UpdateStatus(ctx, id, types.OrderStatus(status))
	if err != nil {
		return errors.InternalError(err)
	}
	return nil
}

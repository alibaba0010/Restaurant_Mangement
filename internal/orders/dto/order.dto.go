package dto

import "github.com/shopspring/decimal"

type CreateOrderItemInput struct {
	MenuID   string `json:"menu_id" validate:"required,uuid"`
	Quantity int    `json:"quantity" validate:"required,min=1"`
}

type CreateOrderInput struct {
	RestaurantID    string                 `json:"restaurant_id" validate:"required,uuid"`
	OrderType       string                 `json:"order_type" validate:"required,oneof=delivery pickup dine_in"`
	DeliveryAddress string                 `json:"delivery_address" validate:"required_if=OrderType delivery"`
	Items           []CreateOrderItemInput `json:"items" validate:"required,min=1,dive"`
}

type OrderItemResponse struct {
	ID        string          `json:"id"`
	MenuID    string          `json:"menu_id"`
	Name      string          `json:"name"`
	Quantity  int             `json:"quantity"`
	Price     decimal.Decimal `json:"price"`
}

type OrderResponse struct {
	ID                  string              `json:"id"`
	UserID              string              `json:"user_id"`
	RestaurantID        string              `json:"restaurant_id"`
	OrderType           string              `json:"order_type"`
	Subtotal            decimal.Decimal     `json:"subtotal"`
	ServiceCharge       decimal.Decimal     `json:"service_charge"`
	ServiceChargePercent string             `json:"service_charge_percent"`
	TotalAmount         decimal.Decimal     `json:"total_amount"`
	Currency            string              `json:"currency"`
	Status              string              `json:"status"`
	PaymentStatus       string              `json:"payment_status"`
	DeliveryAddress     string              `json:"delivery_address"`
	CreatedAt           string              `json:"created_at"`
	UpdatedAt           string              `json:"updated_at"`
	ConfirmedAt         *string             `json:"confirmed_at,omitempty"`
	CompletedAt         *string             `json:"completed_at,omitempty"`
	Items               []OrderItemResponse `json:"items,omitempty"`
}

type UpdateOrderStatusInput struct {
	Status string `json:"status" validate:"required,oneof=confirmed preparing ready completed cancelled"`
}

type UserOrdersResponse struct {
	Orders     []OrderResponse `json:"orders"`
	NextCursor string          `json:"next_cursor"`
	HasMore    bool            `json:"has_more"`
}

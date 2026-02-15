package dto

type CreateOrderItemInput struct {
	MenuID   string `json:"menu_id" validate:"required,uuid"`
	Quantity int    `json:"quantity" validate:"required,min=1"`
}

type CreateOrderInput struct {
	RestaurantID    string                 `json:"restaurant_id" validate:"required,uuid"`
	DeliveryAddress string                 `json:"delivery_address" validate:"required"`
	Items           []CreateOrderItemInput `json:"items" validate:"required,min=1,dive"`
}

type OrderItemResponse struct {
	ID        string  `json:"id"`
	MenuID    string  `json:"menu_id"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

type OrderResponse struct {
	ID              string              `json:"id"`
	UserID          string              `json:"user_id"`
	RestaurantID    string              `json:"restaurant_id"`
	TotalAmount     float64             `json:"total_amount"`
	Status          string              `json:"status"`
	DeliveryAddress string              `json:"delivery_address"`
	CreatedAt       string              `json:"created_at"`
	UpdatedAt       string              `json:"updated_at"`
	Items           []OrderItemResponse `json:"items,omitempty"`
}

type UpdateOrderStatusInput struct {
	Status string `json:"status" validate:"required,oneof=pending processing completed cancelled"`
}

type UserOrdersResponse struct {
	Orders     []OrderResponse `json:"orders"`
	NextCursor string          `json:"next_cursor"`
	HasMore    bool            `json:"has_more"`
}

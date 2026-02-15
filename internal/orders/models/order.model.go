package models

import (
	"time"

	userModel "github.com/alibaba0010/postgres-api/internal/auth/models"
	menuModel "github.com/alibaba0010/postgres-api/internal/restaurants/models"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusProcessing OrderStatus = "processing"
	OrderStatusCompleted  OrderStatus = "completed"
	OrderStatusCancelled  OrderStatus = "cancelled"
)

type Order struct {
	bun.BaseModel `bun:"table:orders"`

	ID              uuid.UUID    `bun:"type:uuid,pk,default:gen_random_uuid()" json:"id"`
	UserID          uuid.UUID    `bun:"type:uuid,notnull" json:"user_id"`
	RestaurantID    uuid.UUID    `bun:"type:uuid,notnull" json:"restaurant_id"`
	TotalAmount     float64      `bun:",notnull" json:"total_amount"`
	Status          OrderStatus  `bun:",notnull,default:'pending'" json:"status"`
	DeliveryAddress string       `bun:",nullzero" json:"delivery_address,omitempty"`
	CreatedAt       time.Time    `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt       time.Time    `bun:",nullzero,notnull,default:current_timestamp" json:"updated_at"`

	// Relations
	OrderItems []*OrderItem         `bun:"rel:has-many,join:id=order_id" json:"items,omitempty"`
	Restaurant *menuModel.Restaurant `bun:"rel:belongs-to,join:restaurant_id=id" json:"restaurant,omitempty"`
	User       *userModel.User      `bun:"rel:belongs-to,join:user_id=id" json:"user,omitempty"`
}

type OrderItem struct {
	bun.BaseModel `bun:"table:order_items"`

	ID        uuid.UUID `bun:"type:uuid,pk,default:gen_random_uuid()" json:"id"`
	OrderID   uuid.UUID `bun:"type:uuid,notnull" json:"order_id"`
	MenuID    uuid.UUID `bun:"type:uuid,nullzero" json:"menu_id,omitempty"`
	Name      string    `bun:",notnull" json:"name"`
	Quantity  int       `bun:",notnull" json:"quantity"`
	Price     float64   `bun:",notnull" json:"price"`
	CreatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"updated_at"`

	// Relations
	Order *Order           `bun:"rel:belongs-to,join:order_id=id" json:"-"`
	Menu  *menuModel.Menu `bun:"rel:belongs-to,join:menu_id=id" json:"menu,omitempty"`
}

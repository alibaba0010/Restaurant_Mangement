package models

import (
	"time"

	userModel "github.com/alibaba0010/postgres-api/internal/auth/models"
	"github.com/alibaba0010/postgres-api/internal/common/types"
	menuModel "github.com/alibaba0010/postgres-api/internal/restaurants/models"
	
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

type Order struct {
	bun.BaseModel `bun:"table:orders"`

	ID              uuid.UUID    `bun:"type:uuid,pk,default:gen_random_uuid()" json:"id"`
	UserID          uuid.UUID    `bun:"type:uuid,notnull" json:"user_id"`
	RestaurantID    uuid.UUID    `bun:"type:uuid,notnull" json:"restaurant_id"`
	OrderType       types.OrderType `bun:",notnull,default:'delivery'" json:"order_type"`
	TotalAmount     decimal.Decimal `bun:"type:decimal(12,2),notnull" json:"total_amount"`
	Currency        string       `bun:",notnull,default:'USD'" json:"currency"`
	Status          types.OrderStatus `bun:",notnull,default:'pending'" json:"status"`
	
	// Payment Information
	PaymentStatus    types.PaymentStatus `bun:",notnull,default:'pending'" json:"payment_status"`

	PaymentMethod    string        `bun:",nullzero" json:"payment_method,omitempty"`
	PaymentReference string        `bun:",nullzero" json:"payment_reference,omitempty"`
	PaidAt           *time.Time    `bun:",nullzero" json:"paid_at,omitempty"`

	DeliveryAddress string       `bun:",nullzero" json:"delivery_address,omitempty"`
	
	// Lifecycle Timestamps
	ConfirmedAt  *time.Time `bun:",nullzero" json:"confirmed_at,omitempty"`
	PreparingAt  *time.Time `bun:",nullzero" json:"preparing_at,omitempty"`
	ReadyAt      *time.Time `bun:",nullzero" json:"ready_at,omitempty"`
	CompletedAt  *time.Time `bun:",nullzero" json:"completed_at,omitempty"`
	CancelledAt  *time.Time `bun:",nullzero" json:"cancelled_at,omitempty"`

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
	MenuID    uuid.UUID `bun:"type:uuid,notnull" json:"menu_id"` // P0: Make non-nullable
	Name      string    `bun:",notnull" json:"name"`
	Quantity  int       `bun:",notnull" json:"quantity"`
	Price     decimal.Decimal `bun:"type:decimal(10,2),notnull" json:"price"` // P0: Decimal for money
	CreatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"updated_at"`

	// Relations
	Order *Order           `bun:"rel:belongs-to,join:order_id=id" json:"-"`
	Menu  *menuModel.Menu `bun:"rel:belongs-to,join:menu_id=id" json:"menu,omitempty"`
}

package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

type SettlementStatus string

const (
	SettlementStatusPending   SettlementStatus = "pending"
	SettlementStatusProcessed SettlementStatus = "processed"
	SettlementStatusFailed    SettlementStatus = "failed"
)

type Settlement struct {
	bun.BaseModel `bun:"table:settlements"`

	ID              uuid.UUID       `bun:"type:uuid,pk,default:gen_random_uuid()" json:"id"`
	OrderID         uuid.UUID       `bun:"type:uuid,notnull" json:"order_id"`
	RestaurantID    uuid.UUID       `bun:"type:uuid,notnull" json:"restaurant_id"`
	TotalAmount     decimal.Decimal `bun:"type:decimal(12,2),notnull" json:"total_amount"`
	PlatformFee     decimal.Decimal `bun:"type:decimal(12,2),notnull" json:"platform_fee"`
	RestaurantShare decimal.Decimal `bun:"type:decimal(12,2),notnull" json:"restaurant_share"`
	
	Status          SettlementStatus `bun:",notnull,default:'pending'" json:"status"`
	PayoutReference string           `bun:",nullzero" json:"payout_reference,omitempty"`
	
	ProcessedAt     *time.Time `bun:",nullzero" json:"processed_at,omitempty"`
	CreatedAt       time.Time  `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt       time.Time  `bun:",nullzero,notnull,default:current_timestamp" json:"updated_at"`
}

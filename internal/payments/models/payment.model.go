package models

import (
	"time"

	authModels "github.com/alibaba0010/postgres-api/internal/auth/models"
	"github.com/alibaba0010/postgres-api/internal/common/types"
	orderModels "github.com/alibaba0010/postgres-api/internal/orders/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

// Payment represents a payment transaction
type Payment struct {
	bun.BaseModel `bun:"table:payments"`

	ID                uuid.UUID       `bun:"type:uuid,pk,default:gen_random_uuid()" json:"id"`
	OrderID           uuid.UUID       `bun:"type:uuid,notnull" json:"order_id"`
	UserID            uuid.UUID       `bun:"type:uuid,notnull" json:"user_id"`
	Amount            decimal.Decimal `bun:"type:decimal(12,2),notnull" json:"amount"`
	Currency          string          `bun:",notnull,default:'NGN'" json:"currency"`
	Provider          types.PaymentProvider `bun:",notnull" json:"provider"`
	Status            types.PaymentStatus   `bun:",notnull,default:'pending'" json:"status"`
	Reference         string          `bun:",unique,notnull" json:"reference"`          // Our internal reference

	ExternalReference string          `bun:",nullzero" json:"external_reference,omitempty"` // Provider's transaction ID
	
	// Payment method details
	PaymentMethod     string          `bun:",nullzero" json:"payment_method,omitempty"` // card, bank_transfer, mobile_money
	CustomerEmail     string          `bun:",nullzero" json:"customer_email,omitempty"`
	CustomerPhone     string          `bun:",nullzero" json:"customer_phone,omitempty"`
	CustomerName      string          `bun:",nullzero" json:"customer_name,omitempty"`
	
	// Refund tracking
	RefundAmount      *decimal.Decimal `bun:"type:decimal(12,2),nullzero" json:"refund_amount,omitempty"`
	RefundReason      string          `bun:",nullzero" json:"refund_reason,omitempty"`

	Metadata          map[string]any  `bun:"type:jsonb,nullzero" json:"metadata,omitempty"`
	
	// Timestamps
	CreatedAt         time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt         time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"updated_at"`
	CompletedAt       *time.Time      `bun:",nullzero" json:"completed_at,omitempty"`
	FailedAt          *time.Time      `bun:",nullzero" json:"failed_at,omitempty"`

	// Relations
	Order *orderModels.Order `bun:"rel:belongs-to,join:order_id=id" json:"order,omitempty"`
	User  *authModels.User   `bun:"rel:belongs-to,join:user_id=id" json:"user,omitempty"`
}

// PaymentRefund represents a refund transaction
type PaymentRefund struct {
	bun.BaseModel `bun:"table:payment_refunds"`

	ID                  uuid.UUID       `bun:"type:uuid,pk,default:gen_random_uuid()" json:"id"`
	PaymentID           uuid.UUID       `bun:"type:uuid,notnull" json:"payment_id"`
	ProviderRefundID    string          `bun:",nullzero" json:"provider_refund_id"` // Provider's refund ID
	Amount              decimal.Decimal `bun:"type:decimal(12,2),notnull" json:"amount"`
	Reason              string          `bun:",notnull" json:"reason"`
	Status              types.PaymentStatus   `bun:",notnull,default:'pending'" json:"status"`
	Metadata            map[string]interface{} `bun:"metadata,type:jsonb" json:"metadata,omitempty"`
	
	CreatedAt           time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt           time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"updated_at"`
	CompletedAt         *time.Time      `bun:",nullzero" json:"completed_at,omitempty"`
	
	// Relations
	Payment             *Payment        `bun:"rel:belongs-to,join:payment_id=id" json:"payment,omitempty"`
}

// PaymentWebhookLog stores webhook events from payment providers
type PaymentWebhookLog struct {
	bun.BaseModel `bun:"table:payment_webhook_logs"`

	ID                  uuid.UUID             `bun:"type:uuid,pk,default:gen_random_uuid()" json:"id"`
	Provider            types.PaymentProvider `bun:",notnull" json:"provider"`
	PaymentID           *uuid.UUID      `bun:"type:uuid,nullzero" json:"payment_id,omitempty"`
	EventType           string          `bun:",notnull" json:"event_type"` // charge.success, charge.failed, etc.
	ProviderEventID     string          `bun:",nullzero,unique" json:"provider_event_id"`
	Payload             map[string]interface{} `bun:"payload,type:jsonb" json:"payload"`
	Processed           bool            `bun:",default:false" json:"processed"`
	ProcessedAt         *time.Time      `bun:",nullzero" json:"processed_at,omitempty"`
	ErrorMessage        string          `bun:",nullzero" json:"error_message,omitempty"`
	
	CreatedAt           time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
}

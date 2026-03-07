package dto

import (
	"time"

	"github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/alibaba0010/postgres-api/internal/payments/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// InitiatePaymentRequest is the request to initialize a payments for an order
type InitiatePaymentRequest struct {
	OrderID     uuid.UUID              `json:"order_id" validate:"required"`
	Provider    types.PaymentProvider `json:"provider" validate:"required,oneof=monnify paystack flutterwave"`
	CallbackURL string                 `json:"callback_url" validate:"required,url"`
}

// InitiatePaymentResponse is the response after initiating a payments
type InitiatePaymentResponse struct {
	PaymentID        uuid.UUID `json:"payment_id"`
	AuthorizationURL string    `json:"authorization_url"`
	Reference        string    `json:"reference"`
	AccessCode       string    `json:"access_code,omitempty"`
}

// InitializePaymentRequest is a generic request to initialize a payments (without direct order link in body)
type InitializePaymentRequest struct {
	Amount        decimal.Decimal        `json:"amount" validate:"required"`
	Currency      string                 `json:"currency" validate:"omitempty,len=3"`
	Description   string                 `json:"description" validate:"required,min=3,max=255"`
	CustomerEmail string                 `json:"customer_email" validate:"required,email"`
	CustomerPhone string                 `json:"customer_phone" validate:"required"`
	CustomerName  string                 `json:"customer_name" validate:"required"`
	Provider      types.PaymentProvider `json:"provider" validate:"required"`
	CallbackURL   string                 `json:"callback_url" validate:"omitempty,url"`
	Metadata      map[string]interface{} `json:"metadata" validate:"omitempty"`
}

// PaymentResponse represents a payments in responses
type PaymentResponse struct {
	ID                uuid.UUID              `json:"id"`
	OrderID           uuid.UUID              `json:"order_id"`
	Amount            decimal.Decimal        `json:"amount"`
	Currency          string                 `json:"currency"`
	Provider          types.PaymentProvider `json:"provider"`
	Status            types.PaymentStatus   `json:"status"`

	Reference         string                 `json:"reference"`
	ExternalReference string                 `json:"external_reference,omitempty"`
	CustomerEmail     string                 `json:"customer_email,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	CompletedAt       *time.Time             `json:"completed_at,omitempty"`
}

// WebhookPayload is the generic webhook payload
type WebhookPayload struct {
	Event string                 `json:"event"`
	Data  map[string]interface{} `json:"data"`
}

// PaginationMeta contains pagination metadata
type PaginationMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// MapPaymentToResponse converts a Payment model to PaymentResponse DTO
func MapPaymentToResponse(p *models.Payment) PaymentResponse {
	return PaymentResponse{
		ID:                p.ID,
		OrderID:           p.OrderID,
		Amount:            p.Amount,
		Currency:          p.Currency,
		Provider:          p.Provider,
		Status:            p.Status,
		Reference:         p.Reference,
		ExternalReference: p.ExternalReference,
		CustomerEmail:     p.CustomerEmail,
		CreatedAt:         p.CreatedAt,
		CompletedAt:       p.CompletedAt,
	}
}

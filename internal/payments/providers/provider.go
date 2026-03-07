package providers

import (
	"context"

	"github.com/shopspring/decimal"
)

// PaymentProvider defines the interface that all payment providers must implement
type PaymentProvider interface {
	// Initialize initiates a payment transaction
	InitializePayment(ctx context.Context, req *InitializeRequest) (*InitializeResponse, error)

	// Verify checks the status of a payment transaction
	VerifyPayment(ctx context.Context, reference string) (*VerifyResponse, error)

	// Refund processes a refund for a payment
	RefundPayment(ctx context.Context, reference string, amount decimal.Decimal, reason string) (*RefundResponse, error)

	// ValidateWebhook validates webhook signature/payload from provider
	ValidateWebhook(ctx context.Context, body []byte, headers map[string]string) (bool, error)

	// GetName returns the provider name
	GetName() string
}

type InitializeRequest struct {
	Amount         decimal.Decimal
	Currency       string
	Email          string
	Phone          string
	Name           string
	Description    string
	Reference      string // Our internal reference
	CallbackURL    string
	Metadata       map[string]interface{}
}

type InitializeResponse struct {
	Status           string
	AuthorizationURL string
	AccessCode       string
	Reference        string // Provider's reference
}

type VerifyResponse struct {
	Status            string
	Amount            decimal.Decimal
	Currency          string
	Reference         string
	ExternalReference string
	CustomerEmail     string
	Success           bool
	Metadata          map[string]interface{}
}

type RefundResponse struct {
	Status           string
	ProviderRefundID string
	Amount           decimal.Decimal
	Message          string
}

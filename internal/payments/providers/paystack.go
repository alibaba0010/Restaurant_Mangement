package providers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const (
	paystackAPIBaseURL = "https://api.paystack.co"
	paystackTimeout    = 30 * time.Second
)

// Paystack structs
type paystackMetadata struct {
	Phone            string                 `json:"phone"`
	Name             string                 `json:"name"`
	CustomFields     map[string]interface{} `json:"custom_fields"`
	OurTransactionID string                 `json:"our_transaction_id"`
}

type paystackInitRequest struct {
	Amount      int64            `json:"amount"`
	Email       string           `json:"email"`
	Reference   string           `json:"reference"`
	CallbackURL string           `json:"callback_url"`
	Metadata    paystackMetadata `json:"metadata"`
}

type paystackInitResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		AuthorizationURL string `json:"authorization_url"`
		AccessCode       string `json:"access_code"`
		Reference        string `json:"reference"`
	} `json:"data"`
}

type paystackVerifyResponse struct {
	Status bool   `json:"status"`
	Message string `json:"message"`
	Data   struct {
		Amount       int64  `json:"amount"`
		Currency     string `json:"currency"`
		Status       string `json:"status"`
		Reference    string `json:"reference"`
		CreatedAt    string `json:"created_at"`
		Customer     struct {
			Email string `json:"email"`
			Name  string `json:"customer_code"`
		} `json:"customer"`
		Metadata map[string]interface{} `json:"metadata"`
	} `json:"data"`
}

type paystackWebhookPayload struct {
	Event string `json:"event"`
	Data  struct {
		Amount    int64  `json:"amount"`
		Currency  string `json:"currency"`
		Status    string `json:"status"`
		Reference string `json:"reference"`
		Customer  struct {
			Email string `json:"email"`
		} `json:"customer"`
		Metadata map[string]interface{} `json:"metadata"`
	} `json:"data"`
}

type paystackRefundRequest struct {
	Transaction string `json:"transaction"`
	Amount      int64  `json:"amount"`
	Note        string `json:"note,omitempty"`
}

type paystackRefundResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		Reference string `json:"reference"`
		Amount    int64  `json:"amount"`
		Status    string `json:"status"`
	} `json:"data"`
}

// PaystackProvider implements the PaymentProvider interface for Paystack
type PaystackProvider struct {
	apiKey string
	client *http.Client
}

// NewPaystackProvider creates a new Paystack provider instance
func NewPaystackProvider(apiKey string) *PaystackProvider {
	return &PaystackProvider{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: paystackTimeout,
		},
	}
}

// GetName returns the provider name
func (p *PaystackProvider) GetName() string {
	return "paystack"
}

// InitializePayment initiates a payment with Paystack
func (p *PaystackProvider) InitializePayment(ctx context.Context, req *InitializeRequest) (*InitializeResponse, error) {
	if req.Amount.LessThanOrEqual(decimal.Zero) || req.Email == "" || req.Reference == "" {
		return nil, fmt.Errorf("invalid payment request: amount, email, and reference are required")
	}

	// Paystack expects amount in kobo (cents * 100)
	amountInKobo := req.Amount.Mul(decimal.NewFromInt(100)).IntPart()

	body := paystackInitRequest{
		Amount:      amountInKobo,
		Email:       req.Email,
		Reference:   req.Reference,
		CallbackURL: req.CallbackURL,
		Metadata: paystackMetadata{
			Phone:            req.Phone,
			Name:             req.Name,
			CustomFields:     req.Metadata,
			OurTransactionID: req.Reference,
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	logger.Log.Warn("Paystack request", zap.String("Reference", req.Reference), zap.Int64("amount in kobo", amountInKobo), zap.String("url", req.CallbackURL))
	
	request, err := http.NewRequestWithContext(ctx, "POST", paystackAPIBaseURL+"/transaction/initialize", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	request.Header.Set("Authorization", "Bearer "+p.apiKey)
	request.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("paystack request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.Log.Warn("Paystack API error", zap.Int("status", resp.StatusCode), zap.String("body", string(respBody)))
		return nil, fmt.Errorf("paystack API error: status %d", resp.StatusCode)
	}

	var paystackResp paystackInitResponse
	if err := json.Unmarshal(respBody, &paystackResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !paystackResp.Status {
		return nil, fmt.Errorf("paystack response indicates failure: %s", paystackResp.Message)
	}

	return &InitializeResponse{
		Status:           "pending",
		AuthorizationURL: paystackResp.Data.AuthorizationURL,
		AccessCode:       paystackResp.Data.AccessCode,
		Reference:        paystackResp.Data.Reference,
	}, nil
}

// VerifyPayment checks payment status with Paystack
func (p *PaystackProvider) VerifyPayment(ctx context.Context, reference string) (*VerifyResponse, error) {
	if reference == "" {
		return nil, fmt.Errorf("reference is required")
	}

	url := fmt.Sprintf("%s/transaction/verify/%s", paystackAPIBaseURL, reference)
	request, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	request.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("paystack request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return &VerifyResponse{
				Status:    "NOT_FOUND",
				Reference: reference,
				Success:   false,
			}, nil
		}
		logger.Log.Warn("Paystack verification error", zap.Int("status", resp.StatusCode), zap.String("body", string(respBody)))
		return nil, fmt.Errorf("paystack verification failed: status %d", resp.StatusCode)
	}


	var paystackResp paystackVerifyResponse
	if err := json.Unmarshal(respBody, &paystackResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !paystackResp.Status {
		return nil, fmt.Errorf("paystack verification returned false status")
	}

	return &VerifyResponse{
		Status:            paystackResp.Data.Status,
		Amount:            decimal.NewFromInt(paystackResp.Data.Amount).Div(decimal.NewFromInt(100)), // Convert from kobo
		Currency:          paystackResp.Data.Currency,
		Reference:         reference,             // Internal reference
		ExternalReference: paystackResp.Data.Reference, // External reference
		CustomerEmail:     paystackResp.Data.Customer.Email,
		Success:           paystackResp.Data.Status == "success",
		Metadata:          paystackResp.Data.Metadata,
	}, nil
}

// RefundPayment processes a refund with Paystack
func (p *PaystackProvider) RefundPayment(ctx context.Context, reference string, amount decimal.Decimal, reason string) (*RefundResponse, error) {
	if reference == "" || amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("reference and positive amount are required")
	}

	body := paystackRefundRequest{
		Transaction: reference,
		Amount:      amount.Mul(decimal.NewFromInt(100)).IntPart(),
		Note:        reason,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, "POST", paystackAPIBaseURL+"/refund", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	request.Header.Set("Authorization", "Bearer "+p.apiKey)
	request.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("paystack request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.Log.Warn("Paystack refund error", zap.Int("status", resp.StatusCode), zap.String("body", string(respBody)))
		return nil, fmt.Errorf("paystack refund failed: status %d", resp.StatusCode)
	}

	var paystackResp paystackRefundResponse
	if err := json.Unmarshal(respBody, &paystackResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &RefundResponse{
		Status:           paystackResp.Data.Status,
		ProviderRefundID: paystackResp.Data.Reference,
		Amount:           decimal.NewFromInt(paystackResp.Data.Amount).Div(decimal.NewFromInt(100)), // Convert from kobo
		Message:          paystackResp.Message,
	}, nil
}

// ValidateWebhook validates the signature of Paystack webhook
func (p *PaystackProvider) ValidateWebhook(ctx context.Context, body []byte, headers map[string]string) (bool, error) {
	signature := headers["x-paystack-signature"]
	if signature == "" {
		return false, nil
	}

	hash := hmac.New(sha512.New, []byte(p.apiKey))
	hash.Write(body)
	computedSignature := hex.EncodeToString(hash.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(computedSignature)), nil
}

// ParseWebhook parses Paystack webhook payload
func (p *PaystackProvider) ParseWebhook(payload []byte) (*VerifyResponse, error) {
	var event paystackWebhookPayload
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}

	// Internal reference is usually stored in metadata or is the reference itself
	reference := event.Data.Reference
	if val, ok := event.Data.Metadata["reference"].(string); ok {
		reference = val
	} else if val, ok := event.Data.Metadata["payment_reference"].(string); ok {
		reference = val
	}

	return &VerifyResponse{
		Status:            event.Data.Status,
		Amount:            decimal.NewFromInt(event.Data.Amount).Div(decimal.NewFromInt(100)),
		Currency:          event.Data.Currency,
		Reference:         reference,
		ExternalReference: event.Data.Reference,
		CustomerEmail:     event.Data.Customer.Email,
		Success:           event.Data.Status == "success" || event.Event == "charge.success",
		Metadata:          event.Data.Metadata,
	}, nil
}

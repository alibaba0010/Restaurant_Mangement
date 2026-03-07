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

	body := map[string]interface{}{
		"amount":       amountInKobo,
		"email":        req.Email,
		"reference":    req.Reference,
		"callback_url": req.CallbackURL,
		"metadata": map[string]interface{}{
			"phone":              req.Phone,
			"name":               req.Name,
			"custom_fields":      req.Metadata,
			"our_transaction_id": req.Reference,
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

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

	var paystackResp struct {
		Status  bool   `json:"status"`
		Message string `json:"message"`
		Data    struct {
			AuthorizationURL string `json:"authorization_url"`
			AccessCode       string `json:"access_code"`
			Reference        string `json:"reference"`
		} `json:"data"`
	}

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
		logger.Log.Warn("Paystack verification error", zap.Int("status", resp.StatusCode), zap.String("body", string(respBody)))
		return nil, fmt.Errorf("paystack verification failed: status %d", resp.StatusCode)
	}

	var paystackResp struct {
		Status bool `json:"status"`
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
		Reference:         reference, // Internal reference
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

	body := map[string]interface{}{
		"transaction": reference,
		"amount":      amount.Mul(decimal.NewFromInt(100)).IntPart(), // Convert to kobo
	}

	if reason != "" {
		body["note"] = reason
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

	var paystackResp struct {
		Status  bool   `json:"status"`
		Message string `json:"message"`
		Data    struct {
			Reference string `json:"reference"`
			Amount    int64  `json:"amount"`
			Status    string `json:"status"`
		} `json:"data"`
	}

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

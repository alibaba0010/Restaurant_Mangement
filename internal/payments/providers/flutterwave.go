package providers

import (
	"bytes"
	"context"
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
	flutterwaveAPIBaseURL = "https://api.flutterwave.com/v3"
	flutterwaveTimeout    = 30 * time.Second
)

// FlutterwaveProvider implements the PaymentProvider interface for Flutterwave
type FlutterwaveProvider struct {
	secretKey     string
	encryptionKey string
	client        *http.Client
}

// NewFlutterwaveProvider creates a new Flutterwave provider instance
func NewFlutterwaveProvider(secretKey, encryptionKey string) *FlutterwaveProvider {
	return &FlutterwaveProvider{
		secretKey:     secretKey,
		encryptionKey: encryptionKey,
		client: &http.Client{
			Timeout: flutterwaveTimeout,
		},
	}
}

// GetName returns the provider name
func (fw *FlutterwaveProvider) GetName() string {
	return "flutterwave"
}

// InitializePayment initiates a payment with Flutterwave
func (fw *FlutterwaveProvider) InitializePayment(ctx context.Context, req *InitializeRequest) (*InitializeResponse, error) {
	if req.Amount.LessThanOrEqual(decimal.Zero) || req.Email == "" || req.Reference == "" {
		return nil, fmt.Errorf("invalid payment request: amount, email, and reference are required")
	}

	body := map[string]interface{}{
		"tx_ref":          req.Reference,
		"amount":          req.Amount,
		"currency":        req.Currency,
		"payment_options": "card,banktransfer,ussd",
		"customer": map[string]interface{}{
			"email":       req.Email,
			"phonenumber": req.Phone,
			"name":        req.Name,
		},
		"customizations": map[string]interface{}{
			"title":       "Payment",
			"description": req.Description,
		},
		"redirect_url": req.CallbackURL,
		"meta":         req.Metadata,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, "POST", flutterwaveAPIBaseURL+"/payments", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	request.Header.Set("Authorization", "Bearer "+fw.secretKey)
	request.Header.Set("Content-Type", "application/json")

	resp, err := fw.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("flutterwave request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.Log.Warn("Flutterwave API error", zap.Int("status", resp.StatusCode), zap.String("body", string(respBody)))
		return nil, fmt.Errorf("flutterwave API error: status %d", resp.StatusCode)
	}

	var flutterwaveResp struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Data    struct {
			Link string `json:"link"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &flutterwaveResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if flutterwaveResp.Status != "success" {
		return nil, fmt.Errorf("flutterwave response indicates failure: %s", flutterwaveResp.Message)
	}

	return &InitializeResponse{
		Status:           "pending",
		AuthorizationURL: flutterwaveResp.Data.Link,
		Reference:        req.Reference,
	}, nil
}

// VerifyPayment checks payment status with Flutterwave
func (fw *FlutterwaveProvider) VerifyPayment(ctx context.Context, reference string) (*VerifyResponse, error) {
	if reference == "" {
		return nil, fmt.Errorf("reference is required")
	}

	url := fmt.Sprintf("%s/transactions/%s/verify", flutterwaveAPIBaseURL, reference)
	request, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	request.Header.Set("Authorization", "Bearer "+fw.secretKey)

	resp, err := fw.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("flutterwave request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.Log.Warn("Flutterwave verification error", zap.Int("status", resp.StatusCode), zap.String("body", string(respBody)))
		return nil, fmt.Errorf("flutterwave verification failed: status %d", resp.StatusCode)
	}

	var flutterwaveResp struct {
		Status string `json:"status"`
		Data   struct {
			ID       int             `json:"id"`
			TxRef    string          `json:"tx_ref"`
			FlwRef   string          `json:"flw_ref"`
			Amount   decimal.Decimal `json:"amount"`
			Currency string          `json:"currency"`
			Status   string          `json:"status"`
			Customer struct {
				Email string `json:"email"`
			} `json:"customer"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &flutterwaveResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if flutterwaveResp.Status != "success" {
		return nil, fmt.Errorf("flutterwave verification returned failure status")
	}

	return &VerifyResponse{
		Status:            flutterwaveResp.Data.Status,
		Amount:            flutterwaveResp.Data.Amount,
		Currency:          flutterwaveResp.Data.Currency,
		Reference:         reference,
		ExternalReference: flutterwaveResp.Data.FlwRef,
		CustomerEmail:     flutterwaveResp.Data.Customer.Email,
		Success:           flutterwaveResp.Data.Status == "successful",
	}, nil
}

// RefundPayment processes a refund with Flutterwave
func (fw *FlutterwaveProvider) RefundPayment(ctx context.Context, reference string, amount decimal.Decimal, reason string) (*RefundResponse, error) {
	if reference == "" || amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("reference and positive amount are required")
	}

	body := map[string]interface{}{
		"amount": amount,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/transactions/%s/refund", flutterwaveAPIBaseURL, reference)
	request, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	request.Header.Set("Authorization", "Bearer "+fw.secretKey)
	request.Header.Set("Content-Type", "application/json")

	resp, err := fw.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("flutterwave request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.Log.Warn("Flutterwave refund error", zap.Int("status", resp.StatusCode), zap.String("body", string(respBody)))
		return nil, fmt.Errorf("flutterwave refund failed: status %d", resp.StatusCode)
	}

	var flutterwaveResp struct {
		Status string `json:"status"`
		Data   struct {
			ID     int             `json:"id"`
			Status string          `json:"status"`
			Amount decimal.Decimal `json:"amount"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &flutterwaveResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &RefundResponse{
		Status:           flutterwaveResp.Data.Status,
		ProviderRefundID: fmt.Sprintf("%d", flutterwaveResp.Data.ID),
		Amount:           flutterwaveResp.Data.Amount,
		Message:          "Refund processed successfully",
	}, nil
}

// ValidateWebhook validates the signature of Flutterwave webhook
func (fw *FlutterwaveProvider) ValidateWebhook(ctx context.Context, body []byte, headers map[string]string) (bool, error) {
	signature := headers["verif-hash"]
	if signature == "" {
		return false, nil
	}

	// Flutterwave uses a simple hash verification from settings
	// For production, you should use the secret hash configured in dashboard
	return signature == fw.encryptionKey, nil
}

func (fw *FlutterwaveProvider) ParseWebhook(payload []byte) (*VerifyResponse, error) {
	var event struct {
		Event string `json:"event"`
		Data  struct {
			ID           int64                  `json:"id"`
			TxRef        string                 `json:"tx_ref"`
			FlwRef       string                 `json:"flw_ref"`
			Amount       float64                `json:"amount"`
			Currency     string                 `json:"currency"`
			Status       string                 `json:"status"`
			Customer     struct {
				Email string `json:"email"`
			} `json:"customer"`
			Metadata map[string]interface{} `json:"meta"`
		} `json:"data"`
	}

	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}

	return &VerifyResponse{
		Status:            event.Data.Status,
		Amount:            decimal.NewFromFloat(event.Data.Amount),
		Currency:          event.Data.Currency,
		Reference:         event.Data.TxRef,
		ExternalReference: event.Data.FlwRef,
		CustomerEmail:     event.Data.Customer.Email,
		Success:           event.Data.Status == "successful" || event.Event == "charge.completed",
		Metadata:          event.Data.Metadata,
	}, nil
}


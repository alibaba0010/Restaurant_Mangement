package providers

import (
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/base64"
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
	monnifyAPIBaseURL = "https://api.monnify.com/api/v1"
	monnifyTimeout    = 30 * time.Second
)

// MonnifyProvider implements the PaymentProvider interface for Monnify
type MonnifyProvider struct {
	apiKey       string
	secretKey    string
	contractCode string
	client       *http.Client
}

// NewMonnifyProvider creates a new Monnify provider instance
func NewMonnifyProvider(apiKey, secretKey, contractCode string) *MonnifyProvider {
	return &MonnifyProvider{
		apiKey:       apiKey,
		secretKey:    secretKey,
		contractCode: contractCode,
		client: &http.Client{
			Timeout: monnifyTimeout,
		},
	}
}

// GetName returns the provider name
func (m *MonnifyProvider) GetName() string {
	return "monnify"
}

// InitializePayment initiates a payment with Monnify
func (m *MonnifyProvider) InitializePayment(ctx context.Context, req *InitializeRequest) (*InitializeResponse, error) {
	if req.Amount.LessThanOrEqual(decimal.Zero) || req.Email == "" || req.Reference == "" {
		return nil, fmt.Errorf("invalid payment request: amount, email, and reference are required")
	}

	body := map[string]interface{}{
		"amount":           req.Amount,
		"currency":         req.Currency,
		"contractCode":     m.contractCode,
		"reference":        req.Reference,
		"description":      req.Description,
		"customerName":     req.Name,
		"customerEmail":    req.Email,
		"customerPhone":    req.Phone,
		"paymentMethod":    "CARD,BANK_TRANSFER",
		"redirectUrl":      req.CallbackURL,
		"metadata":         req.Metadata,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, "POST", monnifyAPIBaseURL+"/merchant/transactions/init", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	token, err := m.getBearerToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get monnify access token: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("monnify request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		logger.Log.Warn("Monnify API error", zap.Int("status", resp.StatusCode), zap.String("body", string(respBody)))
		return nil, fmt.Errorf("monnify API error: status %d", resp.StatusCode)
	}

	var monnifyResp struct {
		RequestSuccessful bool   `json:"requestSuccessful"`
		ResponseMessage   string `json:"responseMessage"`
		Data              struct {
			PaymentReference string `json:"paymentReference"`
			CheckoutURL      string `json:"checkoutUrl"`
			TransactionRef   string `json:"transactionReference"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &monnifyResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !monnifyResp.RequestSuccessful {
		return nil, fmt.Errorf("monnify response indicates failure: %s", monnifyResp.ResponseMessage)
	}

	return &InitializeResponse{
		Status:           "pending",
		AuthorizationURL: monnifyResp.Data.CheckoutURL,
		Reference:        monnifyResp.Data.PaymentReference,
	}, nil
}

// VerifyPayment checks payment status with Monnify
func (m *MonnifyProvider) VerifyPayment(ctx context.Context, reference string) (*VerifyResponse, error) {
	if reference == "" {
		return nil, fmt.Errorf("reference is required")
	}

	url := fmt.Sprintf("%s/merchant/transactions/query?paymentReference=%s", monnifyAPIBaseURL, reference)
	request, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	token, err := m.getBearerToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get monnify access token: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+token)

	resp, err := m.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("monnify request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.Log.Warn("Monnify verification error", zap.Int("status", resp.StatusCode), zap.String("body", string(respBody)))
		return nil, fmt.Errorf("monnify verification failed: status %d", resp.StatusCode)
	}

	var monnifyResp struct {
		RequestSuccessful bool `json:"requestSuccessful"`
		Data              struct {
			PaymentStatus   string          `json:"paymentStatus"`
			Amount          decimal.Decimal `json:"amount"`
			Currency        string          `json:"currency"`
			PaidOn          string  `json:"paidOn"`
			CustomerEmail   string  `json:"customerEmail"`
			TransactionRef  string  `json:"transactionReference"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &monnifyResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !monnifyResp.RequestSuccessful {
		return nil, fmt.Errorf("monnify verification returned failure status")
	}

	return &VerifyResponse{
		Status:            monnifyResp.Data.PaymentStatus,
		Amount:            monnifyResp.Data.Amount,
		Currency:          monnifyResp.Data.Currency,
		Reference:         reference,
		ExternalReference: monnifyResp.Data.TransactionRef,
		CustomerEmail:     monnifyResp.Data.CustomerEmail,
		Success:           monnifyResp.Data.PaymentStatus == "PAID",
	}, nil
}

// RefundPayment processes a refund with Monnify
func (m *MonnifyProvider) RefundPayment(ctx context.Context, reference string, amount decimal.Decimal, reason string) (*RefundResponse, error) {
	if reference == "" || amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("reference and positive amount are required")
	}

	body := map[string]interface{}{
		"refundReference":  fmt.Sprintf("REFUND-%s-%d", reference, time.Now().UnixMilli()),
		"paymentReference": reference,
		"refundAmount":     amount,
		"reason":           reason,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, "POST", monnifyAPIBaseURL+"/merchant/transactions/refund", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	token, err := m.getBearerToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get monnify access token: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("monnify request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.Log.Warn("Monnify refund error", zap.Int("status", resp.StatusCode), zap.String("body", string(respBody)))
		return nil, fmt.Errorf("monnify refund failed: status %d", resp.StatusCode)
	}

	var monnifyResp struct {
		RequestSuccessful bool   `json:"requestSuccessful"`
		ResponseMessage   string `json:"responseMessage"`
		Data              struct {
			RefundRef string          `json:"refundReference"`
			Status    string          `json:"refundStatus"`
			Amount    decimal.Decimal `json:"amount"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &monnifyResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &RefundResponse{
		Status:           monnifyResp.Data.Status,
		ProviderRefundID: monnifyResp.Data.RefundRef,
		Amount:           monnifyResp.Data.Amount,
		Message:          monnifyResp.ResponseMessage,
	}, nil
}

// ValidateWebhook validates the signature of Monnify webhook
func (m *MonnifyProvider) ValidateWebhook(ctx context.Context, body []byte, headers map[string]string) (bool, error) {
	signature := headers["monnify-signature"]
	if signature == "" {
		return false, nil
	}

	hash := sha512.Sum512(append(body, []byte(m.apiKey)...))
	computedSignature := base64.StdEncoding.EncodeToString(hash[:])

	return computedSignature == signature, nil
}

func (m *MonnifyProvider) getBearerToken(ctx context.Context) (string, error) {
	authString := fmt.Sprintf("%s:%s", m.apiKey, m.secretKey)
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(authString))

	req, err := http.NewRequestWithContext(ctx, "POST", monnifyAPIBaseURL+"/auth/login", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Basic "+encodedAuth)

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("monnify auth failed with status %d", resp.StatusCode)
	}

	var authResp struct {
		RequestSuccessful bool `json:"requestSuccessful"`
		Data              struct {
			AccessToken string `json:"accessToken"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return "", err
	}

	if !authResp.RequestSuccessful {
		return "", fmt.Errorf("monnify auth response unsuccessful")
	}

	return authResp.Data.AccessToken, nil
}

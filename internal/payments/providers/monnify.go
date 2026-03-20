package providers

import (
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const (
	monnifyTimeout      = 30 * time.Second
)
// production url: https://api.monnify.com/api/v1
// production url: https://api.monnify.com/api/v2
// Monnify structs
type monnifyAuthResponse struct {
	RequestSuccessful bool   `json:"requestSuccessful"`
	ResponseMessage   string `json:"responseMessage"`
	ResponseBody      struct {
		AccessToken string `json:"accessToken"`
		ExpiresIn   int    `json:"expiresIn"`
	} `json:"responseBody"`
}

type monnifyInitRequest struct {
	Amount             decimal.Decimal        `json:"amount"`
	CurrencyCode       string                 `json:"currencyCode"`
	ContractCode       string                 `json:"contractCode"`
	PaymentReference   string                 `json:"paymentReference"`
	PaymentDescription string                 `json:"paymentDescription"`
	CustomerName       string                 `json:"customerName"`
	CustomerEmail      string                 `json:"customerEmail"`
	CustomerPhone      string                 `json:"customerPhone,omitempty"`
	PaymentMethods     []string               `json:"paymentMethods,omitempty"`
	RedirectUrl        string                 `json:"redirectUrl"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	IncomeSplitConfig  interface{}            `json:"incomeSplitConfig,omitempty"`
}

type monnifyInitResponse struct {
	RequestSuccessful bool   `json:"requestSuccessful"`
	ResponseMessage   string `json:"responseMessage"`
	ResponseBody      struct {
		PaymentReference string `json:"paymentReference"`
		CheckoutURL      string `json:"checkoutUrl"`
		TransactionRef   string `json:"transactionReference"`
	} `json:"responseBody"`
}

type monnifyVerifyResponse struct {
	RequestSuccessful bool   `json:"requestSuccessful"`
	ResponseMessage   string `json:"responseMessage"`
	ResponseBody      struct {
		PaymentStatus  string          `json:"paymentStatus"`
		AmountPaid     decimal.Decimal `json:"amountPaid"`
		TotalPayable   decimal.Decimal `json:"totalPayable"`
		Currency       string          `json:"currencyCode"`
		PaidOn         string          `json:"paidOn"`
		CustomerEmail  string          `json:"customerEmail"`
		TransactionRef string          `json:"transactionReference"`
	} `json:"responseBody"`
}

type monnifyWebhookPayload struct {
	EventType string `json:"eventType"`
	EventData struct {
		AmountPaid           decimal.Decimal        `json:"amountPaid"`
		Amount               decimal.Decimal        `json:"amount"` 
		CurrencyCode         string                 `json:"currencyCode"`
		PaymentStatus        string                 `json:"paymentStatus"`
		PaymentReference     string                 `json:"paymentReference"`
		TransactionReference string                 `json:"transactionReference"`
		CustomerEmail        string                 `json:"customerEmail"`
		Customer             struct {
			Email string `json:"email"`
		} `json:"customer"`
		Meta                 map[string]interface{} `json:"meta"`
		MetaData             map[string]interface{} `json:"metaData"`
	} `json:"eventData"`
}

// MonnifyProvider implements the PaymentProvider interface for Monnify
type MonnifyProvider struct {
	apiKey       string
	secretKey    string
	contractCode string
	baseURLV1    string
	baseURLV2    string
	client       *http.Client
}

// NewMonnifyProvider creates a new Monnify provider instance
func NewMonnifyProvider(apiKey, secretKey, contractCode string, isProd bool) *MonnifyProvider {
	baseURLV1 := "https://sandbox.monnify.com/api/v1"
	baseURLV2 := "https://sandbox.monnify.com/api/v2"
	if isProd {
		baseURLV1 = "https://api.monnify.com/api/v1"
		baseURLV2 = "https://api.monnify.com/api/v2"
	}
	return &MonnifyProvider{
		apiKey:       apiKey,
		secretKey:    secretKey,
		contractCode: contractCode,
		baseURLV1:    baseURLV1,
		baseURLV2:    baseURLV2,
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

	// Since we are using the Web SDK inline popup on the frontend,
	// we do not call the init-transaction API on the backend. 
	// The frontend SDK relies on making the initialization call, and if we make it here, 
	// we consume the unique reference and the user will incorrectly get a "Transaction Failed!" error.
	// NOTE: This is perfectly secure because our `verifyPaymentInternal` checks `verifyResp.Amount.Equal(payment.Amount)`,
	// completely negating any frontend tampering vectors.
	
	return &InitializeResponse{
		Status:           "pending",
		AuthorizationURL: "", // Leave blank to tell the frontend to strictly use the inline popup
		Reference:        req.Reference,
	}, nil
}

// VerifyPayment checks payment status with Monnify
func (m *MonnifyProvider) VerifyPayment(ctx context.Context, reference string) (*VerifyResponse, error) {
	if reference == "" {
		return nil, fmt.Errorf("reference is required")
	}

	url := fmt.Sprintf("%s/merchant/transactions/query?paymentReference=%s", m.baseURLV2, reference)
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
		if resp.StatusCode == http.StatusNotFound {
			return &VerifyResponse{
				Status:    "NOT_FOUND",
				Reference: reference,
				Success:   false,
			}, nil
		}
		logger.Log.Warn("Monnify verification error", zap.Int("status", resp.StatusCode), zap.String("body", string(respBody)))
		return nil, fmt.Errorf("monnify verification failed: status %d", resp.StatusCode)
	}


	var monnifyResp monnifyVerifyResponse
	if err := json.Unmarshal(respBody, &monnifyResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !monnifyResp.RequestSuccessful {
		logger.Log.Warn("Monnify verification failed", zap.String("message", monnifyResp.ResponseMessage), zap.String("body", string(respBody)))
		return nil, fmt.Errorf("monnify verification returned failure status: %s", monnifyResp.ResponseMessage)
	}

	status := monnifyResp.ResponseBody.PaymentStatus
	success := strings.EqualFold(status, "PAID") || 
			   strings.EqualFold(status, "SUCCESS") || 
			   strings.EqualFold(status, "OVERPAID")

	return &VerifyResponse{
		Status:            status,
		Amount:            monnifyResp.ResponseBody.AmountPaid,
		Currency:          monnifyResp.ResponseBody.Currency,
		Reference:         reference,
		ExternalReference: monnifyResp.ResponseBody.TransactionRef,
		CustomerEmail:     monnifyResp.ResponseBody.CustomerEmail,
		Success:           success,
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

	request, err := http.NewRequestWithContext(ctx, "POST", m.baseURLV1+"/merchant/transactions/refund", bytes.NewBuffer(jsonBody))
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
		ResponseBody      struct {
			RefundRef string          `json:"refundReference"`
			Status    string          `json:"refundStatus"`
			Amount    decimal.Decimal `json:"amount"`
		} `json:"responseBody"`
	}

	if err := json.Unmarshal(respBody, &monnifyResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &RefundResponse{
		Status:           monnifyResp.ResponseBody.Status,
		ProviderRefundID: monnifyResp.ResponseBody.RefundRef,
		Amount:           monnifyResp.ResponseBody.Amount,
		Message:          monnifyResp.ResponseMessage,
	}, nil
}

// ValidateWebhook validates the signature of Monnify webhook
func (m *MonnifyProvider) ValidateWebhook(ctx context.Context, body []byte, headers map[string]string) (bool, error) {
	signature := headers["monnify-signature"]
	if signature == "" {
		return false, nil
	}

	h := sha512.New()
	h.Write([]byte(m.secretKey))
	h.Write(body)
	computedSignature := hex.EncodeToString(h.Sum(nil))

	return computedSignature == signature, nil
}

func (m *MonnifyProvider) ParseWebhook(payload []byte) (*VerifyResponse, error) {
	var event monnifyWebhookPayload
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}

	metadata := event.EventData.Meta
	if len(metadata) == 0 {
		metadata = event.EventData.MetaData
	}

	amount := event.EventData.AmountPaid
	if amount.IsZero() {
		amount = event.EventData.Amount
	}

	email := event.EventData.CustomerEmail
	if email == "" {
		email = event.EventData.Customer.Email
	}

	status := event.EventData.PaymentStatus
	success := strings.EqualFold(status, "PAID") || 
			   strings.EqualFold(status, "SUCCESS") || 
			   strings.EqualFold(status, "OVERPAID") || 
			   strings.EqualFold(event.EventType, "SUCCESSFUL_TRANSACTION")

	return &VerifyResponse{
		Status:            status,
		Amount:            amount,
		Currency:          event.EventData.CurrencyCode,
		Reference:         event.EventData.PaymentReference,
		ExternalReference: event.EventData.TransactionReference,
		CustomerEmail:     email,
		Success:           success,
		Metadata:          metadata,
	}, nil

}

func (m *MonnifyProvider) getBearerToken(ctx context.Context) (string, error) {
	authString := fmt.Sprintf("%s:%s", m.apiKey, m.secretKey)
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(authString))

	req, err := http.NewRequestWithContext(ctx, "POST", m.baseURLV1+"/auth/login", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Basic "+encodedAuth)

	resp, err := m.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("monnify auth request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("monnify auth: failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.Log.Warn("Monnify auth failed",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(respBody)),
		)
		return "", fmt.Errorf("monnify auth failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var authResp monnifyAuthResponse
	if err := json.Unmarshal(respBody, &authResp); err != nil {
		return "", fmt.Errorf("monnify auth: failed to parse response: %w", err)
	}

	if !authResp.RequestSuccessful {
		return "", fmt.Errorf("monnify auth failed: %s", authResp.ResponseMessage)
	}

	if authResp.ResponseBody.AccessToken == "" {
		logger.Log.Warn("Monnify auth returned empty access token", zap.String("raw_response", string(respBody)))
		return "", fmt.Errorf("monnify auth: access token missing from response")
	}

	return authResp.ResponseBody.AccessToken, nil
}

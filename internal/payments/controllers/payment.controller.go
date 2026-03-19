package controllers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/guards"
	"github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/alibaba0010/postgres-api/internal/payments/dto"
	"github.com/alibaba0010/postgres-api/internal/payments/services"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type PaymentController struct {
	service   services.PaymentService
	validator *validator.Validate
}

type PaymentControllerInterface interface {
	InitiatePayment(w http.ResponseWriter, r *http.Request)
	VerifyPayment(w http.ResponseWriter, r *http.Request)
	WebhookHandler(w http.ResponseWriter, r *http.Request)
}


// NewPaymentController creates a new instance of PaymentController
func NewPaymentController(service services.PaymentService) *PaymentController {
	return &PaymentController{
		service:   service,
		validator: validator.New(),
	}
}

// InitiatePayment handles the initiation of a payment
func (c *PaymentController) InitiatePayment(w http.ResponseWriter, r *http.Request) {
	var req dto.InitiatePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.ErrorResponse(w, r, errors.ValidationError("Invalid request body"))
		return
	}

	if err := c.validator.Struct(req); err != nil {
		errors.ErrorResponse(w, r, errors.ValidationError(err.Error()))
		return
	}

	user := guards.ExtractAuthenticatedUser(r)
	if user == nil {
		errors.ErrorResponse(w, r, errors.UnauthorizedError("User not authenticated"))
		return
	}

	userID, err := uuid.Parse(user.UserID)
	if err != nil {
		errors.ErrorResponse(w, r, errors.UnauthorizedError("Invalid user ID"))
		return
	}

	resp, appErr := c.service.InitiatePayment(r.Context(), &req, userID)
	if appErr != nil {
		errors.ErrorResponse(w, r, appErr)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// VerifyPayment handles the verification of a payment
func (c *PaymentController) VerifyPayment(w http.ResponseWriter, r *http.Request) {
	reference := r.URL.Query().Get("reference")
	if reference == "" {
		errors.ErrorResponse(w, r, errors.ValidationError("Reference is required"))
		return
	}

	user := guards.ExtractAuthenticatedUser(r)
	if user == nil {
		errors.ErrorResponse(w, r, errors.UnauthorizedError("User not authenticated"))
		return
	}

	userID, err := uuid.Parse(user.UserID)
	if err != nil {
		errors.ErrorResponse(w, r, errors.UnauthorizedError("Invalid user ID"))
		return
	}

	resp, appErr := c.service.VerifyPayment(r.Context(), reference, userID)
	if appErr != nil {
		errors.ErrorResponse(w, r, appErr)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// WebhookHandler handles the webhook notification from the payment provider
func (c *PaymentController) WebhookHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	providerName := vars["provider"]
	providerEnum := types.PaymentProvider(providerName)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Normalize header keys to lowercase so providers can look them up
	// consistently (Go's http.Header stores keys in canonical PascalCase).
	headers := make(map[string]string)
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[strings.ToLower(k)] = v[0]
		}
	}

	if appErr := c.service.HandleWebhook(r.Context(), providerEnum, body, headers); appErr != nil {
		// Log error and return status from appErr (usually 500 for processing errors so provider retries)
		errors.ErrorResponse(w, r, appErr)
		return
	}

	w.WriteHeader(http.StatusOK)
}

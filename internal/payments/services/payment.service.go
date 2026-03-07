package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alibaba0010/postgres-api/internal/common/events"
	"github.com/alibaba0010/postgres-api/internal/common/guards"
	"github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/alibaba0010/postgres-api/internal/config"
	"github.com/alibaba0010/postgres-api/internal/database"
	orderModels "github.com/alibaba0010/postgres-api/internal/orders/models"
	"github.com/alibaba0010/postgres-api/internal/payments/dto"
	paymentEvents "github.com/alibaba0010/postgres-api/internal/payments/events"
	"github.com/alibaba0010/postgres-api/internal/payments/models"
	"github.com/alibaba0010/postgres-api/internal/payments/providers"
	"github.com/alibaba0010/postgres-api/internal/payments/repositories"
	"github.com/shopspring/decimal"

	"github.com/google/uuid"
)

type PaymentService interface {
	InitiatePayment(ctx context.Context, req *dto.InitiatePaymentRequest, userID uuid.UUID) (*dto.InitiatePaymentResponse, error)
	VerifyPayment(ctx context.Context, reference string, userID uuid.UUID) (*dto.PaymentResponse, error)
	HandleWebhook(ctx context.Context, provider types.PaymentProvider, payload []byte, headers map[string]string) error
}

type paymentService struct {
	repo           *repositories.PaymentRepository
	settlementRepo *repositories.SettlementRepository
	producer       events.Producer
	providers      map[types.PaymentProvider]providers.PaymentProvider
	cfg            config.Config
}

func NewPaymentService(producer events.Producer) PaymentService {
	cfg := config.LoadConfig()
	provMap := providers.InitProviders(cfg)
	repo := repositories.NewPaymentRepository(database.DB)
	settleRepo := repositories.NewSettlementRepository(database.DB)

	return &paymentService{
		repo:           repo,
		settlementRepo: settleRepo,
		producer:       producer,
		providers:      provMap,
		cfg:            cfg,
	}
}

func (s *paymentService) InitiatePayment(ctx context.Context, req *dto.InitiatePaymentRequest, userID uuid.UUID) (*dto.InitiatePaymentResponse, error) {
	userEmail, err := guards.GetUserEmail(ctx, userID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get user email: %w", err)
	}
	// 1. Get Order
	var order orderModels.Order
	if err := database.DB.NewSelect().Model(&order).Where("id = ?", req.OrderID).Scan(ctx); err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}

	if order.UserID != userID {
		return nil, errors.New("unauthorized: order does not belong to user")
	}

	provider, ok := s.providers[req.Provider]
	if !ok {
		return nil, fmt.Errorf("payment provider %s not configured", req.Provider)
	}

	reference := fmt.Sprintf("PAY-%s-%d", uuid.New().String()[:8], time.Now().Unix())
	
	payment := &models.Payment{
		OrderID:       order.ID,
		UserID:        userID,
		Amount:        order.TotalAmount,
		Currency:      "NGN",
		Provider:      req.Provider,
		Status:        types.PaymentStatusPending,
		Reference:     reference,
		CustomerEmail: userEmail,
	}

	if err := s.repo.Create(ctx, payment); err != nil {
		return nil, fmt.Errorf("failed to create payment record: %w", err)
	}

	initReq := &providers.InitializeRequest{
		Amount:      order.TotalAmount,
		Currency:    "NGN",
		Email:       userEmail,
		Reference:   reference,
		CallbackURL: req.CallbackURL,
		Metadata: map[string]interface{}{
			"payment_id": payment.ID.String(),
			"order_id":   order.ID.String(),
		},
	}

	initResp, err := provider.InitializePayment(ctx, initReq)
	if err != nil {
		return nil, fmt.Errorf("provider initialization failed: %w", err)
	}

	if initResp.Reference != "" && initResp.Reference != reference {
		payment.ExternalReference = initResp.Reference
		_ = s.repo.Update(ctx, payment, "external_reference")
	}
	
	resp := &dto.InitiatePaymentResponse{
		PaymentID:        payment.ID,
		AuthorizationURL: initResp.AuthorizationURL,
		AccessCode:       initResp.AccessCode,
		Reference:        reference,
	}

	if s.producer != nil {
		event := paymentEvents.NewPaymentEvent("payment_initiated", payment)
		_ = s.producer.Publish(ctx, event)
	}

	return resp, nil
}

func (s *paymentService) VerifyPayment(ctx context.Context, reference string, userID uuid.UUID) (*dto.PaymentResponse, error) {
	payment, err := s.repo.FindByReference(ctx, reference)
	if err != nil {
		return nil, err
	}

	if payment.UserID != userID {
		return nil, errors.New("unauthorized: payment does not belong to user")
	}

	if payment.Status == types.PaymentStatusSuccess {
		resp := dto.MapPaymentToResponse(payment)
		return &resp, nil
	}

	provider, ok := s.providers[payment.Provider]
	if !ok {
		return nil, errors.New("provider not found")
	}

	verifyResp, err := provider.VerifyPayment(ctx, reference)
	if err != nil {
		return nil, fmt.Errorf("verification failed: %w", err)
	}

	previousStatus := payment.Status
	if verifyResp.Success {
		payment.Status = types.PaymentStatusSuccess
		now := time.Now()
		payment.CompletedAt = &now
	} else if verifyResp.Status == "failed" {
		payment.Status = types.PaymentStatusFailed
	}

	if payment.Status != previousStatus {
		if err := s.repo.Update(ctx, payment, "status", "completed_at", "updated_at"); err != nil {
			return nil, err
		}

		if s.producer != nil {
			eventType := "payment_failed"
			if payment.Status == types.PaymentStatusSuccess {
				eventType = "payment_successful"
				
				// Create Settlement Record
				// 1. Get Order to find RestaurantID and Amount
				var order orderModels.Order
				if err := database.DB.NewSelect().Model(&order).Where("id = ?", payment.OrderID).Scan(ctx); err == nil {
					commissionRate := decimal.NewFromFloat(0.10) // 10% Commission
					platformFee := order.TotalAmount.Mul(commissionRate)
					restaurantShare := order.TotalAmount.Sub(platformFee)

					settlement := &models.Settlement{
						OrderID:         order.ID,
						RestaurantID:    order.RestaurantID,
						TotalAmount:     order.TotalAmount,
						PlatformFee:     platformFee,
						RestaurantShare: restaurantShare,
						Status:          models.SettlementStatusPending,
					}
					
					if err := s.settlementRepo.Create(ctx, settlement); err != nil {
						fmt.Printf("failed to create settlement: %v\n", err)
					}
				}
			}
			event := paymentEvents.NewPaymentEvent(eventType, payment)
			_ = s.producer.Publish(ctx, event)
		}
	}

	resp := dto.MapPaymentToResponse(payment)
	return &resp, nil
}

func (s *paymentService) HandleWebhook(ctx context.Context, providerName types.PaymentProvider, payload []byte, headers map[string]string) error {
	provider, ok := s.providers[providerName]
	if !ok {
		return errors.New("provider not found")
	}

	isValid, err := provider.ValidateWebhook(ctx, payload, headers)
	if err != nil || !isValid {
		return errors.New("invalid webhook signature")
	}

	// This is a simplified webhook handler. 
	// In a real app, you would parse the payload based on the provider, 
	// find the internal reference, and call VerifyPayment or update status directly.
	// Since Paystack/Monnify/etc have different formats, we'd need a sub-factory or parser here.
	
	return nil // To be fully implemented with specific provider parsers
}

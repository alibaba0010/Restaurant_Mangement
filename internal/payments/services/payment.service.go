package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/alibaba0010/postgres-api/internal/common/events"
	"github.com/alibaba0010/postgres-api/internal/common/guards"
	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/alibaba0010/postgres-api/internal/config"
	"github.com/alibaba0010/postgres-api/internal/database"
	orderRepo "github.com/alibaba0010/postgres-api/internal/orders/repositories"
	"github.com/alibaba0010/postgres-api/internal/payments/dto"
	paymentEvents "github.com/alibaba0010/postgres-api/internal/payments/events"
	"github.com/alibaba0010/postgres-api/internal/payments/models"
	"github.com/alibaba0010/postgres-api/internal/payments/providers"
	"github.com/alibaba0010/postgres-api/internal/payments/repositories"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
	"go.uber.org/zap"

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
	orderRepo      *orderRepo.OrderRepository
	producer       events.Producer
	providers      map[types.PaymentProvider]providers.PaymentProvider
	cfg            config.Config
}

func NewPaymentService(producer events.Producer, orderRepo *orderRepo.OrderRepository) PaymentService {
	cfg := config.LoadConfig()
	provMap := providers.InitProviders(cfg)
	repo := repositories.NewPaymentRepository(database.DB)
	settleRepo := repositories.NewSettlementRepository(database.DB)

	return &paymentService{
		repo:           repo,
		settlementRepo: settleRepo,
		orderRepo:      orderRepo,
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
	// 1. Get Order (Issue 1.5: use Repository)
	order, err := s.orderRepo.FindByID(ctx, req.OrderID.String())
	if err != nil {
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

	return s.verifyPaymentInternal(ctx, payment)
}

func (s *paymentService) verifyPaymentInternal(ctx context.Context, payment *models.Payment) (*dto.PaymentResponse, error) {
	if payment.Status == types.PaymentStatusSuccess {
		resp := dto.MapPaymentToResponse(payment)
		return &resp, nil
	}

	provider, ok := s.providers[payment.Provider]
	if !ok {
		return nil, errors.New("provider not found")
	}

	verifyResp, err := provider.VerifyPayment(ctx, payment.Reference)
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
		// Use a transaction for atomic update of payment and creation of settlement
		err := database.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			// 1. Update Payment Status (using transaction)
			repoTx := repositories.NewPaymentRepository(tx)
			if err := repoTx.Update(ctx, payment, "status", "completed_at", "updated_at"); err != nil {
				return err
			}

			if payment.Status == types.PaymentStatusSuccess {
				// 2. Get Order and Create Settlement
				order, err := s.orderRepo.FindByID(ctx, payment.OrderID.String())
				if err != nil {
					return fmt.Errorf("order not found: %w", err)
				}

				// Commission rate from config
				commissionRateRaw := s.cfg.PlatformCommissionRate
				if commissionRateRaw <= 0 {
					commissionRateRaw = 0.10 // Default 10%
				}
				commissionRate := decimal.NewFromFloat(commissionRateRaw)
				
				platformFee := order.TotalAmount.Mul(commissionRate).Round(2)
				restaurantShare := order.TotalAmount.Sub(platformFee)

				settlement := &models.Settlement{
					OrderID:         order.ID,
					RestaurantID:    order.RestaurantID,
					TotalAmount:     order.TotalAmount,
					PlatformFee:     platformFee,
					RestaurantShare: restaurantShare,
					Status:          models.SettlementStatusPending,
				}
				
				settleRepoTx := repositories.NewSettlementRepository(tx)
				if err := settleRepoTx.Create(ctx, settlement); err != nil {
					return fmt.Errorf("failed to create settlement: %w", err)
				}
			}
			return nil
		})

		if err != nil {
			logger.Log.Error("payment verification persistence failed", zap.Error(err))
			return nil, err
		}

		// Publish Event
		if s.producer != nil {
			eventType := "payment_failed"
			if payment.Status == types.PaymentStatusSuccess {
				eventType = "payment_successful"
			}
			event := paymentEvents.NewPaymentEvent(eventType, payment)
			if pubErr := s.producer.Publish(ctx, event); pubErr != nil {
				logger.Log.Error("failed to publish payment event", zap.Error(pubErr))
			}
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

	// 1. Validate Signature (Issue 2.3)
	isValid, err := provider.ValidateWebhook(ctx, payload, headers)
	if err != nil || !isValid {
		return errors.New("invalid webhook signature")
	}

	// 2. Parse basic event data to handle deduplication
	var eventData struct {
		Event string `json:"event"`
		Data  struct {
			ID        interface{} `json:"id"`
			Reference string      `json:"reference"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &eventData); err != nil {
		return fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	eventID := fmt.Sprintf("%s_%v", providerName, eventData.Data.ID)
	
	// 3. Deduplication (Issue 2.3)
	existing, _ := s.repo.FindWebhookByEventID(ctx, eventID)
	if existing != nil && existing.Processed {
		logger.Log.Info("Webhook already processed", zap.String("event_id", eventID))
		return nil
	}

	// Log arrival
	if existing == nil {
		var fullPayload map[string]interface{}
		_ = json.Unmarshal(payload, &fullPayload)

		_ = s.repo.LogWebhook(ctx, &models.PaymentWebhookLog{
			Provider:        providerName,
			ProviderEventID: eventID,
			EventType:       eventData.Event,
			Payload:         fullPayload,
			Processed:       false,
		})
	}

	// 4. Handle Specific Events
	if eventData.Event == "charge.success" || eventData.Event == "successful" || eventNameMatch(eventData.Event, "success") {
		ref := eventData.Data.Reference
		if ref == "" {
			return errors.New("reference not found in webhook data")
		}

		payment, err := s.repo.FindByReference(ctx, ref)
		if err != nil {
			// Try external reference as fallback
			payment, err = s.repo.FindByExternalReference(ctx, ref)
			if err != nil {
				return fmt.Errorf("payment not found for reference %s", ref)
			}
		}

		_, err = s.verifyPaymentInternal(ctx, payment)
		if err != nil {
			return err
		}
	}

	// 5. Mark as processed
	return s.repo.MarkWebhookProcessed(ctx, eventID)
}

func eventNameMatch(event, target string) bool {
	return strings.Contains(strings.ToLower(event), target)
}

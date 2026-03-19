package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alibaba0010/postgres-api/internal/common/events"
	"github.com/alibaba0010/postgres-api/internal/common/guards"
	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/alibaba0010/postgres-api/internal/config"
	"github.com/alibaba0010/postgres-api/internal/database"
	orderModels "github.com/alibaba0010/postgres-api/internal/orders/models"
	orderRepo "github.com/alibaba0010/postgres-api/internal/orders/repositories"
	"github.com/alibaba0010/postgres-api/internal/payments/dto"
	paymentEvents "github.com/alibaba0010/postgres-api/internal/payments/events"
	"github.com/alibaba0010/postgres-api/internal/payments/models"
	"github.com/alibaba0010/postgres-api/internal/payments/providers"
	"github.com/alibaba0010/postgres-api/internal/payments/repositories"
	restRepo "github.com/alibaba0010/postgres-api/internal/restaurants/repositories"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
	"go.uber.org/zap"

	apierrors "github.com/alibaba0010/postgres-api/internal/common/errors"
)

type PaymentService interface {
	InitiatePayment(ctx context.Context, req *dto.InitiatePaymentRequest, userID uuid.UUID) (*dto.InitiatePaymentResponse, *apierrors.AppError)
	VerifyPayment(ctx context.Context, reference string, userID uuid.UUID) (*dto.PaymentResponse, *apierrors.AppError)
	HandleWebhook(ctx context.Context, provider types.PaymentProvider, payload []byte, headers map[string]string) *apierrors.AppError
}

type paymentService struct {
	repo           *repositories.PaymentRepository
	settlementRepo *repositories.SettlementRepository
	orderRepo      *orderRepo.OrderRepository
	menuRepo       *restRepo.MenuRepository
	producer       events.Producer
	providers      map[types.PaymentProvider]providers.PaymentProvider
	cfg            *config.Config
}

func NewPaymentService(producer events.Producer, orderRepo *orderRepo.OrderRepository, menuRepo *restRepo.MenuRepository) PaymentService {
	cfg := config.LoadConfig()
	provMap := providers.InitProviders(*cfg)
	repo := repositories.NewPaymentRepository(database.DB)
	settleRepo := repositories.NewSettlementRepository(database.DB)

	return &paymentService{
		repo:           repo,
		settlementRepo: settleRepo,
		orderRepo:      orderRepo,
		menuRepo:       menuRepo,
		producer:       producer,
		providers:      provMap,
		cfg:            cfg,
	}
}

func (s *paymentService) InitiatePayment(ctx context.Context, req *dto.InitiatePaymentRequest, userID uuid.UUID) (*dto.InitiatePaymentResponse, *apierrors.AppError) {
	userEmail, err := guards.GetUserEmail(ctx, userID.String())
	if err != nil {
		return nil, apierrors.InternalError(fmt.Errorf("failed to get user email: %w", err))
	}
	// 1. Get Order (Issue 1.5: use Repository)
	order, err := s.orderRepo.FindByID(ctx, req.OrderID.String())
	if err != nil {
		return nil, apierrors.NotFoundError("order not found")
	}

	if order.UserID != userID {
		return nil, apierrors.ForbiddenError("order does not belong to you")
	}
	provider, ok := s.providers[req.Provider]
	if !ok {
		return nil, apierrors.ValidationError(fmt.Sprintf("payment provider %s not configured", req.Provider))
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
		return nil, apierrors.InternalError(fmt.Errorf("failed to create payment record: %w", err))
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
		return nil, apierrors.InternalError(fmt.Errorf("provider initialization failed: %w", err))
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

func (s *paymentService) VerifyPayment(ctx context.Context, reference string, userID uuid.UUID) (*dto.PaymentResponse, *apierrors.AppError) {
	payment, err := s.repo.FindByReference(ctx, reference)
	if err != nil {
		return nil, apierrors.NotFoundError("payment record not found")
	}

	if payment.UserID != userID {
		return nil, apierrors.ForbiddenError("payment does not belong to you")
	}

	return s.verifyPaymentInternal(ctx, payment)
}

func (s *paymentService) verifyPaymentInternal(ctx context.Context, payment *models.Payment) (*dto.PaymentResponse, *apierrors.AppError) {
	if payment.Status == types.PaymentStatusSuccess {
		resp := dto.MapPaymentToResponse(payment)
		return &resp, nil
	}

	provider, ok := s.providers[payment.Provider]
	if !ok {
		return nil, apierrors.ValidationError("payment provider not found or disabled")
	}

	verifyResp, err := provider.VerifyPayment(ctx, payment.Reference)
	if err != nil {
		return nil, apierrors.InternalError(fmt.Errorf("verification failed: %w", err))
	}

	previousStatus := payment.Status
	if verifyResp.Success {
		// Verify amount matches exactly to prevent partial payment or client-side manipulation limits
		if verifyResp.Amount.GreaterThan(decimal.Zero) && !verifyResp.Amount.Equal(payment.Amount) {
			logger.Log.Warn("Payment amount mismatch detected", 
				zap.String("reference", payment.Reference), 
				zap.String("expected", payment.Amount.String()), 
				zap.String("actual", verifyResp.Amount.String()))
			payment.Status = types.PaymentStatusFailed
		} else {
			payment.Status = types.PaymentStatusSuccess
			now := time.Now()
			payment.CompletedAt = &now
		}
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
				// 2. Update Order (Mark as Paid and Confirmed)
				_, err = tx.NewUpdate().
					Model((*orderModels.Order)(nil)).
					Set("payment_status = ?", types.PaymentStatusSuccess).
					Set("payment_method = ?", string(payment.Provider)).
					Set("payment_reference = ?", payment.Reference).
					Set("paid_at = ?", payment.CompletedAt).
					Set("status = ?", types.OrderStatusConfirmed).
					Set("confirmed_at = NOW()").
					Set("updated_at = NOW()").
					Where("id = ?", payment.OrderID).
					Exec(ctx)
				if err != nil {
					return fmt.Errorf("failed to update order: %w", err)
				}

				// 3. Get Order and Create Settlement
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
			} else if payment.Status == types.PaymentStatusFailed {
				// 3. Restore Stock on Failure
				order, err := s.orderRepo.FindByID(ctx, payment.OrderID.String())
				if err != nil {
					return fmt.Errorf("order not found for stock restoration: %w", err)
				}

				stockChanges := make(map[uuid.UUID]int)
				for _, item := range order.OrderItems {
					stockChanges[item.MenuID] = item.Quantity // Positive restores stock
				}
				
				if err := s.menuRepo.BatchUpdateStock(ctx, tx, stockChanges); err != nil {
					return fmt.Errorf("failed to restore stock on payment failure: %w", err)
				}
			}
			return nil
		})

		if err != nil {
			logger.Log.Error("payment verification persistence failed", zap.Error(err))
			return nil, apierrors.InternalError(err)
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

func (s *paymentService) HandleWebhook(ctx context.Context, providerName types.PaymentProvider, payload []byte, headers map[string]string) *apierrors.AppError {
	// Dev warning for localhost
	if s.cfg.APP_ENV == "development" {
		logger.Log.Warn("[Dev] Webhook received in development mode",
			zap.String("provider", string(providerName)),
			zap.String("tip", "Use ngrok or smee.io to expose localhost for webhook testing"),
		)
	}

	provider, ok := s.providers[providerName]
	if !ok {
		return apierrors.ValidationError("provider not found")
	}

	// 1. Validate Signature
	isValid, err := provider.ValidateWebhook(ctx, payload, headers)
	if err != nil || !isValid {
		return apierrors.ForbiddenError("invalid webhook signature")
	}

	// 2. Parse Provider Webhook (DRY: each provider knows its own format)
	event, err := provider.ParseWebhook(payload)
	if err != nil {
		return apierrors.ValidationError(fmt.Sprintf("failed to parse webhook payload: %v", err))
	}

	// 3. Deduplication (using provider-specific reference and event hash)
	eventID := fmt.Sprintf("%s_%s_%s", providerName, event.Status, event.ExternalReference)
	
	existing, _ := s.repo.FindWebhookByEventID(ctx, eventID)
	if existing != nil && existing.Processed {
		logger.Log.Info("Webhook already processed", zap.String("event_id", eventID))
		return nil
	}

	// Log arrival if not exist
	if existing == nil {
		var fullPayload map[string]interface{}
		_ = json.Unmarshal(payload, &fullPayload)

		_ = s.repo.LogWebhook(ctx, &models.PaymentWebhookLog{
			Provider:        providerName,
			ProviderEventID: eventID,
			EventType:       event.Status,
			Payload:         fullPayload,
			Processed:       false,
		})
	}

	// 4. Process Payment Update (only if successful)
	if event.Success {
		ref := event.Reference
		if ref == "" {
			return apierrors.ValidationError("reference not found in webhook data")
		}

		payment, err := s.repo.FindByReference(ctx, ref)
		if err != nil {
			// Try external reference as fallback
			payment, err = s.repo.FindByExternalReference(ctx, event.ExternalReference)
			if err != nil {
				return apierrors.NotFoundError(fmt.Sprintf("payment not found for reference %s", ref))
			}
		}

		_, appErr := s.verifyPaymentInternal(ctx, payment)
		if appErr != nil {
			return appErr
		}
	}

	// 5. Mark as processed
	if err := s.repo.MarkWebhookProcessed(ctx, eventID); err != nil {
		return apierrors.InternalError(err)
	}
	return nil
}


package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/alibaba0010/postgres-api/internal/payments/models"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// PaymentRepository handles database operations for payments
type PaymentRepository struct {
	db bun.IDB
}

// NewPaymentRepository creates a new payment repository
func NewPaymentRepository(db bun.IDB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

// Create inserts a new payment record
func (r *PaymentRepository) Create(ctx context.Context, payment *models.Payment) error {
	if payment == nil {
		return fmt.Errorf("payment cannot be nil")
	}

	_, err := r.db.NewInsert().
		Model(payment).
		Exec(ctx)
	return err
}

// FindByID retrieves a payment by ID
func (r *PaymentRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Payment, error) {
	payment := &models.Payment{}
	err := r.db.NewSelect().
		Model(payment).
		Where("id = ?", id).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("payment not found")
	}
	if err != nil {
		return nil, err
	}

	return payment, nil
}

// FindByReference retrieves a payment by our internal reference
func (r *PaymentRepository) FindByReference(ctx context.Context, reference string) (*models.Payment, error) {
	if reference == "" {
		return nil, fmt.Errorf("reference is required")
	}

	payment := &models.Payment{}
	err := r.db.NewSelect().
		Model(payment).
		Where("reference = ?", reference).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("payment not found")
	}
	if err != nil {
		return nil, err
	}

	return payment, nil
}

// FindByExternalReference retrieves a payment by provider reference
func (r *PaymentRepository) FindByExternalReference(ctx context.Context, externalRef string) (*models.Payment, error) {
	if externalRef == "" {
		return nil, fmt.Errorf("external reference is required")
	}

	payment := &models.Payment{}
	err := r.db.NewSelect().
		Model(payment).
		Where("external_reference = ?", externalRef).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("payment not found")
	}
	if err != nil {
		return nil, err
	}

	return payment, nil
}

// FindByUserID retrieves all payments for a user with pagination
func (r *PaymentRepository) FindByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.Payment, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	payments := []models.Payment{}
	count, err := r.db.NewSelect().
		Model(&payments).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		ScanAndCount(ctx)

	if err != nil {
		return nil, 0, err
	}

	return payments, int64(count), nil
}

// Update updates an existing payment
func (r *PaymentRepository) Update(ctx context.Context, payment *models.Payment, columns ...string) error {
	payment.UpdatedAt = time.Now()
	q := r.db.NewUpdate().Model(payment).WherePK()
	if len(columns) > 0 {
		q = q.Column(columns...)
	}
	_, err := q.Exec(ctx)
	return err
}

// CreateRefund inserts a new refund record
func (r *PaymentRepository) CreateRefund(ctx context.Context, refund *models.PaymentRefund) error {
	if refund == nil {
		return fmt.Errorf("refund cannot be nil")
	}

	_, err := r.db.NewInsert().
		Model(refund).
		Exec(ctx)
	return err
}

// LogWebhook logs a webhook event for audit and deduplication
func (r *PaymentRepository) LogWebhook(ctx context.Context, log *models.PaymentWebhookLog) error {
	if log == nil {
		return fmt.Errorf("webhook log cannot be nil")
	}

	_, err := r.db.NewInsert().
		Model(log).
		On("CONFLICT (provider_event_id) DO UPDATE").
		Set("processed = excluded.processed, updated_at = NOW()").
		Exec(ctx)
	return err
}

// FindWebhookByEventID checks if a webhook has been processed
func (r *PaymentRepository) FindWebhookByEventID(ctx context.Context, eventID string) (*models.PaymentWebhookLog, error) {
	if eventID == "" {
		return nil, fmt.Errorf("event id is required")
	}

	log := &models.PaymentWebhookLog{}
	err := r.db.NewSelect().
		Model(log).
		Where("provider_event_id = ?", eventID).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return log, nil
}

// MarkWebhookProcessed marks a webhook as processed
func (r *PaymentRepository) MarkWebhookProcessed(ctx context.Context, eventID string) error {
	if eventID == "" {
		return fmt.Errorf("event id is required")
	}

	_, err := r.db.NewUpdate().
		Model(&models.PaymentWebhookLog{}).
		Set("processed = ?, processed_at = NOW()", true).
		Where("provider_event_id = ?", eventID).
		Exec(ctx)
	return err
}

// BeginTx starts a new transaction
func (r *PaymentRepository) BeginTx(ctx context.Context) (bun.Tx, error) {
    if db, ok := r.db.(*bun.DB); ok {
        return db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
    }
	return bun.Tx{}, fmt.Errorf("underlying db is not *bun.DB")
}

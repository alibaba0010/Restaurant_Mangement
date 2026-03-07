package repositories

import (
	"context"

	"github.com/alibaba0010/postgres-api/internal/payments/models"
	"github.com/uptrace/bun"
)

type SettlementRepository struct {
	db *bun.DB
}

func NewSettlementRepository(db *bun.DB) *SettlementRepository {
	return &SettlementRepository{db: db}
}

func (r *SettlementRepository) Create(ctx context.Context, settlement *models.Settlement) error {
	_, err := r.db.NewInsert().Model(settlement).Exec(ctx)
	return err
}

func (r *SettlementRepository) FindByOrderID(ctx context.Context, orderID string) (*models.Settlement, error) {
	var settlement models.Settlement
	err := r.db.NewSelect().Model(&settlement).Where("order_id = ?", orderID).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &settlement, nil
}

package migration

import (
	"context"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		_, err := db.ExecContext(ctx, `
			CREATE TABLE IF NOT EXISTS settlements (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				order_id UUID NOT NULL,
				restaurant_id UUID NOT NULL,
				total_amount DECIMAL(12,2) NOT NULL,
				platform_fee DECIMAL(12,2) NOT NULL,
				restaurant_share DECIMAL(12,2) NOT NULL,
				status TEXT NOT NULL DEFAULT 'pending',
				payout_reference TEXT,
				processed_at TIMESTAMPTZ,
				created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
			)
		`)
		return err
	}, func(ctx context.Context, db *bun.DB) error {
		_, err := db.ExecContext(ctx, "DROP TABLE IF EXISTS settlements")
		return err
	})
}

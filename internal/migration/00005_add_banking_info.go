package migration

import (
	"context"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		if _, err := db.ExecContext(ctx, "ALTER TABLE restaurants ADD COLUMN IF NOT EXISTS account_number TEXT DEFAULT NULL"); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, "ALTER TABLE restaurants ADD COLUMN IF NOT EXISTS bank_name TEXT DEFAULT NULL"); err != nil {
			return err
		}
		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		if _, err := db.ExecContext(ctx, "ALTER TABLE restaurants DROP COLUMN IF EXISTS account_number"); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, "ALTER TABLE restaurants DROP COLUMN IF EXISTS bank_name"); err != nil {
			return err
		}
		return nil
	})
}

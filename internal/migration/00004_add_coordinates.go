package migration

import (
	"context"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// Add latitude and longitude columns to restaurants
		// Using raw SQL is safer for simple ADD COLUMN operations to avoid model binding issues during migration
		if _, err := db.ExecContext(ctx, "ALTER TABLE restaurants ADD COLUMN IF NOT EXISTS latitude FLOAT8 DEFAULT NULL"); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, "ALTER TABLE restaurants ADD COLUMN IF NOT EXISTS longitude FLOAT8 DEFAULT NULL"); err != nil {
			return err
		}

		// Add latitude and longitude columns to users
		if _, err := db.ExecContext(ctx, "ALTER TABLE users ADD COLUMN IF NOT EXISTS latitude FLOAT8 DEFAULT NULL"); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, "ALTER TABLE users ADD COLUMN IF NOT EXISTS longitude FLOAT8 DEFAULT NULL"); err != nil {
			return err
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// Rollback logic
		if _, err := db.ExecContext(ctx, "ALTER TABLE restaurants DROP COLUMN IF EXISTS latitude"); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, "ALTER TABLE restaurants DROP COLUMN IF EXISTS longitude"); err != nil {
			return err
		}

		if _, err := db.ExecContext(ctx, "ALTER TABLE users DROP COLUMN IF EXISTS latitude"); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, "ALTER TABLE users DROP COLUMN IF EXISTS longitude"); err != nil {
			return err
		}

		return nil
	})
}

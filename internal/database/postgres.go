package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/config"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/extra/bundebug"
	"go.uber.org/zap"
)

var (
	Pool *sql.DB
	DB   *bun.DB
)

func ConnectDB() *bun.DB {
	cfg := config.LoadConfig()
	connectionURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", 
		cfg.DB_USERNAME, cfg.DB_PASSWORD, cfg.DB_HOST, cfg.DB_PORT, cfg.DB_NAME)

	config, err := pgx.ParseConfig(connectionURL)
	if err != nil {
		logger.Log.Fatal("Unable to parse database config", zap.Error(err))
	}

	Pool = stdlib.OpenDB(*config)
	Pool.SetMaxIdleConns(25)
	Pool.SetMaxOpenConns(25)
	Pool.SetConnMaxLifetime(5 * time.Minute)

	// Create a Bun db instance
	DB = bun.NewDB(Pool, pgdialect.New())

	// Add query debug hook in development
	DB.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithVerbose(true),
		bundebug.FromEnv(),
	))

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := DB.PingContext(ctx); err != nil {
		logger.Log.Fatal("Database ping failed", zap.Error(err))
	}

	logger.Log.Info("✅ Connected to PostgreSQL database")
	return DB
}

// Close connection when shutting down
func CloseDB() {
	if Pool != nil {
		Pool.Close()
		logger.Log.Info("🔌 Database connection closed")
	}
}
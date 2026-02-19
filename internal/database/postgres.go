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

	// Parse configuration
	pgConfig, err := pgx.ParseConfig(connectionURL)
	if err != nil {
		logger.Log.Fatal("Unable to parse database config", zap.Error(err))
	}

	// For production: tune connection pool settings
	// 1000+ users might need more than 25 connections depending on request patterns,
	// but 25-50 is usually a good starting point for a single instance.
	Pool = stdlib.OpenDB(*pgConfig)
	Pool.SetMaxIdleConns(10)
	Pool.SetMaxOpenConns(50)
	Pool.SetConnMaxLifetime(10 * time.Minute)
	Pool.SetConnMaxIdleTime(5 * time.Minute)

	// Create a Bun db instance
	DB = bun.NewDB(Pool, pgdialect.New())

	// Add query debug hook in development (based on env)
	if cfg.APP_ENV != "production" {
		DB.AddQueryHook(bundebug.NewQueryHook(
			bundebug.WithVerbose(true),
			bundebug.FromEnv(),
		))
	}

	// Test connection with retries
	ctx := context.Background()
	var (
		pingErr error
		maxRetries = 5
	)

	for i := 0; i < maxRetries; i++ {
		pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		if pingErr = DB.PingContext(pingCtx); pingErr == nil {
			cancel()
			break
		}
		cancel()
		logger.Log.Warn("Database ping failed, retrying...", 
			zap.Int("attempt", i+1), 
			zap.Int("max_retries", maxRetries), 
			zap.Error(pingErr))
		time.Sleep(time.Duration(i+1) * 2 * time.Second)
	}

	if pingErr != nil {
		logger.Log.Fatal("Database connection failed after retries", zap.Error(pingErr))
	}

	logger.Log.Info("✅ Connected to PostgreSQL database")
	return DB
}

// CheckHealth returns an error if the database is unreachable
func CheckHealth(ctx context.Context) error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}
	return DB.PingContext(ctx)
}

// Close connection when shutting down
func CloseDB() {
	if Pool != nil {
		if err := Pool.Close(); err != nil {
			logger.Log.Error("Error closing database connection", zap.Error(err))
		} else {
			logger.Log.Info("🔌 Database connection closed")
		}
	}
}

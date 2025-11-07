package database

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/alibaba0010/postgres-api/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)
var Pool *pgxpool.Pool
func ConnectDB(){
	host:= os.Getenv("DB_HOST")
	port:= os.Getenv("DB_PORT")
	user:= os.Getenv("DB_USERNAME")
	password:= os.Getenv("DB_PASSWORD")
	dbname:= os.Getenv("DB_NAME")

	connectionURL:= fmt.Sprintf("postgres://%s:%s@%s:%s/%s", user, password, host, port, dbname)
	context, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
	defer cancel()
	var err error
	Pool, err = pgxpool.New(context, connectionURL)
	if err != nil {
		logger.Log.Fatal("Unable to connect to database", zap.Error(err))
	}
	// db := bun.NewDB(Pool, pgdialect.New())

	
		// Test connection
	err = Pool.Ping(context)
	if err != nil {
		logger.Log.Fatal("Database ping failed", zap.Error(err))
	}

	logger.Log.Info("✅ Connected to PostgreSQL database")

}

// Close connection when shutting down
func CloseDB() {
	if Pool != nil {
		Pool.Close()
		logger.Log.Info("🔌 Database connection closed")
	}
}
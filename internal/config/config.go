package config

import (
	"os"
	"sync"

	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

var (
	cfg  Config
	once sync.Once
)

type Config struct {
	Port                    string
	DB_HOST                 string
	DB_PORT                 string
	DB_USERNAME             string
	DB_PASSWORD             string
	DB_NAME                 string
	REDIS_HOST              string
	REDIS_PORT              string
	REDIS_PASSWORD          string
	EMAIL_PORT              string
	EMAIL_HOST              string
	EMAIL_USER              string
	EMAIL_PASSWORD          string
	FRONTEND_URL            string
	ACCESS_TOKEN_SECRET     string
	REFRESH_TOKEN_SECRET    string
	PRODUCTION_FRONTEND_URL string
	GOOGLE_CLIENT_ID        string
	GOOGLE_CLIENT_SECRET    string
	GOOGLE_REDIRECT_URI     string
	FACEBOOK_CLIENT_ID      string
	FACEBOOK_CLIENT_SECRET  string
	APPLE_CLIENT_ID         string
	APPLE_CLIENT_SECRET     string
	AWS_ACCESS_KEY_ID       string
	AWS_SECRET_ACCESS_KEY   string
	AWS_REGION              string
	AWS_BUCKET_NAME         string
	AWS_CLOUDFRONT_DOMAIN   string
	REDPANDA_BROKERS        string
}

func LoadConfig() Config {
	once.Do(func() {
		err := godotenv.Load()
		if err != nil {
			logger.Log.Warn("No .env file found, using system environment variables", zap.Error(err))
		}
		cfg = Config{
			Port:                    getEnv("PORT", "2000"),
			DB_HOST:                 getEnv("DB_HOST", "localhost"),
			DB_PORT:                 "5432",
			DB_USERNAME:             getEnv("DB_USERNAME", "postgres"),
			DB_PASSWORD:             getEnv("DB_PASSWORD", "password"),
			DB_NAME:                 getEnv("DB_NAME", "postgres"),
			REDIS_HOST:              getEnv("REDIS_HOST", "localhost"),
			REDIS_PORT:              getEnv("REDIS_PORT", "6379"),
			REDIS_PASSWORD:          getEnv("REDIS_PASSWORD", ""),
			EMAIL_PORT:              getEnv("EMAIL_PORT", "587"),
			EMAIL_HOST:              getEnv("EMAIL_HOST", "smtp.gmail.com"),
			EMAIL_USER:              getEnv("EMAIL_USER", ""),
			EMAIL_PASSWORD:          getEnv("EMAIL_PASSWORD", ""),
			FRONTEND_URL:            getEnv("FRONTEND_URL", "http://localhost:3000"),
			ACCESS_TOKEN_SECRET:     getEnv("ACCESS_TOKEN_SECRET", "default_access_secret"),
			REFRESH_TOKEN_SECRET:    getEnv("REFRESH_TOKEN_SECRET", "default_refresh_secret"),
			PRODUCTION_FRONTEND_URL: getEnv("PRODUCTION_FRONTEND_URL", "http://localhost:3000"),
			GOOGLE_CLIENT_ID:        getEnv("GOOGLE_CLIENT_ID", ""),
			GOOGLE_CLIENT_SECRET:    getEnv("GOOGLE_CLIENT_SECRET", ""),
			GOOGLE_REDIRECT_URI:     getEnv("GOOGLE_REDIRECT_URI", "http://localhost:3000/auth/callback"),
			FACEBOOK_CLIENT_ID:      getEnv("FACEBOOK_CLIENT_ID", ""),
			FACEBOOK_CLIENT_SECRET:  getEnv("FACEBOOK_CLIENT_SECRET", ""),
			APPLE_CLIENT_ID:         getEnv("APPLE_CLIENT_ID", ""),
			APPLE_CLIENT_SECRET:     getEnv("APPLE_CLIENT_SECRET", ""),
			AWS_ACCESS_KEY_ID:       getEnv("AWS_ACCESS_KEY_ID", ""),
			AWS_SECRET_ACCESS_KEY:   getEnv("AWS_SECRET_ACCESS_KEY", ""),
			AWS_REGION:              getEnv("AWS_REGION", "us-east-1"),
			AWS_BUCKET_NAME:         getEnv("AWS_BUCKET_NAME", ""),
			AWS_CLOUDFRONT_DOMAIN:   getEnv("AWS_CLOUDFRONT_DOMAIN", ""),
			REDPANDA_BROKERS:        getEnv("REDPANDA_BROKERS", "localhost:9092"),
		}
	})
	return cfg
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}

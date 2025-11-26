package config

import (
	"os"

	"github.com/alibaba0010/postgres-api/internal/logger"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)


type Config struct {
	Port string
	DB_HOST string
	DB_PORT string
	DB_USERNAME string
	DB_PASSWORD string
	DB_NAME string
	REDIS_HOST string
	REDIS_PORT string
	REDIS_PASSWORD string
	EMAIL_PORT string
	EMAIL_HOST string
	EMAIL_USER string
	EMAIL_PASSWORD string
	FRONTEND_URL string
	ACCESS_TOKEN_SECRET string
	REFRESH_TOKEN_SECRET string
}

func LoadConfig() Config {
		err := godotenv.Load()
	if err != nil {
		// log.Println("No .env file found, using system environment variables...")
		logger.Log.Warn("No .env file found", zap.Error(err))
	}
	return Config{
		Port: getEnv("PORT", "2000"),
		DB_HOST:     getEnv("DB_HOST", "localhost"),
		DB_PORT:     "5432",
		DB_USERNAME: getEnv("DB_USERNAME", "postgres"),
		DB_PASSWORD: getEnv("DB_PASSWORD", "password"),
		DB_NAME:     getEnv("DB_NAME", "postgres"),
		REDIS_HOST:     getEnv("REDIS_HOST", "localhost"),
		REDIS_PORT:     getEnv("REDIS_PORT", "6379"),
		REDIS_PASSWORD: getEnv("REDIS_PASSWORD", ""),
		EMAIL_PORT:     getEnv("EMAIL_PORT", "587"),
		EMAIL_HOST:     getEnv("EMAIL_HOST", "smtp.gmail.com"),
		EMAIL_USER:     getEnv("EMAIL_USER", ""),
		EMAIL_PASSWORD: getEnv("EMAIL_PASSWORD", ""),
		FRONTEND_URL:  getEnv("FRONTEND_URL", "http://localhost:3000"),
		ACCESS_TOKEN_SECRET: getEnv("ACCESS_TOKEN_SECRET", "default_access_secret"),
		REFRESH_TOKEN_SECRET: getEnv("REFRESH_TOKEN_SECRET", "default_refresh_secret"),
	}
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

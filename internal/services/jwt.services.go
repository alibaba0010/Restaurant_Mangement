package services

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/alibaba0010/postgres-api/internal/config"
	"github.com/alibaba0010/postgres-api/internal/database"
	"github.com/alibaba0010/postgres-api/internal/errors"
	"github.com/alibaba0010/postgres-api/internal/logger"
	"github.com/alibaba0010/postgres-api/internal/models"
	"github.com/alibaba0010/postgres-api/internal/utils"
	"go.uber.org/zap"
)
const (
	AccessTokenDuration  = 15 * time.Minute
	RefreshTokenDuration = 7 * 24 * time.Hour // 7 days
)



type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
}
// AccessClaims are the JWT claims stored in access tokens.
type AccessTokenClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type RefreshTokenClaims struct {
	UserID    string    `json:"user_id"`
	Role   	  string `json:"role"`
	Token     string    `json:"token"`
	IPAddress string    `json:"ip_address,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
	jwt.RegisteredClaims
}
type RefreshTokenStorage interface {
	StoreRefreshToken(ctx context.Context, token string, data RefreshTokenClaims) error
	GetRefreshToken(ctx context.Context, token string) (*RefreshTokenClaims, error)
	DeleteRefreshToken(ctx context.Context, token string) error
	DeleteUserRefreshTokens(ctx context.Context, userID string) error
}

func GenerateTokenPair(ctx context.Context, userID, role, ip, userAgent string) (*TokenPair, *errors.AppError) {
	cfg := config.LoadConfig()

	now := time.Now()

	// Access token
	accessClaims := &AccessTokenClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   userID,
		},
	}

	accessTok := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessStr, err := accessTok.SignedString([]byte(cfg.ACCESS_TOKEN_SECRET))
	if err != nil {
		logger.Log.Error("failed to sign access token", zap.Error(err))
		return nil, errors.InternalError(err)
	}

	// Refresh token
	refreshClaims := &RefreshTokenClaims{
		UserID: userID,
		Role:   role,
		IPAddress:     ip,
		UserAgent:     userAgent,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(RefreshTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   userID,
		},
	}

	refreshTok := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshStr, err := refreshTok.SignedString([]byte(cfg.REFRESH_TOKEN_SECRET))
	if err != nil {
		logger.Log.Error("failed to sign refresh token", zap.Error(err))
		return nil, errors.InternalError(err)
	}

	// Persist refresh token in DB
	newUUID, err := utils.GenerateUUIDv7()
	if err != nil {
		logger.Log.Error("failed to generate UUID for refresh token", zap.Error(err))
		return nil, errors.InternalError(err)
	}

	rt := &models.RefreshToken{
		ID:    newUUID.String(),
		UserID:    userID,
		Token:     refreshStr,
		IPAddress: ip,
		UserAgent: userAgent,
		ExpiresAt: now.Add(RefreshTokenDuration),
	}
	if _, err := database.DB.NewInsert().Model(rt).Exec(ctx); err != nil {
		logger.Log.Error("failed to store refresh token", zap.Error(err))
		return nil, errors.InternalError(err)
	}

	return &TokenPair{AccessToken: accessStr, RefreshToken: refreshStr}, nil
}
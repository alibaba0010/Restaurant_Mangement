package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/alibaba0010/postgres-api/internal/auth/models"
	"github.com/alibaba0010/postgres-api/internal/auth/repositories"
	apierrors "github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/alibaba0010/postgres-api/internal/config"
	"github.com/alibaba0010/postgres-api/internal/utils"
	"go.uber.org/zap"
)

const (
	AccessTokenDuration  = 15 * time.Minute   //15
	RefreshTokenDuration = 7 * 24 * time.Hour // 7 days
)

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type AccessTokenClaims struct {
	UserID string         `json:"user_id"`
	Role   types.UserRole `json:"role"`
	jwt.RegisteredClaims
}

// GenerateTokenPair optionally accepts oldHashedToken for transaction-safe rotation
func GenerateTokenPair(ctx context.Context, userID string, role types.UserRole, ip, userAgent string, oldHashedToken ...string) (*TokenPair, *apierrors.AppError) {
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
		return nil, apierrors.InternalError(err)
	}

	// Opaque Refresh token: Generate 32 crypto-secure random bytes
	rawTokenBytes := make([]byte, 32)
	if _, err := rand.Read(rawTokenBytes); err != nil {
		logger.Log.Error("failed to generate random bytes for refresh token", zap.Error(err))
		return nil, apierrors.InternalError(err)
	}
	// The client-facing raw string
	refreshStr := base64.URLEncoding.EncodeToString(rawTokenBytes)

	// Hash it securely for DB storage
	hash := sha256.Sum256([]byte(refreshStr))
	hashedToken := hex.EncodeToString(hash[:])

	// Per your review, we NEVER call DeleteAllForUser here anymore. Sessions are preserved.
	newUUID, err := utils.GenerateUUIDv7()
	if err != nil {
		logger.Log.Error("failed to generate UUID for refresh token", zap.Error(err))
		return nil, apierrors.InternalError(err)
	}

	rt := &models.RefreshToken{
		ID:        newUUID.String(),
		UserID:    userID,
		Token:     hashedToken, // Stored hashed
		IPAddress: ip,
		UserAgent: userAgent,
		ExpiresAt: now.Add(RefreshTokenDuration),
	}

	// If rotating, use RotateToken from TokenRepo explicitly with atomic db operations
	var dbErr error
	if len(oldHashedToken) > 0 && oldHashedToken[0] != "" {
		dbErr = repositories.TokenRepo.RotateToken(ctx, userID, oldHashedToken[0], rt)
	} else {
		dbErr = repositories.TokenRepo.Create(ctx, rt)
	}

	if dbErr != nil {
		logger.Log.Error("failed to store refresh token", zap.Error(dbErr))
		return nil, apierrors.InternalError(dbErr)
	}

	return &TokenPair{AccessToken: accessStr, RefreshToken: refreshStr}, nil
}

func VerifyAccessToken(tokenString string) (*AccessTokenClaims, *apierrors.AppError) {
	cfg := config.LoadConfig()

	claims := &AccessTokenClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, apierrors.UnauthorizedError("invalid token signing method")
		}
		return []byte(cfg.ACCESS_TOKEN_SECRET), nil
	})

	if err != nil {
		// Check if error is due to token expiration
		if errors.Is(err, jwt.ErrTokenExpired) {
			logger.Log.Debug("access token expired", zap.Error(err))
			return nil, apierrors.UnauthorizedError("access token is expired")
		}
		logger.Log.Debug("access token verification failed", zap.Error(err))
		return nil, apierrors.UnauthorizedError("access token is invalid")
	}

	if !token.Valid {
		logger.Log.Debug("access token is not valid")
		return nil, apierrors.UnauthorizedError("access token is invalid")
	}

	return claims, nil
}





package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	redisPkg "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/crypto/argon2"

	"github.com/alibaba0010/postgres-api/internal/auth/dto"
	"github.com/alibaba0010/postgres-api/internal/auth/models"
	"github.com/alibaba0010/postgres-api/internal/auth/repositories"
	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/alibaba0010/postgres-api/internal/config"
	"github.com/alibaba0010/postgres-api/internal/database"
	"github.com/alibaba0010/postgres-api/internal/utils"
)

// Argon2 parameters centralized for readability and maintainability
const (
	argonTimeParam uint32 = 1
	argonMemory    uint32 = 64 * 1024
	argonThreads   uint8  = 4
	argonKeyLen    uint32 = 32
	argonSaltLen   uint32 = 16
)

// mapValidatorErrors converts go-playground/validator errors into AppError
func mapValidatorErrors(err error) *errors.AppError {
	if ves, ok := err.(validator.ValidationErrors); ok {
		var messages []string
		for _, fe := range ves {
			field := fe.Field()
			var msg string
			switch fe.Tag() {
			case "oneof":
				msg = fmt.Sprintf("%s can only either be user, admin or management", field)
			case "required":
				msg = fmt.Sprintf("%s is required", field)
			case "min":
				msg = fmt.Sprintf("%s must be at least %s characters", field, fe.Param())
			case "max":
				msg = fmt.Sprintf("%s must be at most %s characters", field, fe.Param())
			case "email":
				msg = fmt.Sprintf("%s must be a valid email address", field)
			case "password_special":
				msg = "password must contain at least one uppercase letter, one lowercase letter, one digit, and one special character"
			case "eqfield":
				msg = fmt.Sprintf("%s must match %s", field, fe.Param())
			default:
				msg = fmt.Sprintf("%s is invalid", field)
			}
			messages = append(messages, msg)
		}
		return errors.ValidationErrors(messages)
	}
	return errors.ValidationError(err.Error())
}

// RegisterUser handles the DB work for signing up a new user.
// It checks for an existing email, hashes the password and inserts the user.
// Returns the created user (with ID populated) or an AppError for controller to return.
func RegisterUser(ctx context.Context, input dto.SignupInput) (*models.User, *errors.AppError) {
	// Validate input using same validation rules as controllers previously used
	validate := validator.New()
	dto.RegisterValidators(validate)

	// Run validation and convert errors to friendly messages
	if err := validate.Struct(input); err != nil {
		return nil, mapValidatorErrors(err)
	}

	// Set default role if not provided
	role := types.UserRole(input.Role)
	if role == "" {
		role = types.RoleUser
	}

	// Check if user already exists
	exists, err := repositories.UserRepo.ExistsByEmail(ctx, input.Email)
	if err != nil {
		return nil, errors.InternalError(err)
	}
	if exists {
		return nil, errors.DuplicateError("email")
	}

	// Generate UUID and verification token in parallel
	newUUID, err := utils.GenerateUUIDv7()
	if err != nil {
		return nil, errors.InternalError(err)
	}

	token, err := utils.GenerateToken()
	if err != nil {
		return nil, errors.InternalError(err)
	}

	// Hash password
	hashedPwd, err := hashPassword(input.Password)
	if err != nil {
		return nil, errors.InternalError(err)
	}

	// Prepare user data
	user := &models.User{
		ID:    newUUID.String(),
		Name:  input.Name,
		Email: input.Email,
	}

	// Prepare payload for Redis storage
	payload := struct {
		ID       string         `json:"id"`
		Name     string         `json:"name"`
		Email    string         `json:"email"`
		Password string         `json:"password"`
		Role     types.UserRole `json:"role"`
	}{
		ID:       user.ID,
		Name:     user.Name,
		Email:    user.Email,
		Password: hashedPwd,
		Role:     role,
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.InternalError(err)
	}

	key := "verify:" + token
	ttl := 15 * time.Minute
	if err := database.RedisClient.Set(ctx, key, b, ttl).Err(); err != nil {
		return nil, errors.InternalError(err)
	}

	// Build verification URL
	cfg := config.LoadConfig()
	verifyURL := fmt.Sprintf("%s/verify?token=%s", cfg.FRONTEND_URL, token)
	html := VerifyMailHTML(user.Name, verifyURL)
		go func() {
		if err := SendEmail(user.Email, "Verify your email", html); err != nil {
			logger.Log.Error("failed to send verification email", 
				zap.Error(err),
				zap.String("email", user.Email),
				zap.String("token", token),
			)
			// Optionally: Add to a retry queue here  future enhancement
		}
	}()

	// Per new flow, registration doesn't persist the user yet — activation will.
	return nil, nil
}
func ActivateUser(ctx context.Context, token string) (*models.User, *errors.AppError) {
	key := "verify:" + token
	data, err := database.RedisClient.Get(ctx, key).Bytes()
	if err == redisPkg.Nil {
		return nil, errors.ValidationError("invalid or expired token")
	}
	if err != nil {
		return nil, errors.InternalError(err)
	}

	var payload struct {
		ID       string         `json:"id"`
		Name     string         `json:"name"`
		Email    string         `json:"email"`
		Password string         `json:"password"`
		Role     types.UserRole `json:"role"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		_ = database.RedisClient.Del(ctx, key).Err()
		return nil, errors.InternalError(err)
	}

	// The password stored in Redis is already hashed, so use it directly.
	user := &models.User{
		ID:       payload.ID,
		Name:     payload.Name,
		Email:    payload.Email,
		Password: payload.Password,
		Role:     payload.Role,
	}

	// Insert into DB
	err = repositories.UserRepo.Create(ctx, user)
	if err != nil {
		return nil, errors.InternalError(err)
	}

	// Token used within TTL -> remove it
	if err := database.RedisClient.Del(ctx, key).Err(); err != nil {
		logger.Log.Error("failed to delete verification token", zap.Error(err))
	}

	return user, nil
}


// LoginUser authenticates a user by email and password
// Returns the user and generated token pair if successful
func LoginUser(ctx context.Context, email, password string) (*models.User, *TokenPair, *errors.AppError) {
	if email == "" || password == "" {
		return nil, nil, errors.ValidationError("email and password are required")
	}

	// Fetch user by email
	user, err := repositories.UserRepo.FindByEmail(ctx, email)
	if err != nil {
		logger.Log.Debug("user not found for login", zap.String("email", email))
		return nil, nil, errors.UnauthorizedError("invalid email or password")
	}

	// Verify password
	if !verifyPassword(password, user.Password) {
		logger.Log.Warn("invalid password for login", zap.String("email", email))
		return nil, nil, errors.UnauthorizedError("invalid email or password")
	}

	logger.Log.Debug("user authenticated successfully", zap.String("user_id", user.ID), zap.String("email", email))
	return user, nil, nil
}

// VerifyPassword compares a plaintext password with an argon2id hash
func verifyPassword(password, hash string) bool {
	// Parse hash components
	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		return false
	}

	// Extract salt and hash from hash string
	b64Salt := parts[4]
	b64Hash := parts[5]

	salt, err := base64.RawStdEncoding.DecodeString(b64Salt)
	if err != nil {
		return false
	}

	originalHash, err := base64.RawStdEncoding.DecodeString(b64Hash)
	if err != nil {
		return false
	}

	// Hash the provided password with the same salt using centralized params
	newHash := argon2.IDKey([]byte(password), salt, argonTimeParam, argonMemory, argonThreads, argonKeyLen)

	// Compare hashes
	return string(newHash) == string(originalHash)
}

// hashPassword creates an argon2id hash of the password
func hashPassword(password string) (string, error) {
	// Parameters
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, argonTimeParam, argonMemory, argonThreads, argonKeyLen)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", argonMemory, argonTimeParam, argonThreads, b64Salt, b64Hash)
	return encoded, nil
}


func LogoutUser(ctx context.Context, userID string) (*errors.AppError) {
	if userID == "" {
		return errors.ValidationError("user id is required")
	}

	// Use a transaction to ensure atomic deletion of all tokens
	tx, err := database.DB.BeginTx(ctx, nil)
	if err != nil {
		logger.Log.Error("failed to begin transaction for logout", zap.Error(err))
		return errors.InternalError(err)
	}

	defer func() {
		if err := tx.Rollback(); err != nil && err.Error() != "tx: already committed or rolled back" {
			logger.Log.Error("failed to rollback logout transaction", zap.Error(err))
		}
	}()

	// Delete all refresh tokens for the user
	res, err := repositories.TokenRepo.DeleteAllForUserInTx(ctx, tx, userID)

	if err != nil {
		logger.Log.Error("failed to delete refresh tokens", zap.Error(err), zap.String("user_id", userID))
		return errors.InternalError(err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		logger.Log.Error("failed to commit logout transaction", zap.Error(err))
		return errors.InternalError(err)
	}

	logger.Log.Info("user logout successful", 
		zap.String("user_id", userID),
		zap.Int64("tokens_revoked", res),
	)

	return nil
}

func RefreshTokenWithRotation(ctx context.Context, refreshToken, ip, userAgent string) (*TokenPair, *errors.AppError) {
	if refreshToken == "" {
		return nil, errors.UnauthorizedError("refresh token missing; please login again")
	}

	// Validate refresh token signature and expiration
	refreshClaims, appErr := ValidateRefreshToken(refreshToken)
	if appErr != nil {
		return nil, appErr
	}

	userID := refreshClaims.UserID

	// Check if refresh token exists in database
	exists, err := repositories.TokenRepo.Exists(ctx, userID, refreshToken)

	if err != nil {
		logger.Log.Error("failed to query refresh token from DB", zap.Error(err))
		return nil, errors.InternalError(err)
	}

	if !exists {
		logger.Log.Warn("refresh token not found in database", zap.String("user_id", userID))
		return nil, errors.UnauthorizedError("user login")
	}

	// Generate new token pair with rotation
	// This will create a new refresh token and store it in DB
	newTokenPair, appErr := GenerateTokenPair(ctx, userID, refreshClaims.Role, ip, userAgent)
	if appErr != nil {
		return nil, appErr
	}

	return newTokenPair, nil
}

// // LogoutAllDevices is a variant that could accept device fingerprints if needed
// // Currently logs out all devices by clearing all tokens
// func LogoutAllDevices(ctx context.Context, userID string) (*errors.AppError) {
// 	// Reuse the main LogoutUser function
// 	return LogoutUser(ctx, userID)
// }

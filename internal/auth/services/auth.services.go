package services

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

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



// Register User Functionality
func RegisterUser(ctx context.Context, input dto.SignupInput) (*models.User, *errors.AppError) {
	// Validate input
	if err := utils.ValidateAndError(input); err != nil {
		return nil, err
	}

	// Set default role
	role := types.RoleUser

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
	payload := dto.VerificationPayload{
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

	ttl := 15 * time.Minute
	// Store token(key) -> payload(value)
	if err := database.RedisClient.Set(ctx, "verify:"+token, b, ttl).Err(); err != nil {
		return nil, errors.InternalError(err)
	}
	// Also store mapping from email -> token so we can lookup token by email for resends
	if err := database.RedisClient.Set(ctx, "verify:email:"+user.Email, token, ttl).Err(); err != nil {
		// If this secondary set fails, try to clean up the primary key and return error
		_ = database.RedisClient.Del(ctx, "verify:"+token).Err()
		return nil, errors.InternalError(err)
	}
	// Build verification URL
	cfg := config.LoadConfig()
	verifyURL := fmt.Sprintf("%s/verify?token=%s", cfg.FRONTEND_URL, token)
	html := VerifyMailHTML(user.Name, verifyURL)
	// Use centralized async email sender
	SendEmailHandler(user.Email, "Verify your email", html)

	return nil, nil
}

// Activate User Functionality
func ActivateUser(ctx context.Context, token string) (*models.User, *errors.AppError) {
	// Reuse helper to get verification payload (centralizes Redis access)
	payload, err := GetVerificationPayload(ctx, token)
	if err != nil {
		if err == redisPkg.Nil {
			return nil, errors.ValidationError("invalid or expired token")
		}
		return nil, errors.InternalError(err)
	}

	// The password stored in Redis is already hashed.
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

	// Token used within TTL -> remove both keys (token and email->token mapping)
	_ = database.RedisClient.Del(ctx, "verify:"+token).Err()
	_ = database.RedisClient.Del(ctx, "verify:email:"+payload.Email).Err()

	return user, nil
}

func LoginUser(ctx context.Context, email, password string) (*models.User, *TokenPair, *errors.AppError) {
	in := dto.SigninInput{Email: email, Password: password}
	if err := utils.ValidateAndError(in); err != nil {
		return nil, nil, err
	}

	// Fetch user by email
	user, err := repositories.UserRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, nil, errors.NotFoundError("invalid email or password")
	}

	// Verify password
	if !verifyPassword(password, user.Password) {
		return nil, nil, errors.NotFoundError("invalid email or password")
	}

	return user, nil, nil
}

// OAuthLogin handles the social login logic (find or create user, generate tokens)
func OAuthLogin(ctx context.Context, email, name, picture, ip, ua string) (*models.User, *TokenPair, *errors.AppError) {
	// Check if user exists
	user, err := repositories.UserRepo.FindByEmail(ctx, email)
	if err != nil {
		// If not found (or error), we assume not found for now and try to create.
	}

	if user == nil || user.ID == "" {
		// Create new user
		newUUID, err := utils.GenerateUUIDv7()
		if err != nil {
			return nil, nil, errors.InternalError(err)
		}
		user = &models.User{
			ID:        newUUID.String(),
			Name:      name,
			Email:     email,
			Password:  "", // No password for social login
			Status:    types.StatusActive,
			Role:      types.RoleUser,
			AvatarURL: picture,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := repositories.UserRepo.Create(ctx, user); err != nil {
			return nil, nil, errors.InternalError(err)
		}
	} else {
		if user.Status == types.StatusSuspended || user.Status == types.StatusDeleted {
			return nil, nil, errors.ForbiddenError("account is " + string(user.Status))
		}
	}

	tokens, appErr := GenerateTokenPair(ctx, user.ID, user.Role, ip, ua)
	if appErr != nil {
		return nil, nil, appErr
	}

	return user, tokens, nil
}

// GenerateOAuthStateAndURL creates an oauth state token and returns provider auth url
func GenerateOAuthStateAndURL(provider string) (string, string, *errors.AppError) {
	state, err := utils.GenerateToken()
	if err != nil {
		return "", "", errors.InternalError(err)
	}

	var authURL string
	switch provider {
	case "google":
		authURL = GetGoogleAuthURL(state)
	case "facebook":
		authURL = GetFacebookAuthURL(state)
	default:
		return "", "", errors.ValidationError("unsupported provider")
	}

	return state, authURL, nil
}

// HandleOAuthCallback exchanges code, obtains user info and performs social login
func HandleOAuthCallback(ctx context.Context, provider, code, ip, ua string) (*models.User, *TokenPair, *errors.AppError) {
	var email, name, picture string

	switch provider {
	case "google":
		gUser, err := ExchangeGoogleCode(code)
		if err != nil {
			return nil, nil, errors.UnauthorizedError(err.Error())
		}
		email = gUser.Email
		name = gUser.Name
		picture = gUser.Picture
	default:
		return nil, nil, errors.ValidationError("unsupported provider")
	}

	user, tokens, appErr := OAuthLogin(ctx, email, name, picture, ip, ua)
	if appErr != nil {
		return nil, nil, appErr
	}

	return user, tokens, nil
}

// VerifyPassword compares a plaintext password with an argon2id hash
func verifyPassword(password, hash string) bool {
	// Time: O(1) (argon2 runs in fixed time based on parameters)
	// Space: O(1)
	// Parse hash components: expected format: $argon2id$v=19$m=...,t=...,p=...$<salt>$<hash>
	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		return false
	}

	// Extract encoded salt & hash
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

	// Attempt to parse parameters from parts[3] (m=...,t=...,p=...)
	mem := argonMemory
	t := argonTimeParam
	p := argonThreads
	params := parts[3]
	if params != "" {
		kvs := strings.Split(params, ",")
		for _, kv := range kvs {
			if strings.HasPrefix(kv, "m=") {
				var parsed uint32
				_, err := fmt.Sscanf(kv, "m=%d", &parsed)
				if err == nil {
					mem = parsed
				}
			} else if strings.HasPrefix(kv, "t=") {
				var parsed uint32
				_, err := fmt.Sscanf(kv, "t=%d", &parsed)
				if err == nil {
					t = parsed
				}
			} else if strings.HasPrefix(kv, "p=") {
				var parsed uint8
				_, err := fmt.Sscanf(kv, "p=%d", &parsed)
				if err == nil {
					p = parsed
				}
			}
		}
	}

	// Hash the provided password with the same salt using parsed params
	newHash := argon2.IDKey([]byte(password), salt, t, mem, p, argonKeyLen)

	// Use constant-time comparison to avoid timing attacks
	if len(newHash) != len(originalHash) {
		return false
	}
	return subtle.ConstantTimeCompare(newHash, originalHash) == 1
}

// hashPassword creates an argon2id hash of the password
func hashPassword(password string) (string, error) {
	// Time: O(1) (Argon2 costs are constant per call based on configured params)
	// Space: O(1)
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

func LogoutUser(ctx context.Context, userID string) *errors.AppError {
	if userID == "" {
		return errors.ValidationError("user id is required")
	}

	// Time: O(1) - single DELETE statement
	// Space: O(1)
	// A single SQL DELETE is atomic; starting an explicit transaction here
	// adds overhead without benefit because only one statement is executed.
	// Use the repository helper to delete directly.
	_, err := repositories.TokenRepo.DeleteAllForUser(ctx, userID)
	if err != nil {
		logger.Log.Error("failed to delete refresh tokens", zap.Error(err), zap.String("user_id", userID))
		return errors.InternalError(err)
	}

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

	// Time: O(1) Space: O(1)
	// Fetch the stored refresh token record to validate metadata (ip/userAgent)
	rt, err := repositories.TokenRepo.FindOne(ctx, userID, refreshToken)
	if err != nil {
		logger.Log.Error("failed to query refresh token from DB", zap.Error(err))
		return nil, errors.UnauthorizedError("user login")
	}

	// Validate IP and User-Agent if they were recorded with the token
	if rt.IPAddress != "" && ip != "" && strings.TrimSpace(rt.IPAddress) != strings.TrimSpace(ip) {
		// Attempt to parse both IPs to check if they are both loopback addresses
		// This handles the case where one is [::1] and the other is 127.0.0.1 (common in dev/localhost)
		rtIP := net.ParseIP(strings.Trim(rt.IPAddress, "[]"))
		gotIP := net.ParseIP(strings.Trim(ip, "[]"))

		isLoopbackMismatch := false
		if rtIP != nil && gotIP != nil {
			if rtIP.IsLoopback() && gotIP.IsLoopback() {
				// Both are loopback, consider them matching
				isLoopbackMismatch = true
			}
		}

		if !isLoopbackMismatch {
			logger.Log.Warn("refresh token IP mismatch", zap.String("expected", rt.IPAddress), zap.String("got", ip))
			return nil, errors.UnauthorizedError("refresh token validation failed")
		}
	}
	if rt.UserAgent != "" && userAgent != "" && strings.TrimSpace(rt.UserAgent) != strings.TrimSpace(userAgent) {
		logger.Log.Warn("refresh token user-agent mismatch", zap.String("expected", rt.UserAgent), zap.String("got", userAgent))
		return nil, errors.UnauthorizedError("refresh token validation failed")
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

// ForgotPassword generates a password reset token and sends it via email
func ForgotPassword(ctx context.Context, email string) *errors.AppError {
	if email == "" {
		return errors.ValidationError("email is required")
	}

	// Check if user exists
	user, err := repositories.UserRepo.FindByEmail(ctx, email)
	if err != nil {
		return errors.NotFoundError("User not found for Password Reset")
	}

	// Generate reset token
	token, err := utils.GenerateToken()
	if err != nil {
		return errors.InternalError(err)
	}

	// Store token in Redis with user ID
	payload := struct {
		UserID string `json:"user_id"`
		Email  string `json:"email"`
		Name   string `json:"name"`
	}{
		UserID: user.ID,
		Email:  user.Email,
		Name:   user.Name,
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return errors.InternalError(err)
	}

	key := "reset:" + token
	ttl := 15 * time.Minute
	if err := database.RedisClient.Set(ctx, key, b, ttl).Err(); err != nil {
		return errors.InternalError(err)
	}

	// Build reset URL
	cfg := config.LoadConfig()
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", cfg.FRONTEND_URL, token)
	html := ResetPasswordMailHTML(user.Name, resetURL)

	// Send email asynchronously via central handler
	SendEmailHandler(user.Email, "Reset your password", html)

	return nil
}

// ResetPassword validates the reset token and updates the user's password
func ResetPassword(ctx context.Context, token, newPassword string) *errors.AppError {
	if token == "" || newPassword == "" {
		return errors.ValidationError("token and password are required")
	}

	// Retrieve token data from Redis
	key := "reset:" + token
	data, err := database.RedisClient.Get(ctx, key).Bytes()
	if err == redisPkg.Nil {
		return errors.ValidationError("invalid or expired reset token")
	}
	if err != nil {
		return errors.InternalError(err)
	}

	var payload struct {
		UserID string `json:"user_id"`
		Email  string `json:"email"`
		Name   string `json:"name"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		_ = database.RedisClient.Del(ctx, key).Err()
		return errors.InternalError(err)
	}

	// Hash new password
	hashedPwd, err := hashPassword(newPassword)
	if err != nil {
		return errors.InternalError(err)
	}

	// Update user's password in database
	err = repositories.UserRepo.UpdatePassword(ctx, payload.UserID, hashedPwd)
	if err != nil {
		return errors.InternalError(err)
	}

	// Delete reset token from Redis
	if err := database.RedisClient.Del(ctx, key).Err(); err != nil {
		logger.Log.Error("failed to delete reset token", zap.Error(err))
	}

	// Optionally: Revoke all existing sessions for security
	_, _ = repositories.TokenRepo.DeleteAllForUser(ctx, payload.UserID)

	return nil
}

// GetVerificationPayload retrieves the verification data from Redis using the token
func GetVerificationPayload(ctx context.Context, token string) (*dto.VerificationPayload, error) {
	key := "verify:" + token
	data, err := database.RedisClient.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	var payload dto.VerificationPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// ResendVerification handles the logic for resending a verification email
func ResendVerification(ctx context.Context, email string) *errors.AppError {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return errors.ValidationError("email is required")
	}

	// 1. Check if user already exists in DB
	exists, err := repositories.UserRepo.ExistsByEmail(ctx, email)
	if err != nil {
		return errors.InternalError(err)
	}
	if exists {
		// Do not leak account existence, but since they are already activated,
		// they shouldn't be here. We return a generic success message in controller.
		return nil
	}

	// 2. Check if verification token exists in Redis
	token, err := database.RedisClient.Get(ctx, "verify:email:"+email).Result()
	if err == redisPkg.Nil {
		return errors.ValidationError("verification link expired or not found; please sign up again")
	}
	if err != nil {
		return errors.InternalError(err)
	}

	// 3. Get payload to get the user's name
	payload, err := GetVerificationPayload(ctx, token)
	if err != nil {
		return errors.InternalError(err)
	}

	// 4. Send email
	cfg := config.LoadConfig()
	verifyURL := fmt.Sprintf("%s/verify?token=%s", cfg.FRONTEND_URL, token)
	html := VerifyMailHTML(payload.Name, verifyURL)

	// Use centralized async sender
	SendEmailHandler(email, "Verify your email", html)

	return nil
}

// SendEmailHandler is a centralized fire-and-forget email sender with subject
func SendEmailHandler(email, subject, html string) {
	go func() {
		if err := SendEmail(email, subject, html); err != nil {
			logger.Log.Error("failed to send email",
				zap.Error(err),
				zap.String("email", email),
				zap.String("subject", subject),
			)
		}
	}()
}

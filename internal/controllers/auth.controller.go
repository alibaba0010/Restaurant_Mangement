package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"

	"github.com/alibaba0010/postgres-api/internal/config"
	"github.com/alibaba0010/postgres-api/internal/database"
	"github.com/alibaba0010/postgres-api/internal/dto"
	"github.com/alibaba0010/postgres-api/internal/errors"
	"github.com/alibaba0010/postgres-api/internal/models"
	"github.com/alibaba0010/postgres-api/internal/services"
	"github.com/alibaba0010/postgres-api/internal/utils"
)


func SignupHandler(writer http.ResponseWriter, request *http.Request) {
	var input dto.SignupInput

	// Decode JSON
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		errors.ErrorResponse(writer, request, errors.ValidationError("Invalid JSON body"))
		return
	}
	
	_, appErr := services.RegisterUser(request.Context(), input)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}
	// Per new flow we don't persist the user at signup; activation will.
	resp := map[string]string{
		"title":   "Successfully signed up",
		"message": "Please check your email for a verification link",
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(writer).Encode(resp)
}
func ActivateUserHandler(writer http.ResponseWriter, request *http.Request) {
	token := request.URL.Query().Get("token")
	if token == "" {
		errors.ErrorResponse(writer, request, errors.ValidationError("token is required"))
		return
	}

	user, appErr := services.ActivateUser(request.Context(), token)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	// generate token pair and set refresh token cookie
	// extract IP and user-agent
	ip:= utils.ExtractClientIP(request)
	ua := request.Header.Get("User-Agent")

	tokens, appErr := services.GenerateTokenPair(request.Context(), user.ID, user.Role, ip, ua)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}
	request.Header.Set("Authorization", "Bearer "+tokens.AccessToken)

	// set refresh token cookie
	cfg := config.LoadConfig()
	cookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    tokens.RefreshToken,
		HttpOnly: true,
		Path:     "/",
		Expires:  time.Now().Add(services.RefreshTokenDuration),
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	}
	// If running behind TLS, make cookie secure
	if strings.HasPrefix(cfg.FRONTEND_URL, "https") {
		cookie.Secure = true
	}
	http.SetCookie(writer, cookie)

	// Return created user (omit password) + tokens
	resp := dto.SignUpResponse{
		Title: "User activated successfully",
		Data: dto.SignUpData{
			ID:           user.ID,
			Name:         user.Name,
			Email:        user.Email,
			Role:         user.Role,
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
		},
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(writer).Encode(resp)
}

func SigninHandler(writer http.ResponseWriter, request *http.Request) {
	var input dto.SigninInput

	// Decode JSON
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		errors.ErrorResponse(writer, request, errors.ValidationError("Invalid JSON body"))
		return
	}

	// Trim email
	input.Email = strings.TrimSpace(input.Email)
	input.Password = strings.TrimSpace(input.Password)

	// Validate input
	validate := validator.New()
	if err := validate.Struct(input); err != nil {
		errors.ErrorResponse(writer, request, errors.ValidationError("email and password are required"))
		return
	}

	// Authenticate user
	user, _, appErr := services.LoginUser(request.Context(), input.Email, input.Password)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	// Extract client IP and User-Agent
	ip := utils.ExtractClientIP(request)
	userAgent := request.Header.Get("User-Agent")

	// Generate token pair
	tokens, appErr := services.GenerateTokenPair(request.Context(), user.ID, user.Role, ip, userAgent)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}
	
	request.Header.Set("Authorization", "Bearer "+tokens.AccessToken)

	// Set refresh token cookie
	cfg := config.LoadConfig()
	cookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    tokens.RefreshToken,
		HttpOnly: true,
		Path:     "/",
		Expires:  time.Now().Add(services.RefreshTokenDuration),
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	}
	if strings.HasPrefix(cfg.FRONTEND_URL, "https") {
		cookie.Secure = true
	}
	http.SetCookie(writer, cookie)

	// Return response
	resp := dto.SigninResponse{
		Title: "Signin successful",
		Data: dto.SigninData{
			ID:           user.ID,
			Name:         user.Name,
			Email:        user.Email,
			Role:         user.Role,
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
		},
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(writer).Encode(resp)
}

func ResendVerificationHandler(writer http.ResponseWriter, request *http.Request) {
	var body struct{
		Email string `json:"email"`
	}
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		errors.ErrorResponse(writer, request, errors.ValidationError("Invalid JSON body"))
		return
	}
	email := strings.TrimSpace(body.Email)
	if email == "" {
		errors.ErrorResponse(writer, request, errors.ValidationError("email is required"))
		return
	}

	// If a user already exists in DB, they are activated
	exists, err := database.DB.NewSelect().Model((*models.User)(nil)).Where("email = ?", email).Exists(request.Context())
	if err != nil {
		errors.ErrorResponse(writer, request, errors.InternalError(err))
		return
	}
	if exists {
		errors.ErrorResponse(writer, request, errors.ValidationError("user already exists"))
		return
	}

	// Scan Redis for verify:* keys and find matching email
	var cursor uint64
	found := false
	for {
		keys, cur, err := database.RedisClient.Scan(request.Context(), cursor, "verify:*", 100).Result()
		if err != nil {
			errors.ErrorResponse(writer, request, errors.InternalError(err))
			return
		}
		for _, k := range keys {
			b, err := database.RedisClient.Get(request.Context(), k).Bytes()
			if err != nil {
				continue
			}
			var payload struct{
				ID string `json:"id"`
				Name string `json:"name"`
				Email string `json:"email"`
			}
			if err := json.Unmarshal(b, &payload); err != nil {
				continue
			}
			if strings.EqualFold(payload.Email, email) {
				token := strings.TrimPrefix(k, "verify:")
				cfg := config.LoadConfig()
				verifyURL := fmt.Sprintf("%s/api/v1/auth/verify?token=%s", cfg.FRONTEND_URL, token)
				html := services.VerifyMailHTML(payload.Name, verifyURL)
				if err := services.SendEmail(email, "Verify your email", html); err != nil {
					errors.ErrorResponse(writer, request, errors.InternalError(err))
					return
				}
				found = true
				break
			}
		}
		if cur == 0 || found {
			break
		}
		cursor = cur
	}
	if !found {
		errors.ErrorResponse(writer, request, errors.ValidationError("verification token not found or expired"))
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(writer).Encode(map[string]string{"message": "Verification email resent"})
}
package controllers

import (
	"encoding/json"
 	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alibaba0010/postgres-api/internal/config"
	"github.com/alibaba0010/postgres-api/internal/database"
	"github.com/alibaba0010/postgres-api/internal/dto"
	"github.com/alibaba0010/postgres-api/internal/errors"
	"github.com/alibaba0010/postgres-api/internal/models"
	"github.com/alibaba0010/postgres-api/internal/services"
)

// SignupHandler godoc
//
//	@Summary		User Signup
//	@Description	Creates a new user account
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dto.SignupInput	true	"Signup request"
//	@Success		201	{object}	map[string]interface{} "User created successfully"
//	@Failure		400	{object}	map[string]string		"Validation error"
//	@Failure		409	{object}	map[string]string		"Duplicate email"
//	@Failure		500	{object}	map[string]string		"Internal server error"
//	@Router			/auth/signup [post]
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
	var ip string
	if xf := request.Header.Get("X-Forwarded-For"); xf != "" {
		// may be comma-separated
		parts := strings.Split(xf, ",")
		ip = strings.TrimSpace(parts[0])
	} else if xr := request.Header.Get("X-Real-Ip"); xr != "" {
		ip = xr
	} else {
		// remote addr may include port
		remote := request.RemoteAddr
		if i := strings.LastIndex(remote, ":"); i != -1 {
			ip = remote[:i]
		} else {
			ip = remote
		}
	}
	ua := request.Header.Get("User-Agent")

	tokens, appErr := services.GenerateTokenPair(request.Context(), user.ID, user.Role, ip, ua)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

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

// ResendVerificationHandler resends the verification email for a pending signup.
// It looks up pending verification tokens in Redis (scan) by email and resends
// the existing token's verification link.
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
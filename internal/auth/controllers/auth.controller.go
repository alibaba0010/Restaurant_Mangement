package controllers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"

	"github.com/alibaba0010/postgres-api/internal/auth/dto"
	"github.com/alibaba0010/postgres-api/internal/auth/models"
	"github.com/alibaba0010/postgres-api/internal/auth/services"
	commondto "github.com/alibaba0010/postgres-api/internal/common/dto"
	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/guards"
	"github.com/alibaba0010/postgres-api/internal/config"
	"github.com/alibaba0010/postgres-api/internal/utils"
)

// sendAuthResponse is a helper to centralize setting cookies and writing the final success response.
// sendAuthResponse is a helper to centralize setting cookies and writing the final success response.
func sendAuthResponse(writer http.ResponseWriter, user *models.User, tokens *services.TokenPair, title string) {
	utils.SetAuthCookies(writer, tokens.AccessToken, tokens.RefreshToken, services.AccessTokenDuration, services.RefreshTokenDuration)

	resp := dto.SigninResponse{
		Title: title,
		Data: dto.SigninData{
			ID:          user.ID,
			Name:        user.Name,
			Email:       user.Email,
			Role:        user.Role,
			Address:     user.Address,
			AvatarURL:   user.AvatarURL,
			PhoneNumber: user.PhoneNumber,
			Status:      user.Status,
			CreatedAt:   user.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   user.UpdatedAt.Format(time.RFC3339),
			AccessToken: tokens.AccessToken,
		},
	}
	utils.WriteJSON(writer, http.StatusOK, resp)
}

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
	resp := commondto.MessageResponse{
		Title:   "Successfully signed up",
		Message: "Please check your email for a verification link",
	}

	utils.WriteJSON(writer, http.StatusCreated, resp)

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

	// Generate tokens
	ip := utils.ExtractClientIP(request)
	ua := request.Header.Get("User-Agent")
	tokens, appErr := services.GenerateTokenPair(request.Context(), user.ID, user.Role, ip, ua)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	// set cookies and return response
	sendAuthResponse(writer, user, tokens, "User activated successfully")
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
	// log tokens
	// set cookies and return response
	sendAuthResponse(writer, user, tokens, "Signin successful")
}

func ResendVerificationHandler(w http.ResponseWriter, r *http.Request) {
	var body dto.ResendVerificationInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errors.ErrorResponse(w, r, errors.ValidationError("invalid JSON body"))
		return
	}

	appErr := services.ResendVerification(r.Context(), body.Email)
	if appErr != nil {
		errors.ErrorResponse(w, r, appErr)
		return
	}

	utils.WriteJSON(w, http.StatusOK, commondto.MessageResponse{
		Title:   "Email Resend",
		Message: "Verification email sent successfully",
	})
}

func RefreshTokenHandler(writer http.ResponseWriter, request *http.Request) {
	// Extract refresh token from cookie
	refreshCookie, err := request.Cookie("refresh_token")

	if err != nil {
		errors.ErrorResponse(writer, request, errors.UnauthorizedError("refresh token missing; please login again"))
		return
	}

	refreshToken := refreshCookie.Value

	if refreshToken == "" {
		errors.ErrorResponse(writer, request, errors.UnauthorizedError("refresh token missing; please login again"))
		return
	}

	// Extract IP and User-Agent for new token
	ip := utils.ExtractClientIP(request)
	userAgent := request.Header.Get("User-Agent")

	// Refresh token with rotation
	newTokenPair, appErr := services.RefreshTokenWithRotation(request.Context(), refreshToken, ip, userAgent)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	// Set new cookies
	utils.SetAuthCookies(writer, newTokenPair.AccessToken, newTokenPair.RefreshToken, services.AccessTokenDuration, services.RefreshTokenDuration)

	// Return new access token in body
	utils.WriteJSON(writer, http.StatusOK, commondto.MessageResponse{
		Title:       "Token Refresh",
		Message:     "Token refreshed successfully",
		AccessToken: newTokenPair.AccessToken,
	})
}

// LogoutHandler logs out the current user by revoking all their refresh tokens
func LogoutHandler(writer http.ResponseWriter, request *http.Request) {
	authenticatedUser := guards.ExtractAuthenticatedUser(request)
	if authenticatedUser == nil {
		errors.ErrorResponse(writer, request, errors.UnauthorizedError("user not authenticated"))
		return
	}

	// Call service to logout
	appErr := services.LogoutUser(request.Context(), authenticatedUser.UserID)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	// Determine if secure flag should be set based on config
	cfg := config.LoadConfig()
	isSecure := strings.HasPrefix(cfg.FRONTEND_URL, "https")

	utils.ClearAccessTokenCookie(writer, isSecure)
	utils.ClearRefreshTokenCookie(writer, isSecure)

	utils.WriteJSON(writer, http.StatusOK, dto.LogoutResponse{
		Title:   "User Logout",
		Message: "You have been successfully logged out",
	})
}

// InitiateOAuthHandler starts the OAuth flow
func InitiateOAuthHandler(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	provider := vars["provider"]

	// Generate state and provider auth URL using service helper
	state, authURL, appErr := services.GenerateOAuthStateAndURL(provider)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	// Set cookie with state
	cfg := config.LoadConfig()
	cookie := &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(15 * time.Minute),
	}
	if strings.HasPrefix(cfg.FRONTEND_URL, "https") {
		cookie.Secure = true
	}
	http.SetCookie(writer, cookie)

	utils.WriteJSON(writer, http.StatusOK, dto.OAuthLoginResponse{URL: authURL})
}

// VerifyOAuthHandler handles the callback verification from frontend (code & state)
func VerifyOAuthHandler(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	provider := vars["provider"]

	var input dto.VerifyOAuthInput
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		errors.ErrorResponse(writer, request, errors.ValidationError("Invalid JSON body"))
		return
	}

	// 1. Validate State
	cookie, err := request.Cookie("oauth_state")
	if err != nil || cookie.Value == "" {
		errors.ErrorResponse(writer, request, errors.ForbiddenError("missing or invalid state cookie"))
		return
	}
	if input.State != cookie.Value {
		errors.ErrorResponse(writer, request, errors.ForbiddenError("state mismatch"))
		return
	}

	// Clear state cookie
	cfg := config.LoadConfig()
	isSecure := strings.HasPrefix(cfg.FRONTEND_URL, "https")
	http.SetCookie(writer, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isSecure,
	})

	// 2. Handle provider callback (exchange code, create/find user, generate tokens)
	ip := utils.ExtractClientIP(request)
	ua := request.Header.Get("User-Agent")

	user, tokens, appErr := services.HandleOAuthCallback(request.Context(), provider, input.Code, ip, ua)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	sendAuthResponse(writer, user, tokens, "Login successful")
}

// ForgotPasswordHandler handles password reset requests
func ForgotPasswordHandler(writer http.ResponseWriter, request *http.Request) {
	var input dto.ForgotPasswordInput
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		errors.ErrorResponse(writer, request, errors.ValidationError("Invalid JSON body"))
		return
	}

	// Trim and validate email
	input.Email = strings.TrimSpace(input.Email)
	validate := validator.New()
	if err := validate.Struct(input); err != nil {
		errors.ErrorResponse(writer, request, errors.ValidationError("valid email is required"))
		return
	}

	// Call service to send reset email
	appErr := services.ForgotPassword(request.Context(), input.Email)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	// Always return success to avoid email enumeration
	utils.WriteJSON(writer, http.StatusOK, commondto.MessageResponse{
		Title:   "Password Reset",
		Message: "A password reset link has been sent to your email",
	})
}

// ResetPasswordHandler handles password reset with token
func ResetPasswordHandler(writer http.ResponseWriter, request *http.Request) {
	var input dto.ResetPasswordInput
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		errors.ErrorResponse(writer, request, errors.ValidationError("Invalid JSON body"))
		return
	}

	// Trim inputs
	input.Token = strings.TrimSpace(input.Token)
	input.Password = strings.TrimSpace(input.Password)
	input.ConfirmPassword = strings.TrimSpace(input.ConfirmPassword)

	// Validate input
	validate := validator.New()
	dto.RegisterValidators(validate)
	if err := validate.Struct(input); err != nil {
		errors.ErrorResponse(writer, request, errors.ValidationError("invalid input"))
		return
	}

	// Call service to reset password
	appErr := services.ResetPassword(request.Context(), input.Token, input.Password)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	utils.WriteJSON(writer, http.StatusOK, commondto.MessageResponse{
		Title:   "Password Reset",
		Message: "Your password has been reset successfully. Please login with your new password",
	})
}

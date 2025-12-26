package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"

	"github.com/alibaba0010/postgres-api/internal/auth/dto"
	"github.com/alibaba0010/postgres-api/internal/auth/models"
	"github.com/alibaba0010/postgres-api/internal/auth/services"
	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/guards"
	"github.com/alibaba0010/postgres-api/internal/config"
	"github.com/alibaba0010/postgres-api/internal/database"
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
	resp := dto.MessageResponse{
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

	// generate token pair and set refresh token cookie
	// extract IP and user-agent
	ip:= utils.ExtractClientIP(request)
	ua := request.Header.Get("User-Agent")

	tokens, appErr := services.GenerateTokenPair(request.Context(), user.ID, user.Role, ip, ua)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	// set refresh token cookie
	utils.SetAccessTokenCookie(writer, tokens.AccessToken, services.AccessTokenDuration)
	utils.SetRefreshTokenCookie(writer, tokens.RefreshToken, services.RefreshTokenDuration)

	// Return created user (omit password) + tokens
	resp := dto.SignUpResponse{
		Title: "User activated successfully",
		Data: dto.SignUpData{
			ID:           user.ID,
			Name:         user.Name,
			Email:        user.Email,
			Role:         user.Role,		
		},
	}
	utils.WriteJSON(writer, http.StatusOK, resp)
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
	

	// Set refresh token cookie
	utils.SetAccessTokenCookie(writer, tokens.AccessToken, services.AccessTokenDuration)
	utils.SetRefreshTokenCookie(writer, tokens.RefreshToken, services.RefreshTokenDuration)

	// Return response
	resp := dto.SigninResponse{
		Title: "Signin successful",
		Data: dto.SigninData{
			ID:           user.ID,
			Name:         user.Name,
			Email:        user.Email,
			Role:         user.Role,		
		},
	}
	utils.WriteJSON(writer, http.StatusOK, resp)

}

func ResendVerificationHandler(writer http.ResponseWriter, request *http.Request) {
	var body dto.ResendVerificationInput
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
				verifyURL := fmt.Sprintf("%s/verify?token=%s", cfg.FRONTEND_URL, token)
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

	utils.WriteJSON(writer, http.StatusOK, dto.MessageResponse{Title: "Success", Message: "Verification email resent"})
}

// RefreshTokenHandler handles token refresh using the refresh token cookie.
// Extracts refresh token, calls service for validation and rotation,
// sets new cookie, and returns new access token.
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

	// Set new refresh token cookie
	utils.SetRefreshTokenCookie(writer, newTokenPair.RefreshToken, services.RefreshTokenDuration)

	// Return new access token
	utils.WriteJSON(writer, http.StatusOK, dto.AccessTokenResponse{AccessToken: newTokenPair.AccessToken})
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

	// Clear the refresh token cookie
	http.SetCookie(writer, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1, // Instruct browser to delete cookie
		HttpOnly: true,
		Secure:   request.URL.Scheme == "https",
		SameSite: http.SameSiteLaxMode,
	})
	
	utils.ClearAccessTokenCookie(writer, request.URL.Scheme == "https")

	response := dto.LogoutResponse{
		Title:   "Success",
		Message: "You have been successfully logged out",
	}

	utils.WriteJSON(writer, http.StatusOK, response)

}

// InitiateOAuthHandler starts the OAuth flow by generating a state and setting it in a cookie.
func InitiateOAuthHandler(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	provider := vars["provider"]

	// 1. Generate state
	state, err := utils.GenerateRandomState(32)
	if err != nil {
		errors.ErrorResponse(writer, request, errors.InternalError(err))
		return
	}

	// 2. Set cookie with state
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

	// 3. For now, we return the state and provider as a mock response.
	// In a real implementation with known providers, we would construct the Redirect URL here.
	utils.WriteJSON(writer, http.StatusOK, dto.MessageResponse{
		Title:   "OAuth Initiated",
		Message: fmt.Sprintf("Provider: %s. State set in cookie.", provider),
	})
}

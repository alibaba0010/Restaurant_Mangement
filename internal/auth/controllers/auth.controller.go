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

// sendAuthResponse is a helper to centralize setting cookies and writing the final success response.
func sendAuthResponse(writer http.ResponseWriter, user *models.User, tokens *services.TokenPair, title string) {
	utils.SetAuthCookies(writer, tokens.AccessToken, tokens.RefreshToken, services.AccessTokenDuration, services.RefreshTokenDuration)

	resp := dto.SigninResponse{
		Title: title,
		Data: dto.SigninData{
			ID:        user.ID,
			Name:      user.Name,
			Email:     user.Email,
			Role:      user.Role,
			Address:   user.Address,
			AvatarURL: user.AvatarURL,
			PhoneNumber: user.PhoneNumber,
			Status:    user.Status,
			CreatedAt: user.CreatedAt.Format(time.RFC3339),
			UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
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
	// set cookies and return response
	sendAuthResponse(writer, user, tokens, "Signin successful")
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
				go func() {
					if err := services.SendEmail(email, "Verify your email", html); err != nil {
						errors.ErrorResponse(writer, request, errors.InternalError(err))
						return
					}
				}()
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

	// Set new cookies
	utils.SetAuthCookies(writer, newTokenPair.AccessToken, newTokenPair.RefreshToken, services.AccessTokenDuration, services.RefreshTokenDuration)

	// Return new access token in body (optional if frontend uses cookies, but kept for compatibility)
	utils.WriteJSON(writer, http.StatusOK, dto.MessageResponse{Title: "Success", Message: "Token refreshed successfully"})
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
		Title:   "Success",
		Message: "You have been successfully logged out",
	})
}

// InitiateOAuthHandler starts the OAuth flow
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

	// 3. Get Auth URL
	var authURL string
	switch provider {
	case "google":
		authURL = services.GetGoogleAuthURL(state)
	case "facebook":
		authURL = services.GetFacebookAuthURL(state)
	default:
		errors.ErrorResponse(writer, request, errors.ValidationError("unsupported provider"))
		return
	}

	utils.WriteJSON(writer, http.StatusOK, dto.OAuthLoginResponse{
		URL: authURL,
	})
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

	// 2. Exchange Code
	var email, name, picture string
	
	switch provider {
	case "google":
		gUser, err := services.ExchangeGoogleCode(input.Code)
		if err != nil {
			errors.ErrorResponse(writer, request, errors.UnauthorizedError(err.Error()))
			return
		}
		email = gUser.Email
		name = gUser.Name
		picture = gUser.Picture
	default:
		errors.ErrorResponse(writer, request, errors.ValidationError("unsupported provider"))
		return		
	}

	ip := utils.ExtractClientIP(request)
	ua := request.Header.Get("User-Agent")
	
	user, tokens, appErr := services.OAuthLogin(request.Context(), email, name, picture, ip, ua)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	sendAuthResponse(writer, user, tokens, "Login successful")
}

package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/alibaba0010/postgres-api/internal/auth/models"
	"github.com/alibaba0010/postgres-api/internal/auth/repositories"
	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/alibaba0010/postgres-api/internal/config"
	"github.com/alibaba0010/postgres-api/internal/utils"
)

// GetGoogleAuthURL generates the Google OAuth URL
func GetGoogleAuthURL(state string) string {
	cfg := config.LoadConfig()
	v := url.Values{}
	v.Set("client_id", cfg.GOOGLE_CLIENT_ID)
	v.Set("redirect_uri", cfg.GOOGLE_REDIRECT_URI)
	v.Set("response_type", "code")
	v.Set("scope", "https://www.googleapis.com/auth/userinfo.profile https://www.googleapis.com/auth/userinfo.email")
	v.Set("state", state)
	v.Set("access_type", "offline")
	v.Set("prompt", "consent")

	return "https://accounts.google.com/o/oauth2/v2/auth?" + v.Encode()
}

// GetFacebookAuthURL generates the Facebook OAuth URL
func GetFacebookAuthURL(state string) string {
	cfg := config.LoadConfig()
	v := url.Values{}
	v.Set("client_id", cfg.FACEBOOK_CLIENT_ID)
	// Facebook redirects to frontend callback
	v.Set("redirect_uri", cfg.FRONTEND_URL+"/auth/callback")
	v.Set("state", state)
	v.Set("scope", "email,public_profile")

	return "https://www.facebook.com/v12.0/dialog/oauth?" + v.Encode()
}

// ExchangeGoogleCode exchanges the authorization code for an access token and user info
func ExchangeGoogleCode(code string) (*GoogleUser, error) {
	cfg := config.LoadConfig()

	// 1. Exchange code for token
	tokenURL := "https://oauth2.googleapis.com/token"
	v := url.Values{}
	v.Set("client_id", cfg.GOOGLE_CLIENT_ID)
	v.Set("client_secret", cfg.GOOGLE_CLIENT_SECRET)
	v.Set("code", code)
	v.Set("grant_type", "authorization_code")
	v.Set("redirect_uri", cfg.GOOGLE_REDIRECT_URI)

	resp, err := http.PostForm(tokenURL, v)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("google token exchange failed: %s", string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		IdToken     string `json:"id_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	// 2. Get user info
	userInfoURL := "https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + tokenResp.AccessToken
	respInfo, err := http.Get(userInfoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer respInfo.Body.Close()

	var googleUser GoogleUser
	if err := json.NewDecoder(respInfo.Body).Decode(&googleUser); err != nil {
		return nil, err
	}
	return &googleUser, nil
}

type GoogleUser struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
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

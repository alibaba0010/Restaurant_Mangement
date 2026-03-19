package utils

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/config"
	"go.uber.org/zap"
)

const turnstileVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

// TurnstileResponse represents the response from Cloudflare's Turnstile verification API
type TurnstileResponse struct {
	Success     bool      `json:"success"`
	ChallengeTS time.Time `json:"challenge_ts"`
	Hostname    string    `json:"hostname"`
	ErrorCodes  []string  `json:"error-codes"`
	Action      string    `json:"action"`
	Cdata       string    `json:"cdata"`
}

var (
	turnstileHttpClient = &http.Client{
		Timeout: 5 * time.Second,
	}
)

// VerifyTurnstileToken verifies the provided token with Cloudflare Turnstile API
func VerifyTurnstileToken(token string, remoteIP string) (bool, error) {
	cfg := config.LoadConfig()
	
	// If Turnstile is not configured or we are in a non-production environment where it's optional, 
	// we might want to skip. But for now, let's assume if keys are empty we skip and log warning.
	if cfg.TURNSTILE_SECRET_KEY == "" {
		logger.Log.Warn("Turnstile secret key is not configured, skipping verification")
		return true, nil
	}

	data := url.Values{}
	data.Set("secret", cfg.TURNSTILE_SECRET_KEY)
	data.Set("response", token)
	if remoteIP != "" {
		data.Set("remoteip", remoteIP)
	}

	resp, err := turnstileHttpClient.PostForm(turnstileVerifyURL, data)
	if err != nil {
		logger.Log.Error("failed to call Turnstile API", zap.Error(err))
		return false, err
	}
	defer resp.Body.Close()

	var result TurnstileResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		logger.Log.Error("failed to decode Turnstile response", zap.Error(err))
		return false, err
	}

	if !result.Success {
		logger.Log.Warn("Turnstile verification failed", 
			zap.Strings("error_codes", result.ErrorCodes),
			zap.String("remote_ip", remoteIP))
	}

	return result.Success, nil
}

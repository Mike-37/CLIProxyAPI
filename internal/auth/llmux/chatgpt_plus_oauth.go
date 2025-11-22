package llmux

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/util"
	log "github.com/sirupsen/logrus"
)

// TODO: These constants need to be updated with actual LLMux OAuth endpoints
const (
	llmuxChatGPTAuthURL  = "https://llmux.example.com/oauth/chatgpt/authorize" // TODO: Update
	llmuxChatGPTTokenURL = "https://llmux.example.com/oauth/chatgpt/token"     // TODO: Update
	llmuxChatGPTClientID = "llmux-chatgpt-client-id"                           // TODO: Update
	llmuxChatGPTRedirectURI = "http://localhost:54547/callback"                // Different port
)

// ChatGPTPlusTokenResponse represents the OAuth token response from LLMux ChatGPT Plus.
type ChatGPTPlusTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	// LLMux-specific fields
	Provider string `json:"provider"` // Should be "chatgpt-plus"
	User     struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
}

// ChatGPTPlusAuth handles LLMux ChatGPT Plus OAuth2 authentication.
type ChatGPTPlusAuth struct {
	httpClient *http.Client
	cfg        *config.Config
}

// NewChatGPTPlusAuth creates a new LLMux ChatGPT Plus authentication service.
func NewChatGPTPlusAuth(cfg *config.Config) *ChatGPTPlusAuth {
	return &ChatGPTPlusAuth{
		httpClient: util.SetProxy(&cfg.SDKConfig, &http.Client{}),
		cfg:        cfg,
	}
}

// GenerateAuthURL creates the OAuth authorization URL for ChatGPT Plus via LLMux.
func (a *ChatGPTPlusAuth) GenerateAuthURL(state string, pkceCodes *PKCECodes) (string, string, error) {
	if pkceCodes == nil {
		return "", "", fmt.Errorf("PKCE codes are required")
	}

	// Get client ID from config, or use default
	clientID := llmuxChatGPTClientID
	if a.cfg != nil && a.cfg.Providers.LLMux.ChatGPTPlus.OAuth.ClientID != "" {
		clientID = a.cfg.Providers.LLMux.ChatGPTPlus.OAuth.ClientID
	}

	// Get redirect port from config, or use default
	redirectURI := llmuxChatGPTRedirectURI
	if a.cfg != nil && a.cfg.Providers.LLMux.ChatGPTPlus.OAuth.RedirectPort != 0 {
		redirectURI = fmt.Sprintf("http://localhost:%d/callback", a.cfg.Providers.LLMux.ChatGPTPlus.OAuth.RedirectPort)
	}

	params := url.Values{
		"client_id":             {clientID},
		"response_type":         {"code"},
		"redirect_uri":          {redirectURI},
		"scope":                 {"chatgpt.read chatgpt.write user.profile"},
		"code_challenge":        {pkceCodes.CodeChallenge},
		"code_challenge_method": {"S256"},
		"state":                 {state},
	}

	authURL := fmt.Sprintf("%s?%s", llmuxChatGPTAuthURL, params.Encode())
	log.WithFields(log.Fields{
		"provider": "llmux-chatgpt",
		"url":      authURL,
	}).Debug("Generated LLMux ChatGPT Plus auth URL")

	return authURL, state, nil
}

// ExchangeCodeForToken exchanges an authorization code for access and refresh tokens.
func (a *ChatGPTPlusAuth) ExchangeCodeForToken(ctx context.Context, code string, pkceCodes *PKCECodes) (*ChatGPTPlusTokenResponse, error) {
	if pkceCodes == nil {
		return nil, fmt.Errorf("PKCE codes are required for token exchange")
	}

	// Get client ID from config
	clientID := llmuxChatGPTClientID
	if a.cfg != nil && a.cfg.Providers.LLMux.ChatGPTPlus.OAuth.ClientID != "" {
		clientID = a.cfg.Providers.LLMux.ChatGPTPlus.OAuth.ClientID
	}

	// Get redirect URI
	redirectURI := llmuxChatGPTRedirectURI
	if a.cfg != nil && a.cfg.Providers.LLMux.ChatGPTPlus.OAuth.RedirectPort != 0 {
		redirectURI = fmt.Sprintf("http://localhost:%d/callback", a.cfg.Providers.LLMux.ChatGPTPlus.OAuth.RedirectPort)
	}

	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {clientID},
		"redirect_uri":  {redirectURI},
		"code_verifier": {pkceCodes.CodeVerifier},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, llmuxChatGPTTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	log.WithFields(log.Fields{
		"provider": "llmux-chatgpt",
		"url":      llmuxChatGPTTokenURL,
	}).Debug("Exchanging code for token")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"provider":    "llmux-chatgpt",
			"status_code": resp.StatusCode,
			"body":        string(body),
		}).Error("Token exchange failed")
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp ChatGPTPlusTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	log.WithFields(log.Fields{
		"provider":   "llmux-chatgpt",
		"user_email": tokenResp.User.Email,
		"expires_in": tokenResp.ExpiresIn,
	}).Info("Successfully obtained LLMux ChatGPT Plus tokens")

	return &tokenResp, nil
}

// RefreshToken refreshes an expired access token using a refresh token.
func (a *ChatGPTPlusAuth) RefreshToken(ctx context.Context, refreshToken string) (*ChatGPTPlusTokenResponse, error) {
	// Get client ID from config
	clientID := llmuxChatGPTClientID
	if a.cfg != nil && a.cfg.Providers.LLMux.ChatGPTPlus.OAuth.ClientID != "" {
		clientID = a.cfg.Providers.LLMux.ChatGPTPlus.OAuth.ClientID
	}

	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {clientID},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, llmuxChatGPTTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	log.WithFields(log.Fields{
		"provider": "llmux-chatgpt",
	}).Debug("Refreshing access token")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read refresh response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"provider":    "llmux-chatgpt",
			"status_code": resp.StatusCode,
			"body":        string(body),
		}).Error("Token refresh failed")
		return nil, fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp ChatGPTPlusTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse refresh response: %w", err)
	}

	log.WithFields(log.Fields{
		"provider":   "llmux-chatgpt",
		"expires_in": tokenResp.ExpiresIn,
	}).Info("Successfully refreshed LLMux ChatGPT Plus token")

	return &tokenResp, nil
}

// CalculateExpiryTime calculates the absolute expiry time from ExpiresIn seconds.
func (a *ChatGPTPlusAuth) CalculateExpiryTime(expiresIn int) time.Time {
	// Subtract 5 minutes buffer to refresh before actual expiry
	bufferSeconds := 300
	if expiresIn > bufferSeconds {
		expiresIn -= bufferSeconds
	}
	return time.Now().Add(time.Duration(expiresIn) * time.Second)
}

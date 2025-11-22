// Package llmux provides OAuth2 authentication for LLMux providers.
// This package implements OAuth flows for accessing Claude Pro and ChatGPT Plus
// subscriptions through LLMux's unified OAuth interface.
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
// Once the LLMux OAuth service is deployed or documented
const (
	llmuxClaudeAuthURL  = "https://llmux.example.com/oauth/claude/authorize"  // TODO: Update with actual URL
	llmuxClaudeTokenURL = "https://llmux.example.com/oauth/claude/token"      // TODO: Update with actual URL
	llmuxClaudeClientID = "llmux-claude-client-id"                            // TODO: Update with actual client ID
	llmuxClaudeRedirectURI = "http://localhost:54546/callback"                // Different port from regular Claude
)

// ClaudeProTokenResponse represents the OAuth token response from LLMux Claude Pro.
type ClaudeProTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	// LLMux-specific fields
	Provider string `json:"provider"` // Should be "claude-pro"
	User     struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
}

// ClaudeProAuth handles LLMux Claude Pro OAuth2 authentication.
type ClaudeProAuth struct {
	httpClient *http.Client
	cfg        *config.Config
}

// NewClaudeProAuth creates a new LLMux Claude Pro authentication service.
func NewClaudeProAuth(cfg *config.Config) *ClaudeProAuth {
	return &ClaudeProAuth{
		httpClient: util.SetProxy(&cfg.SDKConfig, &http.Client{}),
		cfg:        cfg,
	}
}

// GenerateAuthURL creates the OAuth authorization URL for Claude Pro via LLMux.
//
// Parameters:
//   - state: A random state parameter for CSRF protection
//   - pkceCodes: The PKCE codes for secure code exchange
//
// Returns:
//   - string: The complete authorization URL
//   - string: The state parameter for verification
//   - error: An error if generation fails
func (a *ClaudeProAuth) GenerateAuthURL(state string, pkceCodes *PKCECodes) (string, string, error) {
	if pkceCodes == nil {
		return "", "", fmt.Errorf("PKCE codes are required")
	}

	// Get client ID from config, or use default
	clientID := llmuxClaudeClientID
	if a.cfg != nil && a.cfg.Providers.LLMux.ClaudePro.OAuth.ClientID != "" {
		clientID = a.cfg.Providers.LLMux.ClaudePro.OAuth.ClientID
	}

	// Get redirect port from config, or use default
	redirectURI := llmuxClaudeRedirectURI
	if a.cfg != nil && a.cfg.Providers.LLMux.ClaudePro.OAuth.RedirectPort != 0 {
		redirectURI = fmt.Sprintf("http://localhost:%d/callback", a.cfg.Providers.LLMux.ClaudePro.OAuth.RedirectPort)
	}

	params := url.Values{
		"client_id":             {clientID},
		"response_type":         {"code"},
		"redirect_uri":          {redirectURI},
		"scope":                 {"claude.read claude.write user.profile"},
		"code_challenge":        {pkceCodes.CodeChallenge},
		"code_challenge_method": {"S256"},
		"state":                 {state},
	}

	authURL := fmt.Sprintf("%s?%s", llmuxClaudeAuthURL, params.Encode())
	log.WithFields(log.Fields{
		"provider": "llmux-claude",
		"url":      authURL,
	}).Debug("Generated LLMux Claude Pro auth URL")

	return authURL, state, nil
}

// ExchangeCodeForToken exchanges an authorization code for access and refresh tokens.
//
// Parameters:
//   - ctx: Context for the request
//   - code: The authorization code from the callback
//   - pkceCodes: The PKCE codes used in the authorization request
//
// Returns:
//   - *ClaudeProTokenResponse: The token response containing access and refresh tokens
//   - error: An error if the exchange fails
func (a *ClaudeProAuth) ExchangeCodeForToken(ctx context.Context, code string, pkceCodes *PKCECodes) (*ClaudeProTokenResponse, error) {
	if pkceCodes == nil {
		return nil, fmt.Errorf("PKCE codes are required for token exchange")
	}

	// Get client ID from config
	clientID := llmuxClaudeClientID
	if a.cfg != nil && a.cfg.Providers.LLMux.ClaudePro.OAuth.ClientID != "" {
		clientID = a.cfg.Providers.LLMux.ClaudePro.OAuth.ClientID
	}

	// Get redirect URI
	redirectURI := llmuxClaudeRedirectURI
	if a.cfg != nil && a.cfg.Providers.LLMux.ClaudePro.OAuth.RedirectPort != 0 {
		redirectURI = fmt.Sprintf("http://localhost:%d/callback", a.cfg.Providers.LLMux.ClaudePro.OAuth.RedirectPort)
	}

	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {clientID},
		"redirect_uri":  {redirectURI},
		"code_verifier": {pkceCodes.CodeVerifier},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, llmuxClaudeTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	log.WithFields(log.Fields{
		"provider": "llmux-claude",
		"url":      llmuxClaudeTokenURL,
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
			"provider":    "llmux-claude",
			"status_code": resp.StatusCode,
			"body":        string(body),
		}).Error("Token exchange failed")
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp ClaudeProTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	log.WithFields(log.Fields{
		"provider":   "llmux-claude",
		"user_email": tokenResp.User.Email,
		"expires_in": tokenResp.ExpiresIn,
	}).Info("Successfully obtained LLMux Claude Pro tokens")

	return &tokenResp, nil
}

// RefreshToken refreshes an expired access token using a refresh token.
//
// Parameters:
//   - ctx: Context for the request
//   - refreshToken: The refresh token to use
//
// Returns:
//   - *ClaudeProTokenResponse: The new token response
//   - error: An error if the refresh fails
func (a *ClaudeProAuth) RefreshToken(ctx context.Context, refreshToken string) (*ClaudeProTokenResponse, error) {
	// Get client ID from config
	clientID := llmuxClaudeClientID
	if a.cfg != nil && a.cfg.Providers.LLMux.ClaudePro.OAuth.ClientID != "" {
		clientID = a.cfg.Providers.LLMux.ClaudePro.OAuth.ClientID
	}

	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {clientID},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, llmuxClaudeTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	log.WithFields(log.Fields{
		"provider": "llmux-claude",
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
			"provider":    "llmux-claude",
			"status_code": resp.StatusCode,
			"body":        string(body),
		}).Error("Token refresh failed")
		return nil, fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp ClaudeProTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse refresh response: %w", err)
	}

	log.WithFields(log.Fields{
		"provider":   "llmux-claude",
		"expires_in": tokenResp.ExpiresIn,
	}).Info("Successfully refreshed LLMux Claude Pro token")

	return &tokenResp, nil
}

// CalculateExpiryTime calculates the absolute expiry time from ExpiresIn seconds.
func (a *ClaudeProAuth) CalculateExpiryTime(expiresIn int) time.Time {
	// Subtract 5 minutes buffer to refresh before actual expiry
	bufferSeconds := 300
	if expiresIn > bufferSeconds {
		expiresIn -= bufferSeconds
	}
	return time.Now().Add(time.Duration(expiresIn) * time.Second)
}

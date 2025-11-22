package llmux

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	// OpenAI OAuth endpoints
	openaiOAuthAuthURL   = "https://auth.openai.com/authorize"
	openaiOAuthTokenURL  = "https://auth.openai.com/oauth/token"
	openaiOAuthRevokeURL = "https://auth.openai.com/oauth/revoke"

	// OpenAI OAuth scope for ChatGPT
	openaiOAuthScope = "openid profile email offline_access"
)

// ChatGPTPlusOAuthConfig holds the OAuth configuration for ChatGPT Plus
type ChatGPTPlusOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	HTTPClient   *http.Client
}

// ChatGPTPlusToken represents an OpenAI/ChatGPT OAuth token
type ChatGPTPlusToken struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int64     `json:"expires_in"`
	RefreshToken string    `json:"refresh_token"`
	Scope        string    `json:"scope"`
	ExpiresAt    time.Time `json:"-"`
}

// IsExpired checks if the token is expired
func (t *ChatGPTPlusToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// ChatGPTPlusOAuth handles the OAuth flow for ChatGPT Plus
type ChatGPTPlusOAuth struct {
	config *ChatGPTPlusOAuthConfig
}

// NewChatGPTPlusOAuth creates a new ChatGPT Plus OAuth handler
func NewChatGPTPlusOAuth(config *ChatGPTPlusOAuthConfig) *ChatGPTPlusOAuth {
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &ChatGPTPlusOAuth{config: config}
}

// GetAuthorizationURL returns the URL to redirect the user to for authorization
func (o *ChatGPTPlusOAuth) GetAuthorizationURL(state string) string {
	params := url.Values{
		"client_id":     {o.config.ClientID},
		"redirect_uri":  {o.config.RedirectURI},
		"response_type": {"code"},
		"scope":         {openaiOAuthScope},
		"state":         {state},
	}
	return openaiOAuthAuthURL + "?" + params.Encode()
}

// ExchangeCodeForToken exchanges an authorization code for an access token
func (o *ChatGPTPlusOAuth) ExchangeCodeForToken(ctx context.Context, code string) (*ChatGPTPlusToken, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {o.config.ClientID},
		"client_secret": {o.config.ClientSecret},
		"redirect_uri":  {o.config.RedirectURI},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", openaiOAuthTokenURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	req.PostForm = data
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := o.config.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	var token ChatGPTPlusToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	return &token, nil
}

// RefreshToken refreshes an access token using a refresh token
func (o *ChatGPTPlusOAuth) RefreshToken(ctx context.Context, refreshToken string) (*ChatGPTPlusToken, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {o.config.ClientID},
		"client_secret": {o.config.ClientSecret},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", openaiOAuthTokenURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh token request: %w", err)
	}

	req.PostForm = data
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := o.config.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token refresh failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	var token ChatGPTPlusToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("failed to decode refresh token response: %w", err)
	}

	token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	return &token, nil
}

// RevokeToken revokes an access token
func (o *ChatGPTPlusOAuth) RevokeToken(ctx context.Context, accessToken string) error {
	data := url.Values{
		"token":         {accessToken},
		"client_id":     {o.config.ClientID},
		"client_secret": {o.config.ClientSecret},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", openaiOAuthRevokeURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create revoke token request: %w", err)
	}

	req.PostForm = data
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := o.config.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token revocation failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

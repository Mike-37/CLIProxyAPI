package ctonew

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	// Clerk API endpoint for token exchange
	clerkTokenExchangeURL = "https://api.clerk.com/oauth/token"

	// Default cache TTL for exchanged tokens
	defaultTokenCacheTTL = 5 * time.Minute
)

// TokenExchangeRequest represents a token exchange request to Clerk
type TokenExchangeRequest struct {
	GrantType     string `json:"grant_type"`
	ClientID      string `json:"client_id"`
	ClientSecret  string `json:"client_secret"`
	RotatingToken string `json:"rotating_token"`
	Audience      string `json:"audience,omitempty"`
}

// TokenExchangeResponse represents a token exchange response from Clerk
type TokenExchangeResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// CachedToken represents a cached token with expiry
type CachedToken struct {
	Token     *TokenExchangeResponse
	ExpiresAt time.Time
}

// IsExpired checks if the cached token is expired
func (ct *CachedToken) IsExpired() bool {
	// Consider token expired 30 seconds before actual expiry for safety
	return time.Now().Add(30 * time.Second).After(ct.ExpiresAt)
}

// ClerkTokenExchange handles token exchange with Clerk API
type ClerkTokenExchange struct {
	clientID      string
	clientSecret  string
	httpClient    *http.Client
	jwtParser     *ClerkJWTParser
	cache         map[string]*CachedToken
	cacheMutex    sync.RWMutex
	cacheTTL      time.Duration
}

// NewClerkTokenExchange creates a new Clerk token exchange client
func NewClerkTokenExchange(clientID, clientSecret string, httpClient *http.Client) *ClerkTokenExchange {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	return &ClerkTokenExchange{
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   httpClient,
		jwtParser:    NewClerkJWTParser(),
		cache:        make(map[string]*CachedToken),
		cacheTTL:     defaultTokenCacheTTL,
	}
}

// ExchangeToken exchanges a rotating token for a new access token
func (c *ClerkTokenExchange) ExchangeToken(ctx context.Context, clerkJWT string) (*TokenExchangeResponse, error) {
	// Extract rotating token from JWT
	rotatingToken, err := c.jwtParser.ExtractRotatingToken(clerkJWT)
	if err != nil {
		return nil, fmt.Errorf("failed to extract rotating token: %w", err)
	}

	// Check cache
	cacheKey := rotatingToken // Use rotating token as cache key
	c.cacheMutex.RLock()
	if cached, ok := c.cache[cacheKey]; ok && !cached.IsExpired() {
		c.cacheMutex.RUnlock()
		return cached.Token, nil
	}
	c.cacheMutex.RUnlock()

	// Create exchange request
	exchangeReq := TokenExchangeRequest{
		GrantType:     "urn:ietf:params:oauth:grant-type:token-exchange",
		ClientID:      c.clientID,
		ClientSecret:  c.clientSecret,
		RotatingToken: rotatingToken,
		Audience:      "https://api.enginelabs.ai", // ctonew API audience
	}

	// Marshal request
	reqBody, err := json.Marshal(exchangeReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal exchange request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", clerkTokenExchangeURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "CLIProxyAPI/1.0")

	// Make request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange token: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	// Unmarshal response
	var exchangeResp TokenExchangeResponse
	if err := json.Unmarshal(respBody, &exchangeResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Cache the token
	expiresAt := time.Now().Add(time.Duration(exchangeResp.ExpiresIn) * time.Second)
	c.cacheMutex.Lock()
	c.cache[cacheKey] = &CachedToken{
		Token:     &exchangeResp,
		ExpiresAt: expiresAt,
	}
	c.cacheMutex.Unlock()

	return &exchangeResp, nil
}

// GetCachedToken returns a cached token if available and not expired
func (c *ClerkTokenExchange) GetCachedToken(rotatingToken string) (*TokenExchangeResponse, bool) {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	cached, ok := c.cache[rotatingToken]
	if !ok || cached.IsExpired() {
		return nil, false
	}

	return cached.Token, true
}

// ClearCache clears the token cache
func (c *ClerkTokenExchange) ClearCache() {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	c.cache = make(map[string]*CachedToken)
}

// SetCacheTTL sets the cache time-to-live
func (c *ClerkTokenExchange) SetCacheTTL(ttl time.Duration) {
	c.cacheTTL = ttl
}

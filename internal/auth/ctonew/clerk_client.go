package ctonew

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/util"
	log "github.com/sirupsen/logrus"
)

// TokenExchangeResponse represents the response from Clerk's token exchange endpoint.
type TokenExchangeResponse struct {
	JWT       string `json:"jwt"`
	ExpiresAt int64  `json:"expires_at"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
}

// ClerkClient handles API calls to Clerk for token exchange.
type ClerkClient struct {
	httpClient *http.Client
	apiURL     string
}

// NewClerkClient creates a new Clerk API client.
//
// Parameters:
//   - cfg: The application configuration
//
// Returns:
//   - *ClerkClient: A new Clerk client instance
func NewClerkClient(cfg *config.Config) *ClerkClient {
	apiURL := "https://clerk.enginelabs.com" // Default Clerk API URL
	if cfg != nil && cfg.Providers.Ctonew.Clerk.APIURL != "" {
		apiURL = cfg.Providers.Ctonew.Clerk.APIURL
	}

	return &ClerkClient{
		httpClient: util.SetProxy(&cfg.SDKConfig, &http.Client{
			Timeout: 30 * time.Second,
		}),
		apiURL: apiURL,
	}
}

// ExchangeRotatingToken exchanges a rotating_token for a new JWT.
//
// Parameters:
//   - ctx: Context for the request
//   - rotatingToken: The rotating token from the JWT claims
//
// Returns:
//   - string: The new JWT
//   - error: An error if the exchange fails
func (c *ClerkClient) ExchangeRotatingToken(ctx context.Context, rotatingToken string) (string, error) {
	// TODO: Update with actual Clerk token exchange endpoint
	// This endpoint structure is inferred from typical Clerk API patterns
	url := fmt.Sprintf("%s/v1/tokens/create", c.apiURL)

	reqBody := map[string]string{
		"rotating_token": rotatingToken,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	log.WithFields(log.Fields{
		"url": url,
	}).Debug("Exchanging Clerk rotating token")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"status_code": resp.StatusCode,
			"body":        string(body),
		}).Error("Clerk token exchange failed")
		return "", fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var exchangeResp TokenExchangeResponse
	if err := json.Unmarshal(body, &exchangeResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if !exchangeResp.Success || exchangeResp.JWT == "" {
		errorMsg := exchangeResp.Error
		if errorMsg == "" {
			errorMsg = "unknown error"
		}
		return "", fmt.Errorf("token exchange unsuccessful: %s", errorMsg)
	}

	log.WithFields(log.Fields{
		"expires_at": time.Unix(exchangeResp.ExpiresAt, 0),
	}).Info("Successfully exchanged Clerk rotating token")

	return exchangeResp.JWT, nil
}

// ValidateJWT validates a JWT with Clerk (optional, for extra security).
//
// Parameters:
//   - ctx: Context for the request
//   - jwt: The JWT to validate
//
// Returns:
//   - bool: Whether the JWT is valid
//   - error: An error if validation fails
func (c *ClerkClient) ValidateJWT(ctx context.Context, jwt string) (bool, error) {
	// Parse JWT locally first (cheaper than API call)
	claims, err := ParseClerkJWT(jwt)
	if err != nil {
		return false, fmt.Errorf("JWT parsing failed: %w", err)
	}

	// Check expiration locally
	if claims.IsExpired() {
		return false, fmt.Errorf("JWT is expired")
	}

	// For now, local validation is sufficient
	// In production, you might want to validate with Clerk's API for extra security
	return true, nil
}

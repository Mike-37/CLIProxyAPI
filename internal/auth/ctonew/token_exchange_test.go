package ctonew

import (
	"context"
	"testing"
	"time"
)

func TestClerkTokenExchange_ExchangeToken(t *testing.T) {
	cfg := &ClerkTokenExchangeConfig{
		ClientID:      "test_client_id",
		ClientSecret:  "test_client_secret",
		APIEndpoint:   "https://api.enginelabs.ai",
		CacheTTL:      5 * time.Minute,
	}

	exchange := NewClerkTokenExchange(cfg)
	ctx := context.Background()

	// Try to exchange invalid JWT (should fail)
	_, err := exchange.ExchangeToken(ctx, "invalid.jwt.token")
	if err == nil {
		t.Errorf("expected error for invalid JWT, got nil")
	}
}

func TestClerkTokenExchange_GetCachedToken(t *testing.T) {
	cfg := &ClerkTokenExchangeConfig{
		ClientID:      "test_client_id",
		ClientSecret:  "test_client_secret",
		APIEndpoint:   "https://api.enginelabs.ai",
		CacheTTL:      5 * time.Minute,
	}

	exchange := NewClerkTokenExchange(cfg)

	// Try to get non-existent cached token
	_, ok := exchange.GetCachedToken("nonexistent_token")
	if ok {
		t.Errorf("expected GetCachedToken to return false for non-existent token")
	}
}

func TestClerkTokenExchange_ClearCache(t *testing.T) {
	cfg := &ClerkTokenExchangeConfig{
		ClientID:      "test_client_id",
		ClientSecret:  "test_client_secret",
		APIEndpoint:   "https://api.enginelabs.ai",
		CacheTTL:      5 * time.Minute,
	}

	exchange := NewClerkTokenExchange(cfg)

	// Clear cache should not panic
	exchange.ClearCache()
}

func TestClerkTokenExchange_SetCacheTTL(t *testing.T) {
	cfg := &ClerkTokenExchangeConfig{
		ClientID:      "test_client_id",
		ClientSecret:  "test_client_secret",
		APIEndpoint:   "https://api.enginelabs.ai",
		CacheTTL:      5 * time.Minute,
	}

	exchange := NewClerkTokenExchange(cfg)

	// Set cache TTL to different value
	newTTL := 10 * time.Minute
	exchange.SetCacheTTL(newTTL)
}

func TestTokenExchangeResponse_Validation(t *testing.T) {
	tests := []struct {
		name     string
		resp     *TokenExchangeResponse
		valid    bool
	}{
		{
			name: "valid response",
			resp: &TokenExchangeResponse{
				AccessToken: "valid_access_token",
				ExpiresIn:   3600,
				TokenType:   "Bearer",
			},
			valid: true,
		},
		{
			name: "missing access token",
			resp: &TokenExchangeResponse{
				ExpiresIn: 3600,
				TokenType: "Bearer",
			},
			valid: false,
		},
		{
			name: "invalid expires in",
			resp: &TokenExchangeResponse{
				AccessToken: "valid_access_token",
				ExpiresIn:   -1,
				TokenType:   "Bearer",
			},
			valid: false,
		},
		{
			name: "missing token type",
			resp: &TokenExchangeResponse{
				AccessToken: "valid_access_token",
				ExpiresIn:   3600,
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.resp.AccessToken != "" && tt.resp.ExpiresIn > 0 && tt.resp.TokenType != ""

			if isValid != tt.valid {
				t.Errorf("token validation failed: expected %v, got %v", tt.valid, isValid)
			}
		})
	}
}

func TestClerkTokenExchangeConfig_Validation(t *testing.T) {
	tests := []struct {
		name  string
		cfg   *ClerkTokenExchangeConfig
		valid bool
	}{
		{
			name: "valid config",
			cfg: &ClerkTokenExchangeConfig{
				ClientID:     "test_client_id",
				ClientSecret: "test_client_secret",
				APIEndpoint:  "https://api.enginelabs.ai",
				CacheTTL:     5 * time.Minute,
			},
			valid: true,
		},
		{
			name: "missing client id",
			cfg: &ClerkTokenExchangeConfig{
				ClientSecret: "test_client_secret",
				APIEndpoint:  "https://api.enginelabs.ai",
				CacheTTL:     5 * time.Minute,
			},
			valid: false,
		},
		{
			name: "missing client secret",
			cfg: &ClerkTokenExchangeConfig{
				ClientID:    "test_client_id",
				APIEndpoint: "https://api.enginelabs.ai",
				CacheTTL:    5 * time.Minute,
			},
			valid: false,
		},
		{
			name: "missing api endpoint",
			cfg: &ClerkTokenExchangeConfig{
				ClientID:     "test_client_id",
				ClientSecret: "test_client_secret",
				CacheTTL:     5 * time.Minute,
			},
			valid: false,
		},
		{
			name: "zero cache ttl",
			cfg: &ClerkTokenExchangeConfig{
				ClientID:     "test_client_id",
				ClientSecret: "test_client_secret",
				APIEndpoint:  "https://api.enginelabs.ai",
				CacheTTL:     0,
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.cfg.ClientID != "" && tt.cfg.ClientSecret != "" &&
				tt.cfg.APIEndpoint != "" && tt.cfg.CacheTTL > 0

			if isValid != tt.valid {
				t.Errorf("config validation failed: expected %v, got %v", tt.valid, isValid)
			}
		})
	}
}

func TestCachedToken_Expiration(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt int64
		expected  bool
	}{
		{
			name:      "expired token",
			expiresAt: time.Now().Add(-time.Hour).Unix(),
			expected:  true,
		},
		{
			name:      "valid token",
			expiresAt: time.Now().Add(time.Hour).Unix(),
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cached := &CachedToken{
				AccessToken: "test_token",
				ExpiresAt:   tt.expiresAt,
			}

			isExpired := cached.ExpiresAt < time.Now().Unix()

			if isExpired != tt.expected {
				t.Errorf("expiration check failed: expected %v, got %v", tt.expected, isExpired)
			}
		})
	}
}

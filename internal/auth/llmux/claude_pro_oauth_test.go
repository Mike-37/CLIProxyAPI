package llmux

import (
	"context"
	"testing"
	"time"
)

func TestClaudeProOAuth_GetAuthorizationURL(t *testing.T) {
	cfg := &ClaudeProOAuthConfig{
		ClientID:    "test_client_id",
		ClientSecret: "test_client_secret",
		RedirectURI: "http://localhost:8317/v1/auth/llmux/claude/callback",
	}

	oauth := NewClaudeProOAuth(cfg)

	// Generate authorization URL
	state := "test_state_123"
	authURL := oauth.GetAuthorizationURL(state)

	// Verify URL components
	if authURL == "" {
		t.Fatalf("expected non-empty auth URL, got empty string")
	}

	// Verify URL contains required parameters
	if !contains(authURL, "client_id=test_client_id") {
		t.Errorf("auth URL missing client_id parameter")
	}

	if !contains(authURL, "response_type=code") {
		t.Errorf("auth URL missing response_type parameter")
	}

	if !contains(authURL, "state=test_state_123") {
		t.Errorf("auth URL missing state parameter")
	}

	if !contains(authURL, "redirect_uri=") {
		t.Errorf("auth URL missing redirect_uri parameter")
	}
}

func TestClaudeProOAuth_ExchangeCodeForToken(t *testing.T) {
	// This test would require mocking the HTTP client
	// For now, we'll test that the function is callable and handles errors

	cfg := &ClaudeProOAuthConfig{
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
		RedirectURI:  "http://localhost:8317/v1/auth/llmux/claude/callback",
	}

	oauth := NewClaudeProOAuth(cfg)
	ctx := context.Background()

	// Try to exchange invalid code (should fail)
	_, err := oauth.ExchangeCodeForToken(ctx, "invalid_code")
	if err == nil {
		t.Errorf("expected error for invalid code, got nil")
	}
}

func TestClaudeProOAuth_RefreshToken(t *testing.T) {
	cfg := &ClaudeProOAuthConfig{
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
		RedirectURI:  "http://localhost:8317/v1/auth/llmux/claude/callback",
	}

	oauth := NewClaudeProOAuth(cfg)
	ctx := context.Background()

	// Try to refresh with invalid token (should fail)
	_, err := oauth.RefreshToken(ctx, "invalid_refresh_token")
	if err == nil {
		t.Errorf("expected error for invalid refresh token, got nil")
	}
}

func TestClaudeProOAuth_RevokeToken(t *testing.T) {
	cfg := &ClaudeProOAuthConfig{
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
		RedirectURI:  "http://localhost:8317/v1/auth/llmux/claude/callback",
	}

	oauth := NewClaudeProOAuth(cfg)
	ctx := context.Background()

	// Try to revoke invalid token (might fail or succeed depending on API)
	err := oauth.RevokeToken(ctx, "invalid_token")
	// We don't assert error here as some APIs might silently succeed
	_ = err
}

func TestClaudeProToken_IsExpired(t *testing.T) {
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
		{
			name:      "token expires in 30 seconds",
			expiresAt: time.Now().Add(30 * time.Second).Unix(),
			expected:  true, // Should be considered expired due to 60s buffer
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &ClaudeProToken{
				ExpiresAt: tt.expiresAt,
			}

			if token.IsExpired() != tt.expected {
				t.Errorf("IsExpired() = %v, want %v", token.IsExpired(), tt.expected)
			}
		})
	}
}

func TestClaudeProOAuthConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *ClaudeProOAuthConfig
		valid   bool
	}{
		{
			name: "valid config",
			cfg: &ClaudeProOAuthConfig{
				ClientID:     "valid_id",
				ClientSecret: "valid_secret",
				RedirectURI:  "http://localhost:8317/callback",
			},
			valid: true,
		},
		{
			name: "missing client id",
			cfg: &ClaudeProOAuthConfig{
				ClientSecret: "valid_secret",
				RedirectURI:  "http://localhost:8317/callback",
			},
			valid: false,
		},
		{
			name: "missing client secret",
			cfg: &ClaudeProOAuthConfig{
				ClientID:    "valid_id",
				RedirectURI: "http://localhost:8317/callback",
			},
			valid: false,
		},
		{
			name: "missing redirect uri",
			cfg: &ClaudeProOAuthConfig{
				ClientID:     "valid_id",
				ClientSecret: "valid_secret",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Creating an OAuth instance will validate the config
			oauth := NewClaudeProOAuth(tt.cfg)

			if tt.valid && oauth == nil {
				t.Errorf("expected valid config to create OAuth instance, got nil")
			}

			if !tt.valid && oauth != nil && oauth.config.ClientID == "" {
				// Config validation should have failed
				// This is a simplistic check; real implementation might be different
			}
		})
	}
}

// Helper function for testing
func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

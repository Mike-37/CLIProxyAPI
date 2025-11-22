package llmux

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTokenStorage_SaveAndGetToken(t *testing.T) {
	// Create temporary directory for token storage
	tmpDir, err := os.MkdirTemp("", "token_storage_test")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage := NewTokenStorage(tmpDir)

	// Create a test token
	token := &ClaudeProToken{
		AccessToken:  "test_access_token",
		RefreshToken: "test_refresh_token",
		ExpiresAt:    time.Now().Add(time.Hour).Unix(),
		TokenType:    "Bearer",
	}

	// Save token
	err = storage.SaveToken("claude", "user@example.com", token)
	if err != nil {
		t.Fatalf("failed to save token: %v", err)
	}

	// Retrieve token
	retrieved, err := storage.GetToken("claude", "user@example.com")
	if err != nil {
		t.Fatalf("failed to get token: %v", err)
	}

	// Verify token
	retrievedToken, ok := retrieved.(*ClaudeProToken)
	if !ok {
		t.Fatalf("expected *ClaudeProToken, got %T", retrieved)
	}

	if retrievedToken.AccessToken != token.AccessToken {
		t.Errorf("access token mismatch: expected %s, got %s", token.AccessToken, retrievedToken.AccessToken)
	}

	if retrievedToken.RefreshToken != token.RefreshToken {
		t.Errorf("refresh token mismatch: expected %s, got %s", token.RefreshToken, retrievedToken.RefreshToken)
	}
}

func TestTokenStorage_TokenNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "token_storage_test")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage := NewTokenStorage(tmpDir)

	// Try to retrieve non-existent token
	_, err = storage.GetToken("claude", "nonexistent@example.com")
	if err == nil {
		t.Fatalf("expected error for non-existent token, got nil")
	}
}

func TestTokenStorage_DeleteToken(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "token_storage_test")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage := NewTokenStorage(tmpDir)

	// Save token
	token := &ClaudeProToken{
		AccessToken:  "test_access_token",
		RefreshToken: "test_refresh_token",
		ExpiresAt:    time.Now().Add(time.Hour).Unix(),
		TokenType:    "Bearer",
	}

	err = storage.SaveToken("claude", "user@example.com", token)
	if err != nil {
		t.Fatalf("failed to save token: %v", err)
	}

	// Delete token
	err = storage.DeleteToken("claude", "user@example.com")
	if err != nil {
		t.Fatalf("failed to delete token: %v", err)
	}

	// Verify deletion
	_, err = storage.GetToken("claude", "user@example.com")
	if err == nil {
		t.Fatalf("expected error for deleted token, got nil")
	}
}

func TestTokenStorage_Encryption(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "token_storage_test")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create storage with encryption
	storage := NewTokenStorage(tmpDir)

	// Set encryption key (32 bytes for AES-256)
	encKey := []byte("12345678901234567890123456789012")
	err = storage.SetEncryptionKey(encKey)
	if err != nil {
		t.Fatalf("failed to set encryption key: %v", err)
	}

	// Save token
	token := &ClaudeProToken{
		AccessToken:  "secret_access_token",
		RefreshToken: "secret_refresh_token",
		ExpiresAt:    time.Now().Add(time.Hour).Unix(),
		TokenType:    "Bearer",
	}

	err = storage.SaveToken("claude", "user@example.com", token)
	if err != nil {
		t.Fatalf("failed to save encrypted token: %v", err)
	}

	// Verify token file exists
	tokenPath := filepath.Join(tmpDir, "claude", "user@example.com.json")
	if _, err := os.Stat(tokenPath); err != nil {
		t.Fatalf("token file not found: %v", err)
	}

	// Retrieve and verify encryption
	retrieved, err := storage.GetToken("claude", "user@example.com")
	if err != nil {
		t.Fatalf("failed to get encrypted token: %v", err)
	}

	retrievedToken, ok := retrieved.(*ClaudeProToken)
	if !ok {
		t.Fatalf("expected *ClaudeProToken, got %T", retrieved)
	}

	if retrievedToken.AccessToken != token.AccessToken {
		t.Errorf("encrypted token access token mismatch")
	}
}

func TestTokenStorage_MultipleUsers(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "token_storage_test")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage := NewTokenStorage(tmpDir)

	// Save tokens for multiple users
	token1 := &ClaudeProToken{
		AccessToken:  "token1_access",
		RefreshToken: "token1_refresh",
		ExpiresAt:    time.Now().Add(time.Hour).Unix(),
		TokenType:    "Bearer",
	}

	token2 := &ClaudeProToken{
		AccessToken:  "token2_access",
		RefreshToken: "token2_refresh",
		ExpiresAt:    time.Now().Add(time.Hour).Unix(),
		TokenType:    "Bearer",
	}

	err = storage.SaveToken("claude", "user1@example.com", token1)
	if err != nil {
		t.Fatalf("failed to save user1 token: %v", err)
	}

	err = storage.SaveToken("claude", "user2@example.com", token2)
	if err != nil {
		t.Fatalf("failed to save user2 token: %v", err)
	}

	// Retrieve and verify
	retrieved1, err := storage.GetToken("claude", "user1@example.com")
	if err != nil {
		t.Fatalf("failed to get user1 token: %v", err)
	}

	retrieved2, err := storage.GetToken("claude", "user2@example.com")
	if err != nil {
		t.Fatalf("failed to get user2 token: %v", err)
	}

	token1Retrieved := retrieved1.(*ClaudeProToken)
	token2Retrieved := retrieved2.(*ClaudeProToken)

	if token1Retrieved.AccessToken != token1.AccessToken {
		t.Errorf("user1 token mismatch")
	}

	if token2Retrieved.AccessToken != token2.AccessToken {
		t.Errorf("user2 token mismatch")
	}
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
			name:      "token expiring soon",
			expiresAt: time.Now().Add(30 * time.Second).Unix(),
			expected:  true, // Should be expired due to buffer
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

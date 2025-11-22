package llmux

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TokenStorage manages persistent storage of OAuth tokens
type TokenStorage struct {
	basePath   string
	mu         sync.RWMutex
	cache      map[string]*storedToken
	cacheTime  map[string]time.Time
	cacheTTL   time.Duration
	encryptKey []byte // Optional encryption key
}

// storedToken represents a token with metadata
type storedToken struct {
	Provider    string    `json:"provider"`
	UserEmail   string    `json:"user_email"`
	AccessToken string    `json:"access_token"`
	RefreshToken string   `json:"refresh_token,omitempty"`
	TokenType   string    `json:"token_type"`
	ExpiresAt   time.Time `json:"expires_at"`
	Scope       string    `json:"scope,omitempty"`
	StoredAt    time.Time `json:"stored_at"`
}

// NewTokenStorage creates a new token storage instance
func NewTokenStorage(basePath string) (*TokenStorage, error) {
	if err := os.MkdirAll(basePath, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create token storage directory: %w", err)
	}

	return &TokenStorage{
		basePath:  basePath,
		cache:     make(map[string]*storedToken),
		cacheTime: make(map[string]time.Time),
		cacheTTL:  5 * time.Minute, // 5-minute cache
	}, nil
}

// SetEncryptionKey sets the encryption key for sensitive token storage
func (ts *TokenStorage) SetEncryptionKey(key []byte) error {
	if len(key) != 32 && len(key) != 24 && len(key) != 16 {
		return fmt.Errorf("encryption key must be 16, 24, or 32 bytes")
	}
	ts.encryptKey = key
	return nil
}

// SaveToken saves an OAuth token to disk
func (ts *TokenStorage) SaveToken(provider, userEmail string, token interface{}) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	var st *storedToken

	// Convert different token types to stored format
	switch t := token.(type) {
	case *ClaudeProToken:
		st = &storedToken{
			Provider:     provider,
			UserEmail:    userEmail,
			AccessToken:  t.AccessToken,
			RefreshToken: t.RefreshToken,
			TokenType:    t.TokenType,
			ExpiresAt:    t.ExpiresAt,
			Scope:        t.Scope,
			StoredAt:     time.Now(),
		}
	case *ChatGPTPlusToken:
		st = &storedToken{
			Provider:     provider,
			UserEmail:    userEmail,
			AccessToken:  t.AccessToken,
			RefreshToken: t.RefreshToken,
			TokenType:    t.TokenType,
			ExpiresAt:    t.ExpiresAt,
			Scope:        t.Scope,
			StoredAt:     time.Now(),
		}
	default:
		return fmt.Errorf("unsupported token type")
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	// Encrypt if key is set
	if ts.encryptKey != nil {
		data, err = ts.encrypt(data)
		if err != nil {
			return fmt.Errorf("failed to encrypt token: %w", err)
		}
	}

	// Write to file
	filename := ts.tokenFilePath(provider, userEmail)
	if err := os.MkdirAll(filepath.Dir(filename), 0o700); err != nil {
		return fmt.Errorf("failed to create token directory: %w", err)
	}

	if err := os.WriteFile(filename, data, 0o600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	// Update cache
	cacheKey := ts.cacheKey(provider, userEmail)
	ts.cache[cacheKey] = st
	ts.cacheTime[cacheKey] = time.Now()

	return nil
}

// GetToken retrieves a stored token
func (ts *TokenStorage) GetToken(provider, userEmail string) (interface{}, error) {
	ts.mu.RLock()

	// Check cache first
	cacheKey := ts.cacheKey(provider, userEmail)
	if cached, ok := ts.cache[cacheKey]; ok {
		if lastLoad, ok := ts.cacheTime[cacheKey]; ok {
			if time.Since(lastLoad) < ts.cacheTTL {
				ts.mu.RUnlock()
				return ts.storedToToken(cached, provider)
			}
		}
	}
	ts.mu.RUnlock()

	// Load from disk
	filename := ts.tokenFilePath(provider, userEmail)
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("token not found: %w", err)
	}

	// Decrypt if needed
	if ts.encryptKey != nil {
		var err error
		data, err = ts.decrypt(data)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt token: %w", err)
		}
	}

	// Unmarshal
	var st storedToken
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	// Update cache
	ts.mu.Lock()
	ts.cache[cacheKey] = &st
	ts.cacheTime[cacheKey] = time.Now()
	ts.mu.Unlock()

	return ts.storedToToken(&st, provider)
}

// DeleteToken deletes a stored token
func (ts *TokenStorage) DeleteToken(provider, userEmail string) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	filename := ts.tokenFilePath(provider, userEmail)
	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete token: %w", err)
	}

	// Remove from cache
	cacheKey := ts.cacheKey(provider, userEmail)
	delete(ts.cache, cacheKey)
	delete(ts.cacheTime, cacheKey)

	return nil
}

// ListTokens lists all stored tokens for a provider
func (ts *TokenStorage) ListTokens(provider string) ([]string, error) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	dir := filepath.Join(ts.basePath, provider)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to list tokens: %w", err)
	}

	var emails []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".token" {
			// Extract email from filename (email.token)
			email := entry.Name()[:len(entry.Name())-6]
			emails = append(emails, email)
		}
	}

	return emails, nil
}

// Helper methods

func (ts *TokenStorage) tokenFilePath(provider, userEmail string) string {
	return filepath.Join(ts.basePath, provider, userEmail+".token")
}

func (ts *TokenStorage) cacheKey(provider, userEmail string) string {
	return provider + ":" + userEmail
}

func (ts *TokenStorage) storedToToken(st *storedToken, provider string) (interface{}, error) {
	switch provider {
	case "claude":
		return &ClaudeProToken{
			AccessToken:  st.AccessToken,
			TokenType:    st.TokenType,
			ExpiresIn:    int64(st.ExpiresAt.Sub(time.Now()) / time.Second),
			RefreshToken: st.RefreshToken,
			Scope:        st.Scope,
			ExpiresAt:    st.ExpiresAt,
		}, nil
	case "openai", "chatgpt":
		return &ChatGPTPlusToken{
			AccessToken:  st.AccessToken,
			TokenType:    st.TokenType,
			ExpiresIn:    int64(st.ExpiresAt.Sub(time.Now()) / time.Second),
			RefreshToken: st.RefreshToken,
			Scope:        st.Scope,
			ExpiresAt:    st.ExpiresAt,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// Encryption helpers

func (ts *TokenStorage) encrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(ts.encryptKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, data, nil), nil
}

func (ts *TokenStorage) decrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(ts.encryptKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

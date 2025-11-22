package llmux

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
)

// StoredToken represents a token stored in the auth directory.
type StoredToken struct {
	Provider     string    `json:"provider"`      // "llmux-claude" or "llmux-chatgpt"
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
	Scope        string    `json:"scope"`
	UserEmail    string    `json:"user_email"`
	UserID       string    `json:"user_id"`
	SavedAt      time.Time `json:"saved_at"`
}

// SaveToken saves a token to the auth directory.
//
// Parameters:
//   - authDir: The authentication directory path
//   - provider: The provider name ("llmux-claude" or "llmux-chatgpt")
//   - accessToken: The access token
//   - refreshToken: The refresh token
//   - expiresAt: The token expiry time
//   - userEmail: The user's email address
//   - userID: The user's ID
//
// Returns:
//   - error: An error if saving fails
func SaveToken(authDir, provider, accessToken, refreshToken string, expiresAt time.Time, userEmail, userID string) error {
	// Expand tilde in path
	if authDir[:2] == "~/" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		authDir = filepath.Join(homeDir, authDir[2:])
	}

	// Create auth directory if it doesn't exist
	if err := os.MkdirAll(authDir, 0700); err != nil {
		return fmt.Errorf("failed to create auth directory: %w", err)
	}

	token := StoredToken{
		Provider:     provider,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresAt:    expiresAt,
		UserEmail:    userEmail,
		UserID:       userID,
		SavedAt:      time.Now(),
	}

	// Construct filename: provider-email.json
	filename := fmt.Sprintf("%s-%s.json", provider, userEmail)
	filepath := filepath.Join(authDir, filename)

	// Marshal token to JSON
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	// Write to file with restrictive permissions
	if err := os.WriteFile(filepath, data, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	log.WithFields(log.Fields{
		"provider": provider,
		"email":    userEmail,
		"file":     filepath,
	}).Info("Saved token to storage")

	return nil
}

// LoadToken loads a token from the auth directory.
//
// Parameters:
//   - authDir: The authentication directory path
//   - provider: The provider name
//   - email: The user's email address
//
// Returns:
//   - *StoredToken: The loaded token
//   - error: An error if loading fails
func LoadToken(authDir, provider, email string) (*StoredToken, error) {
	// Expand tilde in path
	if authDir[:2] == "~/" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		authDir = filepath.Join(homeDir, authDir[2:])
	}

	// Construct filename
	filename := fmt.Sprintf("%s-%s.json", provider, email)
	filepath := filepath.Join(authDir, filename)

	// Read file
	data, err := os.ReadFile(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("token not found for %s (%s)", provider, email)
		}
		return nil, fmt.Errorf("failed to read token file: %w", err)
	}

	// Unmarshal token
	var token StoredToken
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to parse token file: %w", err)
	}

	log.WithFields(log.Fields{
		"provider": provider,
		"email":    email,
		"file":     filepath,
		"expired":  token.IsExpired(),
	}).Debug("Loaded token from storage")

	return &token, nil
}

// LoadAnyToken loads any token for the given provider (first found).
// Useful when email is not known.
//
// Parameters:
//   - authDir: The authentication directory path
//   - provider: The provider name
//
// Returns:
//   - *StoredToken: The loaded token
//   - error: An error if loading fails
func LoadAnyToken(authDir, provider string) (*StoredToken, error) {
	// Expand tilde in path
	if authDir[:2] == "~/" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		authDir = filepath.Join(homeDir, authDir[2:])
	}

	// List files in auth directory
	files, err := os.ReadDir(authDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read auth directory: %w", err)
	}

	// Find first matching file
	prefix := provider + "-"
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if len(file.Name()) > len(prefix) && file.Name()[:len(prefix)] == prefix {
			// Found a match, load it
			filepath := filepath.Join(authDir, file.Name())
			data, err := os.ReadFile(filepath)
			if err != nil {
				continue
			}

			var token StoredToken
			if err := json.Unmarshal(data, &token); err != nil {
				continue
			}

			log.WithFields(log.Fields{
				"provider": provider,
				"email":    token.UserEmail,
				"file":     filepath,
			}).Debug("Loaded token from storage")

			return &token, nil
		}
	}

	return nil, fmt.Errorf("no token found for provider: %s", provider)
}

// IsExpired checks if the token is expired.
func (t *StoredToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// NeedsRefresh checks if the token should be refreshed (within 5 minutes of expiry).
func (t *StoredToken) NeedsRefresh() bool {
	return time.Now().Add(5 * time.Minute).After(t.ExpiresAt)
}

// DeleteToken deletes a token file.
//
// Parameters:
//   - authDir: The authentication directory path
//   - provider: The provider name
//   - email: The user's email address
//
// Returns:
//   - error: An error if deletion fails
func DeleteToken(authDir, provider, email string) error {
	// Expand tilde in path
	if authDir[:2] == "~/" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		authDir = filepath.Join(homeDir, authDir[2:])
	}

	// Construct filename
	filename := fmt.Sprintf("%s-%s.json", provider, email)
	filepath := filepath.Join(authDir, filename)

	// Delete file
	if err := os.Remove(filepath); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to delete token file: %w", err)
	}

	log.WithFields(log.Fields{
		"provider": provider,
		"email":    email,
		"file":     filepath,
	}).Info("Deleted token from storage")

	return nil
}

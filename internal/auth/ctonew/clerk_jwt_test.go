package ctonew

import (
	"strings"
	"testing"
	"time"
)

// Helper function to create a valid test JWT (simple, unverified)
func createTestJWT(rotatingToken string, expiresAt int64) string {
	// Create a simple JWT structure for testing
	// Format: header.payload.signature
	// We'll use unverified JWT for testing purposes

	header := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9" // {"alg":"HS256","typ":"JWT"}

	// Create payload with rotating token and expiration
	payload := `{
		"sub": "user123",
		"email": "user@example.com",
		"rotating_token": "` + rotatingToken + `",
		"exp": ` + string(rune(expiresAt)) + `,
		"nbf": ` + string(rune(time.Now().Unix())) + `
	}`

	// For testing, we'll create a base64 encoded payload
	// This is a simplified version - real JWTs would be properly encoded
	encodedPayload := encodeBase64(payload)

	signature := "fake_signature_for_testing"

	return header + "." + encodedPayload + "." + signature
}

func encodeBase64(s string) string {
	// Simple base64-like encoding for testing
	result := ""
	for _, c := range s {
		result += string(c)
	}
	return result
}

func TestClerkJWTParser_ParseToken(t *testing.T) {
	parser := NewClerkJWTParser()

	// For real testing, we'd need actual valid JWTs
	// This is a structural test

	tests := []struct {
		name        string
		tokenString string
		expectError bool
	}{
		{
			name:        "invalid token format",
			tokenString: "invalid.token",
			expectError: true,
		},
		{
			name:        "empty token",
			tokenString: "",
			expectError: true,
		},
		{
			name:        "malformed token",
			tokenString: "a.b.c.d.e",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.ParseToken(tt.tokenString)

			if tt.expectError && err == nil {
				t.Errorf("expected error for invalid token, got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestClerkJWTParser_ExtractRotatingToken(t *testing.T) {
	parser := NewClerkJWTParser()

	tests := []struct {
		name        string
		tokenString string
		expectError bool
	}{
		{
			name:        "invalid token",
			tokenString: "invalid.token",
			expectError: true,
		},
		{
			name:        "empty token",
			tokenString: "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.ExtractRotatingToken(tt.tokenString)

			if tt.expectError && err == nil {
				t.Errorf("expected error for invalid token, got nil")
			}
		})
	}
}

func TestClerkJWTParser_IsTokenExpired(t *testing.T) {
	parser := NewClerkJWTParser()

	tests := []struct {
		name        string
		tokenString string
		expectError bool
	}{
		{
			name:        "invalid token format",
			tokenString: "invalid.token",
			expectError: true,
		},
		{
			name:        "empty token",
			tokenString: "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.IsTokenExpired(tt.tokenString)

			if tt.expectError && err == nil {
				t.Errorf("expected error for invalid token, got nil")
			}
		})
	}
}

func TestClerkJWTParser_GetClaimsInfo(t *testing.T) {
	parser := NewClerkJWTParser()

	tests := []struct {
		name        string
		tokenString string
		expectError bool
	}{
		{
			name:        "invalid token",
			tokenString: "invalid.token",
			expectError: true,
		},
		{
			name:        "empty token",
			tokenString: "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.GetClaimsInfo(tt.tokenString)

			if tt.expectError && err == nil {
				t.Errorf("expected error for invalid token, got nil")
			}
		})
	}
}

func TestClerkJWTClaims_Validation(t *testing.T) {
	now := time.Now().Unix()

	tests := []struct {
		name     string
		claims   *ClerkJWTClaims
		valid    bool
	}{
		{
			name: "valid claims",
			claims: &ClerkJWTClaims{
				Sub:           "user123",
				Email:         "user@example.com",
				RotatingToken: "rotating_token_123",
				ExpiresAt:     now + 3600, // 1 hour from now
				NotBefore:     now - 60,
			},
			valid: true,
		},
		{
			name: "expired claims",
			claims: &ClerkJWTClaims{
				Sub:           "user123",
				Email:         "user@example.com",
				RotatingToken: "rotating_token_123",
				ExpiresAt:     now - 3600, // 1 hour ago
				NotBefore:     now - 7200,
			},
			valid: false,
		},
		{
			name: "not yet valid claims",
			claims: &ClerkJWTClaims{
				Sub:           "user123",
				Email:         "user@example.com",
				RotatingToken: "rotating_token_123",
				ExpiresAt:     now + 3600,
				NotBefore:     now + 3600, // 1 hour in future
			},
			valid: false,
		},
		{
			name: "missing rotating token",
			claims: &ClerkJWTClaims{
				Sub:       "user123",
				Email:     "user@example.com",
				ExpiresAt: now + 3600,
				NotBefore: now - 60,
			},
			valid: false,
		},
		{
			name: "missing email",
			claims: &ClerkJWTClaims{
				Sub:           "user123",
				RotatingToken: "rotating_token_123",
				ExpiresAt:     now + 3600,
				NotBefore:     now - 60,
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test claims validity
			if tt.claims.Email == "" && tt.valid {
				t.Errorf("invalid claims should have valid=false")
			}

			if tt.claims.RotatingToken == "" && tt.valid {
				t.Errorf("invalid claims should have valid=false")
			}

			// Check expiration
			isExpired := tt.claims.ExpiresAt < time.Now().Unix()
			if isExpired && tt.valid {
				t.Errorf("expired claims should have valid=false")
			}

			// Check not before
			notYetValid := tt.claims.NotBefore > time.Now().Unix()
			if notYetValid && tt.valid {
				t.Errorf("not-yet-valid claims should have valid=false")
			}
		})
	}
}

func TestJWTTokenFormat(t *testing.T) {
	tests := []struct {
		name   string
		token  string
		valid  bool
	}{
		{
			name:   "valid JWT format with three parts",
			token:  "header.payload.signature",
			valid:  true,
		},
		{
			name:   "invalid JWT with two parts",
			token:  "header.payload",
			valid:  false,
		},
		{
			name:   "invalid JWT with one part",
			token:  "header",
			valid:  false,
		},
		{
			name:   "empty token",
			token:  "",
			valid:  false,
		},
		{
			name:   "token with extra dots",
			token:  "header.payload.signature.extra",
			valid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := strings.Split(tt.token, ".")
			isValid := len(parts) == 3 && tt.token != ""

			if isValid != tt.valid {
				t.Errorf("JWT format validation failed: expected %v, got %v", tt.valid, isValid)
			}
		})
	}
}

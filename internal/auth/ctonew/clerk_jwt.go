package ctonew

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ClerkJWTClaims represents the claims in a Clerk JWT token
type ClerkJWTClaims struct {
	Sub            string `json:"sub"`
	Email          string `json:"email"`
	EmailVerified  bool   `json:"email_verified"`
	Name           string `json:"name"`
	Picture        string `json:"picture"`
	RotatingToken  string `json:"rotating_token"`
	Aud            string `json:"aud"`
	Iss            string `json:"iss"`
	ExpiresAt      int64  `json:"exp"`
	IssuedAt       int64  `json:"iat"`
	NotBefore      int64  `json:"nbf"`
	SessionId      string `json:"sid"`
	OrgID          string `json:"org_id"`
	OrgSlug        string `json:"org_slug"`
	OrgRole        string `json:"org_role"`
	OrgPermissions []string `json:"org_permissions"`
}

// ClerkJWTParser parses and validates Clerk JWT tokens
type ClerkJWTParser struct {
	skipVerification bool // For development/testing only
}

// NewClerkJWTParser creates a new Clerk JWT parser
func NewClerkJWTParser() *ClerkJWTParser {
	return &ClerkJWTParser{
		skipVerification: false,
	}
}

// SetSkipVerification sets whether to skip signature verification (DEVELOPMENT ONLY)
func (p *ClerkJWTParser) SetSkipVerification(skip bool) {
	p.skipVerification = skip
}

// ParseToken parses and validates a Clerk JWT token
func (p *ClerkJWTParser) ParseToken(tokenString string) (*ClerkJWTClaims, error) {
	// Split JWT into parts
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format: expected 3 parts, got %d", len(parts))
	}

	// Decode header (we don't validate it for now)
	headerData, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT header: %w", err)
	}

	var header map[string]interface{}
	if err := json.Unmarshal(headerData, &header); err != nil {
		return nil, fmt.Errorf("failed to parse JWT header: %w", err)
	}

	// Decode claims
	claimsData, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT claims: %w", err)
	}

	var claims ClerkJWTClaims
	if err := json.Unmarshal(claimsData, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse JWT claims: %w", err)
	}

	// Validate token is not expired
	if claims.ExpiresAt > 0 {
		expiresAt := time.Unix(claims.ExpiresAt, 0)
		if time.Now().After(expiresAt) {
			return nil, fmt.Errorf("token is expired (expired at %v)", expiresAt)
		}
	}

	// Validate token is not used before nbf
	if claims.NotBefore > 0 {
		notBefore := time.Unix(claims.NotBefore, 0)
		if time.Now().Before(notBefore) {
			return nil, fmt.Errorf("token is not yet valid (valid from %v)", notBefore)
		}
	}

	// Check for rotating token
	if claims.RotatingToken == "" {
		return nil, fmt.Errorf("rotating_token not found in JWT claims")
	}

	return &claims, nil
}

// ExtractRotatingToken extracts the rotating_token from a Clerk JWT
func (p *ClerkJWTParser) ExtractRotatingToken(tokenString string) (string, error) {
	claims, err := p.ParseToken(tokenString)
	if err != nil {
		return "", fmt.Errorf("failed to parse JWT: %w", err)
	}

	if claims.RotatingToken == "" {
		return "", fmt.Errorf("rotating_token not found in claims")
	}

	return claims.RotatingToken, nil
}

// IsTokenExpired checks if a token is expired
func (p *ClerkJWTParser) IsTokenExpired(tokenString string) (bool, error) {
	claims, err := p.ParseToken(tokenString)
	if err != nil {
		// Parsing error means token is invalid/expired
		return true, err
	}

	if claims.ExpiresAt > 0 {
		expiresAt := time.Unix(claims.ExpiresAt, 0)
		return time.Now().After(expiresAt), nil
	}

	return false, nil
}

// GetClaimsInfo returns a summary of the token claims
func (p *ClerkJWTParser) GetClaimsInfo(tokenString string) (map[string]interface{}, error) {
	claims, err := p.ParseToken(tokenString)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"sub":              claims.Sub,
		"email":            claims.Email,
		"email_verified":   claims.EmailVerified,
		"name":             claims.Name,
		"session_id":       claims.SessionId,
		"org_id":           claims.OrgID,
		"issued_at":        time.Unix(claims.IssuedAt, 0),
		"expires_at":       time.Unix(claims.ExpiresAt, 0),
		"has_rotating_token": claims.RotatingToken != "",
	}, nil
}

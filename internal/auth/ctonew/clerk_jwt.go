// Package ctonew provides Clerk JWT authentication for the ctonew provider.
// This package implements JWT parsing and token exchange with Clerk's API.
package ctonew

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// ClerkJWTClaims represents the claims within a Clerk JWT.
type ClerkJWTClaims struct {
	// Standard JWT claims
	ISS string `json:"iss"` // Issuer
	SUB string `json:"sub"` // Subject (user ID)
	IAT int64  `json:"iat"` // Issued at
	EXP int64  `json:"exp"` // Expires at
	AZP string `json:"azp"` // Authorized party

	// Clerk-specific claims
	RotatingToken string `json:"rotating_token"`
	SID           string `json:"sid"` // Session ID
	OrgID         string `json:"org_id,omitempty"`
	OrgRole       string `json:"org_role,omitempty"`
	OrgSlug       string `json:"org_slug,omitempty"`
}

// ParseClerkJWT parses a Clerk JWT and extracts the claims.
//
// Parameters:
//   - jwtString: The JWT string to parse (from __client cookie)
//
// Returns:
//   - *ClerkJWTClaims: The parsed claims
//   - error: An error if parsing fails
func ParseClerkJWT(jwtString string) (*ClerkJWTClaims, error) {
	// JWT format: header.payload.signature
	parts := strings.Split(jwtString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format: expected 3 parts, got %d", len(parts))
	}

	// Decode payload (part 1)
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	// Parse claims
	var claims ClerkJWTClaims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse JWT claims: %w", err)
	}

	log.WithFields(log.Fields{
		"sub":             claims.SUB,
		"exp":             time.Unix(claims.EXP, 0),
		"has_rotating_token": claims.RotatingToken != "",
	}).Debug("Parsed Clerk JWT")

	return &claims, nil
}

// ExtractRotatingToken extracts the rotating_token from a Clerk JWT.
// This is a convenience function that parses the JWT and returns just the rotating token.
//
// Parameters:
//   - jwtString: The JWT string from the __client cookie
//
// Returns:
//   - string: The rotating token
//   - error: An error if extraction fails
func ExtractRotatingToken(jwtString string) (string, error) {
	claims, err := ParseClerkJWT(jwtString)
	if err != nil {
		return "", err
	}

	if claims.RotatingToken == "" {
		return "", fmt.Errorf("no rotating_token found in JWT claims")
	}

	return claims.RotatingToken, nil
}

// IsExpired checks if the JWT is expired.
func (c *ClerkJWTClaims) IsExpired() bool {
	return time.Now().Unix() > c.EXP
}

// ExpiresAt returns the expiration time.
func (c *ClerkJWTClaims) ExpiresAt() time.Time {
	return time.Unix(c.EXP, 0)
}

// ExpiresIn returns the duration until expiration.
func (c *ClerkJWTClaims) ExpiresIn() time.Duration {
	return time.Until(c.ExpiresAt())
}

// NeedsRefresh checks if the token should be refreshed (within 5 minutes of expiry).
func (c *ClerkJWTClaims) NeedsRefresh() bool {
	return c.ExpiresIn() < 5*time.Minute
}

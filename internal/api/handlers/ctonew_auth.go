package handlers

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/auth/ctonew"
)

// CtonewAuthHandler handles authentication for ctonew
type CtonewAuthHandler struct {
	jwtParser     *ctonew.ClerkJWTParser
	tokenExchange *ctonew.ClerkTokenExchange
	storedJWTs    map[string]string // Store JWTs keyed by user identifier
	jwtMutex      sync.RWMutex
}

// NewCtonewAuthHandler creates a new ctonew auth handler
func NewCtonewAuthHandler(
	jwtParser *ctonew.ClerkJWTParser,
	tokenExchange *ctonew.ClerkTokenExchange,
) *CtonewAuthHandler {
	return &CtonewAuthHandler{
		jwtParser:     jwtParser,
		tokenExchange: tokenExchange,
		storedJWTs:    make(map[string]string),
	}
}

// SubmitJWTRequest represents a request to submit a Clerk JWT
type SubmitJWTRequest struct {
	JWT       string `json:"jwt" binding:"required"`
	UserEmail string `json:"user_email"`
}

// SubmitJWTHandler handles JWT submission
func (h *CtonewAuthHandler) SubmitJWTHandler(c *gin.Context) {
	var req SubmitJWTRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid request: %v", err),
		})
		return
	}

	// Parse JWT to validate
	claims, err := h.jwtParser.ParseToken(req.JWT)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid JWT: %v", err),
		})
		return
	}

	// Use provided email or fall back to email in JWT
	userEmail := req.UserEmail
	if userEmail == "" {
		userEmail = claims.Email
	}
	if userEmail == "" {
		userEmail = claims.Sub
	}

	// Store JWT
	h.jwtMutex.Lock()
	h.storedJWTs[userEmail] = req.JWT
	h.jwtMutex.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"message": "JWT stored successfully",
		"user":    userEmail,
		"issued_at": claims.IssuedAt,
		"expires_at": claims.ExpiresAt,
	})
}

// StatusRequest represents a status check request
type StatusRequest struct {
	UserEmail string `form:"user_email"`
}

// StatusHandler checks authentication status
func (h *CtonewAuthHandler) StatusHandler(c *gin.Context) {
	userEmail := c.Query("user_email")
	if userEmail == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "user_email is required",
		})
		return
	}

	// Check if JWT is stored
	h.jwtMutex.RLock()
	jwt, exists := h.storedJWTs[userEmail]
	h.jwtMutex.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"authenticated": false,
			"error":         "no JWT stored for this user",
		})
		return
	}

	// Check if JWT is valid
	isExpired, err := h.jwtParser.IsTokenExpired(jwt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"authenticated": false,
			"error":         fmt.Sprintf("failed to validate JWT: %v", err),
		})
		return
	}

	// Get JWT info
	claimsInfo, err := h.jwtParser.GetClaimsInfo(jwt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"authenticated": false,
			"error":         fmt.Sprintf("failed to get JWT info: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"authenticated":  !isExpired,
		"user_email":     userEmail,
		"is_expired":     isExpired,
		"claims":         claimsInfo,
	})
}

// RevokeRequest represents a revoke request
type RevokeRequest struct {
	UserEmail string `form:"user_email"`
}

// RevokeHandler revokes a stored JWT
func (h *CtonewAuthHandler) RevokeHandler(c *gin.Context) {
	userEmail := c.Query("user_email")
	if userEmail == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "user_email is required",
		})
		return
	}

	// Delete stored JWT
	h.jwtMutex.Lock()
	_, exists := h.storedJWTs[userEmail]
	if exists {
		delete(h.storedJWTs, userEmail)
	}
	h.jwtMutex.Unlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "no JWT stored for this user",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "JWT revoked successfully",
		"user":    userEmail,
	})
}

// GetJWTHandler retrieves a stored JWT for execution
func (h *CtonewAuthHandler) GetJWTHandler(c *gin.Context) {
	userEmail := c.Query("user_email")
	if userEmail == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "user_email is required",
		})
		return
	}

	// Get stored JWT
	h.jwtMutex.RLock()
	jwt, exists := h.storedJWTs[userEmail]
	h.jwtMutex.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "no JWT stored for this user",
		})
		return
	}

	// Return only the JWT (not full token for security)
	c.JSON(http.StatusOK, gin.H{
		"jwt": jwt,
	})
}

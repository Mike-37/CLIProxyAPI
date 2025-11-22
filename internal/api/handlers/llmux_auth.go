package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/auth/llmux"
	log "github.com/sirupsen/logrus"
)

// LLMuxAuthHandler handles OAuth for LLMux providers
type LLMuxAuthHandler struct {
	claudeOAuth   *llmux.ClaudeProOAuth
	chatgptOAuth  *llmux.ChatGPTPlusOAuth
	tokenStorage  *llmux.TokenStorage
	callbackURL   string
	oauthStates   map[string]time.Time // Track OAuth states with timestamps
}

// NewLLMuxAuthHandler creates a new LLMux auth handler
func NewLLMuxAuthHandler(
	claudeOAuth *llmux.ClaudeProOAuth,
	chatgptOAuth *llmux.ChatGPTPlusOAuth,
	tokenStorage *llmux.TokenStorage,
	callbackURL string,
) *LLMuxAuthHandler {
	return &LLMuxAuthHandler{
		claudeOAuth:  claudeOAuth,
		chatgptOAuth: chatgptOAuth,
		tokenStorage: tokenStorage,
		callbackURL:  callbackURL,
		oauthStates:  make(map[string]time.Time),
	}
}

// ClaudeLoginHandler initiates Claude Pro OAuth flow
func (h *LLMuxAuthHandler) ClaudeLoginHandler(c *gin.Context) {
	state := generateRandomState()
	h.oauthStates[state] = time.Now().Add(10 * time.Minute) // 10 minute expiry

	authURL := h.claudeOAuth.GetAuthorizationURL(state)

	c.JSON(http.StatusOK, gin.H{
		"auth_url": authURL,
		"state":    state,
		"message":  "Open the URL in your browser to authorize. You will be redirected back here with a code.",
	})
}

// ChatGPTLoginHandler initiates ChatGPT Plus OAuth flow
func (h *LLMuxAuthHandler) ChatGPTLoginHandler(c *gin.Context) {
	state := generateRandomState()
	h.oauthStates[state] = time.Now().Add(10 * time.Minute) // 10 minute expiry

	authURL := h.chatgptOAuth.GetAuthorizationURL(state)

	c.JSON(http.StatusOK, gin.H{
		"auth_url": authURL,
		"state":    state,
		"message":  "Open the URL in your browser to authorize. You will be redirected back here with a code.",
	})
}

// ClaudeCallbackHandler handles Claude OAuth callback
func (h *LLMuxAuthHandler) ClaudeCallbackHandler(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	errStr := c.Query("error")

	if errStr != "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": errStr,
		})
		return
	}

	// Verify state
	if _, ok := h.oauthStates[state]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid or expired state",
		})
		return
	}
	delete(h.oauthStates, state)

	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "no authorization code provided",
		})
		return
	}

	// Exchange code for token
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	token, err := h.claudeOAuth.ExchangeCodeForToken(ctx, code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to exchange code: %v", err),
		})
		return
	}

	// Get user email from request
	userEmail := c.Query("user_email")
	if userEmail == "" {
		userEmail = "default" // Fallback
	}

	// Save token
	if err := h.tokenStorage.SaveToken("claude", userEmail, token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to save token: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Successfully authenticated with Claude Pro",
		"token":   token.AccessToken,
		"expires": token.ExpiresAt,
	})
}

// ChatGPTCallbackHandler handles ChatGPT OAuth callback
func (h *LLMuxAuthHandler) ChatGPTCallbackHandler(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	errStr := c.Query("error")

	if errStr != "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": errStr,
		})
		return
	}

	// Verify state
	if _, ok := h.oauthStates[state]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid or expired state",
		})
		return
	}
	delete(h.oauthStates, state)

	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "no authorization code provided",
		})
		return
	}

	// Exchange code for token
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	token, err := h.chatgptOAuth.ExchangeCodeForToken(ctx, code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to exchange code: %v", err),
		})
		return
	}

	// Get user email from request
	userEmail := c.Query("user_email")
	if userEmail == "" {
		userEmail = "default" // Fallback
	}

	// Save token
	if err := h.tokenStorage.SaveToken("openai", userEmail, token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to save token: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Successfully authenticated with ChatGPT Plus",
		"token":   token.AccessToken,
		"expires": token.ExpiresAt,
	})
}

// ClaudeStatusHandler returns Claude authentication status
func (h *LLMuxAuthHandler) ClaudeStatusHandler(c *gin.Context) {
	userEmail := c.Query("user_email")
	if userEmail == "" {
		userEmail = "default"
	}

	tokenInterface, err := h.tokenStorage.GetToken("claude", userEmail)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"authenticated": false,
			"error":         "no token found",
		})
		return
	}

	token := tokenInterface.(*llmux.ClaudeProToken)
	isExpired := token.IsExpired()

	c.JSON(http.StatusOK, gin.H{
		"authenticated": !isExpired,
		"user_email":    userEmail,
		"expires_at":    token.ExpiresAt,
		"is_expired":    isExpired,
	})
}

// ChatGPTStatusHandler returns ChatGPT authentication status
func (h *LLMuxAuthHandler) ChatGPTStatusHandler(c *gin.Context) {
	userEmail := c.Query("user_email")
	if userEmail == "" {
		userEmail = "default"
	}

	tokenInterface, err := h.tokenStorage.GetToken("openai", userEmail)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"authenticated": false,
			"error":         "no token found",
		})
		return
	}

	token := tokenInterface.(*llmux.ChatGPTPlusToken)
	isExpired := token.IsExpired()

	c.JSON(http.StatusOK, gin.H{
		"authenticated": !isExpired,
		"user_email":    userEmail,
		"expires_at":    token.ExpiresAt,
		"is_expired":    isExpired,
	})
}

// ClaudeRevokeHandler revokes Claude authentication
func (h *LLMuxAuthHandler) ClaudeRevokeHandler(c *gin.Context) {
	userEmail := c.Query("user_email")
	if userEmail == "" {
		userEmail = "default"
	}

	// Get token to revoke
	tokenInterface, err := h.tokenStorage.GetToken("claude", userEmail)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "no token found",
		})
		return
	}

	token := tokenInterface.(*llmux.ClaudeProToken)

	// Revoke token with provider
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := h.claudeOAuth.RevokeToken(ctx, token.AccessToken); err != nil {
		// Log but continue - token will be deleted anyway
		fmt.Printf("warning: failed to revoke token with provider: %v\n", err)
	}

	// Delete local token
	if err := h.tokenStorage.DeleteToken("claude", userEmail); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to delete token: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Successfully revoked Claude authentication",
	})
}

// ChatGPTRevokeHandler revokes ChatGPT authentication
func (h *LLMuxAuthHandler) ChatGPTRevokeHandler(c *gin.Context) {
	userEmail := c.Query("user_email")
	if userEmail == "" {
		userEmail = "default"
	}

	// Get token to revoke
	tokenInterface, err := h.tokenStorage.GetToken("openai", userEmail)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "no token found",
		})
		return
	}

	token := tokenInterface.(*llmux.ChatGPTPlusToken)

	// Revoke token with provider
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := h.chatgptOAuth.RevokeToken(ctx, token.AccessToken); err != nil {
		// Log but continue - token will be deleted anyway
		fmt.Printf("warning: failed to revoke token with provider: %v\n", err)
	}

	// Delete local token
	if err := h.tokenStorage.DeleteToken("openai", userEmail); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to delete token: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Successfully revoked ChatGPT authentication",
	})
}

// Helper function
func generateRandomState() string {
	// Use current time and nanoseconds for state generation
	// In production, should use crypto/rand with proper entropy
	return fmt.Sprintf("state_%d", time.Now().UnixNano())
}

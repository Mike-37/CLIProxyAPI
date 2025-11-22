package executor

import (
	"context"
	"fmt"
	"net/http"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/auth/llmux"
	"github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/executor"
)

// LLMuxClaudeExecutor implements the Executor interface for LLMux Claude Pro
type LLMuxClaudeExecutor struct {
	oauthHandler *llmux.ClaudeProOAuth
	tokenStorage *llmux.TokenStorage
	userEmail    string
	httpClient   *http.Client
}

// NewLLMuxClaudeExecutor creates a new LLMux Claude executor
func NewLLMuxClaudeExecutor(
	oauthHandler *llmux.ClaudeProOAuth,
	tokenStorage *llmux.TokenStorage,
	userEmail string,
	httpClient *http.Client,
) *LLMuxClaudeExecutor {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &LLMuxClaudeExecutor{
		oauthHandler: oauthHandler,
		tokenStorage: tokenStorage,
		userEmail:    userEmail,
		httpClient:   httpClient,
	}
}

// Name returns the executor name
func (e *LLMuxClaudeExecutor) Name() string {
	return "llmux-claude"
}

// Execute executes a request using LLMux Claude Pro
func (e *LLMuxClaudeExecutor) Execute(ctx context.Context, request *executor.ExecuteRequest) (*executor.ExecuteResponse, error) {
	// Get or refresh token
	token, err := e.getValidToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get valid token: %w", err)
	}

	// Call Claude API with token
	return e.callClaudeAPI(ctx, token, request)
}

// Stream executes a streaming request using LLMux Claude Pro
func (e *LLMuxClaudeExecutor) Stream(ctx context.Context, request *executor.ExecuteRequest) (<-chan executor.StreamChunk, error) {
	// Get or refresh token
	token, err := e.getValidToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get valid token: %w", err)
	}

	// Call Claude API with streaming
	return e.streamClaudeAPI(ctx, token, request)
}

// GetValidToken retrieves a valid access token, refreshing if necessary
func (e *LLMuxClaudeExecutor) getValidToken(ctx context.Context) (*llmux.ClaudeProToken, error) {
	// Try to get stored token
	tokenInterface, err := e.tokenStorage.GetToken("claude", e.userEmail)
	if err != nil {
		return nil, fmt.Errorf("no stored token for user: %w", err)
	}

	token := tokenInterface.(*llmux.ClaudeProToken)

	// Check if token is expired
	if token.IsExpired() {
		// Try to refresh
		if token.RefreshToken != "" {
			newToken, err := e.oauthHandler.RefreshToken(ctx, token.RefreshToken)
			if err != nil {
				return nil, fmt.Errorf("failed to refresh token: %w", err)
			}

			// Save refreshed token
			if err := e.tokenStorage.SaveToken("claude", e.userEmail, newToken); err != nil {
				// Log warning but continue with new token
				fmt.Printf("warning: failed to save refreshed token: %v\n", err)
			}

			return newToken, nil
		}

		return nil, fmt.Errorf("token expired and no refresh token available")
	}

	return token, nil
}

// callClaudeAPI makes a request to the Claude API
func (e *LLMuxClaudeExecutor) callClaudeAPI(ctx context.Context, token *llmux.ClaudeProToken, request *executor.ExecuteRequest) (*executor.ExecuteResponse, error) {
	// TODO: Implement Claude API call
	// This would typically:
	// 1. Transform request format to Claude API format
	// 2. Add Bearer token to Authorization header
	// 3. Make HTTP request to Claude API
	// 4. Transform response back to executor format

	return nil, fmt.Errorf("not yet implemented")
}

// streamClaudeAPI makes a streaming request to the Claude API
func (e *LLMuxClaudeExecutor) streamClaudeAPI(ctx context.Context, token *llmux.ClaudeProToken, request *executor.ExecuteRequest) (<-chan executor.StreamChunk, error) {
	// TODO: Implement Claude API streaming
	// This would typically:
	// 1. Transform request format to Claude API format
	// 2. Add Bearer token to Authorization header
	// 3. Make streaming HTTP request to Claude API
	// 4. Transform streaming response chunks to executor format

	return nil, fmt.Errorf("not yet implemented")
}

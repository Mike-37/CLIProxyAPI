package executor

import (
	"context"
	"fmt"
	"net/http"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/auth/llmux"
	"github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/executor"
)

// LLMuxChatGPTExecutor implements the Executor interface for LLMux ChatGPT Plus
type LLMuxChatGPTExecutor struct {
	oauthHandler *llmux.ChatGPTPlusOAuth
	tokenStorage *llmux.TokenStorage
	userEmail    string
	httpClient   *http.Client
}

// NewLLMuxChatGPTExecutor creates a new LLMux ChatGPT executor
func NewLLMuxChatGPTExecutor(
	oauthHandler *llmux.ChatGPTPlusOAuth,
	tokenStorage *llmux.TokenStorage,
	userEmail string,
	httpClient *http.Client,
) *LLMuxChatGPTExecutor {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &LLMuxChatGPTExecutor{
		oauthHandler: oauthHandler,
		tokenStorage: tokenStorage,
		userEmail:    userEmail,
		httpClient:   httpClient,
	}
}

// Name returns the executor name
func (e *LLMuxChatGPTExecutor) Name() string {
	return "llmux-chatgpt"
}

// Execute executes a request using LLMux ChatGPT Plus
func (e *LLMuxChatGPTExecutor) Execute(ctx context.Context, request *executor.ExecuteRequest) (*executor.ExecuteResponse, error) {
	// Get or refresh token
	token, err := e.getValidToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get valid token: %w", err)
	}

	// Call OpenAI API with token
	return e.callOpenAIAPI(ctx, token, request)
}

// Stream executes a streaming request using LLMux ChatGPT Plus
func (e *LLMuxChatGPTExecutor) Stream(ctx context.Context, request *executor.ExecuteRequest) (<-chan executor.StreamChunk, error) {
	// Get or refresh token
	token, err := e.getValidToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get valid token: %w", err)
	}

	// Call OpenAI API with streaming
	return e.streamOpenAIAPI(ctx, token, request)
}

// GetValidToken retrieves a valid access token, refreshing if necessary
func (e *LLMuxChatGPTExecutor) getValidToken(ctx context.Context) (*llmux.ChatGPTPlusToken, error) {
	// Try to get stored token
	tokenInterface, err := e.tokenStorage.GetToken("openai", e.userEmail)
	if err != nil {
		return nil, fmt.Errorf("no stored token for user: %w", err)
	}

	token := tokenInterface.(*llmux.ChatGPTPlusToken)

	// Check if token is expired
	if token.IsExpired() {
		// Try to refresh
		if token.RefreshToken != "" {
			newToken, err := e.oauthHandler.RefreshToken(ctx, token.RefreshToken)
			if err != nil {
				return nil, fmt.Errorf("failed to refresh token: %w", err)
			}

			// Save refreshed token
			if err := e.tokenStorage.SaveToken("openai", e.userEmail, newToken); err != nil {
				// Log warning but continue with new token
				fmt.Printf("warning: failed to save refreshed token: %v\n", err)
			}

			return newToken, nil
		}

		return nil, fmt.Errorf("token expired and no refresh token available")
	}

	return token, nil
}

// callOpenAIAPI makes a request to the OpenAI API
func (e *LLMuxChatGPTExecutor) callOpenAIAPI(ctx context.Context, token *llmux.ChatGPTPlusToken, request *executor.ExecuteRequest) (*executor.ExecuteResponse, error) {
	// TODO: Implement OpenAI API call
	// This would typically:
	// 1. Transform request format to OpenAI API format
	// 2. Add Bearer token to Authorization header
	// 3. Make HTTP request to OpenAI API
	// 4. Transform response back to executor format

	return nil, fmt.Errorf("not yet implemented")
}

// streamOpenAIAPI makes a streaming request to the OpenAI API
func (e *LLMuxChatGPTExecutor) streamOpenAIAPI(ctx context.Context, token *llmux.ChatGPTPlusToken, request *executor.ExecuteRequest) (<-chan executor.StreamChunk, error) {
	// TODO: Implement OpenAI API streaming
	// This would typically:
	// 1. Transform request format to OpenAI API format
	// 2. Add Bearer token to Authorization header
	// 3. Make streaming HTTP request to OpenAI API
	// 4. Transform streaming response chunks to executor format

	return nil, fmt.Errorf("not yet implemented")
}

package executor

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/auth/llmux"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/executor"
	sdktranslator "github.com/router-for-me/CLIProxyAPI/v6/sdk/translator"
	log "github.com/sirupsen/logrus"
)

// LLMuxClaudeExecutor implements the Executor interface for LLMux Claude Pro OAuth
type LLMuxClaudeExecutor struct {
	oauthHandler *llmux.ClaudeProOAuth
	tokenStorage *llmux.TokenStorage
	httpClient   *http.Client
}

// NewLLMuxClaudeExecutor creates a new LLMux Claude executor
func NewLLMuxClaudeExecutor(
	oauthHandler *llmux.ClaudeProOAuth,
	tokenStorage *llmux.TokenStorage,
) *LLMuxClaudeExecutor {
	return &LLMuxClaudeExecutor{
		oauthHandler: oauthHandler,
		tokenStorage: tokenStorage,
		httpClient:   &http.Client{Timeout: 0},
	}
}

// Identifier returns the executor identifier
func (e *LLMuxClaudeExecutor) Identifier() string {
	return "llmux-claude"
}

// PrepareRequest prepares the HTTP request (no-op for this executor)
func (e *LLMuxClaudeExecutor) PrepareRequest(_ *http.Request, _ *cliproxyauth.Auth) error {
	return nil
}

// Execute executes a non-streaming request using LLMux Claude Pro
func (e *LLMuxClaudeExecutor) Execute(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (resp cliproxyexecutor.Response, err error) {
	// Extract user email from auth context
	userEmail := extractUserEmail(auth)
	if userEmail == "" {
		return resp, fmt.Errorf("no user email found in auth context")
	}

	// Get or refresh token
	token, err := e.getValidToken(ctx, userEmail)
	if err != nil {
		return resp, fmt.Errorf("failed to get valid token: %w", err)
	}

	// Transform request to Claude API format
	claudeReq, err := e.transformRequestToClaude(req, opts)
	if err != nil {
		return resp, fmt.Errorf("failed to transform request: %w", err)
	}

	// Make HTTP request to Claude API
	httpResp, err := e.callClaudeAPI(ctx, token, claudeReq)
	if err != nil {
		return resp, err
	}

	// Transform response back to executor format
	resp.Payload = httpResp
	return resp, nil
}

// ExecuteStream executes a streaming request using LLMux Claude Pro
func (e *LLMuxClaudeExecutor) ExecuteStream(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (<-chan cliproxyexecutor.StreamChunk, error) {
	// Extract user email from auth context
	userEmail := extractUserEmail(auth)
	if userEmail == "" {
		ch := make(chan cliproxyexecutor.StreamChunk, 1)
		ch <- cliproxyexecutor.StreamChunk{Err: fmt.Errorf("no user email found in auth context")}
		close(ch)
		return ch, nil
	}

	// Get or refresh token
	token, err := e.getValidToken(ctx, userEmail)
	if err != nil {
		ch := make(chan cliproxyexecutor.StreamChunk, 1)
		ch <- cliproxyexecutor.StreamChunk{Err: fmt.Errorf("failed to get valid token: %w", err)}
		close(ch)
		return ch, nil
	}

	// Transform request to Claude API format (with streaming enabled)
	claudeReq, err := e.transformRequestToClaude(req, opts)
	if err != nil {
		ch := make(chan cliproxyexecutor.StreamChunk, 1)
		ch <- cliproxyexecutor.StreamChunk{Err: fmt.Errorf("failed to transform request: %w", err)}
		close(ch)
		return ch, nil
	}

	// Make streaming HTTP request to Claude API
	return e.streamClaudeAPI(ctx, token, claudeReq), nil
}

// getValidToken retrieves a valid access token, refreshing if necessary
func (e *LLMuxClaudeExecutor) getValidToken(ctx context.Context, userEmail string) (*llmux.ClaudeProToken, error) {
	// Try to get stored token
	tokenInterface, err := e.tokenStorage.GetToken("claude", userEmail)
	if err != nil {
		return nil, fmt.Errorf("no stored token for user: %w", err)
	}

	token, ok := tokenInterface.(*llmux.ClaudeProToken)
	if !ok {
		return nil, fmt.Errorf("invalid token type")
	}

	// Check if token is expired
	if token.IsExpired() {
		// Try to refresh
		if token.RefreshToken != "" {
			newToken, err := e.oauthHandler.RefreshToken(ctx, token.RefreshToken)
			if err != nil {
				return nil, fmt.Errorf("failed to refresh token: %w", err)
			}

			// Save refreshed token
			if err := e.tokenStorage.SaveToken("claude", userEmail, newToken); err != nil {
				log.Warnf("failed to save refreshed token: %v", err)
			}

			return newToken, nil
		}

		return nil, fmt.Errorf("token expired and no refresh token available")
	}

	return token, nil
}

// transformRequestToClaude transforms an executor request to Claude API format
func (e *LLMuxClaudeExecutor) transformRequestToClaude(req cliproxyexecutor.Request, opts cliproxyexecutor.Options) ([]byte, error) {
	// Transform from source format to Claude format
	from := opts.SourceFormat
	to := sdktranslator.FromString("claude")
	stream := opts.Stream

	// Use streaming translation to preserve function calling
	body := sdktranslator.TranslateRequest(from, to, req.Model, bytes.Clone(req.Payload), stream)
	return body, nil
}

// callClaudeAPI makes an HTTP request to the Claude API
func (e *LLMuxClaudeExecutor) callClaudeAPI(ctx context.Context, token *llmux.ClaudeProToken, body []byte) ([]byte, error) {
	url := "https://api.anthropic.com/v1/messages?beta=true"

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	// Add authorization header
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Anthropic-Version", "2023-06-01")

	// Make request
	httpResp, err := e.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	// Check for HTTP errors
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return nil, &statusErr{code: httpResp.StatusCode, msg: string(respBody)}
	}

	return respBody, nil
}

// streamClaudeAPI makes a streaming HTTP request to the Claude API
func (e *LLMuxClaudeExecutor) streamClaudeAPI(ctx context.Context, token *llmux.ClaudeProToken, body []byte) <-chan cliproxyexecutor.StreamChunk {
	ch := make(chan cliproxyexecutor.StreamChunk, 10)

	go func() {
		defer close(ch)

		url := "https://api.anthropic.com/v1/messages?beta=true"

		// Create HTTP request
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			ch <- cliproxyexecutor.StreamChunk{Err: err}
			return
		}

		// Add authorization header
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Anthropic-Version", "2023-06-01")

		// Make request
		httpResp, err := e.httpClient.Do(httpReq)
		if err != nil {
			ch <- cliproxyexecutor.StreamChunk{Err: err}
			return
		}
		defer httpResp.Body.Close()

		// Check for HTTP errors
		if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
			body, _ := io.ReadAll(httpResp.Body)
			ch <- cliproxyexecutor.StreamChunk{Err: &statusErr{code: httpResp.StatusCode, msg: string(body)}}
			return
		}

		// Stream lines from response
		scanner := bufio.NewScanner(httpResp.Body)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			// Parse SSE format: "data: {...}"
			if bytes.HasPrefix(line, []byte("data: ")) {
				chunk := bytes.TrimPrefix(line, []byte("data: "))
				ch <- cliproxyexecutor.StreamChunk{Payload: chunk}
			}
		}

		if err := scanner.Err(); err != nil {
			ch <- cliproxyexecutor.StreamChunk{Err: err}
		}
	}()

	return ch
}

// extractUserEmail extracts user email from auth context
func extractUserEmail(auth *cliproxyauth.Auth) string {
	if auth == nil {
		return ""
	}

	// Try to get from auth label (user@example.com format)
	if auth.Label != "" {
		// Label might be "user@example.com" or "Provider:user@example.com"
		if strings.Contains(auth.Label, "@") {
			// Extract email from label
			parts := strings.Split(auth.Label, ":")
			if len(parts) > 0 {
				email := parts[len(parts)-1]
				if strings.Contains(email, "@") {
					return email
				}
			}
		}
	}

	// Try to get from auth metadata
	if auth.AccountInfo != nil {
		accountType, accountValue := auth.AccountInfo()
		if accountType == "email" {
			return accountValue
		}
	}

	return ""
}

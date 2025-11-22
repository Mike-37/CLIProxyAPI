package executor

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/auth/llmux"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/executor"
	sdktranslator "github.com/router-for-me/CLIProxyAPI/v6/sdk/translator"
	log "github.com/sirupsen/logrus"
)

// LLMuxChatGPTExecutor implements the Executor interface for LLMux ChatGPT Plus OAuth
type LLMuxChatGPTExecutor struct {
	oauthHandler *llmux.ChatGPTPlusOAuth
	tokenStorage *llmux.TokenStorage
	httpClient   *http.Client
}

// NewLLMuxChatGPTExecutor creates a new LLMux ChatGPT executor
func NewLLMuxChatGPTExecutor(
	oauthHandler *llmux.ChatGPTPlusOAuth,
	tokenStorage *llmux.TokenStorage,
) *LLMuxChatGPTExecutor {
	return &LLMuxChatGPTExecutor{
		oauthHandler: oauthHandler,
		tokenStorage: tokenStorage,
		httpClient:   &http.Client{Timeout: 0},
	}
}

// Identifier returns the executor identifier
func (e *LLMuxChatGPTExecutor) Identifier() string {
	return "llmux-chatgpt"
}

// PrepareRequest prepares the HTTP request (no-op for this executor)
func (e *LLMuxChatGPTExecutor) PrepareRequest(_ *http.Request, _ *cliproxyauth.Auth) error {
	return nil
}

// Execute executes a non-streaming request using LLMux ChatGPT Plus
func (e *LLMuxChatGPTExecutor) Execute(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (resp cliproxyexecutor.Response, err error) {
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

	// Transform request to OpenAI API format
	openaiReq, err := e.transformRequestToOpenAI(req, opts)
	if err != nil {
		return resp, fmt.Errorf("failed to transform request: %w", err)
	}

	// Make HTTP request to OpenAI API
	httpResp, err := e.callOpenAIAPI(ctx, token, openaiReq)
	if err != nil {
		return resp, err
	}

	// Transform response back to executor format
	resp.Payload = httpResp
	return resp, nil
}

// ExecuteStream executes a streaming request using LLMux ChatGPT Plus
func (e *LLMuxChatGPTExecutor) ExecuteStream(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (<-chan cliproxyexecutor.StreamChunk, error) {
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

	// Transform request to OpenAI API format (with streaming enabled)
	openaiReq, err := e.transformRequestToOpenAI(req, opts)
	if err != nil {
		ch := make(chan cliproxyexecutor.StreamChunk, 1)
		ch <- cliproxyexecutor.StreamChunk{Err: fmt.Errorf("failed to transform request: %w", err)}
		close(ch)
		return ch, nil
	}

	// Make streaming HTTP request to OpenAI API
	return e.streamOpenAIAPI(ctx, token, openaiReq), nil
}

// getValidToken retrieves a valid access token, refreshing if necessary
func (e *LLMuxChatGPTExecutor) getValidToken(ctx context.Context, userEmail string) (*llmux.ChatGPTPlusToken, error) {
	// Try to get stored token
	tokenInterface, err := e.tokenStorage.GetToken("openai", userEmail)
	if err != nil {
		return nil, fmt.Errorf("no stored token for user: %w", err)
	}

	token, ok := tokenInterface.(*llmux.ChatGPTPlusToken)
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
			if err := e.tokenStorage.SaveToken("openai", userEmail, newToken); err != nil {
				log.Warnf("failed to save refreshed token: %v", err)
			}

			return newToken, nil
		}

		return nil, fmt.Errorf("token expired and no refresh token available")
	}

	return token, nil
}

// transformRequestToOpenAI transforms an executor request to OpenAI API format
func (e *LLMuxChatGPTExecutor) transformRequestToOpenAI(req cliproxyexecutor.Request, opts cliproxyexecutor.Options) ([]byte, error) {
	// Transform from source format to OpenAI format
	from := opts.SourceFormat
	to := sdktranslator.FromString("openai")
	stream := opts.Stream

	// Use streaming translation to preserve function calling
	body := sdktranslator.TranslateRequest(from, to, req.Model, bytes.Clone(req.Payload), stream)
	return body, nil
}

// callOpenAIAPI makes an HTTP request to the OpenAI API
func (e *LLMuxChatGPTExecutor) callOpenAIAPI(ctx context.Context, token *llmux.ChatGPTPlusToken, body []byte) ([]byte, error) {
	url := "https://api.openai.com/v1/chat/completions"

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	// Add authorization header
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
	httpReq.Header.Set("Content-Type", "application/json")

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

// streamOpenAIAPI makes a streaming HTTP request to the OpenAI API
func (e *LLMuxChatGPTExecutor) streamOpenAIAPI(ctx context.Context, token *llmux.ChatGPTPlusToken, body []byte) <-chan cliproxyexecutor.StreamChunk {
	ch := make(chan cliproxyexecutor.StreamChunk, 10)

	go func() {
		defer close(ch)

		url := "https://api.openai.com/v1/chat/completions"

		// Create HTTP request
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			ch <- cliproxyexecutor.StreamChunk{Err: err}
			return
		}

		// Add authorization header
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
		httpReq.Header.Set("Content-Type", "application/json")

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
				// Skip [DONE] marker
				if string(chunk) != "[DONE]" {
					ch <- cliproxyexecutor.StreamChunk{Payload: chunk}
				}
			}
		}

		if err := scanner.Err(); err != nil {
			ch <- cliproxyexecutor.StreamChunk{Err: err}
		}
	}()

	return ch
}

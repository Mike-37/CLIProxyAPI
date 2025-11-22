package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/auth/llmux"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/util"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/executor"
	sdktranslator "github.com/router-for-me/CLIProxyAPI/v6/sdk/translator"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/sjson"
)

// LLMuxChatGPTExecutor executes requests to ChatGPT Plus via LLMux OAuth.
// This executor handles in-process OAuth token management and makes direct
// API calls to OpenAI's API using LLMux-obtained credentials.
type LLMuxChatGPTExecutor struct {
	cfg         *config.Config
	authDir     string
	chatgptAuth *llmux.ChatGPTPlusAuth
}

// NewLLMuxChatGPTExecutor creates a new LLMux ChatGPT executor.
func NewLLMuxChatGPTExecutor(cfg *config.Config) *LLMuxChatGPTExecutor {
	authDir := cfg.AuthDir
	if authDir == "" {
		authDir = "~/.cli-proxy-api"
	}

	return &LLMuxChatGPTExecutor{
		cfg:         cfg,
		authDir:     authDir,
		chatgptAuth: llmux.NewChatGPTPlusAuth(cfg),
	}
}

// Identifier returns the executor identifier.
func (e *LLMuxChatGPTExecutor) Identifier() string {
	return "llmux-chatgpt"
}

// PrepareRequest is called before execution (no-op for OAuth).
func (e *LLMuxChatGPTExecutor) PrepareRequest(_ *http.Request, _ *cliproxyauth.Auth) error {
	return nil
}

// Execute performs a non-streaming request to ChatGPT via LLMux.
func (e *LLMuxChatGPTExecutor) Execute(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (resp cliproxyexecutor.Response, err error) {
	// Load or refresh token
	accessToken, err := e.getValidAccessToken(ctx)
	if err != nil {
		return resp, fmt.Errorf("failed to get access token: %w", err)
	}

	// Translate request to OpenAI format (codex)
	from := opts.SourceFormat
	to := sdktranslator.FromString("codex")
	stream := from != to
	body := sdktranslator.TranslateRequest(from, to, req.Model, bytes.Clone(req.Payload), stream)

	// Apply payload config
	body = applyPayloadConfig(e.cfg, req.Model, body)

	// TODO: Update with actual OpenAI API endpoint (may be proxied through LLMux)
	// For now, using standard OpenAI API endpoint
	baseURL := "https://api.openai.com"
	url := fmt.Sprintf("%s/v1/chat/completions", baseURL)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return resp, err
	}

	// Set headers with LLMux OAuth token
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	httpReq.Header.Set("X-LLMux-Provider", "chatgpt-plus") // LLMux identifier

	log.WithFields(log.Fields{
		"provider": e.Identifier(),
		"model":    req.Model,
		"url":      url,
	}).Debug("Executing LLMux ChatGPT request")

	httpClient := util.SetProxy(&e.cfg.SDKConfig, &http.Client{})
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		return resp, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		body, _ := io.ReadAll(httpResp.Body)
		log.Errorf("LLMux ChatGPT request failed: status=%d, body=%s", httpResp.StatusCode, string(body))
		return resp, fmt.Errorf("request failed with status %d: %s", httpResp.StatusCode, string(body))
	}

	data, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return resp, err
	}

	// Translate response back to source format
	out := sdktranslator.TranslateNonStream(ctx, to, from, req.Model, bytes.Clone(opts.OriginalRequest), body, data, nil)
	resp = cliproxyexecutor.Response{Payload: []byte(out)}

	log.WithFields(log.Fields{
		"provider": e.Identifier(),
		"model":    req.Model,
		"status":   httpResp.StatusCode,
	}).Info("LLMux ChatGPT request completed")

	return resp, nil
}

// ExecuteStream performs a streaming request to ChatGPT via LLMux.
func (e *LLMuxChatGPTExecutor) ExecuteStream(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (<-chan cliproxyexecutor.StreamChunk, error) {
	// Load or refresh token
	accessToken, err := e.getValidAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	// Translate request to OpenAI format
	from := opts.SourceFormat
	to := sdktranslator.FromString("codex")
	body := sdktranslator.TranslateRequest(from, to, req.Model, bytes.Clone(req.Payload), true)

	// Enable streaming
	body, _ = sjson.SetBytes(body, "stream", true)
	body = applyPayloadConfig(e.cfg, req.Model, body)

	baseURL := "https://api.openai.com"
	url := fmt.Sprintf("%s/v1/chat/completions", baseURL)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	httpReq.Header.Set("X-LLMux-Provider", "chatgpt-plus")

	log.WithFields(log.Fields{
		"provider":  e.Identifier(),
		"model":     req.Model,
		"streaming": true,
	}).Debug("Executing LLMux ChatGPT streaming request")

	httpClient := util.SetProxy(&e.cfg.SDKConfig, &http.Client{})
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		body, _ := io.ReadAll(httpResp.Body)
		httpResp.Body.Close()
		log.Errorf("LLMux ChatGPT streaming request failed: status=%d, body=%s", httpResp.StatusCode, string(body))
		return nil, fmt.Errorf("streaming request failed with status %d: %s", httpResp.StatusCode, string(body))
	}

	// Create channel and start streaming
	streamChan := make(chan cliproxyexecutor.StreamChunk, 10)

	go func() {
		defer close(streamChan)
		defer httpResp.Body.Close()

		// Stream translator handles SSE -> format conversion
		sdktranslator.StreamTranslator(ctx, to, from, req.Model, bytes.Clone(opts.OriginalRequest), body, httpResp.Body, streamChan, nil)
	}()

	return streamChan, nil
}

// getValidAccessToken retrieves a valid access token, refreshing if necessary.
func (e *LLMuxChatGPTExecutor) getValidAccessToken(ctx context.Context) (string, error) {
	// Try to load existing token
	token, err := llmux.LoadAnyToken(e.authDir, "llmux-chatgpt")
	if err != nil {
		return "", fmt.Errorf("no LLMux ChatGPT token found. Please authenticate via /v1/auth/llmux/chatgpt/login")
	}

	// Check if token needs refresh
	if token.NeedsRefresh() {
		log.WithFields(log.Fields{
			"provider": "llmux-chatgpt",
			"email":    token.UserEmail,
		}).Info("Refreshing LLMux ChatGPT token")

		newTokenResp, err := e.chatgptAuth.RefreshToken(ctx, token.RefreshToken)
		if err != nil {
			return "", fmt.Errorf("failed to refresh token: %w", err)
		}

		// Calculate new expiry
		expiresAt := e.chatgptAuth.CalculateExpiryTime(newTokenResp.ExpiresIn)

		// Save refreshed token
		if err := llmux.SaveToken(
			e.authDir,
			"llmux-chatgpt",
			newTokenResp.AccessToken,
			newTokenResp.RefreshToken,
			expiresAt,
			newTokenResp.User.Email,
			newTokenResp.User.ID,
		); err != nil {
			log.Warnf("Failed to save refreshed token: %v", err)
		}

		return newTokenResp.AccessToken, nil
	}

	return token.AccessToken, nil
}

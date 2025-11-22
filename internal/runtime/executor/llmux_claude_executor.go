package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/auth/llmux"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/util"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/executor"
	sdktranslator "github.com/router-for-me/CLIProxyAPI/v6/sdk/translator"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/sjson"
)

// LLMuxClaudeExecutor executes requests to Claude Pro via LLMux OAuth.
// This executor handles in-process OAuth token management and makes direct
// API calls to Claude's API using LLMux-obtained credentials.
type LLMuxClaudeExecutor struct {
	cfg       *config.Config
	authDir   string
	claudeAuth *llmux.ClaudeProAuth
}

// NewLLMuxClaudeExecutor creates a new LLMux Claude executor.
func NewLLMuxClaudeExecutor(cfg *config.Config) *LLMuxClaudeExecutor {
	authDir := cfg.AuthDir
	if authDir == "" {
		authDir = "~/.cli-proxy-api"
	}

	return &LLMuxClaudeExecutor{
		cfg:       cfg,
		authDir:   authDir,
		claudeAuth: llmux.NewClaudeProAuth(cfg),
	}
}

// Identifier returns the executor identifier.
func (e *LLMuxClaudeExecutor) Identifier() string {
	return "llmux-claude"
}

// PrepareRequest is called before execution (no-op for OAuth).
func (e *LLMuxClaudeExecutor) PrepareRequest(_ *http.Request, _ *cliproxyauth.Auth) error {
	return nil
}

// Execute performs a non-streaming request to Claude via LLMux.
func (e *LLMuxClaudeExecutor) Execute(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (resp cliproxyexecutor.Response, err error) {
	// Load or refresh token
	accessToken, err := e.getValidAccessToken(ctx)
	if err != nil {
		return resp, fmt.Errorf("failed to get access token: %w", err)
	}

	// Translate request to Claude format
	from := opts.SourceFormat
	to := sdktranslator.FromString("claude")
	stream := from != to
	body := sdktranslator.TranslateRequest(from, to, req.Model, bytes.Clone(req.Payload), stream)

	// Apply payload config
	body = applyPayloadConfig(e.cfg, req.Model, body)

	// TODO: Update with actual Claude API endpoint (may be proxied through LLMux)
	// For now, using standard Claude API endpoint
	baseURL := "https://api.anthropic.com"
	url := fmt.Sprintf("%s/v1/messages", baseURL)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return resp, err
	}

	// Set headers with LLMux OAuth token
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("X-LLMux-Provider", "claude-pro") // LLMux identifier

	log.WithFields(log.Fields{
		"provider": e.Identifier(),
		"model":    req.Model,
		"url":      url,
	}).Debug("Executing LLMux Claude request")

	httpClient := util.SetProxy(&e.cfg.SDKConfig, &http.Client{})
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		return resp, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		body, _ := io.ReadAll(httpResp.Body)
		log.Errorf("LLMux Claude request failed: status=%d, body=%s", httpResp.StatusCode, string(body))
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
	}).Info("LLMux Claude request completed")

	return resp, nil
}

// ExecuteStream performs a streaming request to Claude via LLMux.
func (e *LLMuxClaudeExecutor) ExecuteStream(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (<-chan cliproxyexecutor.StreamChunk, error) {
	// Load or refresh token
	accessToken, err := e.getValidAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	// Translate request to Claude format
	from := opts.SourceFormat
	to := sdktranslator.FromString("claude")
	body := sdktranslator.TranslateRequest(from, to, req.Model, bytes.Clone(req.Payload), true)

	// Enable streaming
	body, _ = sjson.SetBytes(body, "stream", true)
	body = applyPayloadConfig(e.cfg, req.Model, body)

	baseURL := "https://api.anthropic.com"
	url := fmt.Sprintf("%s/v1/messages", baseURL)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("X-LLMux-Provider", "claude-pro")

	log.WithFields(log.Fields{
		"provider": e.Identifier(),
		"model":    req.Model,
		"streaming": true,
	}).Debug("Executing LLMux Claude streaming request")

	httpClient := util.SetProxy(&e.cfg.SDKConfig, &http.Client{})
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		body, _ := io.ReadAll(httpResp.Body)
		httpResp.Body.Close()
		log.Errorf("LLMux Claude streaming request failed: status=%d, body=%s", httpResp.StatusCode, string(body))
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
func (e *LLMuxClaudeExecutor) getValidAccessToken(ctx context.Context) (string, error) {
	// Try to load existing token
	token, err := llmux.LoadAnyToken(e.authDir, "llmux-claude")
	if err != nil {
		return "", fmt.Errorf("no LLMux Claude token found. Please authenticate via /v1/auth/llmux/claude/login")
	}

	// Check if token needs refresh
	if token.NeedsRefresh() {
		log.WithFields(log.Fields{
			"provider": "llmux-claude",
			"email":    token.UserEmail,
		}).Info("Refreshing LLMux Claude token")

		newTokenResp, err := e.claudeAuth.RefreshToken(ctx, token.RefreshToken)
		if err != nil {
			return "", fmt.Errorf("failed to refresh token: %w", err)
		}

		// Calculate new expiry
		expiresAt := e.claudeAuth.CalculateExpiryTime(newTokenResp.ExpiresIn)

		// Save refreshed token
		if err := llmux.SaveToken(
			e.authDir,
			"llmux-claude",
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

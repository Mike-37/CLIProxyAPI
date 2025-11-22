package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/auth/ctonew"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/util"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/executor"
	sdktranslator "github.com/router-for-me/CLIProxyAPI/v6/sdk/translator"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/sjson"
)

// CtonewExecutor executes requests to EngineLabs via Clerk JWT authentication.
// This executor handles Clerk JWT parsing, rotating token exchange, and API calls.
type CtonewExecutor struct {
	cfg          *config.Config
	clerkClient  *ctonew.ClerkClient
	cachedJWT    string
	jwtExpiresAt time.Time
}

// NewCtonewExecutor creates a new ctonew executor.
func NewCtonewExecutor(cfg *config.Config) *CtonewExecutor {
	return &CtonewExecutor{
		cfg:         cfg,
		clerkClient: ctonew.NewClerkClient(cfg),
	}
}

// Identifier returns the executor identifier.
func (e *CtonewExecutor) Identifier() string {
	return "ctonew"
}

// PrepareRequest is called before execution (no-op).
func (e *CtonewExecutor) PrepareRequest(_ *http.Request, _ *cliproxyauth.Auth) error {
	return nil
}

// Execute performs a non-streaming request via ctonew.
func (e *CtonewExecutor) Execute(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (resp cliproxyexecutor.Response, err error) {
	// Get valid JWT (from cache or exchange)
	jwt, err := e.getValidJWT(ctx, auth)
	if err != nil {
		return resp, fmt.Errorf("failed to get valid JWT: %w", err)
	}

	// Determine which upstream provider to use based on model
	// ctonew supports both Claude and GPT models
	var apiURL string
	var authHeader string

	if isClaudeModel(req.Model) {
		// Route to Claude via EngineLabs
		apiURL = "https://api.enginelabs.ai/v1/claude/messages"
		authHeader = fmt.Sprintf("Bearer %s", jwt)
	} else {
		// Route to OpenAI via EngineLabs
		apiURL = "https://api.enginelabs.ai/v1/chat/completions"
		authHeader = fmt.Sprintf("Bearer %s", jwt)
	}

	// Translate request to appropriate format
	from := opts.SourceFormat
	var to string
	if isClaudeModel(req.Model) {
		to = "claude"
	} else {
		to = "codex"
	}

	stream := from != sdktranslator.FromString(to)
	body := sdktranslator.TranslateRequest(from, sdktranslator.FromString(to), req.Model, bytes.Clone(req.Payload), stream)
	body = applyPayloadConfig(e.cfg, req.Model, body)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return resp, err
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", authHeader)
	httpReq.Header.Set("X-Provider", "ctonew")

	log.WithFields(log.Fields{
		"provider": e.Identifier(),
		"model":    req.Model,
		"url":      apiURL,
	}).Debug("Executing ctonew request")

	httpClient := util.SetProxy(&e.cfg.SDKConfig, &http.Client{})
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		return resp, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		body, _ := io.ReadAll(httpResp.Body)
		log.Errorf("ctonew request failed: status=%d, body=%s", httpResp.StatusCode, string(body))
		return resp, fmt.Errorf("request failed with status %d: %s", httpResp.StatusCode, string(body))
	}

	data, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return resp, err
	}

	// Translate response back
	toFormat := sdktranslator.FromString(to)
	out := sdktranslator.TranslateNonStream(ctx, toFormat, from, req.Model, bytes.Clone(opts.OriginalRequest), body, data, nil)
	resp = cliproxyexecutor.Response{Payload: []byte(out)}

	log.WithFields(log.Fields{
		"provider": e.Identifier(),
		"model":    req.Model,
		"status":   httpResp.StatusCode,
	}).Info("ctonew request completed")

	return resp, nil
}

// ExecuteStream performs a streaming request via ctonew.
func (e *CtonewExecutor) ExecuteStream(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (<-chan cliproxyexecutor.StreamChunk, error) {
	// Get valid JWT
	jwt, err := e.getValidJWT(ctx, auth)
	if err != nil {
		return nil, fmt.Errorf("failed to get valid JWT: %w", err)
	}

	// Determine API URL
	var apiURL string
	var authHeader string

	if isClaudeModel(req.Model) {
		apiURL = "https://api.enginelabs.ai/v1/claude/messages"
		authHeader = fmt.Sprintf("Bearer %s", jwt)
	} else {
		apiURL = "https://api.enginelabs.ai/v1/chat/completions"
		authHeader = fmt.Sprintf("Bearer %s", jwt)
	}

	// Translate request
	from := opts.SourceFormat
	var to string
	if isClaudeModel(req.Model) {
		to = "claude"
	} else {
		to = "codex"
	}

	body := sdktranslator.TranslateRequest(from, sdktranslator.FromString(to), req.Model, bytes.Clone(req.Payload), true)
	body, _ = sjson.SetBytes(body, "stream", true)
	body = applyPayloadConfig(e.cfg, req.Model, body)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", authHeader)
	httpReq.Header.Set("X-Provider", "ctonew")

	log.WithFields(log.Fields{
		"provider":  e.Identifier(),
		"model":     req.Model,
		"streaming": true,
	}).Debug("Executing ctonew streaming request")

	httpClient := util.SetProxy(&e.cfg.SDKConfig, &http.Client{})
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		body, _ := io.ReadAll(httpResp.Body)
		httpResp.Body.Close()
		log.Errorf("ctonew streaming request failed: status=%d, body=%s", httpResp.StatusCode, string(body))
		return nil, fmt.Errorf("streaming request failed with status %d: %s", httpResp.StatusCode, string(body))
	}

	streamChan := make(chan cliproxyexecutor.StreamChunk, 10)

	go func() {
		defer close(streamChan)
		defer httpResp.Body.Close()

		toFormat := sdktranslator.FromString(to)
		sdktranslator.StreamTranslator(ctx, toFormat, from, req.Model, bytes.Clone(opts.OriginalRequest), body, httpResp.Body, streamChan, nil)
	}()

	return streamChan, nil
}

// getValidJWT retrieves a valid JWT, exchanging the rotating token if necessary.
func (e *CtonewExecutor) getValidJWT(ctx context.Context, auth *cliproxyauth.Auth) (string, error) {
	// Check if we have a cached JWT that's still valid
	if e.cachedJWT != "" && time.Now().Before(e.jwtExpiresAt.Add(-5*time.Minute)) {
		return e.cachedJWT, nil
	}

	// Get Clerk JWT from auth
	if auth == nil || auth.Attributes == nil {
		return "", fmt.Errorf("no auth provided. Please provide Clerk JWT cookie via /v1/auth/ctonew")
	}

	clerkJWT, ok := auth.Attributes["clerk_jwt_cookie"]
	if !ok || clerkJWT == "" {
		return "", fmt.Errorf("no clerk_jwt_cookie found in auth attributes")
	}

	// Parse JWT to get rotating token
	claims, err := ctonew.ParseClerkJWT(clerkJWT)
	if err != nil {
		return "", fmt.Errorf("failed to parse Clerk JWT: %w", err)
	}

	// Check if JWT needs refresh
	if !claims.NeedsRefresh() {
		// JWT is still valid, use it directly
		e.cachedJWT = clerkJWT
		e.jwtExpiresAt = claims.ExpiresAt()
		return clerkJWT, nil
	}

	// Exchange rotating token for new JWT
	log.WithFields(log.Fields{
		"provider": e.Identifier(),
	}).Info("Exchanging Clerk rotating token for new JWT")

	newJWT, err := e.clerkClient.ExchangeRotatingToken(ctx, claims.RotatingToken)
	if err != nil {
		return "", fmt.Errorf("failed to exchange rotating token: %w", err)
	}

	// Parse new JWT to get expiry
	newClaims, err := ctonew.ParseClerkJWT(newJWT)
	if err != nil {
		return "", fmt.Errorf("failed to parse new JWT: %w", err)
	}

	// Cache the new JWT
	e.cachedJWT = newJWT
	e.jwtExpiresAt = newClaims.ExpiresAt()

	return newJWT, nil
}

// isClaudeModel checks if a model name indicates a Claude model.
func isClaudeModel(model string) bool {
	// Check for Claude model patterns
	return len(model) >= 6 && model[:6] == "claude"
}

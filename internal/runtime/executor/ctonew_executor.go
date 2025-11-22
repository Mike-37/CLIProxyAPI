package executor

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/auth/ctonew"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/executor"
	sdktranslator "github.com/router-for-me/CLIProxyAPI/v6/sdk/translator"
	log "github.com/sirupsen/logrus"
)

const (
	// ctonew API endpoint
	ctonewAPIEndpoint = "https://api.enginelabs.ai/v1/chat/completions"
)

// CtonewExecutor implements the Executor interface for ctonew JWT-based authentication
type CtonewExecutor struct {
	jwtParser     *ctonew.ClerkJWTParser
	tokenExchange *ctonew.ClerkTokenExchange
	httpClient    *http.Client
}

// NewCtonewExecutor creates a new ctonew executor
func NewCtonewExecutor(
	jwtParser *ctonew.ClerkJWTParser,
	tokenExchange *ctonew.ClerkTokenExchange,
) *CtonewExecutor {
	return &CtonewExecutor{
		jwtParser:     jwtParser,
		tokenExchange: tokenExchange,
		httpClient:    &http.Client{Timeout: 0},
	}
}

// Identifier returns the executor identifier
func (e *CtonewExecutor) Identifier() string {
	return "ctonew"
}

// PrepareRequest prepares the HTTP request (no-op for this executor)
func (e *CtonewExecutor) PrepareRequest(_ *http.Request, _ *cliproxyauth.Auth) error {
	return nil
}

// Execute executes a non-streaming request using ctonew
func (e *CtonewExecutor) Execute(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (resp cliproxyexecutor.Response, err error) {
	// Extract stored JWT from auth context or metadata
	clerkJWT := extractClerkJWT(auth)
	if clerkJWT == "" {
		return resp, fmt.Errorf("no Clerk JWT found in auth context")
	}

	// Exchange JWT for access token
	accessToken, err := e.getAccessToken(ctx, clerkJWT)
	if err != nil {
		return resp, fmt.Errorf("failed to get access token: %w", err)
	}

	// Transform request to ctonew API format
	ctonewReq, err := e.transformRequestToCtonew(req, opts)
	if err != nil {
		return resp, fmt.Errorf("failed to transform request: %w", err)
	}

	// Make HTTP request to ctonew API
	httpResp, err := e.callCtonewAPI(ctx, accessToken, ctonewReq)
	if err != nil {
		return resp, err
	}

	// Transform response back to executor format
	resp.Payload = httpResp
	return resp, nil
}

// ExecuteStream executes a streaming request using ctonew
func (e *CtonewExecutor) ExecuteStream(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (<-chan cliproxyexecutor.StreamChunk, error) {
	// Extract stored JWT from auth context or metadata
	clerkJWT := extractClerkJWT(auth)
	if clerkJWT == "" {
		ch := make(chan cliproxyexecutor.StreamChunk, 1)
		ch <- cliproxyexecutor.StreamChunk{Err: fmt.Errorf("no Clerk JWT found in auth context")}
		close(ch)
		return ch, nil
	}

	// Exchange JWT for access token
	accessToken, err := e.getAccessToken(ctx, clerkJWT)
	if err != nil {
		ch := make(chan cliproxyexecutor.StreamChunk, 1)
		ch <- cliproxyexecutor.StreamChunk{Err: fmt.Errorf("failed to get access token: %w", err)}
		close(ch)
		return ch, nil
	}

	// Transform request to ctonew API format (with streaming enabled)
	ctonewReq, err := e.transformRequestToCtonew(req, opts)
	if err != nil {
		ch := make(chan cliproxyexecutor.StreamChunk, 1)
		ch <- cliproxyexecutor.StreamChunk{Err: fmt.Errorf("failed to transform request: %w", err)}
		close(ch)
		return ch, nil
	}

	// Make streaming HTTP request to ctonew API
	return e.streamCtonewAPI(ctx, accessToken, ctonewReq), nil
}

// getAccessToken gets a valid access token, exchanging the JWT if necessary
func (e *CtonewExecutor) getAccessToken(ctx context.Context, clerkJWT string) (string, error) {
	// Parse JWT to extract rotating token
	claims, err := e.jwtParser.ParseToken(clerkJWT)
	if err != nil {
		return "", fmt.Errorf("failed to parse JWT: %w", err)
	}

	// Check if we have a cached token
	if cached, ok := e.tokenExchange.GetCachedToken(claims.RotatingToken); ok {
		return cached.AccessToken, nil
	}

	// Exchange token
	exchanged, err := e.tokenExchange.ExchangeToken(ctx, clerkJWT)
	if err != nil {
		return "", fmt.Errorf("failed to exchange token: %w", err)
	}

	return exchanged.AccessToken, nil
}

// transformRequestToCtonew transforms an executor request to ctonew API format
func (e *CtonewExecutor) transformRequestToCtonew(req cliproxyexecutor.Request, opts cliproxyexecutor.Options) ([]byte, error) {
	// Transform from source format to ctonew format
	from := opts.SourceFormat
	to := sdktranslator.FromString("claude")  // ctonew uses Claude-compatible format
	stream := opts.Stream

	// Use streaming translation to preserve function calling
	body := sdktranslator.TranslateRequest(from, to, req.Model, bytes.Clone(req.Payload), stream)
	return body, nil
}

// callCtonewAPI makes an HTTP request to the ctonew API
func (e *CtonewExecutor) callCtonewAPI(ctx context.Context, accessToken string, body []byte) ([]byte, error) {
	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, ctonewAPIEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	// Add authorization header
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
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

// streamCtonewAPI makes a streaming HTTP request to the ctonew API
func (e *CtonewExecutor) streamCtonewAPI(ctx context.Context, accessToken string, body []byte) <-chan cliproxyexecutor.StreamChunk {
	ch := make(chan cliproxyexecutor.StreamChunk, 10)

	go func() {
		defer close(ch)

		// Create HTTP request
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, ctonewAPIEndpoint, bytes.NewReader(body))
		if err != nil {
			ch <- cliproxyexecutor.StreamChunk{Err: err}
			return
		}

		// Add authorization header
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
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
				ch <- cliproxyexecutor.StreamChunk{Payload: chunk}
			}
		}

		if err := scanner.Err(); err != nil {
			ch <- cliproxyexecutor.StreamChunk{Err: err}
		}
	}()

	return ch
}

// extractClerkJWT extracts the Clerk JWT from auth context or metadata
func extractClerkJWT(auth *cliproxyauth.Auth) string {
	if auth == nil {
		return ""
	}

	// Try to get from metadata first
	if auth.Metadata != nil {
		if jwt, ok := auth.Metadata["clerk_jwt"].(string); ok && jwt != "" {
			return jwt
		}
	}

	// Try to get from auth label
	if auth.Label != "" {
		return auth.Label
	}

	return ""
}

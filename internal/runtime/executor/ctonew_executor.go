package executor

import (
	"context"
	"fmt"
	"net/http"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/auth/ctonew"
	"github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/executor"
)

const (
	// ctonew API endpoint
	ctonewAPIEndpoint = "https://api.enginelabs.ai"
)

// CtonewExecutor implements the Executor interface for ctonew
type CtonewExecutor struct {
	jwtParser       *ctonew.ClerkJWTParser
	tokenExchange   *ctonew.ClerkTokenExchange
	clerkJWT        string
	httpClient      *http.Client
}

// NewCtonewExecutor creates a new ctonew executor
func NewCtonewExecutor(
	jwtParser *ctonew.ClerkJWTParser,
	tokenExchange *ctonew.ClerkTokenExchange,
	clerkJWT string,
	httpClient *http.Client,
) *CtonewExecutor {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &CtonewExecutor{
		jwtParser:     jwtParser,
		tokenExchange: tokenExchange,
		clerkJWT:      clerkJWT,
		httpClient:    httpClient,
	}
}

// Name returns the executor name
func (e *CtonewExecutor) Name() string {
	return "ctonew"
}

// Execute executes a request using ctonew
func (e *CtonewExecutor) Execute(ctx context.Context, request *executor.ExecuteRequest) (*executor.ExecuteResponse, error) {
	// Exchange JWT for access token
	accessToken, err := e.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	// Call ctonew API
	return e.callCtonewAPI(ctx, accessToken, request)
}

// Stream executes a streaming request using ctonew
func (e *CtonewExecutor) Stream(ctx context.Context, request *executor.ExecuteRequest) (<-chan executor.StreamChunk, error) {
	// Exchange JWT for access token
	accessToken, err := e.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	// Call ctonew API with streaming
	return e.streamCtonewAPI(ctx, accessToken, request)
}

// getAccessToken gets a valid access token, exchanging the JWT if necessary
func (e *CtonewExecutor) getAccessToken(ctx context.Context) (string, error) {
	// Parse JWT to extract rotating token
	claims, err := e.jwtParser.ParseToken(e.clerkJWT)
	if err != nil {
		return "", fmt.Errorf("failed to parse JWT: %w", err)
	}

	// Check if we have a cached token
	if cached, ok := e.tokenExchange.GetCachedToken(claims.RotatingToken); ok {
		return cached.AccessToken, nil
	}

	// Exchange token
	exchanged, err := e.tokenExchange.ExchangeToken(ctx, e.clerkJWT)
	if err != nil {
		return "", fmt.Errorf("failed to exchange token: %w", err)
	}

	return exchanged.AccessToken, nil
}

// callCtonewAPI makes a request to the ctonew API
func (e *CtonewExecutor) callCtonewAPI(ctx context.Context, accessToken string, request *executor.ExecuteRequest) (*executor.ExecuteResponse, error) {
	// TODO: Implement ctonew API call
	// This would typically:
	// 1. Transform request format to ctonew API format
	// 2. Add Bearer token to Authorization header
	// 3. Make HTTP request to ctonew API
	// 4. Transform response back to executor format

	return nil, fmt.Errorf("not yet implemented")
}

// streamCtonewAPI makes a streaming request to the ctonew API
func (e *CtonewExecutor) streamCtonewAPI(ctx context.Context, accessToken string, request *executor.ExecuteRequest) (<-chan executor.StreamChunk, error) {
	// TODO: Implement ctonew API streaming
	// This would typically:
	// 1. Transform request format to ctonew API format
	// 2. Add Bearer token to Authorization header
	// 3. Make streaming HTTP request to ctonew API
	// 4. Transform streaming response chunks to executor format

	return nil, fmt.Errorf("not yet implemented")
}

# Provider Integration Analysis & Design Recommendations

## Executive Summary

You have 4 new provider proxies to integrate into CLIProxyAPI:

1. **WebAI-to-API** - Gemini browser cookies + gpt4free fallback (FastAPI, Python)
2. **AIstudioProxyAPI** - Google AI Studio with Playwright automation (FastAPI, Python)
3. **LLMux** - Claude Pro + ChatGPT Plus via official OAuth (FastAPI, Python)
4. **ctonew-proxy** - Ctonew/EngineLabs Clerk JWT cookies (Deno/TypeScript)

The core question: **Should these be merged into the existing auth system or implemented as separate parallel providers?**

**RECOMMENDATION: Hybrid Approach**
- **Merge**: WebAI, LLMux (OAuth-like patterns already exist in CLIProxyAPI)
- **Parallel**: AIstudioProxyAPI, ctonew-proxy (stateful browser/Clerk integration too different)

---

## Part 1: Current CLIProxyAPI Architecture Review

### Auth System Design
```
Layer 1: Request Authentication (AccessManager)
  └─ Validates API keys in HTTP headers
  └─ Controls access, not credentials

Layer 2: Provider Authentication (Auth Manager + Authenticators)
  ├─ Token Storage (FileStore, Postgres, Git, Object)
  ├─ Authenticators per Provider (Claude, Codex, Gemini, Qwen, etc.)
  ├─ OAuth Callback Handlers (local HTTP server per provider)
  └─ Auto-Refresh Mechanism (every 5 seconds)

Layer 3: Execution (Executors)
  └─ ProviderExecutor interface for each provider
  └─ Makes actual API calls with credentials from Layer 2
```

### Current Provider Pattern
```go
// Every provider implements this flow:

1. Authenticator (sdk/auth/provider.go)
   - Login(ctx, cfg) → Token
   - RefreshLead() → Duration
   - Refresh(ctx, token) → Token

2. Executor (internal/runtime/executor/provider_executor.go)
   - Execute(ctx, auth, req) → Response
   - ExecuteStream(ctx, auth, req) → <-chan StreamChunk
   - Refresh(ctx, auth) → *Auth

3. Registration (cmd/server/main.go + sdk/cliproxy/service.go)
   - NewAuthenticator()
   - NewExecutor()
   - RegisterModelsForAuth()

4. Token Persistence (via TokenStore)
   - File: ~/.cli-proxy-api/{provider}-{email}.json
   - Database: Postgres with provider/email keys
   - Git: Repository-backed
   - Object: S3-compatible storage
```

### Strengths
✅ Clean separation of concerns (auth vs execution)
✅ Pluggable token storage backends
✅ Built-in OAuth callback infrastructure
✅ Auto-refresh with provider-specific lead times
✅ Model registry per provider
✅ Round-robin load balancing across multiple accounts
✅ Extensible executor pattern
✅ No database required (file-based by default)

### Challenges for New Providers
❌ Assumes token-based OAuth (not cookie-based)
❌ Assumes simple token → API call (not stateful browser sessions)
❌ Assumes credentials = identity (not true for Clerk JWT or shared accounts)
❌ Auto-refresh assumes long-lived tokens (breaks with per-request token generation)
❌ Model registry tied to auth entries (not suitable for service-wide models)

---

## Part 2: New Providers Detailed Comparison

### 1. WebAI-to-API (MERGEABLE ✓)
**Type**: Browser cookie extraction + fallback proxy
**Auth Pattern**: Similar to existing Gemini API key approach (static credentials)

**Current State**:
```python
- Reads __Secure_1PSID cookie from browser
- Falls back to auto-extract if manual cookie empty
- Can use gpt4free for 50+ model providers
- Stores in config.conf
```

**Challenges**:
- Cookies are opaque, time-limited, browser-specific
- gpt4free providers have varying auth mechanisms
- Not suitable for production without cookie rotation strategy

**Integration Effort**: MEDIUM
- Cookie extraction similar to existing "api-key" approach
- Needs cookie validation/refresh logic
- gpt4free requires provider selection per model

---

### 2. AIstudioProxyAPI (PARALLEL ONLY ✗)
**Type**: Browser automation (Camoufox) with stateful session
**Auth Pattern**: Requires running browser instance

**Current State**:
```python
- Launches Camoufox (anti-fingerprint Firefox)
- Maintains auth_profiles/ with cookies/localStorage
- Supports three streaming tiers for low latency
- Requires interactive login once per session
```

**Why NOT to Merge**:
- Requires persistent Playwright/Camoufox process
- Maintains per-request stateful page interactions
- Browser fingerprinting is environment-specific
- Cannot auto-refresh tokens (requires new login)
- Expensive resource-wise (browser per instance)

**Integration Approach**: Run as separate service
```
┌─────────────────┐
│ CLIProxyAPI     │  ← Main router
└────────┬────────┘
         │
    Route to AIstudio
    model requests
         │
    ┌────▼──────────────────┐
    │ AIstudioProxyAPI       │  ← Separate micro-service
    │ (Playwright + Camoufox)│  ← Handles AI Studio only
    └───────────────────────┘
```

**Integration Effort**: HIGH (but isolated)

---

### 3. LLMux (MERGEABLE ✓)
**Type**: Official OAuth broker for Claude Pro + ChatGPT Plus
**Auth Pattern**: Extends existing OAuth pattern, superior tokens

**Current State**:
```python
- Uses official Anthropic OAuth (custom CLIENT_ID)
- Uses official OpenAI OAuth (auth.openai.com)
- Generates 1-year tokens (vs typical 30-90 days)
- Stores in ~/.llmux/tokens.json
- Supports reasoning/thinking models
- Routes based on model prefix
```

**Why MERGE**:
- OAuth pattern already exists in CLIProxyAPI (Claude, Codex)
- Token storage is identical (JSON with Bearer tokens)
- Auto-refresh is same pattern (check expiry, call Refresh)
- Multiple auth entries per provider already supported
- Model registry per auth entry already works

**Key Difference**: Supports reasoning models with thinking budgets

**Integration Effort**: LOW-MEDIUM
- Just add Gemini OAuth, ChatGPT OAuth authenticators
- Reuse existing OAuth callback infrastructure
- Add reasoning budget handling to Claude executor

---

### 4. ctonew-proxy (PARALLEL ONLY ✗)
**Type**: Stateless Clerk JWT cookie broker
**Auth Pattern**: Per-request JWT extraction, no persistent tokens

**Current State**:
```typescript
- Extracts rotating_token from __client JWT cookie
- Exchanges for new JWT via Clerk endpoint
- All stateless (no database/file storage)
- Single TypeScript file
```

**Why NOT to Merge**:
- Completely stateless (no token storage needed)
- Per-request token generation (not suitable for long-running connections)
- Clerk is external service dependency (not part of CLI auth model)
- Fixed single backend (enginelabs.ai)
- Very different credential model (shared Clerk JWT, not personal OAuth)

**Integration Approach**: Run as separate service
```
┌─────────────────┐
│ CLIProxyAPI     │  ← Main router
└────────┬────────┘
         │
    Route to ctonew
    model requests
         │
    ┌────▼──────────────┐
    │ ctonew-proxy      │  ← Separate micro-service
    │ (Deno + Oak)      │  ← Stateless JWT extraction
    └───────────────────┘
```

**Integration Effort**: MEDIUM (but isolated)

---

## Part 3: Recommended Architecture

### High-Level Design

```
┌──────────────────────────────────────────────────────────────────┐
│                        CLIProxyAPI Router                         │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌─ Request Entry Points                                        │
│  │  ├─ /v1/chat/completions (OpenAI)                          │
│  │  ├─ /v1/messages (Anthropic)                               │
│  │  ├─ /v1beta/* (Gemini)                                     │
│  │  └─ /v0/management/* (Admin)                               │
│  │                                                              │
│  ├─ Access Control (API Key validation)                        │
│  │                                                              │
│  ├─ Authentication Manager (merged providers)                  │
│  │  ├─ FileStore (token persistence)                          │
│  │  ├─ Authenticators:                                        │
│  │  │  ├─ Claude (existing)                                   │
│  │  │  ├─ Codex (existing)                                    │
│  │  │  ├─ Gemini (existing)                                   │
│  │  │  ├─ Qwen (existing)                                     │
│  │  │  ├─ iFlow (existing)                                    │
│  │  │  ├─ Antigravity (existing)                              │
│  │  │  ├─ Claude+ChatGPT via LLMux OAuth (NEW) ✨            │
│  │  │  ├─ Gemini Web via WebAI cookies (NEW) ✨              │
│  │  │  └─ OpenAI-compatible providers (NEW) ✨               │
│  │  └─ Auto-refresh mechanism                                 │
│  │                                                              │
│  ├─ Executor Registry (provider implementations)               │
│  │  ├─ ClaudeExecutor (existing)                             │
│  │  ├─ CodexExecutor (existing)                              │
│  │  ├─ GeminiExecutor (existing)                             │
│  │  ├─ QwenExecutor (existing)                               │
│  │  ├─ iFlowExecutor (existing)                              │
│  │  ├─ AntigravityExecutor (existing)                        │
│  │  ├─ WebAIExecutor (for WebAI-to-API) (NEW) ✨            │
│  │  ├─ LLMuxExecutor (for LLMux providers) (NEW) ✨          │
│  │  └─ OpenAICompatExecutor (for custom providers) (NEW) ✨  │
│  │                                                              │
│  ├─ Model Registry (global + per-auth)                        │
│  │  └─ Supports reasoning budget tracking                     │
│  │                                                              │
│  └─ Routing/Load Balancing (round-robin, model-aware)        │
│                                                                   │
└────────────────────────────────────────────────────────────────┘

Separate Micro-Services (NOT merged):

┌─────────────────────────────┐   ┌──────────────────────────────┐
│   AIstudioProxyAPI          │   │   ctonew-proxy Service       │
│  (Playwright + Camoufox)    │   │  (Deno + Clerk JWT)          │
│                             │   │                              │
│ ├─ Browser automation       │   │ ├─ JWT extraction            │
│ ├─ Stateful AI Studio API   │   │ ├─ Clerk token exchange      │
│ ├─ Session management       │   │ ├─ Stateless pass-through    │
│ ├─ 3-tier streaming         │   │ └─ EngineLabs backend        │
│ ├─ Model switching          │   │                              │
│ └─ Anti-fingerprint layer   │   │                              │
│                             │   │                              │
│ Accessed via:               │   │ Accessed via:                │
│ - Internal service calls    │   │ - Internal service calls     │
│ - gRPC or HTTP routes       │   │ - HTTP routes                │
└─────────────────────────────┘   └──────────────────────────────┘

                    ▲                              ▲
                    │                              │
        Route AI Studio requests      Route Ctonew/EngineLabs
                                                requests
```

---

## Part 4: Detailed Integration Plan

### 4.1 MERGED PROVIDERS (WebAI-to-API + LLMux)

#### Step 1: Analyze Current OAuth Pattern
**Files to Review**:
- `/sdk/auth/claude.go` - OAuth implementation pattern
- `/internal/auth/claude/openai_auth.go` - Token exchange
- `/sdk/auth/manager.go` - AuthenticatorInterface

**Key Patterns**:
```go
type AuthenticatorInterface interface {
    Provider() string
    RefreshLead() *time.Duration
    Login(ctx, cfg, opts) (*Auth, error)
    // Token refresh handled by executor
}

type Auth struct {
    ID           string
    Provider     string
    Storage      TokenStorage
    Metadata     map[string]string  // email, project_id, etc
    Attributes   map[string]string  // api_key, base_url, etc
    Status       AuthStatus
    AccessToken  string
    RefreshToken string
    ExpiresAt    time.Time
    // ...
}
```

#### Step 2: Create WebAI-to-API Authenticator

**File**: `/sdk/auth/webai.go`

**Implementation**:
```go
type WebAIAuthenticator struct{}

func (a *WebAIAuthenticator) Provider() string {
    return "webai" // or "gemini-web"
}

func (a *WebAIAuthenticator) RefreshLead() *time.Duration {
    lead := 6 * time.Hour // cookies expire after ~24 hours
    return &lead
}

func (a *WebAIAuthenticator) Login(ctx context.Context, cfg *config.Config, opts map[string]interface{}) (*coreauth.Auth, error) {
    // Option 1: Read from config (pre-extracted cookies)
    // Option 2: Auto-extract from browser using go-based cookie reader
    // Returns Auth with:
    // - Metadata: {"browser": "firefox", "cookies_path": "..."}
    // - Attributes: {"__Secure_1PSID": "...", "__Secure_1PSIDTS": "..."}
    // - ExpiresAt: now + 24 hours
}

func (a *WebAIAuthenticator) Validate(ctx context.Context, auth *coreauth.Auth) error {
    // Try a simple request to Gemini API to verify cookie validity
}
```

**Challenges to Handle**:
- ❌ Cookies are browser/machine-specific (not portable)
- ❌ Cookies have opaque expiration times
- ❌ Can't programmatically refresh cookies (need browser automation)
- ✓ Solution: Move to separate AIstudioProxyAPI for stateful cookie management
  OR: Accept manual re-login per cookie expiration (6-12 hours)

**Recommendation**: Use WebAI-to-API only as fallback, support gpt4free passthrough
- Treat gpt4free providers as OpenAI-compatible endpoints
- Let WebAI-to-API run as separate service if cookies required

#### Step 3: Create LLMux Authenticators

**Files**:
- `/sdk/auth/claude_oauth.go` - Anthropic official OAuth (upgrade existing)
- `/sdk/auth/chatgpt_oauth.go` - OpenAI official OAuth (upgrade existing)
- `/internal/auth/claude_oauth/` - Token exchange implementation
- `/internal/auth/chatgpt_oauth/` - Token exchange implementation

**Key Insight**: LLMux uses same pattern as existing Codex/Claude authenticators
- Just uses different CLIENT_ID for Anthropic
- Different OAuth issuer for OpenAI
- Same token format (Bearer token)
- Same refresh logic (check expiry, POST refresh_token)

**Implementation**:
```go
// /sdk/auth/claude_oauth.go
type ClaudeOAuthAuthenticator struct{}

func (a *ClaudeOAuthAuthenticator) Provider() string {
    return "claude-oauth" // distinct from api-key based
}

func (a *ClaudeOAuthAuthenticator) RefreshLead() *time.Duration {
    lead := 30 * time.Minute // Refresh 30 min before 1-year expiry
    return &lead
}

func (a *ClaudeOAuthAuthenticator) Login(ctx context.Context, cfg *config.Config, opts map[string]interface{}) (*coreauth.Auth, error) {
    // Use LLMux's Anthropic OAuth CLIENT_ID
    // Generate PKCE codes
    // Start callback server on port (config)
    // Open browser with:
    //   https://accounts.anthropic.com/login?
    //     client_id=9d1c250a-e61b-44d9-88ed-5944d1962f5e&
    //     redirect_uri=http://localhost:54546/anthropic/callback&
    //     response_type=code&
    //     scope=... &
    //     state=... &
    //     code_challenge=...
    // Wait for callback
    // Exchange code for token (1-year)
    // Store token with expiry
}

// Similar for ChatGPT OAuth
```

**Executor Changes**:
- Existing ClaudeExecutor already supports OAuth tokens
- Add reasoning budget support (thinking tokens parameter)
- Add beta header for reasoning models

**Models to Add**:
```go
// Add to model registry
claude-sonnet-4-5-20250929     (Claude Pro)
claude-opus-4-1-20250805       (Claude Pro)
claude-haiku-4-5-20251001      (Claude Pro)
gpt-5                          (ChatGPT Plus)
gpt-5-1                        (ChatGPT Plus)
gpt-5-codex                    (ChatGPT Plus)
```

---

### 4.2 PARALLEL PROVIDERS (AIstudioProxyAPI + ctonew-proxy)

#### Option A: Run as Separate Services

**Architecture**:
```
CLIProxyAPI (main router)
├─ Existing providers (Claude, Codex, Gemini, etc.) → local executors
├─ LLMux providers (Claude+ChatGPT OAuth) → local executors
└─ WebAI providers (Gemini web cookies) → local executors + gpt4free

AIstudioProxyAPI (separate service)
└─ AI Studio Gemini → Playwright + Camoufox

ctonew-proxy (separate service)
└─ Ctonew/EngineLabs → Clerk JWT
```

**Router Changes in CLIProxyAPI**:
```go
// In config:
aiStudioProxy:
  enabled: true
  endpoint: "http://localhost:8318" // separate service port

ctonewProxy:
  enabled: true
  endpoint: "http://localhost:8319"  // separate service port

// In executor registry (sdk/cliproxy/service.go):
case "aistudio":
    m.RegisterExecutor(executor.NewHTTPProxyExecutor(
        cfg.AIStudioProxy.Endpoint,
        "aistudio",
    ))
case "ctonew":
    m.RegisterExecutor(executor.NewHTTPProxyExecutor(
        cfg.CtonewProxy.Endpoint,
        "ctonew",
    ))
```

**HTTP Proxy Executor**:
```go
// /internal/runtime/executor/http_proxy_executor.go
type HTTPProxyExecutor struct {
    endpoint string
    provider string
    client   *http.Client
}

func (e *HTTPProxyExecutor) Execute(ctx context.Context, auth *coreauth.Auth, req *RequestPayload, opts ...) (Response, error) {
    // Pass-through request to upstream service
    // Auth.Attributes["endpoint"] contains API key if needed
    // Return response as-is
}

func (e *HTTPProxyExecutor) ExecuteStream(ctx context.Context, auth *coreauth.Auth, req *RequestPayload, opts ...) (<-chan StreamChunk, error) {
    // Stream pass-through
}
```

**Pros**:
- ✅ No coupling between router and complex browser/stateful services
- ✅ Can scale independently
- ✅ Easy to update/restart without affecting main router
- ✅ Different tech stacks are isolated
- ✅ Can add more backends without bloating main codebase

**Cons**:
- ❌ Network latency between services
- ❌ More complex deployment
- ❌ More processes to manage

---

#### Option B: Embedded Components (NOT RECOMMENDED)

Would require:
- ❌ Adding Go bindings for Playwright
- ❌ Embedding Camoufox or similar browser
- ❌ Adding stateful session management to Go
- ❌ Managing browser lifecycle in same process
- ❌ Memory overhead (browser = 200-500MB per instance)
- ❌ Complexity explosion

**Conclusion**: Option A (separate services) is much cleaner.

---

### 4.3 WebAI-to-API Detailed Approach

**Option 1: Merge as gpt4free fallback**
- Keep WebAI in `/sdk/auth/webai.go`
- Use cookies only for Gemini web access
- Support gpt4free providers via OpenAI-compatible executor
- Problem: gpt4free models need provider selection (model="gpt4free:chatgpt" ?)

**Option 2: Keep as separate service** (RECOMMENDED)
- Run WebAI-to-API separately (it's already FastAPI)
- Route webai-* models to it via HTTPProxyExecutor
- Let WebAI handle cookie management and gpt4free internally
- Cleaner separation: each service does one thing well

**Recommendation**: Option 2 (keep separate)
```
CLIProxyAPI config:
webaiProxy:
  enabled: true
  endpoint: "http://localhost:8320"  // WebAI-to-API service

Then in auth:
[
  {
    "provider": "webai",
    "model_regex": "webai:.*",  // models like "webai:gemini-2.0-flash"
    "endpoint": "http://localhost:8320"
  }
]
```

---

## Part 5: Architecture Decision Table

| Provider | Merge? | Reason | Integration | Complexity |
|----------|--------|--------|-------------|-----------|
| **LLMux (Claude+ChatGPT OAuth)** | ✅ YES | Token-based OAuth, matches existing pattern | New Authenticator + Executor | LOW-MEDIUM |
| **WebAI-to-API** | ⚠️ OPTIONAL | Cookie-based, could work but better separate | Keep as separate service | LOW |
| **AIstudioProxyAPI** | ❌ NO | Stateful browser automation required | HTTP proxy service | MEDIUM |
| **ctonew-proxy** | ❌ NO | Stateless JWT broker, different model | HTTP proxy service | LOW |

---

## Part 6: Implementation Roadmap

### Phase 1: Merge LLMux OAuth Providers (1-2 weeks)

**Step-by-step**:
1. Create `/sdk/auth/claude_oauth.go` (Anthropic OAuth)
2. Create `/sdk/auth/chatgpt_oauth.go` (OpenAI OAuth)
3. Create `/internal/auth/claude_oauth/` (token exchange)
4. Create `/internal/auth/chatgpt_oauth/` (token exchange)
5. Update ClaudeExecutor to support reasoning budgets
6. Add ChatGPT executor (`/internal/runtime/executor/chatgpt_executor.go`)
7. Add CLI login commands (`-claude-oauth-login`, `-chatgpt-oauth-login`)
8. Register models in model registry
9. Test end-to-end flows

**Key Consideration**: Handle "thinking" tokens in reasoning models
```go
type Request struct {
    // ... existing fields
    ThinkingBudgetTokens int    // NEW: for reasoning models
    MaxThinkingTokens    int    // NEW: max thinking tokens
}

// In executor, check if model is reasoning variant:
// If so, include thinking parameters in API call
```

### Phase 2: Implement Parallel Service Infrastructure (1 week)

**Step-by-step**:
1. Create `/internal/runtime/executor/http_proxy_executor.go`
2. Add config sections for parallel services
3. Test HTTP proxy pattern with dummy service
4. Document deployment approach

### Phase 3: Integrate AIstudioProxyAPI (1-2 weeks)

**Step-by-step**:
1. Deploy AIstudioProxyAPI to separate port
2. Create auth entry for AIstudio in config
3. Register models for aistudio
4. Test routing and proxy pass-through
5. Document setup and authentication flow

**Key**: AIstudioProxyAPI auth setup is manual (browser automation first-time)

### Phase 4: Integrate ctonew-proxy (1 week)

**Step-by-step**:
1. Deploy ctonew-proxy (already single file)
2. Create auth entry for ctonew in config
3. Register models for ctonew
4. Test routing and proxy pass-through

**Key**: ctonew-proxy requires Clerk JWT cookie from user

### Phase 5: Integrate WebAI-to-API (1 week) - OPTIONAL

**Step-by-step**:
1. Deploy WebAI-to-API separately
2. Create auth entry in config
3. Register webai models
4. Test cookie management and fallback providers

---

## Part 7: Implementation Details for Merged Providers

### 7.1 Enhanced Auth System

**New auth storage structure** (for reasoning models):
```json
{
  "id": "auth-claude-oauth-user@example.com",
  "provider": "claude-oauth",
  "access_token": "sk-ant-...",
  "refresh_token": "sk-ant-...",
  "expires_at": "2025-11-22T10:00:00Z",
  "metadata": {
    "email": "user@example.com",
    "name": "User Name",
    "organization": "org-123"
  },
  "attributes": {
    "thinking_budget_default": "8000",
    "thinking_budget_max": "32000"
  },
  "model_capabilities": {
    "claude-sonnet-4-5": {
      "supports_reasoning": true,
      "default_thinking_tokens": "8000"
    }
  }
}
```

### 7.2 Reasoning Model Support

**In ClaudeExecutor** (`/internal/runtime/executor/claude_executor.go`):
```go
func (e *ClaudeExecutor) Execute(ctx context.Context, auth *coreauth.Auth, req *api.ChatRequest, opts ...ExecutorOption) (*api.ChatResponse, error) {
    // Check if model supports reasoning
    if isReasoningModel(req.Model) {
        // Build thinking configuration
        thinking := map[string]interface{}{
            "type": "enabled",
            "budget_tokens": getThinkingBudget(auth, req.Model),
        }

        // Add to Anthropic API request
        body["thinking"] = thinking

        // Parse response including thinking blocks
        resp, err := e.parseReasoningResponse(body)
        // Translate thinking blocks back to OpenAI format
    }

    // Regular execution
    return e.executeRequest(ctx, auth, body)
}

func (e *ClaudeExecutor) parseReasoningResponse(resp *anthropic.MessageResponse) (*api.ChatResponse, error) {
    // Map Anthropic thinking blocks to OpenAI format
    // thinking → assistant with <thinking> tags
    // OR: store in response metadata
}
```

### 7.3 OAuth Callback Infrastructure

**Extend existing callback system** (in `/internal/api/server.go`):
```go
// Add new callback routes
r.POST("/anthropic-oauth/callback", handlers.AnthropicOAuthCallback)
r.POST("/openai-oauth/callback", handlers.OpenAIOAuthCallback)

// These reuse existing infrastructure:
// - Local HTTP server on different ports
// - State validation (CSRF protection)
// - PKCE code exchange
// - Token persistence via TokenStore
```

### 7.4 Model Registry Updates

**Extend model registry** (`/internal/registry/model_definitions.go`):
```go
var LLMuxModels = []*cliproxy.ModelInfo{
    // Claude models
    {
        ID:          "claude-sonnet-4-5-20250929",
        Alias:       []string{"claude-sonnet-4-5", "claude-sonnet"},
        Provider:    "claude-oauth",
        Type:        "chat",
        ContextSize: 200000,
        MaxOutput:   65536,
        Features:    []string{"reasoning", "vision", "code_execution"},
    },
    // Reasoning variants
    {
        ID:          "claude-sonnet-4-5-reasoning-low",
        Alias:       []string{"claude-sonnet-reasoning"},
        Provider:    "claude-oauth",
        Type:        "chat",
        ContextSize: 200000,
        MaxOutput:   65536,
        Features:    []string{"reasoning-only"},
        Config: map[string]interface{}{
            "reasoning_effort": "low",
            "thinking_budget": 8000,
        },
    },
    // GPT models
    {
        ID:          "gpt-5",
        Provider:    "chatgpt-oauth",
        Type:        "chat",
        ContextSize: 400000,
        MaxOutput:   128000,
        Features:    []string{"reasoning", "code_execution"},
    },
    // ... more models
}
```

---

## Part 8: Configuration Examples

### 8.1 Merged Providers Config

```yaml
# config.yaml

port: 8317
debug: false

auth-dir: "~/.cli-proxy-api"

# Existing providers
claude-api-key:
  - api-key: "sk-ant-..."
    email: "user@example.com"

# NEW: Merged LLMux OAuth providers
claude-oauth:
  enabled: true

chatgpt-oauth:
  enabled: true

# Web AI (optional separate service)
webai-proxy:
  enabled: false
  endpoint: "http://localhost:8320"
```

### 8.2 Full Multi-Provider Setup

```yaml
# config.yaml - Complete setup with all providers

server:
  port: 8317
  debug: false

auth:
  dir: "~/.cli-proxy-api"
  store: "file"  # or "postgres", "git", "object"

# Existing providers
claude-api-key:
  - api-key: "sk-ant-..."
    email: "claude-user@example.com"

gemini-api-key:
  - api-key: "AIza..."

codex-api-key:
  - api-key: "sk-..."

# Merged OAuth providers (NEW)
claude-oauth:
  enabled: true
  # Credentials stored automatically in auth-dir after login

chatgpt-oauth:
  enabled: true
  # Credentials stored automatically in auth-dir after login

# Parallel services (external)
ai-studio-proxy:
  enabled: true
  endpoint: "http://localhost:8318"  # AIstudioProxyAPI
  # Auth: User manages in AIstudioProxyAPI separately
  # Models: ai-studio-*

ctonew-proxy:
  enabled: true
  endpoint: "http://localhost:8319"  # ctonew-proxy service
  # Auth: User provides Clerk JWT cookie
  # Models: ctonew-*

webai-proxy:
  enabled: true
  endpoint: "http://localhost:8320"  # WebAI-to-API service
  # Auth: Cookies managed by WebAI service
  # Models: webai-*

openai-compatibility:
  - name: "openrouter"
    base-url: "https://openrouter.io/api/v1"
    api-keys: ["sk-or-..."]
    models:
      - name: "anthropic/claude-3.5-sonnet"
        alias: "claude-3.5-sonnet"
```

### 8.3 CLI Usage Examples

```bash
# Login to LLMux OAuth providers
./cli-proxy-api -claude-oauth-login
./cli-proxy-api -chatgpt-oauth-login

# Start router with all providers
./cli-proxy-api -claude -codex -gemini -chatgpt-oauth

# Query models from different providers
curl -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"model": "claude-sonnet-4-5", "messages": [...]}' \
  http://localhost:8317/v1/chat/completions

curl -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"model": "gpt-5", "messages": [...]}' \
  http://localhost:8317/v1/chat/completions

curl -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"model": "ai-studio-gemini-2-flash", "messages": [...]}' \
  http://localhost:8317/v1/chat/completions  # Proxied to AIstudioProxyAPI

curl -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"model": "webai-gemini-2-flash", "messages": [...]}' \
  http://localhost:8317/v1/chat/completions  # Proxied to WebAI-to-API
```

---

## Part 9: Handling Differences & Special Cases

### 9.1 Token Format & Storage Differences

| Provider | Token Type | Storage | Refresh |
|----------|-----------|---------|---------|
| Claude OAuth | Bearer token (sk-ant-...) | FileStore | 1-year, use refresh_token |
| ChatGPT OAuth | Bearer token | FileStore | 30-90 days (auto-refresh) |
| WebAI | Cookie pair | config.conf or auto-extract | Manual or session-based |
| AIStudio | Cookies + localStorage | auth_profiles/ | Browser session refresh |
| Ctonew | Clerk JWT | None (stateless) | Per-request generation |

**Solution**:
```go
// In TokenStore interface, add provider-specific methods:
type TokenStore interface {
    // Existing
    Save(ctx, auth) error
    Load(ctx, provider, email) (*Auth, error)

    // NEW: Provider-specific handling
    ValidateToken(ctx, auth) error  // Check token validity
    RefreshToken(ctx, auth) (*Auth, error)  // Provider-specific refresh
    GetCookies(ctx, auth) (map[string]string, error)  // For cookie-based
}
```

### 9.2 Model Availability Differences

**Problem**:
- Claude OAuth provides same models as API-key Claude (different auth only)
- ChatGPT OAuth provides models NOT in OpenAI API (Plus/Pro tier)
- WebAI provides Gemini web models (different from Gemini API)
- AIStudio provides Gemini models via web UI
- Ctonew provides specific models via enginelabs

**Solution**: Model registry distinguishes by provider+auth type
```go
type ModelInfo struct {
    ID       string  // claude-sonnet-4-5
    Provider string  // "claude-oauth" vs "claude" vs "claude-api-key"
    AuthType string  // "oauth", "api-key", "cookie", "jwt"
    // Model is available if auth type matches
}

// Router logic:
// 1. User requests model "claude-sonnet-4-5"
// 2. Check which auth types support it (both "claude" and "claude-oauth")
// 3. Pick available auth entry (round-robin)
// 4. Use matching executor
```

### 9.3 Request Format Differences

**Problem**: Reasoning models need special request format

**Solution**: Extend request normalization
```go
func NormalizeRequest(req *RequestPayload, executor ProviderExecutor) *RequestPayload {
    provider := executor.Identifier()

    switch provider {
    case "claude-oauth":
        // Check if model is reasoning variant
        if isReasoningModel(req.Model) {
            // Extract thinking budget from model name or request
            // Add thinking budget to request metadata
            req.Metadata["thinking_budget"] = extractThinkingBudget(req)
        }
    case "chatgpt-oauth":
        // Handle reasoning effort parameter
        if req.ExtraParams != nil {
            reasoning_effort := req.ExtraParams["reasoning_effort"]
            // Pass to ChatGPT executor
        }
    }

    return req
}
```

### 9.4 Error Handling for Different Providers

**Problem**: Different providers return different error formats

**Example**:
```
Anthropic: {"error": {"type": "invalid_request_error", "message": "..."}}
OpenAI: {"error": {"code": "...", "message": "...", "param": "..."}}
WebAI (gpt4free): {"error": "error message"}
Ctonew: {"error": "message"}
```

**Solution**: Standardize in executors
```go
func (e *ClaudeOAuthExecutor) handleError(err error) error {
    // Parse Anthropic error format
    // Translate to standard CLIProxyAPI error format
    return &APIError{
        Code:    "invalid_request",
        Message: err.Error(),
        Status:  400,
    }
}
```

---

## Part 10: Security Considerations

### 10.1 Cookie Security (WebAI, AIStudio)

**Risks**:
- Cookies are session-specific and browser-specific
- Can't be rotated like tokens
- Vulnerable to cookie theft if stored insecurely
- May violate ToS if shared across multiple machines

**Mitigations**:
1. Store in encrypted file storage
2. Require HTTPS-only communication
3. Document that sharing credentials violates ToS
4. Implement cookie expiry detection and re-prompt for login
5. Use separate service (not embedded) to isolate cookie storage

### 10.2 JWT/Token Security (Ctonew, OAuth)

**Risks**:
- Tokens visible in API responses (for debugging)
- Token leakage in logs or error messages
- Tokens stored in plaintext in FileStore

**Mitigations**:
1. Encrypt tokens in FileStore (add encryption layer)
2. Implement audit logging for token use
3. Mask tokens in logs and error messages
4. Implement token rotation strategies
5. Document token lifetime and refresh behavior

### 10.3 Multi-Provider Credential Management

**Risks**:
- User confusion about credential scope
- Mixing credentials across providers
- Credentials accessible to wrong consumer

**Mitigations**:
1. Clear labeling in auth entries (provider, auth type, scope)
2. Separate auth directories per provider
3. Implement RBAC for credential access
4. Audit trail for auth usage
5. Clear documentation of credential security

---

## Part 11: Recommended Final Architecture Summary

### ✅ MERGED INTO CLIProxyAPI:

**1. LLMux OAuth Providers**
```
New Files:
- /sdk/auth/claude_oauth.go
- /sdk/auth/chatgpt_oauth.go
- /internal/auth/claude_oauth/*
- /internal/auth/chatgpt_oauth/*
- /internal/runtime/executor/chatgpt_executor.go

Changes:
- Update ClaudeExecutor for reasoning budgets
- Extend model registry with new models
- Add CLI login commands
- Update TokenStore if needed
```

**2. WebAI-to-API (OPTIONAL - as separate service)**
```
If merged:
- /sdk/auth/webai.go
- /internal/runtime/executor/webai_executor.go

Recommendation: Keep separate for cookie management
- Run WebAI-to-API on port 8320
- Use HTTPProxyExecutor for routing
- Simpler auth model (just endpoint config)
```

### ❌ SEPARATE SERVICES:

**1. AIstudioProxyAPI**
```
Deployment:
- Run on port 8318 (separate process)
- Manages browser automation (Camoufox)
- Handles stateful AI Studio sessions

Integration:
- Add HTTPProxyExecutor support
- Add config.ai-studio-proxy
- Route ai-studio-* models to it
```

**2. ctonew-proxy**
```
Deployment:
- Run on port 8319 (separate process)
- Already Deno-based single file
- Handles Clerk JWT extraction

Integration:
- Add HTTPProxyExecutor support
- Add config.ctonew-proxy
- Route ctonew-* models to it
```

### Request Routing Flow

```
User Request:
  ↓
CLIProxyAPI HTTP Handler
  ├─ Parse model name
  ├─ Find matching provider
  └─ Determine auth type
       ├─ If "claude-oauth" or "chatgpt-oauth" → Local executor
       ├─ If "ai-studio-*" → HTTPProxyExecutor → AIstudioProxyAPI service
       ├─ If "ctonew-*" → HTTPProxyExecutor → ctonew-proxy service
       ├─ If "webai-*" → HTTPProxyExecutor → WebAI-to-API service
       └─ If "claude-api-key" or others → Existing executors
            ↓
        Execute request
            ↓
        Return response
```

---

## Part 12: Timeline & Effort Estimation

| Phase | Task | Effort | Duration |
|-------|------|--------|----------|
| 1 | Merge LLMux OAuth (Claude + ChatGPT) | 8 days | 1-2 weeks |
| 2 | Implement HTTP Proxy Executor | 3 days | 3-4 days |
| 3 | Integrate AIstudioProxyAPI | 5 days | 1-2 weeks |
| 4 | Integrate ctonew-proxy | 3 days | 3-4 days |
| 5 | Integrate WebAI-to-API (optional) | 4 days | 4-5 days |
| 6 | Testing & Documentation | 5 days | 1 week |
| **Total** | | **25+ days** | **4-6 weeks** |

**Critical Path**: Phase 1 (LLMux OAuth) - most valuable feature
**Quick Wins**: Phase 4 (ctonew-proxy) - already mostly built
**Optional**: Phase 5 (WebAI) - lower priority

---

## Conclusion & Recommendation

### Best Approach: **Hybrid Integration**

**DO MERGE**:
- ✅ **LLMux OAuth** (Claude Pro + ChatGPT Plus official access)
  - Extends existing OAuth pattern
  - Adds reasoning model support
  - Improves token lifetime
  - Effort: 1-2 weeks, High value

**DO NOT MERGE** (keep separate):
- ❌ **AIstudioProxyAPI** (Playwright browser automation)
  - Requires stateful browser instance
  - Complex authentication flow
  - Better as isolated service
  - Effort: 1-2 weeks integration

- ❌ **ctonew-proxy** (Clerk JWT broker)
  - Stateless operation doesn't fit token-based model
  - Already complete Deno service
  - Better as isolated service
  - Effort: few days integration

- ⚠️ **WebAI-to-API** (gpt4free + Gemini cookies)
  - Could merge but recommend separate
  - Cookie management complex
  - Better as optional fallback service
  - Effort: days to integrate, weeks to maintain

### Key Advantages of This Approach

1. **Clean separation of concerns**
   - Token-based providers stay in router
   - Stateful services run independently

2. **Minimal disruption**
   - Existing providers unchanged
   - New features added, not modified

3. **Scalability**
   - Each service can scale independently
   - Can restart services without affecting others

4. **Maintainability**
   - Reduced complexity in main router
   - Services use native tech stacks
   - Clear interfaces between components

5. **User flexibility**
   - Can use any subset of providers
   - Easy to add/remove providers
   - No forced dependencies

### Implementation Priority

1. **Week 1-2**: Merge LLMux OAuth (highest ROI)
2. **Week 2-3**: HTTP Proxy Executor infrastructure
3. **Week 3-4**: AIstudioProxyAPI + ctonew-proxy integration
4. **Week 4-5**: WebAI-to-API (if needed)
5. **Week 5-6**: Testing, documentation, deployment

This roadmap provides a coherent, maintainable router with professional support for 10+ providers without complexity explosion.

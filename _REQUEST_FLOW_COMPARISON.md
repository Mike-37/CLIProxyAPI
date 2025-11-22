# Request Flow Comparison: Web Providers vs CLI Agents

## Quick Distinction

**CLI Agent Providers** (existing):
- Direct access to official APIs (Claude API, OpenAI API, Google AI API)
- Use official OAuth flows (when available)
- Can issue their own API keys or developer credentials
- Purpose: Developer/API access

**Web Providers** (new):
- Access via web UI automation or unofficial APIs
- Use browser cookies, shared accounts, or unofficial OAuth
- Emulate user accounts (Claude Pro, ChatGPT Plus subscriptions)
- Purpose: User/subscriber access (not official API)

---

## Detailed Request Flows

### FLOW 1: CLI Agent Provider (Existing) - Example: Claude Code

```
┌─────────────────────────────────────────────────────────────────┐
│                         USER/CLIENT                              │
│                    (CLI or HTTP client)                          │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼ (1) HTTP POST /v1/chat/completions
                    ┌────────────────────────┐
                    │   CLIProxyAPI Router   │
                    │   (Port 8317)          │
                    └────────────┬───────────┘
                                 │
                    (2) API Key validation (Access Layer)
                    ├─ Header: Authorization: Bearer {api-key}
                    └─ Check against configured keys
                                 │
                    (3) Parse model: "claude-sonnet-4-5"
                    ├─ Find matching provider: "claude"
                    └─ Get auth entry for claude
                                 │
                    ┌────────────▼───────────────────────┐
                    │  Auth Manager                      │
                    │  (Manages credentials)             │
                    │                                    │
                    │  Token Storage:                    │
                    │  ~/.cli-proxy-api/                 │
                    │    ├─ claude-user@example.json     │
                    │    └─ Contains:                     │
                    │       - access_token               │
                    │       - refresh_token              │
                    │       - expires_at                 │
                    └────────────┬───────────────────────┘
                                 │
                    (4) Check if token valid
                    ├─ If expired: Auto-refresh
                    │  └─ POST to Anthropic refresh endpoint
                    └─ Return valid token
                                 │
                    ┌────────────▼───────────────────────┐
                    │  ClaudeExecutor                    │
                    │  (Provider implementation)         │
                    │                                    │
                    │  1. Format request to Anthropic   │
                    │  2. Add auth header (Bearer token) │
                    │  3. POST /v1/messages              │
                    │     to api.anthropic.com           │
                    │                                    │
                    │  4. Parse response                 │
                    │  5. Translate to OpenAI format     │
                    │  6. Track usage                    │
                    └────────────┬───────────────────────┘
                                 │
                    (5) ┌────────▼──────────────────┐
                        │  Anthropic Official API   │
                        │  (Claude Sonnet 4.5)      │
                        │  api.anthropic.com        │
                        │                           │
                        │  Requires:                │
                        │  - Valid API key/token    │
                        │  - Developer account      │
                        │  - API quota              │
                        └────────────┬──────────────┘
                                     │
                        (6) ┌────────▼──────────┐
                            │  Model Response   │
                            │  (Tokens used)    │
                            └────────┬──────────┘
                                     │
                    (7) ┌───────────▼────────────┐
                        │  Response Translator   │
                        │  (Anthropic → OpenAI) │
                        │                        │
                        │  Transform:            │
                        │  - Message format      │
                        │  - Token counting      │
                        │  - Streaming format    │
                        │  - Error codes         │
                        └────────────┬───────────┘
                                     │
                        (8) ┌────────▼──────────────┐
                            │  HTTP Response       │
                            │  (JSON with answer)  │
                            └────────┬──────────────┘
                                     │
                        ┌────────────▼─────────────┐
                        │       CLIENT             │
                        │  (Receives response)     │
                        └──────────────────────────┘

KEY CHARACTERISTICS:
✓ Direct API call to official provider
✓ Token in Router → official API (1 hop)
✓ Single executor handles everything
✓ Clear credential ownership (developer account)
✓ Predictable behavior
✓ Official rate limits and quotas
```

---

### FLOW 2: Web Provider - Example: LLMux (Claude Pro OAuth)

```
┌──────────────────────────────────────────────────────┐
│              USER/CLIENT                             │
│         (CLI or HTTP client)                         │
└────────────────────────┬─────────────────────────────┘
                         │
                         ▼ (1) HTTP POST /v1/chat/completions
                ┌────────────────────────┐
                │   CLIProxyAPI Router   │
                │   (Port 8317)          │
                └────────────┬───────────┘
                             │
                (2) API Key validation (same as CLI)
                             │
                (3) Parse model: "claude-sonnet-4-5-oauth"
                ├─ Find matching provider: "claude-oauth"
                └─ Get auth entry for claude-oauth
                             │
                ┌────────────▼────────────────────┐
                │  Auth Manager                   │
                │  (Manages user credentials)     │
                │                                 │
                │  Token Storage:                 │
                │  ~/.cli-proxy-api/              │
                │    └─ claude-oauth-{email}.json │
                │       - access_token            │
                │       - refresh_token           │
                │       - user_email              │
                │       - expires_at              │
                │       - organization            │
                │       - 1-YEAR lifetime!        │
                └────────────┬────────────────────┘
                             │
                (4) Check if token valid
                ├─ If expired: Auto-refresh
                │  └─ POST to Anthropic OAuth endpoint
                │     with refresh_token
                └─ Return valid token
                             │
                ┌────────────▼─────────────────────┐
                │  ClaudeOAuthExecutor             │
                │  (Handles OAuth-based access)   │
                │                                  │
                │  1. Check if reasoning model    │
                │  2. Extract thinking budget     │
                │  3. Format request              │
                │  4. Add OAuth Bearer token      │
                │  5. POST /v1/messages           │
                │     to api.anthropic.com        │
                │                                 │
                │  4. Parse response              │
                │  5. Handle reasoning blocks     │
                │  6. Translate to OpenAI format  │
                │  7. Track usage                 │
                └────────────┬────────────────────┘
                             │
                (5) ┌────────▼──────────────────────┐
                    │  Anthropic API                │
                    │  (via OAuth, not API key)     │
                    │  api.anthropic.com            │
                    │                               │
                    │  Difference from CLI:         │
                    │  - Uses OAuth token           │
                    │  - May access paid features   │
                    │    (Claude Pro)               │
                    │  - Different rate limits      │
                    │  - User-specific context      │
                    └────────────┬──────────────────┘
                                 │
                    (6) ┌────────▼──────────────┐
                        │  Model Response       │
                        │  (from paid tier)     │
                        └────────┬──────────────┘
                                 │
                    ┌────────────▼──────────────┐
                    │  Response Translator      │
                    │  (Anthropic → OpenAI)    │
                    │  + Reasoning handling     │
                    └────────────┬──────────────┘
                                 │
                        ┌────────▼──────────────┐
                        │  HTTP Response        │
                        │  (JSON with answer)   │
                        └────────┬──────────────┘
                                 │
                        ┌────────▼────────────┐
                        │       CLIENT         │
                        │  (Receives response) │
                        └────────────────────┘

KEY CHARACTERISTICS:
✓ Direct API call (same as CLI) but uses OAuth token
✓ Token in Router → official API (1 hop, same as CLI)
✓ Single executor (reuses Claude one + reasoning support)
✓ Credential is USER account (Claude Pro subscription)
✓ Accesses paid features
✓ Same rate limits as official API
✓ Reasoning model support
```

---

### FLOW 3: Web Provider - Example: AIstudioProxyAPI (Browser Automation)

```
┌──────────────────────────────────────────────────────┐
│              USER/CLIENT                             │
│         (CLI or HTTP client)                         │
└────────────────────────┬─────────────────────────────┘
                         │
                         ▼ (1) HTTP POST /v1/chat/completions
                ┌────────────────────────┐
                │   CLIProxyAPI Router   │
                │   (Port 8317)          │
                └────────────┬───────────┘
                             │
                (2) API Key validation
                             │
                (3) Parse model: "ai-studio-gemini-2-flash"
                ├─ Find matching provider: "aistudio"
                └─ Get auth entry for aistudio
                             │
                ┌────────────▼────────────────────┐
                │  Auth Manager                   │
                │  (Minimal - just endpoint URL)  │
                │                                 │
                │  Token Storage:                 │
                │  ~/.cli-proxy-api/              │
                │    └─ aistudio-user.json        │
                │       - endpoint: http://...:8318
                │       - configured: true        │
                │       (auth is REMOTE)          │
                └────────────┬────────────────────┘
                             │
                ┌────────────▼──────────────────────┐
                │  HTTPProxyExecutor                │
                │  (Pass-through to service)       │
                │                                  │
                │  1. Forward request to:          │
                │     http://localhost:8318        │
                │  2. Pass through all params      │
                │  3. Stream response back         │
                │  4. Return as-is                 │
                └────────────┬──────────────────────┘
                             │
                (4) ┌────────▼──────────────────────┐
                    │  AIstudioProxyAPI Service     │
                    │  (Port 8318 - SEPARATE)       │
                    │                               │
                    │  Manages own auth:            │
                    │  - Camoufox browser instance  │
                    │  - auth_profiles/ directory   │
                    │  - Cookies + localStorage     │
                    │  - Session state              │
                    │                               │
                    │  Request processing:          │
                    │  1. Receive request           │
                    │  2. Interact with browser     │
                    │  3. Navigate AI Studio        │
                    │  4. Fill form inputs          │
                    │  5. Switch models             │
                    │  6. Get response              │
                    │  7. Stream back chunks        │
                    └────────────┬──────────────────┘
                                 │
                    (5) ┌────────▼──────────────────┐
                        │  Google AI Studio         │
                        │  (Web UI)                 │
                        │  ai.studio.google.com     │
                        │                           │
                        │  Requires:                │
                        │  - Session cookies        │
                        │  - Browser automation     │
                        │  - User account           │
                        │  - Anti-fingerprinting    │
                        └────────────┬──────────────┘
                                     │
                        ┌────────────▼──────────────┐
                        │  Model Response           │
                        │  (from browser page)      │
                        └────────────┬──────────────┘
                                     │
                    ┌────────────────▼──────────────┐
                    │  Stream back to HTTPProxy     │
                    │  (chunks in real-time)        │
                    └────────────────┬──────────────┘
                                     │
                    ┌────────────────▼──────────────┐
                    │  HTTPProxyExecutor            │
                    │  (Transparent pass-through)   │
                    └────────────────┬──────────────┘
                                     │
                        ┌────────────▼────────────┐
                        │  HTTP Response (streamed)│
                        │  (SSE chunks)           │
                        └────────────┬────────────┘
                                     │
                        ┌────────────▼──────────┐
                        │       CLIENT           │
                        │  (Receives response)   │
                        └────────────────────────┘

KEY CHARACTERISTICS:
❌ NOT a direct API call
✓ 2 HOP: Client → Router → AIStudio Service
✓ Router is TRANSPARENT (just proxy)
✓ Actual work happens in separate service
✓ Service manages own browser state
✓ Router knows nothing about browser/Camoufox
✓ Different auth model (session/cookies, not tokens)
✓ Can restart service independently
✓ Latency: Client → Router (1ms) + Router → Service (local, 1ms) + Service → Browser → Web (100-500ms)
✓ Service handles:
  - Browser lifecycle
  - Session management
  - Model switching
  - Parameter handling
  - Response streaming
```

---

### FLOW 4: Web Provider - Example: ctonew-proxy (Stateless JWT)

```
┌──────────────────────────────────────────────────────┐
│              USER/CLIENT                             │
│         (CLI or HTTP client)                         │
└────────────────────────┬─────────────────────────────┘
                         │
                         ▼ (1) HTTP POST /v1/chat/completions
                ┌────────────────────────┐
                │   CLIProxyAPI Router   │
                │   (Port 8317)          │
                └────────────┬───────────┘
                             │
                (2) API Key validation
                             │
                (3) Parse model: "ctonew-claude-sonnet"
                ├─ Find matching provider: "ctonew"
                └─ Get auth entry for ctonew
                             │
                ┌────────────▼────────────────────┐
                │  Auth Manager                   │
                │  (Just stores Clerk JWT)        │
                │                                 │
                │  Token Storage:                 │
                │  ~/.cli-proxy-api/              │
                │    └─ ctonew-user.json          │
                │       - clerk_jwt_cookie:       │
                │         "__client=eyJ..."       │
                │       - endpoint: http://...:8319
                │       (NO REFRESH - PER-REQUEST)
                └────────────┬────────────────────┘
                             │
                ┌────────────▼──────────────────────┐
                │  HTTPProxyExecutor                │
                │  (With JWT injection)             │
                │                                  │
                │  1. Get stored Clerk JWT cookie  │
                │  2. Add to request headers:      │
                │     Cookie: __client=eyJ...      │
                │  3. Forward to ctonew service:   │
                │     http://localhost:8319        │
                │  4. Stream response back         │
                │  5. Return as-is                 │
                └────────────┬──────────────────────┘
                             │
                (4) ┌────────▼──────────────────────┐
                    │  ctonew-proxy Service         │
                    │  (Port 8319 - SEPARATE)       │
                    │                               │
                    │  Single TypeScript file       │
                    │  (Deno-based, Oak framework)  │
                    │                               │
                    │  Per-request processing:      │
                    │  1. Receive request           │
                    │  2. Extract __client JWT      │
                    │  3. GET /v1/client (Clerk)    │
                    │  4. Parse rotating_token      │
                    │  5. POST to Clerk for token   │
                    │  6. Get new JWT token         │
                    │  7. Use for API call          │
                    │  8. Stream response           │
                    │                               │
                    │  (NO persistent token storage)│
                    │  (Fresh token per request)    │
                    └────────────┬──────────────────┘
                                 │
                    (5) ┌────────▼──────────────────┐
                        │  Clerk Service             │
                        │  (Token exchange)          │
                        │                           │
                        │  HTTP:                    │
                        │  POST /v1/tokens/create   │
                        │  → returns new JWT        │
                        └────────────┬──────────────┘
                                     │
                        ┌────────────▼──────────────┐
                        │  EngineLabs API            │
                        │  (ctonew backend)          │
                        │  api.enginelabs.ai         │
                        │                           │
                        │  Uses JWT token:          │
                        │  Bearer {jwt_from_clerk}  │
                        │                           │
                        │  Models:                  │
                        │  - Claude Sonnet 4.5      │
                        │  - GPT-5                  │
                        └────────────┬──────────────┘
                                     │
                        ┌────────────▼──────────────┐
                        │  Model Response           │
                        │  (streaming chunks)       │
                        └────────────┬──────────────┘
                                     │
                    ┌────────────────▼──────────────┐
                    │  ctonew-proxy forwards to     │
                    │  HTTPProxyExecutor            │
                    │  (SSE stream)                 │
                    └────────────────┬──────────────┘
                                     │
                        ┌────────────▼────────────┐
                        │  HTTP Response (streamed)│
                        │  (SSE chunks)           │
                        └────────────┬────────────┘
                                     │
                        ┌────────────▼──────────┐
                        │       CLIENT           │
                        │  (Receives response)   │
                        └────────────────────────┘

KEY CHARACTERISTICS:
❌ NOT a direct API call (not persistent token model)
✓ 2 HOP: Client → Router → ctonew Service
✓ ctonew service is STATELESS
✓ Per-request JWT generation (not stored)
✓ Router just injects Cookie header
✓ ctonew handles:
  - JWT extraction from cookie
  - Clerk token exchange
  - API call to EngineLabs
  - Response streaming
✓ Zero token management in Router
✓ Clerk is single point of auth
✓ Can deploy on Deno Deploy (serverless!)
```

---

### FLOW 5: Web Provider - Example: WebAI-to-API (Fallback Approach)

```
Option A: Run as separate service (RECOMMENDED)

┌──────────────────────────────────────────────────────┐
│              USER/CLIENT                             │
└────────────────────────┬─────────────────────────────┘
                         │
                         ▼ (1) HTTP POST /v1/chat/completions
                ┌────────────────────────┐
                │   CLIProxyAPI Router   │
                └────────────┬───────────┘
                             │
                (2) Model: "webai-gemini-2-flash"
                → provider: "webai"
                             │
                ┌────────────▼────────────────────┐
                │  Auth: Just endpoint config      │
                │  ~/.cli-proxy-api/               │
                │    └─ webai-config.json          │
                │       - endpoint: http://...:8320
                │       - has_cookies: true        │
                │       (no token management)      │
                └────────────┬────────────────────┘
                             │
                ┌────────────▼──────────────────────┐
                │  HTTPProxyExecutor                │
                │                                  │
                │  Forward to:                     │
                │  http://localhost:8320           │
                │  /v1/chat/completions            │
                └────────────┬──────────────────────┘
                             │
                (3) ┌────────▼──────────────────────┐
                    │  WebAI-to-API Service         │
                    │  (Port 8320 - SEPARATE)       │
                    │  (FastAPI + Python)           │
                    │                               │
                    │  Handles:                     │
                    │  - config.conf (cookie cfg)   │
                    │  - browser-cookie3 (auto)     │
                    │  - gemini-webapi client       │
                    │  - gpt4free fallback          │
                    │  - session management         │
                    │                               │
                    │  Request processing:          │
                    │  1. Parse model: gemini-2...  │
                    │  2. Load cookies from config  │
                    │  3. Call Gemini API           │
                    │  4. Fall back to gpt4free?    │
                    │  5. Stream response           │
                    └────────────┬──────────────────┘
                                 │
                    (4) ┌────────▼──────────────────┐
                        │  Gemini Web API            │
                        │  (via cookies)             │
                        │  OR                        │
                        │  gpt4free providers        │
                        │  (50+ model fallback)      │
                        └────────────┬──────────────┘
                                     │
                        ┌────────────▼──────────────┐
                        │  Model Response            │
                        │  (streaming)               │
                        └────────────┬──────────────┘
                                     │
                        ┌────────────▼────────────┐
                        │  HTTP Response (streamed)│
                        └────────────┬────────────┘
                                     │
                        ┌────────────▼──────────┐
                        │       CLIENT           │
                        └────────────────────────┘

KEY CHARACTERISTICS:
✓ Simplest separate service
✓ Router just passes through
✓ Service manages config.conf + cookies
✓ Supports dual fallback (Gemini + gpt4free)
✓ Can auto-extract cookies from browser
✓ No token refresh (manual or per-session)
```

---

## Summary Comparison Table

| Aspect | CLI Agent (Claude) | Web OAuth (LLMux) | Browser Auto (AIStudio) | JWT Broker (ctonew) | Cookie Fallback (WebAI) |
|--------|-------------------|-------------------|------------------------|---------------------|----------------------|
| **Token in Router** | ✅ Yes | ✅ Yes | ❌ No (endpoint only) | ❌ No (cookie only) | ❌ No (config only) |
| **Hops to API** | 1 | 1 | 2 | 2 | 2 |
| **Router Complexity** | Low | Low | Minimal (proxy) | Minimal (proxy) | Minimal (proxy) |
| **Auth Management** | In Router | In Router | In Service | In Service | In Service |
| **Token Storage** | FileStore | FileStore | N/A | N/A | config.conf |
| **Auto-Refresh** | ✅ Yes (5s loop) | ✅ Yes (5s loop) | ❌ No | ❌ No | ❌ No |
| **Service Process** | N/A (local) | N/A (local) | Separate | Separate | Separate |
| **Latency** | Low (direct) | Low (direct) | Med (2 hops) | Med (2 hops) | Med (2 hops) |
| **Stateful** | ✅ No | ✅ No | ❌ Yes (browser) | ✅ No | ⚠️ Config-based |
| **Credential Type** | API Key/Token | User OAuth | Browser Cookies | JWT Cookie | Browser Cookies |
| **User Type** | Developer | Subscriber (Pro) | Subscriber (free) | Subscriber | Subscriber |

---

## Key Architectural Insights

### Why Separate Services for Web Providers?

1. **Different Auth Models**
   - CLI: Token-based (stateless)
   - Web: Session/Cookie-based (stateful) OR JWT-based (ephemeral)

2. **Different Resource Models**
   - CLI: Memory-light (just HTTP client)
   - Web Browser: Memory-heavy (500MB+ per browser instance)
   - Web JWT: Stateless (no resources needed)

3. **Different Credential Ownership**
   - CLI: Developer account (API key)
   - Web: User account (subscriber credentials)

4. **Different Failure Models**
   - CLI: Token expires → auto-refresh → works
   - Web Browser: Browser crashes → manual restart needed
   - Web JWT: Clerk service down → no auth possible

5. **Different Scaling Models**
   - CLI: Can scale horizontally (no state)
   - Web Browser: Must keep per-instance (state = browser)
   - Web JWT: Can scale horizontally (stateless)

### Why Merge LLMux OAuth?

Because it's fundamentally the same as existing OAuth:
- Token-based (stateless)
- Auto-refreshable (same pattern)
- Same credential ownership model (user tokens)
- Just different issuer (Anthropic OAuth vs Codex OAuth)
- No new infrastructure needed (reuse existing)

### Why Keep AIStudio + ctonew Separate?

Because they fundamentally differ:
- **AIStudio**: Requires browser instance (can't be embedded in Go)
- **ctonew**: Completely stateless (doesn't fit token storage model)
- Both better as dedicated services

---

## Request Path Decision Tree

```
Request arrives:
  │
  ├─ Merge with existing auth? (token-based, long-lived)
  │  ├─ YES: Claude OAuth, ChatGPT OAuth, future official APIs
  │  │  └─ Execute locally
  │  │
  │  └─ NO: Different auth model
  │     └─ Route to separate service?
  │        ├─ Browser-based? → AIStudioProxyAPI
  │        ├─ Cookie-based? → WebAI-to-API
  │        ├─ JWT-based? → ctonew-proxy
  │        └─ Custom OAuth? → Create new service
```


# CLIProxyAPI - Architecture Diagrams

> **Visual Reference**: Quick architecture understanding through diagrams

---

## 1. System Overview

### 1.1 High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         CLIENT LAYER                            │
│  OpenAI SDK | curl | Custom Apps | Claude Code | Continue.dev  │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             │ OpenAI-compatible API
                             ↓
┌─────────────────────────────────────────────────────────────────┐
│                    ROUTER (Go - Single Binary)                  │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ HTTP Server (:8317)                                     │   │
│  │ ├─ /v1/chat/completions    (OpenAI format)             │   │
│  │ ├─ /v1/messages            (Anthropic format)          │   │
│  │ ├─ /v1beta/models          (Model registry)            │   │
│  │ ├─ /v1/auth/*              (Auth management)           │   │
│  │ └─ /v1/ws                  (WebSocket for services)    │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ Model Router                                           │   │
│  │ - Parse model name                                     │   │
│  │ - Match regex patterns                                 │   │
│  │ - Select provider(s)                                   │   │
│  │ - Automatic failover                                   │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌───────────────────────────────────┬─────────────────────┐   │
│  │  DIRECT EXECUTORS (In-Process)    │  RELAY EXECUTORS    │   │
│  │                                   │                     │   │
│  │  ✅ LLMux Claude OAuth            │  ✅ AIstudio (WS)    │   │
│  │  ✅ LLMux ChatGPT OAuth           │  ⚠️  WebAI (HTTP)    │   │
│  │  ✅ ctonew (Clerk JWT)            │                     │   │
│  │  ✅ Claude OAuth (existing)       │                     │   │
│  │  ✅ Codex OAuth (existing)        │                     │   │
│  │  ✅ Gemini API (existing)         │                     │   │
│  └───────────────────────────────────┴─────────────────────┘   │
└────────────┬──────────────────────────┬───────────────────────┘
             │                          │
             │ Direct HTTPS             │ WebSocket / HTTP
             ↓                          ↓
┌────────────────────────┐   ┌─────────────────────────────────┐
│  UPSTREAM PROVIDERS    │   │  EXTERNAL SERVICES              │
│                        │   │                                 │
│  • api.anthropic.com   │   │  ┌─────────────────────────┐   │
│  • api.openai.com      │   │  │ AIstudio (Python)       │   │
│  • generativelanguage  │   │  │ - Browser pool          │   │
│  • api.enginelabs.ai   │   │  │ - WebSocket client      │   │
│  • etc.                │   │  │ - Session management    │   │
└────────────────────────┘   │  └─────────────────────────┘   │
                             │                                 │
                             │  ┌─────────────────────────┐   │
                             │  │ WebAI (Python) OPTIONAL │   │
                             │  │ - HTTP server           │   │
                             │  │ - Cookie management     │   │
                             │  │ - gpt4free integration  │   │
                             │  └─────────────────────────┘   │
                             └─────────────────────────────────┘
```

---

## 2. Process Architecture Comparison

### 2.1 Before (Over-Engineered)

```
┌────────────────────────────────────────────────┐
│ PROCESS 1: Router (Go)                         │
│ └─ Port: 8317                                  │
└────────────────────────────────────────────────┘
         │
         ├─── HTTP ──→ ┌──────────────────────────┐
         │             │ PROCESS 2: LLMux Service │
         │             │ └─ Port: 8401            │
         │             └──────────────────────────┘
         │
         ├─── HTTP ──→ ┌──────────────────────────┐
         │             │ PROCESS 3: ctonew (Deno) │
         │             │ └─ Port: 8405            │
         │             └──────────────────────────┘
         │
         ├─── WS ────→ ┌──────────────────────────┐
         │             │ PROCESS 4: AIstudio (Py) │
         │             │ └─ WebSocket client      │
         │             └──────────────────────────┘
         │
         └─── HTTP ──→ ┌──────────────────────────┐
                       │ PROCESS 5: WebAI (Py)    │
                       │ └─ Port: 8406            │
                       └──────────────────────────┘

Total: 5 processes
Overhead: 3 HTTP hops for LLMux/ctonew
Latency: +200-300ms per hop
Complexity: High (5 services to manage)
```

### 2.2 After (Optimized)

```
┌─────────────────────────────────────────────────────┐
│ PROCESS 1: Router (Go)                              │
│ ├─ Port: 8317                                       │
│ ├─ LLMux Claude (in-process)    ✅                  │
│ ├─ LLMux ChatGPT (in-process)   ✅                  │
│ ├─ ctonew (in-process)          ✅                  │
│ └─ Direct API calls to upstream                     │
└─────────────────────────────────────────────────────┘
         │
         ├─── WS ────→ ┌──────────────────────────┐
         │             │ PROCESS 2: AIstudio (Py) │
         │             │ └─ WebSocket client      │
         │             │ └─ Browser pool          │
         │             └──────────────────────────┘
         │
         └─── HTTP ──→ ┌──────────────────────────┐
              (opt)    │ PROCESS 3: WebAI (Py)    │
                       │ └─ Port: 8406 (OPTIONAL) │
                       └──────────────────────────┘

Total: 2-3 processes (down from 5)
Overhead: No HTTP hops for LLMux/ctonew
Latency: Direct execution (faster)
Complexity: Low (2-3 services)

IMPROVEMENTS:
✅ -40% processes
✅ -60% latency (LLMux/ctonew)
✅ -50% complexity
```

---

## 3. Request Flow Diagrams

### 3.1 Direct Executor Flow (LLMux, ctonew)

```
┌─────────┐
│ CLIENT  │
└────┬────┘
     │
     │ POST /v1/chat/completions
     │ {"model": "gpt-5", "messages": [...]}
     ↓
┌──────────────────────────────────────┐
│ ROUTER - HTTP Server                 │
│ ├─ Parse JSON                        │  1. Validate request
│ ├─ Validate                          │  2. Extract model name
│ └─ Extract model: "gpt-5"            │  3. Auth check
└────┬─────────────────────────────────┘
     │
     ↓
┌──────────────────────────────────────┐
│ ROUTER - Model Router                │
│ ├─ Match: "^gpt-5"                   │  4. Pattern matching
│ ├─ Providers: [                      │  5. Priority selection
│ │    "llmux-chatgpt",  ← Priority 1  │
│ │    "ctonew"          ← Priority 2  │
│ │  ]                                 │
│ └─ Select: "llmux-chatgpt"           │
└────┬─────────────────────────────────┘
     │
     ↓
┌──────────────────────────────────────┐
│ ROUTER - LLMuxChatGPTExecutor        │
│ (In-Process Go Code)                 │
│                                      │
│ 1. Load tokens:                      │  6. Auth management
│    ~/.cli-proxy-api/                 │     (automatic)
│    llmux-chatgpt-user@ex.json        │
│                                      │
│ 2. Check expiry:                     │  7. Token refresh
│    if expired → refresh_token flow   │     (if needed)
│                                      │
│ 3. Make HTTPS request:               │  8. Direct API call
│    POST https://api.openai.com       │     (no proxy)
│    Authorization: Bearer <token>     │
│                                      │
│ 4. Stream response                   │  9. Stream back
└────┬─────────────────────────────────┘
     │
     │ Direct HTTPS
     ↓
┌──────────────────────────────────────┐
│ UPSTREAM API                         │
│ api.openai.com                       │  10. Process request
└────┬─────────────────────────────────┘
     │
     │ SSE stream
     ↓
┌──────────────────────────────────────┐
│ ROUTER - Response Formatter          │
│ ├─ Convert to OpenAI SSE format      │  11. Format response
│ └─ Forward to client                 │  12. Return to client
└────┬─────────────────────────────────┘
     │
     ↓
┌─────────┐
│ CLIENT  │
│ data: {"choices":[...]}               │
└─────────┘

⏱️  LATENCY: ~100-200ms
🔧  PROCESSES: 1 (router only)
✅  OPTIMIZATION: No HTTP proxy overhead
```

---

### 3.2 WebSocket Relay Flow (AIstudio)

```
┌─────────┐
│ CLIENT  │
└────┬────┘
     │
     │ POST /v1/chat/completions
     │ {"model": "gemini-2-flash-aistudio", ...}
     ↓
┌──────────────────────────────────────┐
│ ROUTER - HTTP Server                 │
│ └─ Extract model                     │  1. Parse request
└────┬─────────────────────────────────┘
     │
     ↓
┌──────────────────────────────────────┐
│ ROUTER - Model Router                │
│ ├─ Match: "gemini-.*-aistudio$"      │  2. Pattern match
│ └─ Select: "aistudio"                │  3. Select executor
└────┬─────────────────────────────────┘
     │
     ↓
┌──────────────────────────────────────┐
│ ROUTER - AIStudioExecutor            │
│ └─ Send WebSocket message:           │  4. Create WS message
│    {                                 │
│      "type": "http_request",         │
│      "request_id": "req-123",        │
│      "url": "ai.studio.google.com",  │
│      "method": "POST",               │
│      "body": {...}                   │
│    }                                 │
└────┬─────────────────────────────────┘
     │
     │ WebSocket message
     │ ws://localhost:8317/v1/ws
     ↓
┌──────────────────────────────────────┐
│ AISTUDIO SERVICE (Python)            │
│                                      │
│ 1. Receive WS message                │  5. Get message
│                                      │
│ 2. Get browser from pool:            │  6. Browser management
│    ├─ Check pool                     │     - Reuse if available
│    ├─ Restore session                │     - Create if needed
│    └─ Validate cookies               │
│                                      │
│ 3. Navigate to URL:                  │  7. Browser interaction
│    ├─ Open ai.studio.google.com     │
│    ├─ Fill textarea                  │
│    └─ Click send button              │
│                                      │
│ 4. Observe DOM for response:         │  8. Stream detection
│    ├─ MutationObserver              │
│    └─ Extract text chunks            │
│                                      │
│ 5. Send WS messages back:            │  9. Stream back to router
│    {"type": "stream_chunk",          │
│     "request_id": "req-123",         │
│     "data": "Hello"}                 │
└────┬─────────────────────────────────┘
     │
     │ Browser automation
     ↓
┌──────────────────────────────────────┐
│ UPSTREAM WEB UI                      │
│ ai.studio.google.com                 │  10. Web interface
└────┬─────────────────────────────────┘
     │
     │ WebSocket chunks
     ↓
┌──────────────────────────────────────┐
│ ROUTER - AIStudioExecutor            │
│ ├─ Receive WS stream chunks          │  11. Receive chunks
│ └─ Convert to OpenAI SSE             │  12. Format response
└────┬─────────────────────────────────┘
     │
     │ SSE stream
     ↓
┌─────────┐
│ CLIENT  │
│ data: {"choices":[...]}               │
└─────────┘

⏱️  LATENCY: ~1-2s (browser overhead)
🔧  PROCESSES: 2 (router + aistudio)
⚠️  COMPLEXITY: Browser needed for automation
```

---

### 3.3 HTTP Proxy Flow (WebAI - Optional)

```
┌─────────┐
│ CLIENT  │
└────┬────┘
     │
     │ POST /v1/chat/completions
     │ {"model": "gemini-webai", ...}
     ↓
┌──────────────────────────────────────┐
│ ROUTER - HTTP Server                 │
│ └─ Extract model: "gemini-webai"     │  1. Parse request
└────┬─────────────────────────────────┘
     │
     ↓
┌──────────────────────────────────────┐
│ ROUTER - Model Router                │
│ ├─ Match: ".*-webai$"                │  2. Pattern match
│ └─ Select: "webai"                   │  3. Select executor
└────┬─────────────────────────────────┘
     │
     ↓
┌──────────────────────────────────────┐
│ ROUTER - HTTPProxyExecutor           │
│ └─ Forward HTTP POST to:             │  4. HTTP proxy
│    http://localhost:8406             │
└────┬─────────────────────────────────┘
     │
     │ HTTP POST
     ↓
┌──────────────────────────────────────┐
│ WEBAI SERVICE (Python)               │
│                                      │
│ 1. Receive HTTP request              │  5. Receive request
│                                      │
│ 2. Extract cookies from config       │  6. Load credentials
│                                      │
│ 3. Try Gemini web API:               │  7. Primary attempt
│    ├─ Inject cookies                 │
│    ├─ Call gemini.google.com        │
│    └─ If success → return            │
│                                      │
│ 4. If fails → gpt4free fallback:     │  8. Fallback chain
│    ├─ Try provider 1                 │
│    ├─ Try provider 2                 │
│    ├─ ...                            │
│    └─ Try provider N                 │
│                                      │
│ 5. Return HTTP response              │  9. Return result
└────┬─────────────────────────────────┘
     │
     │ HTTP response
     ↓
┌──────────────────────────────────────┐
│ ROUTER - HTTPProxyExecutor           │
│ └─ Forward response to client        │  10. Proxy back
└────┬─────────────────────────────────┘
     │
     ↓
┌─────────┐
│ CLIENT  │
└─────────┘

⏱️  LATENCY: ~300-500ms
🔧  PROCESSES: 3 (router + aistudio + webai)
ℹ️  NOTE: Disabled by default (optional)
```

---

## 4. Authentication Flows

### 4.1 OAuth Flow (LLMux Claude/ChatGPT)

```
┌──────────┐
│  USER    │
└────┬─────┘
     │
     │ 1. Initiate auth
     │ GET /v1/auth/llmux/claude/login
     ↓
┌──────────────────────────────────────┐
│ ROUTER - OAuth Handler               │
│                                      │
│ 1. Generate PKCE challenge:          │
│    - code_verifier (random)          │
│    - code_challenge (SHA256)         │
│                                      │
│ 2. Start local callback server:      │
│    http://localhost:random/callback  │
│                                      │
│ 3. Build OAuth URL:                  │
│    https://claude.ai/oauth/authorize │
│    ?client_id=...                    │
│    &redirect_uri=http://localhost... │
│    &code_challenge=...               │
│                                      │
│ 4. Open system browser               │
└────┬─────────────────────────────────┘
     │
     ↓
┌──────────────────────────────────────┐
│ SYSTEM BROWSER                       │
│ https://claude.ai/oauth/authorize    │
│                                      │
│ ┌────────────────────────────────┐  │
│ │ Login to Claude                │  │ ← User enters credentials
│ │ Email: ________________        │  │
│ │ Password: ________________     │  │
│ │                                │  │
│ │ [Authorize]                    │  │ ← User clicks
│ └────────────────────────────────┘  │
└────┬─────────────────────────────────┘
     │
     │ Redirect to callback
     │ http://localhost:random/callback?code=AUTH_CODE
     ↓
┌──────────────────────────────────────┐
│ ROUTER - OAuth Callback Handler      │
│                                      │
│ 1. Receive auth code                 │
│                                      │
│ 2. Exchange code for tokens:         │
│    POST https://claude.ai/oauth/token│
│    code=AUTH_CODE                    │
│    code_verifier=...                 │
│                                      │
│ 3. Receive tokens:                   │
│    {                                 │
│      "access_token": "...",          │
│      "refresh_token": "...",         │
│      "expires_in": 3600              │
│    }                                 │
│                                      │
│ 4. Save tokens:                      │
│    ~/.cli-proxy-api/                 │
│    llmux-claude-user@example.com.json│
│    {                                 │
│      "access_token": "...",          │
│      "refresh_token": "...",         │
│      "expires_at": "2025-11-23T..." │
│    }                                 │
│                                      │
│ 5. Return success to user            │
└──────────────────────────────────────┘

✅ ONE-TIME: User only does this once
✅ AUTO-REFRESH: Router refreshes tokens automatically
✅ PERSISTENT: Tokens survive restarts
```

---

### 4.2 Browser Session Flow (AIstudio)

```
┌──────────┐
│  USER    │
└────┬─────┘
     │
     │ 1. Initiate auth
     │ POST /v1/auth/aistudio/login
     ↓
┌──────────────────────────────────────┐
│ ROUTER - AIStudio Auth Handler       │
│ └─ Send WebSocket message:           │
│    {"type": "auth_request",          │
│     "profile": "default"}            │
└────┬─────────────────────────────────┘
     │
     │ WebSocket message
     ↓
┌──────────────────────────────────────┐
│ AISTUDIO SERVICE                     │
│                                      │
│ 1. Receive auth request              │
│                                      │
│ 2. Launch VISIBLE browser:           │
│    - NOT headless                    │
│    - User can see and interact       │
│                                      │
│ 3. Navigate to:                      │
│    https://ai.studio.google.com      │
└────┬─────────────────────────────────┘
     │
     │ Opens visible browser window
     ↓
┌──────────────────────────────────────┐
│ VISIBLE BROWSER (Camoufox)           │
│                                      │
│ ┌────────────────────────────────┐  │
│ │ Sign in to Google              │  │
│ │ ____________________________   │  │ ← User enters email
│ │                                │  │
│ │ ____________________________   │  │ ← User enters password
│ │                                │  │
│ │ ☐ Verify with 2FA              │  │ ← User completes 2FA
│ │                                │  │
│ │ [Sign In]                      │  │ ← User clicks
│ └────────────────────────────────┘  │
│                                      │
│ ┌────────────────────────────────┐  │
│ │ ✓ Signed in successfully       │  │
│ │                                │  │
│ │ Welcome to AI Studio!          │  │
│ └────────────────────────────────┘  │
└────┬─────────────────────────────────┘
     │
     │ Browser signed in
     ↓
┌──────────────────────────────────────┐
│ AISTUDIO SERVICE                     │
│                                      │
│ 1. Detect successful login:          │
│    - Check for user profile          │
│    - Verify session active           │
│                                      │
│ 2. Extract session data:             │
│    - Browser cookies                 │
│    - localStorage tokens             │
│    - sessionStorage data             │
│                                      │
│ 3. Save session:                     │
│    providers/aistudio/               │
│    auth_profiles/default.json        │
│    {                                 │
│      "cookies": [...],               │
│      "localStorage": {...},          │
│      "profile": "default"            │
│    }                                 │
│                                      │
│ 4. Keep browser alive in pool:       │
│    - Reuse for subsequent requests   │
│    - Auto-refresh on timeout         │
│                                      │
│ 5. Send success WebSocket message    │
└────┬─────────────────────────────────┘
     │
     │ Success message
     ↓
┌──────────────────────────────────────┐
│ ROUTER                               │
│ └─ Return success to user            │
└──────────────────────────────────────┘

✅ MANUAL LOGIN: User logs in via browser UI
✅ PERSISTENT: Session saved, browser reused
✅ AUTO-REFRESH: Browser refreshed on idle
⚠️  REQUIRES DISPLAY: Needs X11/Wayland for visible browser
```

---

### 4.3 JWT Cookie Flow (ctonew)

```
┌──────────┐
│  USER    │
└────┬─────┘
     │
     │ 1. User extracts Clerk JWT cookie from browser
     │    (DevTools → Application → Cookies → __client)
     │
     │ 2. Provide to router
     │ POST /v1/auth/ctonew
     │ {"clerk_jwt_cookie": "eyJhbGciOiJSUzI1Ni..."}
     ↓
┌──────────────────────────────────────┐
│ ROUTER - ctonew Auth Handler         │
│                                      │
│ 1. Validate JWT format:              │
│    - Check 3 parts (header.payload.sig)
│    - Verify not expired             │
│                                      │
│ 2. Save to auth storage:             │
│    ~/.cli-proxy-api/ctonew-default.json
│    {                                 │
│      "clerk_jwt_cookie": "eyJ...",   │
│      "saved_at": "2025-11-22T..."   │
│    }                                 │
│                                      │
│ 3. Return success                    │
└──────────────────────────────────────┘

✅ SIMPLE: Just paste JWT cookie
✅ PERSISTENT: Saved for future requests
⚠️  MANUAL REFRESH: User must update if cookie expires
```

---

## 5. Configuration Architecture

### 5.1 Configuration Hierarchy

```
┌─────────────────────────────────────────────────────────┐
│ config.yaml (Master Configuration)                      │
├─────────────────────────────────────────────────────────┤
│                                                         │
│ server:                      ← Router settings          │
│   port: 8317                                            │
│   host: "0.0.0.0"                                       │
│                                                         │
│ auth:                        ← Auth storage             │
│   dir: "~/.cli-proxy-api"                               │
│   store: "file"                                         │
│                                                         │
│ providers:                   ← Provider configs         │
│   llmux:                     ← In-process providers     │
│     claude_pro: {...}                                   │
│     chatgpt_plus: {...}                                 │
│                                                         │
│   ctonew: {...}              ← In-process provider      │
│                                                         │
│   aistudio:                  ← External service         │
│     service:                                            │
│       command: "python providers/aistudio/main.py"      │
│     browser:                                            │
│       type: "camoufox"                                  │
│       max_instances: 3                                  │
│                                                         │
│   webai:                     ← External service (opt)   │
│     enabled: false           ← Disabled by default      │
│     service:                                            │
│       command: "python providers/webai/main.py"         │
│                                                         │
│ models:                      ← Model routing            │
│   routing:                                              │
│     - pattern: "^gpt-5"                                 │
│       providers: ["llmux-chatgpt", "ctonew"]            │
│     - pattern: "^claude-sonnet"                         │
│       providers: ["llmux-claude", "claude-oauth"]       │
│     - pattern: "gemini-.*-aistudio$"                    │
│       providers: ["aistudio"]                           │
└─────────────────────────────────────────────────────────┘
            │
            ├─────→ providers/aistudio/config.yaml
            │       (Service-specific config)
            │
            └─────→ providers/webai/config.yaml
                    (Service-specific config)
```

---

### 5.2 Auth Storage Structure

```
~/.cli-proxy-api/
├── llmux-claude-user@example.com.json
│   {
│     "access_token": "eyJ...",
│     "refresh_token": "eyJ...",
│     "expires_at": "2025-11-23T10:30:00Z",
│     "provider": "llmux-claude"
│   }
│
├── llmux-chatgpt-user@example.com.json
│   {
│     "access_token": "eyJ...",
│     "refresh_token": "eyJ...",
│     "expires_at": "2025-11-23T11:00:00Z",
│     "provider": "llmux-chatgpt"
│   }
│
├── ctonew-default.json
│   {
│     "clerk_jwt_cookie": "eyJhbGciOi...",
│     "saved_at": "2025-11-22T09:00:00Z",
│     "provider": "ctonew"
│   }
│
├── aistudio-default.json
│   {
│     "cookies": [...],
│     "localStorage": {...},
│     "profile": "default",
│     "provider": "aistudio"
│   }
│
└── webai-default.json (optional)
    {
      "cookies": {...},
      "provider": "webai"
    }

Permissions: 600 (-rw-------)
Owner: Current user
```

---

## 6. Deployment Scenarios

### 6.1 Local Development

```
Developer Machine
├─ cli-proxy-api (router)       ← Running in terminal 1
├─ aistudio service             ← Running in terminal 2
└─ Logs visible in real-time    ← Easy debugging

Usage:
./scripts/dev/start-router.sh       # Terminal 1
./scripts/dev/start-aistudio.sh     # Terminal 2

Benefits:
✅ Easy debugging (visible logs)
✅ Fast iteration (restart individual services)
✅ No process management needed
```

---

### 6.2 Production Server

```
Server
├─ Process Manager (start.sh)
│  ├─ cli-proxy-api (PID: 12345)
│  ├─ aistudio (PID: 12346)
│  └─ Health checks every 30s
│
├─ Logs
│  ├─ logs/router.log
│  └─ logs/aistudio.log
│
└─ Auto-restart on failure

Usage:
./scripts/start.sh          # Start all
./scripts/stop.sh           # Stop all
./scripts/status.sh         # Check health
./scripts/logs.sh router    # View logs

Benefits:
✅ Automatic management
✅ Health monitoring
✅ Auto-restart
✅ Centralized logging
```

---

### 6.3 Docker Deployment

```
Docker Containers
├─ cli-proxy-api container
│  └─ Port: 8317
│
├─ aistudio container
│  ├─ Visible browser (X11 forwarding)
│  └─ WebSocket to router
│
└─ webai container (optional)
   └─ HTTP server on :8406

Usage:
docker-compose up -d        # Start all
docker-compose logs -f      # View logs
docker-compose down         # Stop all

Benefits:
✅ Isolated environment
✅ Easy deployment
✅ Version control
✅ Reproducible
```

---

## 7. Scaling Considerations

### 7.1 Horizontal Scaling

```
Load Balancer
    ↓
┌───────────────────────────────────────┐
│ Instance 1                            │
│ ├─ Router (shared auth storage)      │
│ ├─ AIstudio service                  │
│ └─ Local browser pool                │
└───────────────────────────────────────┘
┌───────────────────────────────────────┐
│ Instance 2                            │
│ ├─ Router (shared auth storage)      │
│ ├─ AIstudio service                  │
│ └─ Local browser pool                │
└───────────────────────────────────────┘

Shared Auth Storage
├─ PostgreSQL
└─ Centralized token storage

Benefits:
✅ Handle more requests
✅ Redundancy
✅ Browser pool per instance
```

---

### 7.2 Service Scaling

```
Router (Single Instance)
    ↓
Multiple AIstudio Services
├─ AIstudio 1 (browsers 1-3)
├─ AIstudio 2 (browsers 4-6)
└─ AIstudio 3 (browsers 7-9)

Router load-balances across services
WebSocket connections to all

Benefits:
✅ More concurrent browser sessions
✅ Isolated browser pools
✅ Better resource utilization
```

---

## 8. Key Metrics

### 8.1 Performance Comparison

| Metric | Before (Over-eng) | After (Optimized) | Improvement |
|--------|------------------|-------------------|-------------|
| **Processes** | 5 | 2-3 | -40% to -60% |
| **LLMux Latency** | ~400ms | ~150ms | -62% |
| **ctonew Latency** | ~500ms | ~250ms | -50% |
| **Memory (Total)** | ~800MB | ~600MB | -25% |
| **Setup Time** | 45 min | 25 min | -44% |
| **Deployment Complexity** | High | Low | N/A |

---

### 8.2 Capacity Estimates

```
Single Instance Capacity:

Router (Go):
- Throughput: 100-200 req/s
- Memory: ~100 MB
- CPU: 2 cores

AIstudio (Python + Browser):
- Concurrent browsers: 3
- Memory: ~500 MB per browser
- Throughput: ~30 req/min (browser limited)

WebAI (Python):
- Throughput: 50-100 req/s
- Memory: ~200 MB
- CPU: 1 core
```

---

**END OF ARCHITECTURE DIAGRAMS**

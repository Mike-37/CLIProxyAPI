# Comprehensive Multi-Provider Router Redesign

## Part 1: Current State - What We Have

### Current Provider Model
```
┌────────────────────────────────────────────┐
│         CLIProxyAPI Router v1.0             │
│         (Single-Auth-Type Per Provider)     │
└────────────────────────────────────────────┘

Auth Entries:                  Can Return:
├─ claude (API key)       →    Claude models only
├─ codex (API key)        →    GPT models only
├─ gemini (API key)       →    Gemini models only
├─ qwen (Device Flow)     →    Qwen models only
├─ iflow (OAuth)          →    iFlow models only
├─ antigravity (OAuth)    →    Antigravity models only
└─ openai-compat (custom) →    Custom provider models

Model Registry:
  claude-3.5-sonnet        → provider: "claude"
  gpt-4-turbo              → provider: "codex"
  gemini-2-flash           → provider: "gemini"
  qwen-max                 → provider: "qwen"

Request Flow:
  User: "I want claude-3.5-sonnet"
    ↓
  Router: Find provider = "claude"
    ↓
  Router: Get claude auth entry (only 1 per email)
    ↓
  ClaudeExecutor: Make API call
    ↓
  Return response

LIMITATION:
❌ Cannot have multiple auth sources for same model type
❌ Cannot distinguish "claude via API key" vs "claude via OAuth" vs "claude via ctonew"
❌ Routing is 1:1 (model → provider → executor)
```

---

## Part 2: Web Providers Problem - Multiple Models Per Auth

### The Real Issue

```
LLMux OAuth auth entry can return:
├─ claude-sonnet-4-5       ✓
├─ claude-opus-4-1         ✓
├─ claude-haiku-4-5        ✓
├─ gpt-5                   ✓
├─ gpt-5-1                 ✓
└─ gpt-5-codex             ✓

AIStudioProxyAPI can return:
├─ gemini-2-flash          ✓
├─ gemini-1.5-pro          ✓
└─ gemini-1.5-flash        ✓

ctonew-proxy can return:
├─ claude-sonnet-4.5       ✓
└─ gpt-5                   ✓

WebAI-to-API can return:
├─ gemini-2-flash          ✓
├─ gemini-1.5-pro          ✓
└─ gpt4free:chatgpt        ✓
└─ gpt4free:claude         ✓
└─ gpt4free:deepseek       ✓
└─ ... (50+ models)        ✓

PROBLEM:
User wants: "claude-3.5-sonnet"

Where can they get it?
├─ claude (API key) ✓
├─ claude-oauth (LLMux) ✓
└─ ctonew-proxy ✓

Which one to use?
  - Priority?
  - Load balancing?
  - Different behavior?
  - Different latency/cost?

Same problem for GPT, Gemini, etc.
```

---

## Part 3: Design Options for Expansion

### OPTION A: Merge All Auth into Router (Everything In-Process)

```
┌──────────────────────────────────────────────────────┐
│            CLIProxyAPI Router v2.0                   │
│       (All Auth Types, Single Process)               │
└──────────────────────────────────────────────────────┘

Auth Manager:
├─ claude-api-key         → Auth + Executor (direct)
├─ claude-oauth           → Auth + Executor (direct)
├─ claude-ctonew          → Auth + Executor (JWT call to ctonew service)
├─ codex-api-key          → Auth + Executor (direct)
├─ codex-oauth            → Auth + Executor (direct)
├─ codex-ctonew           → Auth + Executor (JWT call to ctonew service)
├─ gemini-api-key         → Auth + Executor (direct)
├─ gemini-aistudio        → Auth + Executor (HTTP proxy to AIStudio service)
├─ gemini-webai           → Auth + Executor (HTTP proxy to WebAI service)
├─ qwen-device-flow       → Auth + Executor (direct)
├─ gpt4free-*             → Auth + Executor (HTTP proxy to WebAI service)
└─ ... (all combinations)

Model Registry (now N:M mapping):
  claude-3.5-sonnet:
    ├─ Source 1: claude (API key)
    ├─ Source 2: claude-oauth (LLMux)
    ├─ Source 3: ctonew-proxy
    └─ Fallback: gpt4free:claude (WebAI)

  gpt-4-turbo:
    ├─ Source 1: codex (API key)
    ├─ Source 2: codex-oauth (LLMux)
    ├─ Source 3: ctonew-proxy
    └─ Fallback: gpt4free:gpt4 (WebAI)

  gemini-2-flash:
    ├─ Source 1: gemini (API key)
    ├─ Source 2: gemini-aistudio (browser)
    └─ Fallback: gemini-webai (cookies)

Request Routing Logic:
  User: "I want gpt-4-turbo with reasoning"
    ↓
  Router: Find all sources for gpt-4-turbo
    ├─ codex (no reasoning support)
    ├─ codex-oauth (YES, reasoning)
    ├─ ctonew-proxy (YES, reasoning)
    └─ gpt4free:gpt4 (no reasoning)
    ↓
  Router: Filter by capabilities (reasoning needed)
    ├─ codex-oauth ✓
    └─ ctonew-proxy ✓
    ↓
  Router: Pick one (load balance, priority, latency)
    ↓
  Execute via selected executor

PROS:
✅ Single process, no inter-service communication
✅ Easier to implement (no service discovery)
✅ Faster (no network hops between services)
✅ Unified logging and monitoring
✅ Simpler deployment (1 service to manage)

CONS:
❌ MASSIVE complexity increase:
   - Need to implement ALL auth mechanisms in Go
   - Browser automation (Camoufox) → Go binding issues
   - Per-request JWT generation (Clerk) → complex state
   - Cookie management → browser cookies in Go
   - Need 20+ different executor implementations
   - Each executor has different auth flow

❌ Resource overhead:
   - If embedding browser (AIStudio), memory explosion
   - Can't scale browser instances
   - One process dies = everything dies

❌ Mixing concerns:
   - Official APIs (direct token)
   - Browser automation (stateful)
   - JWT brokers (ephemeral)
   - Cookie managers (opaque)
   - All in one process = nightmare

❌ Maintenance nightmare:
   - Each provider update breaks router
   - Provider auth changes require router redeploy
   - Can't isolate failures

EFFORT: ⭐⭐⭐⭐⭐ (extremely high)
MAINTAINABILITY: ⭐ (extremely low)
```

---

### OPTION B: Separate Services + Service Registry (Full Microservices)

```
┌─────────────────────────────────────────────────────┐
│       CLIProxyAPI Router (Service Registry)         │
│           (Orchestration Only)                      │
└─────────────────────────────────────────────────────┘
             │
  Manages:   ├─ Service discovery
             ├─ Routing logic
             ├─ Model mapping
             ├─ Load balancing
             ├─ Request normalization
             └─ Response translation
             │
             ▼

Separate Auth/Executor Services:

Service 1: Claude Direct Service         (Port 8400)
├─ Auth: API key, OAuth
├─ Handles: claude-api-key, claude-oauth
├─ Returns: Claude models only
└─ Simple HTTP API

Service 2: ChatGPT Direct Service        (Port 8401)
├─ Auth: API key, OAuth
├─ Handles: codex-api-key, codex-oauth
├─ Returns: GPT models only
└─ Simple HTTP API

Service 3: Gemini Direct Service         (Port 8402)
├─ Auth: API key
├─ Handles: gemini-api-key
├─ Returns: Gemini models only
└─ Simple HTTP API

Service 4: LLMux Unified Service         (Port 8403)
├─ Auth: OAuth for Claude + ChatGPT
├─ Handles: claude-oauth, codex-oauth
├─ Returns: Claude + GPT models
└─ Single OAuth → multiple model types

Service 5: AIStudio Service              (Port 8404)
├─ Auth: Browser automation (Camoufox)
├─ Handles: gemini-aistudio
├─ Returns: Gemini models
├─ Manages: Playwright browser
└─ Complex stateful service

Service 6: Ctonew Service                (Port 8405)
├─ Auth: Clerk JWT
├─ Handles: ctonew-proxy
├─ Returns: Claude + GPT models
└─ Stateless, can scale

Service 7: WebAI Service                 (Port 8406)
├─ Auth: Browser cookies + gpt4free
├─ Handles: webai-gemini, gpt4free:*
├─ Returns: Gemini + 50+ model types
└─ Fallback system

Service 8: Qwen Service                  (Port 8407)
├─ Auth: Device Flow
├─ Handles: qwen-device-flow
├─ Returns: Qwen models only
└─ Device auth implemented

Service 9: iFlow Service                 (Port 8408)
├─ Auth: OAuth
├─ Handles: iflow-oauth
├─ Returns: iFlow models only
└─ Custom provider

Service 10: Antigravity Service          (Port 8409)
├─ Auth: OAuth
├─ Handles: antigravity-oauth
├─ Returns: Antigravity models only
└─ Custom provider

Router Architecture:
┌─────────────────────────────────────────────────────┐
│          CLIProxyAPI Router (Port 8317)             │
├─────────────────────────────────────────────────────┤
│                                                      │
│  1. Request Handler                                 │
│     - API key validation                            │
│     - Parse model name                              │
│                                                      │
│  2. Model Resolution Engine                         │
│     - Model → [Services] mapping                    │
│     - Filter by capabilities                        │
│     - Track availability                            │
│                                                      │
│  3. Service Discovery                               │
│     - Health check all services                     │
│     - Detect failures                               │
│     - Track load per service                        │
│                                                      │
│  4. Routing Strategy                                │
│     - Load balancing (round-robin, least-busy)      │
│     - Priority (user-configured)                    │
│     - Fallback chains                               │
│                                                      │
│  5. Request Translator                              │
│     - Normalize request format                      │
│     - Add provider-specific params                  │
│     - Handle streaming                              │
│                                                      │
│  6. Response Translator                             │
│     - Normalize to OpenAI format                    │
│     - Handle errors from services                   │
│     - Track usage                                   │
│                                                      │
│  7. Auth Manager (Minimal)                          │
│     - Store which services are enabled              │
│     - Store service endpoints                       │
│     - Delegate auth to each service                 │
│                                                      │
└─────────────────────────────────────────────────────┘

Model Registry (in Router):
```yaml
models:
  claude-3.5-sonnet:
    providers:
      - name: claude-direct
        service: "http://localhost:8400"
        auth_type: "api-key"
        priority: 1
        features: [vision, tool-use, batch]
      - name: claude-oauth
        service: "http://localhost:8403"  # LLMux service
        auth_type: "oauth"
        priority: 2
        features: [vision, tool-use, batch, reasoning]
      - name: ctonew
        service: "http://localhost:8405"
        auth_type: "jwt"
        priority: 3
        features: [vision, tool-use]

  gpt-4-turbo:
    providers:
      - name: codex-direct
        service: "http://localhost:8401"
        auth_type: "api-key"
        priority: 1
        features: [vision, tool-use]
      - name: codex-oauth
        service: "http://localhost:8403"  # LLMux service
        auth_type: "oauth"
        priority: 2
        features: [vision, tool-use, reasoning]
      - name: ctonew
        service: "http://localhost:8405"
        auth_type: "jwt"
        priority: 3
        features: [vision, tool-use]

  gemini-2-flash:
    providers:
      - name: gemini-direct
        service: "http://localhost:8402"
        auth_type: "api-key"
        priority: 1
        features: [vision, tool-use]
      - name: gemini-aistudio
        service: "http://localhost:8404"
        auth_type: "browser"
        priority: 2
        features: [vision, tool-use, thinking]
      - name: gemini-webai
        service: "http://localhost:8406"
        auth_type: "cookie"
        priority: 3
        features: [vision, tool-use]

  gpt4free-chatgpt:
    providers:
      - name: webai-gpt4free
        service: "http://localhost:8406"
        auth_type: "none"
        priority: 1
        features: []
```

Request Flow:
```
User Request
    ↓
Router: Validate API key
    ↓
Router: Parse model "claude-3.5-sonnet"
    ↓
Router: Look up providers for "claude-3.5-sonnet"
    ├─ claude-direct (priority 1, 8400)
    ├─ claude-oauth (priority 2, 8403)
    └─ ctonew (priority 3, 8405)
    ↓
Router: Health check each
    ├─ 8400: Up, healthy, last checked 1s ago
    ├─ 8403: Up, healthy, load 0.2
    └─ 8405: Down (last check failed)
    ↓
Router: Filter by capabilities needed
    (e.g., if request needs reasoning)
    ├─ claude-direct: No reasoning
    └─ claude-oauth: YES reasoning ✓
    ├─ ctonew: Unavailable
    ↓
Router: Pick one (if multiple available)
    Using strategy: [priority, load-balance, latency]
    ↓
Router: Normalize request for selected service
    (e.g., add auth header, convert params)
    ↓
POST to selected service:
    http://localhost:8403/v1/messages
    {
      "model": "claude-3.5-sonnet",
      "messages": [...],
      "max_tokens": 1024,
      "_internal_auth_id": "claude-oauth-user@example.com"
    }
    ↓
Service processes & returns response
    ↓
Router: Translate response (if needed)
    ↓
Return to client
```

PROS:
✅ Clean separation of concerns
✅ Each service independent
✅ Can deploy/update services separately
✅ Easy to add new providers (just add new service)
✅ Can scale services independently
✅ Language flexibility (Go, Python, TypeScript, etc.)
✅ Fault isolation (one service down ≠ others down)
✅ Reduces router complexity significantly
✅ Clear interfaces between components

CONS:
❌ Inter-service communication overhead
❌ Network latency (local though, ~1-5ms)
❌ More complex deployment (many services to manage)
❌ Service discovery needed
❌ Distributed tracing complexity
❌ More moving parts to monitor
❌ Health check overhead

EFFORT: ⭐⭐⭐⭐ (high, need service architecture)
MAINTAINABILITY: ⭐⭐⭐⭐⭐ (excellent)
```

---

### OPTION C: Hybrid - Merge Direct APIs, Separate Web Providers (RECOMMENDED)

```
┌──────────────────────────────────────────────────────┐
│    CLIProxyAPI Router (Hybrid Architecture)          │
│  Merges direct APIs + proxies to web provider        │
│              services                                │
└──────────────────────────────────────────────────────┘

IN-PROCESS (Direct Executors):
├─ claude-api-key        → ClaudeAPIExecutor
├─ claude-oauth          → ClaudeOAuthExecutor (LLMux)
├─ codex-api-key         → CodexAPIExecutor
├─ codex-oauth           → CodexOAuthExecutor (LLMux)
├─ gemini-api-key        → GeminiAPIExecutor
├─ qwen-device-flow      → QwenExecutor
├─ iflow-oauth           → iFlowExecutor
├─ antigravity-oauth     → AntigravityExecutor
└─ openai-compat         → OpenAICompatExecutor

OUT-OF-PROCESS (HTTP Proxy Executors):
├─ gemini-aistudio       → HTTPProxyExecutor → AIStudioProxyAPI (8404)
├─ gemini-webai          → HTTPProxyExecutor → WebAI-to-API (8406)
├─ gpt4free-*            → HTTPProxyExecutor → WebAI-to-API (8406)
├─ ctonew-claude         → HTTPProxyExecutor → ctonew-proxy (8405)
└─ ctonew-gpt            → HTTPProxyExecutor → ctonew-proxy (8405)

Auth Manager (in Router):
├─ Direct tokens: FileStore (existing)
├─ Service configs: Simple endpoint + auth type
└─ Auto-refresh: Only for direct APIs

Model Registry (in Router):
```yaml
models:
  claude-3.5-sonnet:
    sources:
      - auth: "claude-api-key"      # Direct
        service: null               # In-process
        priority: 1
      - auth: "claude-oauth"        # Direct (LLMux)
        service: null               # In-process
        priority: 2
      - auth: "ctonew-claude"       # Proxy
        service: "http://localhost:8405"
        priority: 3

  gpt-4-turbo:
    sources:
      - auth: "codex-api-key"       # Direct
        service: null               # In-process
        priority: 1
      - auth: "codex-oauth"         # Direct (LLMux)
        service: null               # In-process
        priority: 2
      - auth: "ctonew-gpt"          # Proxy
        service: "http://localhost:8405"
        priority: 3

  gemini-2-flash:
    sources:
      - auth: "gemini-api-key"      # Direct
        service: null               # In-process
        priority: 1
      - auth: "gemini-aistudio"     # Proxy
        service: "http://localhost:8404"
        priority: 2
      - auth: "gemini-webai"        # Proxy
        service: "http://localhost:8406"
        priority: 3
```

Router Components:

```
┌──────────────────────────────────────────────────────┐
│            CLIProxyAPI Router (Port 8317)            │
├──────────────────────────────────────────────────────┤
│                                                       │
│  ACCESS LAYER                                        │
│  ├─ API Key validation                              │
│  └─ Rate limiting                                   │
│                                                       │
│  REQUEST PARSING                                     │
│  ├─ Extract model name                              │
│  ├─ Parse parameters                                │
│  └─ Detect capabilities needed (reasoning, etc)     │
│                                                       │
│  MODEL RESOLUTION                                    │
│  ├─ Find all available sources for model            │
│  ├─ Filter by availability (service health)         │
│  ├─ Filter by capabilities (needs reasoning?)       │
│  └─ Filter by user config (allowed providers)       │
│                                                       │
│  ROUTING DECISION                                    │
│  ├─ Load balance across available sources           │
│  ├─ Apply priority rules                            │
│  ├─ Consider latency/cost                           │
│  └─ Fallback strategy                               │
│                                                       │
│  REQUEST NORMALIZATION                              │
│  ├─ Convert to provider format (if direct)          │
│  ├─ Add auth headers/tokens                         │
│  ├─ Handle provider-specific params                 │
│  └─ Stream setup                                    │
│                                                       │
│  EXECUTION                                           │
│  │                                                  │
│  ├─ Direct: Call local executor                     │
│  │  └─ Use token from FileStore                     │
│  │  └─ Make API call directly                       │
│  │                                                  │
│  └─ Proxy: Call HTTPProxyExecutor                   │
│     ├─ Determine service endpoint                   │
│     ├─ Forward request to service                   │
│     ├─ Stream/buffer response                       │
│     └─ Handle service errors                        │
│                                                       │
│  RESPONSE HANDLING                                   │
│  ├─ Parse response (streaming or buffered)          │
│  ├─ Translate to OpenAI format (if needed)          │
│  ├─ Add usage tracking                              │
│  └─ Error handling/translation                      │
│                                                       │
│  AUTH MANAGEMENT                                     │
│  ├─ Direct providers: Token storage + auto-refresh  │
│  ├─ Service providers: Endpoint config only         │
│  └─ Health monitoring                               │
│                                                       │
└──────────────────────────────────────────────────────┘
```

Separate Services (Minimal):

Service 1: AIstudioProxyAPI (Port 8404)
├─ Manages Playwright + Camoufox
├─ Handles Google AI Studio authentication
├─ Returns Gemini models only
└─ Completely isolated

Service 2: ctonew-proxy (Port 8405)
├─ Stateless JWT extraction
├─ Clerk integration
├─ Returns Claude + GPT
└─ Can run on Deno Deploy

Service 3: WebAI-to-API (Port 8406) [Optional]
├─ Browser cookie management
├─ gpt4free fallback
├─ Returns Gemini + 50+ model types
└─ Optional if no cookie-based access needed

PROS:
✅ Best of both worlds:
   - Direct APIs stay fast (no network hop)
   - Complex services stay isolated
   - Easy to manage
✅ Low complexity increase:
   - Only add new direct executors for LLMux OAuth
   - Reuse existing auth patterns
   - HTTPProxyExecutor handles all web services
✅ Easy to evolve:
   - Can add new direct APIs easily
   - Can add new web services easily
   - No forced merges
✅ Good resource usage:
   - Direct APIs minimal memory
   - Web services in separate processes
   - Can scale each independently
✅ Practical deployment:
   - Router + 2-3 small services
   - Can run everything on one machine
   - Or separate machines
✅ Best maintainability:
   - Router stays clean (~2000 lines)
   - Web services unchanged
   - Clear ownership boundaries

CONS:
⚠️ Some network latency:
   - Direct: None (local executor)
   - Web: 1-2ms (local service)
   - Overall impact: minimal
✅ Trade-off worth it for cleanliness

EFFORT: ⭐⭐⭐ (medium - most work is direct APIs)
MAINTAINABILITY: ⭐⭐⭐⭐⭐ (excellent)
```

---

## Part 4: Recommendation: Option C (Hybrid)

### Why Option C Wins

| Aspect | Option A | Option B | Option C |
|--------|----------|----------|----------|
| **Complexity** | ⭐ (nightmare) | ⭐⭐⭐⭐⭐ (complex) | ⭐⭐⭐ (manageable) |
| **Speed** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **Maintainability** | ⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **Deployment** | ⭐⭐ (1 monster) | ⭐⭐ (10 services) | ⭐⭐⭐⭐ (3-4 services) |
| **Scalability** | ⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| **Fault Isolation** | ⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| **Learning Curve** | ⭐ (very hard) | ⭐⭐⭐⭐ (hard) | ⭐⭐⭐ (moderate) |
| **Effort to Implement** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ |
| **Cost/Benefit Ratio** | Poor | Good | **Excellent** |

---

## Part 5: Final Architecture Design (Option C)

### Complete System Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           EXTERNAL CLIENTS                                   │
│                      (CLI, Python, JavaScript, etc)                          │
└────────────────────────────────────┬────────────────────────────────────────┘
                                     │
                     HTTP POST /v1/chat/completions
                                     │
                    ┌────────────────▼────────────────┐
                    │    CLIProxyAPI Router           │
                    │         (Port 8317)             │
                    │   Unified Entry Point           │
                    └────────────────┬────────────────┘
                                     │
                    ┌────────────────▼────────────────────────────────┐
                    │  1. AUTH LAYER (Access Control)                 │
                    │  ├─ Validate API key from header                │
                    │  ├─ Check rate limits                           │
                    │  └─ Set security headers                        │
                    └────────────────┬────────────────────────────────┘
                                     │
                    ┌────────────────▼────────────────────────────────┐
                    │  2. REQUEST PARSING                             │
                    │  ├─ Extract: model, messages, parameters        │
                    │  ├─ Detect: reasoning, vision, streaming, etc   │
                    │  └─ Normalize: request format                   │
                    └────────────────┬────────────────────────────────┘
                                     │
                    ┌────────────────▼────────────────────────────────┐
                    │  3. MODEL RESOLUTION ENGINE                     │
                    │  ├─ Parse model: "claude-3.5-sonnet"            │
                    │  ├─ Lookup: model → [sources] from registry     │
                    │  │   Returns:                                   │
                    │  │   - claude-api-key (priority 1)              │
                    │  │   - claude-oauth (priority 2)                │
                    │  │   - ctonew (priority 3)                      │
                    │  ├─ Filter: by availability (health check)      │
                    │  │   - claude-api-key: UP                       │
                    │  │   - claude-oauth: UP                         │
                    │  │   - ctonew: DOWN (skip)                      │
                    │  ├─ Filter: by capabilities (reasoning needed?) │
                    │  │   - claude-api-key: No reasoning (remove)    │
                    │  │   - claude-oauth: YES reasoning ✓            │
                    │  └─ Result: [claude-oauth]                      │
                    └────────────────┬────────────────────────────────┘
                                     │
                    ┌────────────────▼────────────────────────────────┐
                    │  4. ROUTING DECISION                            │
                    │  ├─ Available sources after filtering:          │
                    │  │   [claude-oauth]                             │
                    │  ├─ If multiple:                                │
                    │  │   ├─ Load balance: least-busy-first          │
                    │  │   ├─ Round-robin: rotate requests            │
                    │  │   └─ Latency-aware: pick fastest             │
                    │  ├─ Fallback: if primary fails, try next        │
                    │  └─ Selected: "claude-oauth"                    │
                    │     ├─ Type: "direct" (in-process executor)     │
                    │     ├─ Executor: ClaudeOAuthExecutor            │
                    │     └─ Auth: token from FileStore               │
                    └────────────────┬────────────────────────────────┘
                                     │
                    ┌────────────────▼────────────────────────────────┐
                    │  5. REQUEST NORMALIZATION                       │
                    │  ├─ If direct executor:                         │
                    │  │   ├─ Get token from FileStore                │
                    │  │   ├─ Check if expired (auto-refresh if so)   │
                    │  │   ├─ Translate to Anthropic format           │
                    │  │   └─ Add reasoning params if needed           │
                    │  │                                              │
                    │  └─ If proxy executor:                          │
                    │      ├─ Determine service endpoint              │
                    │      ├─ Pass request to service                 │
                    │      └─ Let service handle auth                 │
                    └────────────────┬────────────────────────────────┘
                                     │
                    ┌────────────────▼────────────────────────────────┐
                    │  6. EXECUTION                                   │
                    │                                                  │
                    │  BRANCH A: Direct Execution                    │
                    │  ├─ Call ClaudeOAuthExecutor.Execute()         │
                    │  ├─ Executor: POST to api.anthropic.com        │
                    │  │   Headers: Authorization: Bearer {token}     │
                    │  │   Body: Anthropic format with reasoning      │
                    │  └─ Return response                             │
                    │                                                  │
                    │  BRANCH B: Proxy Execution                     │
                    │  ├─ Call HTTPProxyExecutor.Execute()           │
                    │  ├─ Executor: POST to service                  │
                    │  │   URL: http://localhost:8404/v1/messages    │
                    │  │   Headers/Body: Pass through as-is           │
                    │  ├─ Service processes:                          │
                    │  │   ├─ AIStudio service: Browser automation    │
                    │  │   ├─ ctonew service: JWT generation + call   │
                    │  │   └─ WebAI service: Cookie management        │
                    │  └─ Return response                             │
                    │                                                  │
                    └────────────────┬────────────────────────────────┘
                                     │
                    ┌────────────────▼────────────────────────────────┐
                    │  7. RESPONSE HANDLING                           │
                    │  ├─ Handle streaming:                           │
                    │  │   ├─ Direct: SSE stream from API             │
                    │  │   └─ Proxy: SSE stream from service          │
                    │  ├─ Parse response:                             │
                    │  │   ├─ Extract tokens, reasoning blocks        │
                    │  │   └─ Normalize format                        │
                    │  ├─ Translate to OpenAI format:                 │
                    │  │   ├─ Anthropic → OpenAI conversion           │
                    │  │   ├─ Gemini → OpenAI conversion              │
                    │  │   └─ Custom → OpenAI conversion              │
                    │  ├─ Track usage:                                │
                    │  │   ├─ Input tokens                            │
                    │  │   ├─ Output tokens                           │
                    │  │   └─ Cache hits/writes                       │
                    │  ├─ Error handling:                             │
                    │  │   ├─ 401/403 → Update token status           │
                    │  │   ├─ 429 → Backoff + retry                   │
                    │  │   ├─ 5xx → Try fallback provider             │
                    │  │   └─ Translate error to OpenAI format        │
                    │  └─ Add response headers                        │
                    │                                                  │
                    └────────────────┬────────────────────────────────┘
                                     │
                    ┌────────────────▼────────────────────────────────┐
                    │  8. RESPONSE RETURN                             │
                    │  ├─ Streaming: SSE stream to client             │
                    │  ├─ Non-streaming: JSON response to client      │
                    │  └─ Status: 200 (success) or error code         │
                    └────────────────┬────────────────────────────────┘
                                     │
                    ┌────────────────▼────────────────────────────────┐
                    │    LOGGING & MONITORING                         │
                    │  ├─ Log: provider, model, tokens, latency       │
                    │  ├─ Metrics: success rate, latency, load        │
                    │  └─ Alerts: service down, high error rate       │
                    └────────────────┬────────────────────────────────┘
                                     │
                                     ▼
                    ┌────────────────────────────────┐
                    │    Response to Client           │
                    │  {                              │
                    │    "choices": [...],            │
                    │    "usage": {...}               │
                    │  }                              │
                    └────────────────────────────────┘

SEPARATE SERVICES (Background):

┌─────────────────────────────────────┐
│  Direct API Executors (in-process)  │
├─────────────────────────────────────┤
│ ✓ ClaudeAPIExecutor                 │
│ ✓ ClaudeOAuthExecutor               │
│ ✓ CodexAPIExecutor                  │
│ ✓ CodexOAuthExecutor                │
│ ✓ GeminiAPIExecutor                 │
│ ✓ QwenExecutor                      │
│ ✓ iFlowExecutor                     │
│ ✓ AntigravityExecutor               │
│ ✓ OpenAICompatExecutor              │
│ ✓ HTTPProxyExecutor (for services)  │
└─────────────────────────────────────┘

┌────────────────────────────────────────────┐
│  Optional Web Provider Services            │
├────────────────────────────────────────────┤
│                                            │
│  AIstudioProxyAPI (Port 8404)             │
│  ├─ Playwright + Camoufox                │
│  ├─ Google AI Studio automation          │
│  ├─ Returns: Gemini models               │
│  └─ Health: Checked every 10s             │
│                                            │
│  ctonew-proxy (Port 8405)                 │
│  ├─ Deno + Oak framework                 │
│  ├─ Clerk JWT extraction                 │
│  ├─ Returns: Claude + GPT                │
│  └─ Health: Checked every 10s             │
│                                            │
│  WebAI-to-API (Port 8406) [Optional]     │
│  ├─ FastAPI + Python                     │
│  ├─ Cookie management + gpt4free         │
│  ├─ Returns: Gemini + 50+ models         │
│  └─ Health: Checked every 10s             │
│                                            │
└────────────────────────────────────────────┘

Auth Management (Hybrid):

Direct Providers (in Router):
├─ claude-api-key: FileStore
├─ claude-oauth: FileStore + auto-refresh
├─ codex-api-key: FileStore
├─ codex-oauth: FileStore + auto-refresh
├─ gemini-api-key: FileStore
├─ qwen: FileStore + device flow
├─ iflow: FileStore + OAuth
└─ antigravity: FileStore + OAuth

Service Providers (minimal):
├─ aistudio: endpoint config only
├─ ctonew: endpoint config only
└─ webai: endpoint config only
```

---

## Part 6: Detailed Model Registry

### Single Source of Truth

```yaml
# model_registry.yaml (in router)

model_groups:
  # Claude Models
  claude-sonnet-4-5:
    display_name: "Claude Sonnet 4.5"
    source: "anthropic"

    providers:
      # Option 1: Direct API
      - id: "claude-api-key"
        auth_type: "api-key"
        executor_type: "direct"
        executor_class: "ClaudeAPIExecutor"
        priority: 1
        available: true

      # Option 2: OAuth (LLMux)
      - id: "claude-oauth"
        auth_type: "oauth"
        executor_type: "direct"
        executor_class: "ClaudeOAuthExecutor"
        priority: 2
        features: [reasoning]  # Only this one has reasoning
        available: true

      # Option 3: Via Ctonew service
      - id: "ctonew-claude"
        auth_type: "jwt"
        executor_type: "proxy"
        executor_class: "HTTPProxyExecutor"
        service_url: "http://localhost:8405"
        priority: 3
        available: true

  claude-opus-4-1:
    display_name: "Claude Opus 4.1"
    source: "anthropic"

    providers:
      - id: "claude-api-key"
        priority: 1
      - id: "claude-oauth"
        priority: 2
      - id: "ctonew-claude"
        priority: 3

  # GPT Models
  gpt-4-turbo:
    display_name: "GPT-4 Turbo"
    source: "openai"

    providers:
      - id: "codex-api-key"
        auth_type: "api-key"
        executor_type: "direct"
        executor_class: "CodexAPIExecutor"
        priority: 1
        available: true

      - id: "codex-oauth"
        auth_type: "oauth"
        executor_type: "direct"
        executor_class: "CodexOAuthExecutor"
        priority: 2
        features: [reasoning]
        available: true

      - id: "ctonew-gpt"
        auth_type: "jwt"
        executor_type: "proxy"
        executor_class: "HTTPProxyExecutor"
        service_url: "http://localhost:8405"
        priority: 3
        available: true

  gpt-5:
    display_name: "GPT-5"
    source: "openai"

    providers:
      - id: "codex-oauth"
        priority: 1
        features: [reasoning]

      - id: "ctonew-gpt"
        priority: 2

  # Gemini Models
  gemini-2-flash:
    display_name: "Gemini 2.0 Flash"
    source: "google"

    providers:
      - id: "gemini-api-key"
        auth_type: "api-key"
        executor_type: "direct"
        executor_class: "GeminiAPIExecutor"
        priority: 1
        available: true

      - id: "gemini-aistudio"
        auth_type: "none"  # Service manages auth
        executor_type: "proxy"
        executor_class: "HTTPProxyExecutor"
        service_url: "http://localhost:8404"
        priority: 2
        features: [thinking]  # Only web version has thinking
        available: true

      - id: "gemini-webai"
        auth_type: "none"  # Service manages auth
        executor_type: "proxy"
        executor_class: "HTTPProxyExecutor"
        service_url: "http://localhost:8406"
        priority: 3
        available: true

  gemini-1.5-pro:
    display_name: "Gemini 1.5 Pro"
    source: "google"

    providers:
      - id: "gemini-api-key"
        priority: 1
      - id: "gemini-aistudio"
        priority: 2
      - id: "gemini-webai"
        priority: 3

  # Qwen Models
  qwen-max:
    display_name: "Qwen Max"
    source: "alibaba"

    providers:
      - id: "qwen-device-flow"
        auth_type: "device-flow"
        executor_type: "direct"
        executor_class: "QwenExecutor"
        priority: 1
        available: true

  # iFlow Models
  iflow-pro:
    display_name: "iFlow Pro"
    source: "iflow"

    providers:
      - id: "iflow-oauth"
        auth_type: "oauth"
        executor_type: "direct"
        executor_class: "iFlowExecutor"
        priority: 1
        available: true

  # Antigravity Models
  antigravity-large:
    display_name: "Antigravity Large"
    source: "antigravity"

    providers:
      - id: "antigravity-oauth"
        auth_type: "oauth"
        executor_type: "direct"
        executor_class: "AntigravityExecutor"
        priority: 1
        available: true

  # Open Router & Custom Providers
  openrouter-claude-opus:
    display_name: "Claude Opus (via OpenRouter)"
    source: "openrouter"

    providers:
      - id: "openai-compat-openrouter"
        auth_type: "api-key"
        executor_type: "direct"
        executor_class: "OpenAICompatExecutor"
        base_model: "anthropic/claude-opus"
        priority: 1
        available: true

  # Fallback/Free Models
  gpt4free-chatgpt:
    display_name: "ChatGPT (via gpt4free)"
    source: "gpt4free"

    providers:
      - id: "webai-gpt4free"
        auth_type: "none"
        executor_type: "proxy"
        executor_class: "HTTPProxyExecutor"
        service_url: "http://localhost:8406"
        priority: 1
        available: false  # Requires WebAI service

  gpt4free-claude:
    display_name: "Claude (via gpt4free)"
    source: "gpt4free"

    providers:
      - id: "webai-gpt4free"
        priority: 1
```

---

## Part 7: Configuration for Hybrid Architecture

### config.yaml (Main Router)

```yaml
# CLIProxyAPI Config for Hybrid Architecture

server:
  port: 8317
  debug: false
  log_level: "info"

# Auth Storage (for direct providers)
auth:
  storage:
    type: "file"  # or "postgres", "git", "object"
    file:
      dir: "~/.cli-proxy-api"

# Direct API Providers (in-process)
providers:
  # Claude
  claude:
    enabled: true
    type: "direct"
    api_endpoint: "https://api.anthropic.com"

  claude_oauth:
    enabled: true
    type: "direct"
    oauth:
      client_id: "9d1c250a-e61b-44d9-88ed-5944d1962f5e"
      issuer: "https://accounts.anthropic.com"
      scopes: [claude]
      callback_port: 54546

  # ChatGPT/Codex
  codex:
    enabled: true
    type: "direct"
    api_endpoint: "https://api.openai.com"

  codex_oauth:
    enabled: true
    type: "direct"
    oauth:
      issuer: "https://auth.openai.com"
      scopes: [openai-oauth]
      callback_port: 54547

  # Gemini
  gemini:
    enabled: true
    type: "direct"
    api_endpoint: "https://generativelanguage.googleapis.com"

  # Qwen
  qwen:
    enabled: false  # Optional
    type: "direct"
    device_flow: true

  # iFlow
  iflow:
    enabled: false  # Optional
    type: "direct"
    oauth:
      issuer: "https://oauth.iflow.com"
      callback_port: 54548

  # Antigravity
  antigravity:
    enabled: false  # Optional
    type: "direct"
    oauth:
      issuer: "https://oauth.antigravity.com"
      callback_port: 54549

# External Service Providers (proxy)
services:
  aistudio:
    enabled: false  # Disable if AIStudioProxyAPI not running
    type: "proxy"
    endpoint: "http://localhost:8404"
    health_check_interval: "10s"
    health_check_timeout: "2s"

  ctonew:
    enabled: false  # Disable if ctonew-proxy not running
    type: "proxy"
    endpoint: "http://localhost:8405"
    health_check_interval: "10s"
    health_check_timeout: "2s"

  webai:
    enabled: false  # Disable if WebAI-to-API not running
    type: "proxy"
    endpoint: "http://localhost:8406"
    health_check_interval: "10s"
    health_check_timeout: "2s"

# Model Routing Preferences
routing:
  strategy: "priority"  # or "load-balance", "latency-aware", "random"

  # Per-model preferences (override defaults)
  model_overrides:
    "claude-3.5-sonnet":
      providers: ["claude-api-key", "claude-oauth"]  # Don't use ctonew for this
      preferred: "claude-oauth"  # Always use this if available

    "gpt-4-turbo":
      providers: ["codex-api-key", "codex-oauth"]
      preferred: "codex-oauth"

    "gemini-2-flash":
      providers: ["gemini-api-key", "gemini-aistudio", "gemini-webai"]
      preferred: "gemini-api-key"  # Fast, official

  # Global fallback behavior
  fallback:
    enabled: true
    max_retries: 3  # Try up to 3 providers
    backoff: "exponential"  # 1s, 2s, 4s

# API Access Control
access_control:
  type: "api-key"  # How clients authenticate to router
  keys:
    - key: "sk-router-dev"
      rate_limit: "1000/hour"
    - key: "sk-router-prod"
      rate_limit: "10000/hour"

# Monitoring & Observability
observability:
  logging:
    format: "json"  # or "text"
    level: "info"

  metrics:
    enabled: true
    export_interval: "60s"

  tracing:
    enabled: false  # Set to true if using Jaeger/Otel
    sampler: 0.1  # 10% of requests
```

### services.yaml (Service Configurations)

```yaml
# Config for external provider services

services:
  aistudio:
    name: "AIStudioProxyAPI"
    description: "Google AI Studio with Playwright automation"
    endpoint: "http://localhost:8404"

    # What this service provides
    models:
      - "gemini-2-flash"
      - "gemini-1.5-pro"
      - "gemini-1.5-flash"

    # How to authenticate to this service
    auth_type: "none"  # Service handles auth internally

    # Service capabilities
    capabilities:
      streaming: true
      reasoning: false  # AI Studio doesn't expose thinking
      vision: true

    # Health check
    health_check:
      path: "/health"
      interval: "10s"
      timeout: "2s"

    # Fallback behavior
    auto_retry_on_failure: true
    max_retries: 2

  ctonew:
    name: "ctonew-proxy"
    description: "Ctonew/EngineLabs via Clerk JWT"
    endpoint: "http://localhost:8405"

    models:
      - "claude-3.5-sonnet"
      - "gpt-4-turbo"
      - "gpt-5"

    auth_type: "none"  # Service handles Clerk auth internally

    capabilities:
      streaming: true
      reasoning: true
      vision: true

    health_check:
      path: "/health"
      interval: "10s"
      timeout: "2s"

    auto_retry_on_failure: true
    max_retries: 2

  webai:
    name: "WebAI-to-API"
    description: "Gemini web + gpt4free fallback"
    endpoint: "http://localhost:8406"

    models:
      - "gemini-2-flash"
      - "gemini-1.5-pro"
      - "gpt4free-chatgpt"
      - "gpt4free-claude"
      - "gpt4free-deepseek"  # ...50+ models

    auth_type: "none"  # Service handles cookie auth internally

    capabilities:
      streaming: true
      reasoning: false
      vision: true

    health_check:
      path: "/"
      interval: "10s"
      timeout: "2s"

    auto_retry_on_failure: true
    max_retries: 2
```

---

## Part 8: Request Routing Examples

### Example 1: User Requests Claude with Reasoning

```
REQUEST:
POST /v1/chat/completions
{
  "model": "claude-3.5-sonnet",
  "messages": [...],
  "parameters": {
    "thinking_budget_tokens": 8000  # Requires reasoning
  }
}

ROUTER FLOW:

1. Parse: model = "claude-3.5-sonnet", needs_reasoning = true

2. Lookup model_registry:
   Sources:
   - claude-api-key (priority 1)
   - claude-oauth (priority 2) - HAS reasoning ✓
   - ctonew (priority 3)

3. Filter by availability:
   - claude-api-key: UP ✓
   - claude-oauth: UP ✓
   - ctonew: DOWN ✗

4. Filter by capabilities (reasoning):
   - claude-api-key: NO reasoning support ✗ (REMOVE)
   - claude-oauth: YES reasoning ✓

5. Remaining options: [claude-oauth]

6. Select executor:
   - Type: "direct"
   - Class: "ClaudeOAuthExecutor"
   - Auth: Load token from ~/.cli-proxy-api/claude-oauth-user.json
   - Check if expired: No (auto-refresh if so)

7. Normalize request:
   - Add thinking_budget_tokens: 8000
   - Add Claude-specific headers
   - Convert format to Anthropic API

8. Execute:
   POST https://api.anthropic.com/v1/messages
   Headers: Authorization: Bearer {oauth_token}
   Body: Anthropic format with thinking

9. Handle response:
   - Parse thinking blocks
   - Return in OpenAI format

RESULT: ✓ User gets Claude response with reasoning
```

### Example 2: User Requests Gemini (Prefer API, Fallback to Browser)

```
REQUEST:
POST /v1/chat/completions
{
  "model": "gemini-2-flash",
  "messages": [...]
}

ROUTER FLOW:

1. Parse: model = "gemini-2-flash", needs_nothing_special

2. Lookup model_registry:
   Sources:
   - gemini-api-key (priority 1)
   - gemini-aistudio (priority 2)
   - gemini-webai (priority 3)

3. Filter by availability:
   - gemini-api-key: UP ✓
   - gemini-aistudio: DOWN ✗ (Playwright/Camoufox crashed)
   - gemini-webai: UP ✓

4. Remaining: [gemini-api-key, gemini-webai]

5. Select executor (by priority):
   - Select: gemini-api-key
   - Type: "direct"
   - Class: "GeminiAPIExecutor"
   - Auth: Load API key from FileStore
   - Check if valid: Yes

6. Execute:
   POST https://generativelanguage.googleapis.com/v1beta/...
   Headers: x-goog-api-key: {api_key}

7. Response: ✓ Successful

RESULT: Fast response from official API

---

SCENARIO: gemini-api-key goes DOWN

Later request for same model:

1-4. Same as above
5. Select executor:
   - Try: gemini-api-key → FAILURE (503)
   - Fallback to next: gemini-webai

6. Execute (proxy):
   POST http://localhost:8406/v1/chat/completions
   (HTTPProxyExecutor passes through to WebAI service)

7. WebAI service handles:
   - Extract cookies from config.conf
   - Call Gemini web API
   - Return response

8. Response: ✓ Successful (slightly slower but works)

RESULT: Graceful fallback when primary fails
```

### Example 3: User Requests GPT (Multiple Options Available)

```
REQUEST:
POST /v1/chat/completions
{
  "model": "gpt-4-turbo",
  "messages": [...]
}

ROUTER FLOW:

1. Parse: model = "gpt-4-turbo"

2. Lookup model_registry:
   Sources:
   - codex-api-key (priority 1)
   - codex-oauth (priority 2)
   - ctonew (priority 3)

3. Filter by availability:
   - codex-api-key: UP (load: 0.8/1.0) ✓
   - codex-oauth: UP (load: 0.2/1.0) ✓
   - ctonew: UP (load: 0.1/1.0) ✓

4. All available, no capability filtering needed

5. Routing strategy: "load-balance" (least-busy)
   Best: ctonew (0.1 load)

6. Select executor:
   - Type: "proxy"
   - Class: "HTTPProxyExecutor"
   - Service: "http://localhost:8405" (ctonew)

7. Execute (proxy):
   POST http://localhost:8405/v1/chat/completions
   Body: Pass through as-is

8. ctonew service processes:
   - Extract Clerk JWT from stored cookie
   - Exchange for fresh token via Clerk
   - Call enginelabs API
   - Return response

9. Router translates response (if needed)

RESULT: Request routed to least-busy service
        Automatic load balancing
```

---

## Part 9: Finalized Architecture Summary

### The Clean Design

```
┌─────────────────────────────────────────────────────────────────┐
│  CLIProxyAPI Router (Hybrid, ~2500 lines of code)               │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  COMPOSITION:                                                    │
│  ├─ Access Control Layer (API key validation)                   │
│  ├─ Request Parsing (model, params, capabilities)               │
│  ├─ Model Resolution Engine (model → providers lookup)          │
│  ├─ Provider Filtering (availability, capabilities, health)     │
│  ├─ Routing Engine (select executor, load balance)              │
│  ├─ Request Normalizer (convert to provider format)             │
│  ├─ Executor Dispatcher (direct vs proxy)                       │
│  ├─ Response Handler (streaming, error handling)                │
│  ├─ Response Translator (normalize to OpenAI format)            │
│  ├─ Auth Manager (token storage + auto-refresh for direct)      │
│  └─ Monitoring (health checks, metrics, logging)                │
│                                                                   │
│  DIRECT EXECUTORS (in-process):                                 │
│  ├─ ClaudeAPIExecutor (api key)                                │
│  ├─ ClaudeOAuthExecutor (OAuth - LLMux) [NEW]                  │
│  ├─ CodexAPIExecutor (api key)                                 │
│  ├─ CodexOAuthExecutor (OAuth - LLMux) [NEW]                   │
│  ├─ GeminiAPIExecutor (api key)                                │
│  ├─ QwenExecutor (device flow)                                 │
│  ├─ iFlowExecutor (OAuth)                                      │
│  ├─ AntigravityExecutor (OAuth)                                │
│  ├─ OpenAICompatExecutor (custom providers)                    │
│  └─ HTTPProxyExecutor (routes to services) [NEW]               │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
                              │
                ┌─────────────┼─────────────┐
                │             │             │
                ▼             ▼             ▼
        ┌──────────────┐  ┌─────────────┐  ┌─────────────┐
        │Direct APIs   │  │AIStudio     │  │Ctonew       │
        │(Fast)        │  │Service      │  │Service      │
        │              │  │(Port 8404)  │  │(Port 8405)  │
        │Official:     │  │             │  │             │
        │- Anthropic   │  │Playwright + │  │Stateless    │
        │- OpenAI      │  │Camoufox     │  │JWT Broker   │
        │- Google      │  │             │  │             │
        │- Alibaba     │  │Gemini only  │  │Claude + GPT │
        │              │  │             │  │             │
        │<1ms latency  │  │2-5s latency │  │1-2s latency │
        └──────────────┘  └─────────────┘  └─────────────┘
                                │
                        (Optional:)
                        ┌─────────────┐
                        │WebAI Service│
                        │(Port 8406)  │
                        │             │
                        │Cookies +    │
                        │gpt4free     │
                        │             │
                        │Gemini +     │
                        │50+ models   │
                        └─────────────┘
```

### Key Characteristics

```
ARCHITECTURE CHARACTERISTICS:

Flexibility:
✓ Each provider (direct) is independent
✓ Each service (proxy) is independent
✓ Can enable/disable any provider via config
✓ Can change routing strategy without code change
✓ Can add new services without touching router
✓ Can add new direct APIs by implementing executor interface

Scalability:
✓ Direct APIs: Scale with router (shared resource)
✓ Services: Scale independently (separate processes)
✓ Can run on different machines
✓ Can use Kubernetes/Docker for orchestration
✓ Health checks enable automatic failover

Maintainability:
✓ Router: ~2500 lines (manageable)
✓ Each executor: ~200-400 lines (simple)
✓ Each service: Independent codebase (not our concern)
✓ Clear interfaces between components
✓ Easy to understand request flow
✓ Isolated failures (one service down ≠ router down)

Robustness:
✓ Multiple providers per model = automatic fallback
✓ Health checks detect failures immediately
✓ Load balancing distributes traffic
✓ Retry logic handles transient failures
✓ Error translation provides consistent responses

Performance:
✓ Direct APIs: Fast (no network hops)
✓ Services: 1-2ms local network latency
✓ Streaming support for all providers
✓ No blocking I/O
✓ Async request handling
```

---

## Part 10: Implementation Phases

```
PHASE 1: Core Infrastructure (1 week)
├─ Model Resolution Engine
├─ Provider Filtering Logic
├─ Routing Strategy
├─ HTTPProxyExecutor
├─ Service Health Checks
└─ Basic Config System

PHASE 2: Direct Auth Merges (2 weeks)
├─ ClaudeOAuthExecutor (LLMux)
├─ CodexOAuthExecutor (LLMux)
├─ Anthropic OAuth integration
├─ OpenAI OAuth integration
├─ Reasoning budget support
└─ Testing

PHASE 3: Service Integration (1 week)
├─ AIStudioProxyAPI integration
├─ ctonew-proxy integration
├─ WebAI-to-API integration (optional)
├─ Service discovery config
└─ Fallback logic

PHASE 4: Testing & Optimization (1 week)
├─ End-to-end tests (all providers)
├─ Load testing
├─ Failover testing
├─ Documentation
└─ Performance optimization

TOTAL: 5 weeks (or 3 weeks if skip WebAI)

EFFORT:
- Phase 1: 5 days (medium difficulty)
- Phase 2: 10 days (medium difficulty - mostly OAuth setup)
- Phase 3: 5 days (easy - mostly config)
- Phase 4: 5 days (easy - testing)
```

This is the recommended **Option C: Hybrid Architecture** - the sweet spot between simplicity and capability!

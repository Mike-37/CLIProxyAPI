# CLIProxyAPI - Final Comprehensive Specification

> **Version**: 2.0 (Optimized Architecture)
> **Date**: 2025-11-22
> **Status**: Design Phase - Ready for Implementation

---

## Table of Contents

1. [Project Concept & Intentions](#1-project-concept--intentions)
2. [User Workflows](#2-user-workflows)
3. [Final Architecture Design](#3-final-architecture-design)
4. [Implementation Plan](#4-implementation-plan)
5. [Testing & Validation Strategy](#5-testing--validation-strategy)
6. [Success Metrics](#6-success-metrics)

---

# 1. PROJECT CONCEPT & INTENTIONS

## 1.1 The Problem We're Solving

### Current Pain Points

**Problem 1: Fragmented AI Provider Access**
- Developers need to integrate multiple AI providers (OpenAI, Anthropic, Google)
- Each has different APIs, authentication methods, and response formats
- Managing multiple API keys, billing accounts, and rate limits is complex

**Problem 2: Access Barriers**
- Some providers require paid subscriptions (ChatGPT Plus, Claude Pro)
- API access may be more expensive than web UI subscriptions
- Users already paying for web access can't leverage it programmatically

**Problem 3: Deployment Complexity**
- Existing solutions scattered across multiple repositories
- Each provider proxy requires separate installation, configuration, deployment
- No unified management or orchestration
- Over-engineered architectures with unnecessary microservices

**Problem 4: Lack of Flexibility**
- Hard to add new providers without forking/modifying code
- No automatic failover between providers
- Difficult to use same model from multiple sources

### Our Solution

**CLIProxyAPI: A unified, OpenAI-compatible router for multiple AI providers**

✅ **Single Interface**: One API endpoint, OpenAI format for all providers
✅ **Multiple Auth Methods**: API keys, OAuth, browser automation, cookies
✅ **Smart Routing**: Automatic provider selection based on model name
✅ **Simple Deployment**: One repository, one install script, 2-3 processes
✅ **Extensible**: Easy to add new providers via executors
✅ **Optimized**: In-process execution for simple providers, services only when needed

## 1.2 Target Users

### Primary Users

1. **Application Developers**
   - Building AI-powered applications
   - Want unified interface across providers
   - Need reliable failover and load balancing

2. **Individual Power Users**
   - Already have ChatGPT Plus / Claude Pro subscriptions
   - Want programmatic access without separate API billing
   - Need local tools integration (Claude Code, Continue.dev, etc.)

3. **Research Teams**
   - Comparing outputs across models
   - Need access to latest models quickly
   - Want flexible provider switching

4. **Self-Hosters**
   - Run own infrastructure
   - Want control over authentication and routing
   - Need offline-capable solution

### Secondary Users

5. **Extension Developers**
   - Building custom providers
   - Need clear extension points
   - Want minimal code to add new integrations

## 1.3 Core Principles

### 1. Simplicity Over Complexity
- **Port simple providers to Go** (in-process, fast)
- **Use services only when necessary** (browser automation, complex logic)
- **Minimize process count** (2-3 instead of 5+)
- **One command to start** everything

### 2. Single Source of Truth
- **One repository** contains all code
- **Submodules for reference** to original repos
- **Unified configuration** in one YAML file
- **Centralized authentication** storage

### 3. Developer Experience First
- **Quick setup**: < 30 minutes from clone to first request
- **Clear errors**: Actionable error messages
- **Easy debugging**: Structured logs, health checks
- **Good defaults**: Works out-of-box for common cases

### 4. Production Ready
- **Reliable failover**: Automatic provider switching on failure
- **Health monitoring**: Continuous service health checks
- **Graceful degradation**: Core features work even if services fail
- **Observable**: Comprehensive logging and metrics

### 5. Extensible Architecture
- **Plugin system**: Add providers without modifying core
- **Clear interfaces**: Executor pattern for provider implementations
- **Multiple execution modes**: Direct (Go), WebSocket (browser), HTTP (proxies)
- **Configuration-driven**: Enable/disable features via config

## 1.4 Success Criteria

### Must Have (MVP)
✅ Single OpenAI-compatible endpoint accepting all provider models
✅ LLMux providers (Claude Pro, ChatGPT Plus) working via OAuth
✅ ctonew provider working via Clerk JWT
✅ AIstudio provider working via browser automation
✅ Automatic model → provider routing
✅ Authentication management endpoints
✅ Health checks for all services
✅ One-command install and start
✅ < 3 processes running

### Should Have (V1.0)
✅ Automatic failover between providers
✅ WebAI provider integration (optional)
✅ Request/response logging
✅ Service auto-restart on failure
✅ Docker support
✅ Comprehensive documentation

### Nice to Have (Future)
⚠️ Token usage tracking
⚠️ Rate limiting per provider
⚠️ Admin UI for management
⚠️ Multiple auth storage backends (Postgres, Git)
⚠️ Load balancing across multiple accounts

---

# 2. USER WORKFLOWS

## 2.1 First-Time Setup Workflow

### Goal
User goes from "git clone" to "making API requests" in < 30 minutes.

### Steps

```bash
# Step 1: Clone repository (includes all provider code)
git clone --recursive https://github.com/Mike-37/CLIProxyAPI.git
cd CLIProxyAPI

# Step 2: Run install script
./scripts/install.sh
# This installs:
# - Go dependencies (for router)
# - Python + Playwright + Camoufox (for AIstudio)
# - Optional: Python + gpt4free (for WebAI if enabled)

# Step 3: Configure
cp config.example.yaml config.yaml
# Edit config.yaml:
# - Enable desired providers
# - Set browser preferences
# - Configure model routing

# Step 4: Start services
./scripts/start.sh
# Output:
# ✓ Router started (PID: 12345)
# ✓ AIstudio service started (PID: 12346)
# ✓ All services healthy

# Step 5: Authenticate providers (one-time per provider)
# For LLMux Claude:
curl http://localhost:8317/v1/auth/llmux/claude/login
# Browser opens → User logs into Claude → Tokens saved

# For ctonew:
# User provides Clerk JWT cookie via config or API

# For AIstudio:
curl -X POST http://localhost:8317/v1/auth/aistudio/login
# Visible browser opens → User logs into Google → Session saved

# Step 6: Make requests!
curl http://localhost:8317/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-sonnet-4-5-20250929",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

### Expected Outcome
✅ User can make requests to multiple providers through one endpoint
✅ Authentication persists across restarts
✅ Clear error messages if something fails
✅ All services healthy and responsive

### Failure Cases & Recovery

| Issue | Symptom | Recovery |
|-------|---------|----------|
| Missing dependencies | Install script fails | Check logs, install manually |
| Port already in use | Router won't start | Change port in config |
| Browser won't open (AIstudio) | Auth fails | Check display, try headless=false |
| OAuth fails (LLMux) | Redirect error | Check network, firewall |

## 2.2 Daily Usage Workflow

### Goal
User starts system and makes requests effortlessly.

### Steps

```bash
# Morning: Start services
./scripts/start.sh
# All services start automatically (router + enabled providers)

# Check status
./scripts/status.sh
# Output:
# ✓ Router: running (PID: 12345)
# ✓ AIstudio: running (PID: 12346)
# ✓ All providers: healthy

# Make requests all day
# The router handles everything:
# - Provider selection
# - Authentication (auto-refresh tokens)
# - Failover if provider fails
# - Response normalization

# Evening: Stop services
./scripts/stop.sh
```

### Expected Outcome
✅ Single command to start
✅ No manual token refreshes
✅ Automatic failover on errors
✅ Consistent API regardless of provider

## 2.3 Provider Integration Workflows

### Workflow A: OAuth Providers (LLMux Claude/ChatGPT, Codex)

**Type**: In-process (Direct executor)
**Authentication**: OAuth 2.0 with PKCE
**State**: Stateless (token-based)

```
┌─────────────────────────────────────────────────┐
│ 1. User Initiates Auth                         │
│    GET /v1/auth/llmux/claude/login              │
└────────────┬────────────────────────────────────┘
             ↓
┌─────────────────────────────────────────────────┐
│ 2. Router Starts OAuth Flow                    │
│    - Generate PKCE challenge                    │
│    - Open browser to provider OAuth page        │
│    - Start local callback server                │
└────────────┬────────────────────────────────────┘
             ↓
┌─────────────────────────────────────────────────┐
│ 3. User Authorizes in Browser                  │
│    - Logs into Claude/ChatGPT                   │
│    - Approves access                            │
└────────────┬────────────────────────────────────┘
             ↓
┌─────────────────────────────────────────────────┐
│ 4. Router Receives Callback                    │
│    - Exchange code for tokens                   │
│    - Save tokens to ~/.cli-proxy-api/           │
│    - Return success                             │
└────────────┬────────────────────────────────────┘
             ↓
┌─────────────────────────────────────────────────┐
│ 5. Runtime Token Management (Automatic)        │
│    - Check token expiry before each request     │
│    - Refresh if needed (transparent)            │
│    - Re-auth only if refresh fails              │
└─────────────────────────────────────────────────┘
```

**Request Flow**:
```
Client Request
    ↓
Router: Parse model → Select provider → LLMuxClaudeExecutor
    ↓
LLMuxClaudeExecutor:
    1. Load tokens from ~/.cli-proxy-api/llmux-claude-user@example.com.json
    2. Check expiry: if expired → refresh_token flow
    3. Make HTTPS request to api.anthropic.com
    4. Return response
    ↓
Client receives OpenAI-formatted response
```

**Latency**: ~100-200ms
**Processes**: 1 (router only)

---

### Workflow B: Browser Automation (AIstudio)

**Type**: External service (WebSocket relay)
**Authentication**: Google session cookies
**State**: Stateful (browser session pool)

```
┌─────────────────────────────────────────────────┐
│ 1. User Initiates Auth                         │
│    POST /v1/auth/aistudio/login                 │
└────────────┬────────────────────────────────────┘
             ↓
┌─────────────────────────────────────────────────┐
│ 2. Router Sends WS Message to AIstudio Service │
│    {                                            │
│      "type": "auth_request",                    │
│      "profile": "default"                       │
│    }                                            │
└────────────┬────────────────────────────────────┘
             ↓
┌─────────────────────────────────────────────────┐
│ 3. AIstudio Service Launches Visible Browser   │
│    - Opens ai.studio.google.com                 │
│    - User manually logs in                      │
│    - User solves any CAPTCHAs                   │
└────────────┬────────────────────────────────────┘
             ↓
┌─────────────────────────────────────────────────┐
│ 4. AIstudio Service Extracts & Saves Session   │
│    - Cookies from browser                       │
│    - localStorage tokens                        │
│    - Save to auth_profiles/default.json         │
│    - Send success WS message to router          │
└────────────┬────────────────────────────────────┘
             ↓
┌─────────────────────────────────────────────────┐
│ 5. Browser Stays Alive in Pool                 │
│    - Reused for subsequent requests             │
│    - Auto-refreshed on idle timeout             │
│    - Session restored from saved profile        │
└─────────────────────────────────────────────────┘
```

**Request Flow**:
```
Client Request (model: "gemini-2-flash-aistudio")
    ↓
Router: Parse model → Select provider → AIStudioExecutor
    ↓
AIStudioExecutor:
    1. Send WS message to AIstudio service:
       {
         "type": "http_request",
         "method": "POST",
         "url": "https://ai.studio.google.com/v1beta/generateContent",
         "body": { ... }
       }
    ↓
AIstudio Service:
    1. Receive WS message
    2. Get browser from pool (or create new)
    3. Restore session (cookies + localStorage)
    4. Navigate to URL
    5. Fill form / click buttons
    6. Stream response via DOM observation
    7. Send WS messages back:
       {"type": "stream_chunk", "data": "token"}
    ↓
Router:
    1. Receive WS stream chunks
    2. Forward as Server-Sent Events (SSE) to client
    ↓
Client receives streaming response
```

**Latency**: ~1-2s (browser overhead)
**Processes**: 2 (router + aistudio service)

---

### Workflow C: Stateless Proxy (ctonew - Ported to Go)

**Type**: In-process (Direct executor)
**Authentication**: Clerk JWT cookie
**State**: Stateless (per-request credential extraction)

```
┌─────────────────────────────────────────────────┐
│ 1. User Provides Clerk JWT Cookie              │
│    Via config.yaml or API:                      │
│    POST /v1/auth/ctonew                         │
│    { "clerk_jwt_cookie": "eyJ..." }             │
└────────────┬────────────────────────────────────┘
             ↓
┌─────────────────────────────────────────────────┐
│ 2. Router Saves to Auth Storage                │
│    ~/.cli-proxy-api/ctonew-default.json         │
└─────────────────────────────────────────────────┘
```

**Request Flow**:
```
Client Request (model: "ctonew-claude-sonnet")
    ↓
Router: Parse model → Select provider → CtonewExecutor
    ↓
CtonewExecutor (In-process Go):
    1. Load Clerk JWT from ~/.cli-proxy-api/ctonew-default.json
    2. Extract rotating_token from JWT payload:
       parts := strings.Split(jwt, ".")
       payload := base64Decode(parts[1])
       claims := json.Unmarshal(payload)
       rotatingToken := claims["rotating_token"]

    3. Exchange rotating_token for new JWT:
       POST https://clerk.com/v1/tokens/create
       { "rotating_token": "..." }
       Response: { "jwt": "new_jwt_..." }

    4. Call EngineLabs API:
       POST https://api.enginelabs.ai/v1/chat/completions
       Authorization: Bearer new_jwt_...
       { "model": "claude-sonnet", "messages": [...] }

    5. Stream response
    ↓
Client receives OpenAI-formatted streaming response
```

**Latency**: ~200-300ms (Clerk exchange + API call)
**Processes**: 1 (router only)

**Key Optimization**: No separate service needed! All logic ported to Go.

---

### Workflow D: HTTP Proxy (WebAI - Optional)

**Type**: External service (HTTP proxy)
**Authentication**: Browser cookies + gpt4free fallback
**State**: Stateless (per-request cookie injection)

```
┌─────────────────────────────────────────────────┐
│ 1. User Provides Cookies                       │
│    Manual: Extracted from browser               │
│    Auto: WebAI service can extract via browser  │
└────────────┬────────────────────────────────────┘
             ↓
┌─────────────────────────────────────────────────┐
│ 2. Cookies Saved to WebAI Config               │
│    providers/webai/cookies.json                 │
└─────────────────────────────────────────────────┘
```

**Request Flow**:
```
Client Request (model: "gemini-webai")
    ↓
Router: Parse model → Select provider → HTTPProxyExecutor
    ↓
HTTPProxyExecutor:
    1. Forward HTTP request to WebAI service:
       POST http://localhost:8406/v1/chat/completions
       { "model": "gemini-webai", "messages": [...] }
    ↓
WebAI Service (Python):
    1. Receive HTTP request
    2. Extract cookies from config
    3. Try Gemini web API:
       - Inject cookies into request
       - Call https://gemini.google.com/app
       - If success → return response
    4. If fails → fallback to gpt4free:
       - Try 50+ free providers
       - Return first successful response
    ↓
Router:
    1. Receive HTTP response
    2. Forward to client
    ↓
Client receives response
```

**Latency**: ~300-500ms
**Processes**: 3 (router + aistudio + webai)

**Why Keep as Service?**
- gpt4free is complex Python library (100k+ lines)
- Not worth porting to Go
- Optional provider (disabled by default)

---

## 2.4 Adding New Provider Workflow

### Goal
Developer adds support for new AI provider without breaking existing setup.

### Scenario: Adding "Poe" Provider (Example)

**Step 1: Determine Integration Type**

```
Questions to answer:
1. Authentication method?
   → Cookie-based (browser session)

2. API structure?
   → WebSocket streaming

3. Complexity?
   → Medium (need to handle WS + cookies)

Decision: Create as external WebSocket service (similar to AIstudio)
```

**Step 2: Implement Provider Service**

```bash
# Create provider directory
mkdir -p providers/poe

# Create service files
touch providers/poe/main.py
touch providers/poe/ws_client.py
touch providers/poe/poe_client.py
touch providers/poe/requirements.txt
```

**Step 3: Update Configuration**

```yaml
# config.yaml
providers:
  poe:
    enabled: true
    auto_start: true

    service:
      command: "python providers/poe/main.py"
      cwd: "providers/poe"

    health_check:
      websocket_url: "ws://localhost:8317/v1/ws"
      interval: 30

    models:
      - "poe-gpt4"
      - "poe-claude"

models:
  routing:
    - pattern: "^poe-"
      providers: ["poe"]
```

**Step 4: Register Executor**

```go
// internal/runtime/executor/poe_executor.go
package executor

type PoeExecutor struct {
    wsManager *wsrelay.Manager
}

func (e *PoeExecutor) Execute(ctx context.Context, auth *cliproxyauth.Auth,
    req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (cliproxyexecutor.Response, error) {

    // Send WebSocket message to Poe service
    // (similar to AIStudioExecutor)

    return resp, nil
}
```

**Step 5: Test & Deploy**

```bash
# Start only the new service for testing
./scripts/dev/start-service.sh poe

# Test endpoint
curl http://localhost:8317/v1/chat/completions \
  -d '{"model": "poe-gpt4", "messages": [...]}'

# If works, restart all services
./scripts/restart.sh
```

### Expected Outcome
✅ New provider available immediately
✅ No changes to existing providers
✅ Isolated failures (if Poe fails, others still work)
✅ Clear documentation for next provider

---

## 2.5 Development & Debugging Workflow

### Goal
Developer can easily debug issues and contribute changes.

### Debugging Process Flow

```bash
# 1. Start router only (to test in isolation)
./scripts/dev/start-router.sh

# 2. Check logs in real-time
./scripts/logs.sh router
# or
tail -f logs/router.log

# 3. Test specific provider
./scripts/dev/test-provider.sh llmux-claude

# 4. Start service manually with debug logging
cd providers/aistudio
DEBUG=1 python main.py

# 5. Check service health
curl http://localhost:8317/v1/health
# Response:
# {
#   "router": "healthy",
#   "providers": {
#     "llmux-claude": "healthy",
#     "aistudio": "degraded",  // ← Issue here!
#     "ctonew": "healthy"
#   }
# }

# 6. Restart failed service
./scripts/restart.sh aistudio

# 7. View all service statuses
./scripts/status.sh
```

### Common Debugging Scenarios

#### Scenario 1: Provider Not Responding

```bash
# Check if service is running
./scripts/status.sh
# Output: AIstudio: not running

# Check logs for crash reason
./scripts/logs.sh aistudio
# Error: Browser binary not found

# Fix: Reinstall dependencies
./scripts/install/install-aistudio.sh

# Restart
./scripts/restart.sh aistudio
```

#### Scenario 2: Authentication Failing

```bash
# Check auth files
ls -la ~/.cli-proxy-api/
# Output: llmux-claude-user@example.com.json (expired tokens)

# Re-authenticate
curl http://localhost:8317/v1/auth/llmux/claude/login
# Browser opens, user logs in again

# Test request
curl http://localhost:8317/v1/chat/completions \
  -d '{"model": "claude-sonnet-4-5", "messages": [...]}'
# Success!
```

#### Scenario 3: Model Routing Not Working

```bash
# Check model registry
curl http://localhost:8317/v1beta/models
# Lists all available models and their providers

# Test specific model
curl http://localhost:8317/v1/chat/completions \
  -d '{"model": "unknown-model", "messages": [...]}'
# Error: No provider registered for model pattern

# Fix: Update config.yaml routing
# Add pattern match for new model
```

### Expected Outcome
✅ Detailed logging at multiple levels (debug, info, warn, error)
✅ Service health checks show exactly what's failing
✅ Clear error messages with suggested fixes
✅ Easy to test locally before deployment

---

# 3. FINAL ARCHITECTURE DESIGN

## 3.1 High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         CLIENTS                             │
│  (OpenAI SDK, curl, Custom Apps, Claude Code, etc.)        │
└────────────┬────────────────────────────────────────────────┘
             │
             │ HTTP POST /v1/chat/completions
             │ OpenAI-compatible format
             ↓
┌─────────────────────────────────────────────────────────────┐
│                    ROUTER (Go - Port 8317)                  │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ HTTP Server (Gin Framework)                        │   │
│  │ ├─ /v1/chat/completions (OpenAI)                   │   │
│  │ ├─ /v1/messages (Anthropic)                        │   │
│  │ ├─ /v1beta/models (Model registry)                 │   │
│  │ ├─ /v1/auth/* (Auth management)                    │   │
│  │ └─ /v1/ws (WebSocket for services)                 │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Model Router                                       │   │
│  │ - Parse model name                                 │   │
│  │ - Match regex patterns                             │   │
│  │ - Select provider(s)                               │   │
│  │ - Failover on error                                │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌──────────────────────────────────┬──────────────────┐   │
│  │ DIRECT EXECUTORS (In-Process)    │ RELAY EXECUTORS  │   │
│  │                                   │                  │   │
│  │ ├─ LLMuxClaudeExecutor  ✅        │ ├─ AIStudio (WS) │   │
│  │ ├─ LLMuxChatGPTExecutor ✅        │ └─ WebAI (HTTP)  │   │
│  │ ├─ CtonewExecutor       ✅        │                  │   │
│  │ ├─ ClaudeOAuthExecutor  (exist)   │                  │   │
│  │ └─ CodexOAuthExecutor   (exist)   │                  │   │
│  └──────────────────────────────────┴──────────────────┘   │
└─────────────┬────────────────────────┬──────────────────────┘
              │                        │
              │                        │ WebSocket / HTTP
              ↓                        ↓
      Direct API Calls       ┌─────────────────────┐
      (api.anthropic.com,    │  EXTERNAL SERVICES  │
       api.openai.com, etc)  │                     │
                             │  ├─ AIstudio (Py)   │
                             │  │   - Browser pool  │
                             │  │   - WS client     │
                             │  │                   │
                             │  └─ WebAI (Py)      │
                             │      - HTTP server   │
                             │      - gpt4free      │
                             └─────────────────────┘
```

## 3.2 Process Architecture

### Production Mode (./scripts/start.sh)

```
┌─────────────────────────────────────────────┐
│ Process Manager (start.sh)                  │
│ ├─ Starts all enabled services              │
│ ├─ Health checks every 30s                  │
│ ├─ Auto-restart on failure                  │
│ └─ Logs to ./logs/*.log                     │
└────────┬────────────────────────────────────┘
         │
         ├─── PROCESS 1: Router (PID: 12345)
         │    └─ Binary: ./bin/cli-proxy-api
         │    └─ Port: 8317
         │    └─ Includes: LLMux, ctonew executors
         │
         ├─── PROCESS 2: AIstudio Service (PID: 12346)
         │    └─ Command: python providers/aistudio/main.py
         │    └─ WebSocket: ws://localhost:8317/v1/ws
         │    └─ Browser Pool: 1-3 instances
         │
         └─── PROCESS 3: WebAI Service (PID: 12347) [OPTIONAL]
              └─ Command: python providers/webai/main.py
              └─ HTTP: http://localhost:8406
              └─ Fallback: gpt4free
```

**Total Processes**: 2-3 (down from 5+ in over-engineered version)

### Development Mode (./scripts/start.sh --dev)

```
Terminal 1: Router
$ ./scripts/dev/start-router.sh
[INFO] Router started on :8317
[INFO] Loaded providers: llmux-claude, llmux-chatgpt, ctonew
[INFO] WebSocket relay ready at /v1/ws

Terminal 2: AIstudio Service
$ cd providers/aistudio && DEBUG=1 python main.py
[DEBUG] Connecting to router WebSocket...
[INFO] Connected successfully
[DEBUG] Browser pool initialized

Terminal 3: Test Requests
$ curl http://localhost:8317/v1/chat/completions -d '...'
```

## 3.3 Directory Structure (Final)

```
CLIProxyAPI/
│
├── README.md
├── LICENSE
├── .gitignore
├── .gitmodules                    # Submodules for reference repos
│
├── config.example.yaml
├── config.yaml                    # gitignored
│
├── go.mod
├── go.sum
│
├── cmd/
│   └── server/
│       └── main.go               # Router entry point
│
├── internal/                     # Router implementation (Go)
│   ├── api/
│   │   ├── server.go
│   │   ├── handlers/
│   │   │   ├── openai/          # /v1/chat/completions
│   │   │   ├── claude/          # /v1/messages
│   │   │   ├── gemini/          # /v1beta/models
│   │   │   └── management/
│   │   │       ├── auth_management.go
│   │   │       ├── provider_management.go
│   │   │       ├── health.go
│   │   │       └── service_control.go
│   │   └── modules/amp/
│   │
│   ├── runtime/executor/
│   │   ├── executor.go          # Base executor interface
│   │   │
│   │   ├── ---- EXISTING (Direct) ----
│   │   ├── claude_executor.go
│   │   ├── codex_executor.go
│   │   ├── gemini_executor.go
│   │   │
│   │   ├── ---- NEW: PORTED (Direct) ----
│   │   ├── llmux_claude_executor.go     # LLMux Claude OAuth
│   │   ├── llmux_chatgpt_executor.go    # LLMux ChatGPT OAuth
│   │   ├── ctonew_executor.go           # ctonew (from Deno)
│   │   │
│   │   ├── ---- NEW: RELAY ----
│   │   ├── aistudio_executor.go         # WebSocket relay (exists)
│   │   └── http_proxy_executor.go       # Generic HTTP proxy
│   │
│   ├── auth/
│   │   ├── claude/              # Existing
│   │   ├── codex/               # Existing
│   │   ├── gemini/              # Existing
│   │   ├── llmux/               # NEW: LLMux OAuth
│   │   │   ├── claude_pro_oauth.go
│   │   │   ├── chatgpt_plus_oauth.go
│   │   │   └── oauth_server.go
│   │   └── ctonew/              # NEW: ctonew auth
│   │       ├── clerk_jwt.go
│   │       ├── clerk_client.go
│   │       └── token_exchange.go
│   │
│   ├── wsrelay/                 # Existing WebSocket relay
│   ├── config/
│   ├── logging/
│   └── browser/
│
├── sdk/
│   ├── cliproxy/
│   ├── auth/
│   └── translator/
│
├── providers/
│   ├── _reference/              # Git submodules (read-only)
│   │   ├── aistudio/           # Original repo (reference)
│   │   ├── llmux/              # Original repo (ported to Go)
│   │   ├── webai/              # Original repo (reference)
│   │   └── ctonew/             # Original repo (ported to Go)
│   │
│   ├── aistudio/               # Active service
│   │   ├── main.py
│   │   ├── requirements.txt
│   │   ├── ws_client.py
│   │   ├── browser_manager.py
│   │   ├── session_manager.py
│   │   ├── gemini_client.py
│   │   ├── streaming.py
│   │   ├── auth_profiles/      # gitignored
│   │   └── config.yaml
│   │
│   └── webai/                  # Active service (optional)
│       ├── main.py
│       ├── requirements.txt
│       ├── http_server.py
│       ├── cookie_manager.py
│       ├── gemini_web_client.py
│       ├── gpt4free_client.py
│       └── config.yaml
│
├── scripts/
│   ├── install.sh              # Main installer
│   ├── install/
│   │   ├── install-base.sh
│   │   ├── install-aistudio.sh
│   │   └── install-webai.sh
│   ├── start.sh
│   ├── stop.sh
│   ├── restart.sh
│   ├── status.sh
│   ├── logs.sh
│   └── dev/
│       ├── start-router.sh
│       ├── start-aistudio.sh
│       ├── test-provider.sh
│       └── build.sh
│
├── docs/
│   ├── README.md
│   ├── SETUP.md
│   ├── ARCHITECTURE.md
│   ├── PROVIDERS.md
│   ├── API.md
│   ├── TROUBLESHOOTING.md
│   ├── CONTRIBUTING.md
│   └── FINAL_SPECIFICATION.md  # This file
│
├── examples/
│   ├── custom-provider/
│   └── clients/
│       ├── curl.sh
│       ├── python/example.py
│       └── javascript/example.js
│
├── tests/
│   ├── unit/
│   ├── integration/
│   └── e2e/
│
├── .github/workflows/
├── docker/
│
├── logs/                       # gitignored
├── pids/                       # gitignored
└── .cli-proxy-api/            # gitignored
```

## 3.4 Configuration Architecture (Final)

### Master Configuration: config.yaml

```yaml
# ============================================
# CLIProxyAPI Configuration
# ============================================

# Router settings
server:
  port: 8317
  host: "0.0.0.0"
  debug: false

# Authentication storage
auth:
  dir: "~/.cli-proxy-api"
  store: "file"           # Options: file, postgres, git

# WebSocket relay (for services like AIstudio)
websocket:
  enabled: true
  path: "/v1/ws"
  timeout: 300            # 5 minutes
  ping_interval: 30       # seconds

# Logging
logging:
  level: "info"           # debug, info, warn, error
  dir: "./logs"
  max_size_mb: 100
  max_backups: 5

# ============================================
# PROVIDERS
# ============================================

providers:
  # --------------------------------------------------
  # IN-PROCESS PROVIDERS (No external services)
  # --------------------------------------------------

  llmux:
    claude_pro:
      enabled: true
      # OAuth tokens managed automatically

    chatgpt_plus:
      enabled: true
      # OAuth tokens managed automatically

  ctonew:
    enabled: true
    # Clerk JWT cookie provided by user

  # --------------------------------------------------
  # EXTERNAL SERVICE PROVIDERS
  # --------------------------------------------------

  aistudio:
    enabled: true
    auto_start: true

    service:
      command: "python providers/aistudio/main.py"
      cwd: "providers/aistudio"
      env:
        ROUTER_WS_URL: "ws://localhost:8317/v1/ws"
        PROVIDER_NAME: "aistudio"

    health_check:
      enabled: true
      interval: 30
      timeout: 10

    browser:
      type: "camoufox"      # or "playwright"
      headless: true
      idle_timeout: 1800    # 30 min
      max_instances: 3

  webai:
    enabled: false          # Optional
    auto_start: false

    service:
      command: "python providers/webai/main.py"
      cwd: "providers/webai"
      port: 8406

    health_check:
      enabled: true
      url: "http://localhost:8406/health"
      interval: 30

    proxy:
      endpoint: "http://localhost:8406"
      timeout: 60

# ============================================
# MODEL ROUTING
# ============================================

models:
  # Model name pattern → Providers (in order of priority)
  routing:
    # Claude models
    - pattern: "^claude-sonnet-4-5"
      providers:
        - "llmux-claude"      # Try LLMux first
        - "claude-oauth"      # Fallback to direct OAuth
        - "ctonew"            # Last resort

    - pattern: "^claude-opus-4"
      providers: ["llmux-claude", "claude-oauth"]

    # GPT models
    - pattern: "^gpt-5"
      providers:
        - "llmux-chatgpt"     # Try LLMux first
        - "ctonew"            # Fallback

    # ctonew-specific
    - pattern: "^ctonew-"
      providers: ["ctonew"]

    # AIstudio models
    - pattern: "gemini-.*-aistudio$"
      providers: ["aistudio"]

    # WebAI models (optional)
    - pattern: ".*-webai$"
      providers: ["webai"]

  # Default model capabilities
  defaults:
    max_tokens: 200000
    supports_streaming: true
    supports_vision: false
    supports_reasoning: false
```

## 3.5 Data Flow Architecture

### Flow 1: Direct Executor (LLMux, ctonew)

```
┌────────────────────────────────────────────────────────┐
│ CLIENT                                                 │
└────┬───────────────────────────────────────────────────┘
     │
     │ POST /v1/chat/completions
     │ {"model": "gpt-5", "messages": [...]}
     ↓
┌────────────────────────────────────────────────────────┐
│ ROUTER - HTTP Server                                   │
│ ├─ Parse request                                       │
│ ├─ Validate auth                                       │
│ └─ Extract model: "gpt-5"                              │
└────┬───────────────────────────────────────────────────┘
     │
     ↓
┌────────────────────────────────────────────────────────┐
│ ROUTER - Model Router                                  │
│ ├─ Match pattern: "^gpt-5"                             │
│ ├─ Providers: ["llmux-chatgpt", "ctonew"]              │
│ └─ Select: "llmux-chatgpt" (priority 1)                │
└────┬───────────────────────────────────────────────────┘
     │
     ↓
┌────────────────────────────────────────────────────────┐
│ ROUTER - LLMuxChatGPTExecutor (In-Process Go)          │
│ ├─ Load tokens from ~/.cli-proxy-api/                  │
│ ├─ Check expiry → Refresh if needed                    │
│ ├─ Make HTTPS request to api.openai.com                │
│ └─ Stream response                                     │
└────┬───────────────────────────────────────────────────┘
     │
     ↓ Direct HTTPS
┌────────────────────────────────────────────────────────┐
│ UPSTREAM API (api.openai.com)                          │
└────┬───────────────────────────────────────────────────┘
     │
     │ Streaming response
     ↓
┌────────────────────────────────────────────────────────┐
│ ROUTER - Response Formatter                            │
│ └─ Convert to OpenAI SSE format                        │
└────┬───────────────────────────────────────────────────┘
     │
     ↓ SSE stream
┌────────────────────────────────────────────────────────┐
│ CLIENT                                                 │
│ data: {"choices":[{"delta":{"content":"Hello"}}]}     │
└────────────────────────────────────────────────────────┘

Latency: ~100-200ms
Processes: 1 (router only)
```

### Flow 2: WebSocket Relay (AIstudio)

```
┌────────────────────────────────────────────────────────┐
│ CLIENT                                                 │
└────┬───────────────────────────────────────────────────┘
     │
     │ POST /v1/chat/completions
     │ {"model": "gemini-2-flash-aistudio", "messages": [...]}
     ↓
┌────────────────────────────────────────────────────────┐
│ ROUTER - HTTP Server                                   │
│ └─ Extract model: "gemini-2-flash-aistudio"            │
└────┬───────────────────────────────────────────────────┘
     │
     ↓
┌────────────────────────────────────────────────────────┐
│ ROUTER - Model Router                                  │
│ ├─ Match pattern: "gemini-.*-aistudio$"                │
│ └─ Select provider: "aistudio"                         │
└────┬───────────────────────────────────────────────────┘
     │
     ↓
┌────────────────────────────────────────────────────────┐
│ ROUTER - AIStudioExecutor                              │
│ └─ Send WebSocket message to AIstudio service:         │
│    {                                                   │
│      "type": "http_request",                           │
│      "request_id": "req-123",                          │
│      "url": "https://ai.studio.google.com/...",        │
│      "method": "POST",                                 │
│      "body": {...}                                     │
│    }                                                   │
└────┬───────────────────────────────────────────────────┘
     │
     ↓ WebSocket (ws://localhost:8317/v1/ws)
┌────────────────────────────────────────────────────────┐
│ AISTUDIO SERVICE (Python)                              │
│ ├─ Receive WS message                                  │
│ ├─ Get browser from pool (or create)                   │
│ ├─ Restore session (cookies + localStorage)            │
│ ├─ Navigate to URL                                     │
│ ├─ Fill form, click send                               │
│ ├─ Observe DOM for response chunks                     │
│ └─ Send WS messages back:                              │
│    {"type": "stream_chunk", "request_id": "req-123",   │
│     "data": "Hello"}                                   │
└────┬───────────────────────────────────────────────────┘
     │
     ↓ Browser interaction
┌────────────────────────────────────────────────────────┐
│ UPSTREAM (ai.studio.google.com)                        │
└────┬───────────────────────────────────────────────────┘
     │
     ↓ WebSocket stream chunks
┌────────────────────────────────────────────────────────┐
│ ROUTER - AIStudioExecutor                              │
│ ├─ Receive WS stream chunks                            │
│ └─ Convert to OpenAI SSE format                        │
└────┬───────────────────────────────────────────────────┘
     │
     ↓ SSE stream
┌────────────────────────────────────────────────────────┐
│ CLIENT                                                 │
└────────────────────────────────────────────────────────┘

Latency: ~1-2s (browser overhead)
Processes: 2 (router + aistudio service)
```

### Flow 3: HTTP Proxy (WebAI)

```
┌────────────────────────────────────────────────────────┐
│ CLIENT                                                 │
└────┬───────────────────────────────────────────────────┘
     │
     │ POST /v1/chat/completions
     │ {"model": "gemini-webai", "messages": [...]}
     ↓
┌────────────────────────────────────────────────────────┐
│ ROUTER - HTTP Server                                   │
│ └─ Extract model: "gemini-webai"                       │
└────┬───────────────────────────────────────────────────┘
     │
     ↓
┌────────────────────────────────────────────────────────┐
│ ROUTER - Model Router                                  │
│ ├─ Match pattern: ".*-webai$"                          │
│ └─ Select provider: "webai"                            │
└────┬───────────────────────────────────────────────────┘
     │
     ↓
┌────────────────────────────────────────────────────────┐
│ ROUTER - HTTPProxyExecutor                             │
│ └─ Forward HTTP POST to http://localhost:8406          │
└────┬───────────────────────────────────────────────────┘
     │
     ↓ HTTP POST
┌────────────────────────────────────────────────────────┐
│ WEBAI SERVICE (Python)                                 │
│ ├─ Receive HTTP request                                │
│ ├─ Extract cookies from config                         │
│ ├─ Try Gemini web API                                  │
│ │  └─ If fails: fallback to gpt4free                   │
│ └─ Return HTTP response                                │
└────┬───────────────────────────────────────────────────┘
     │
     ↓ HTTP response
┌────────────────────────────────────────────────────────┐
│ ROUTER - HTTPProxyExecutor                             │
│ └─ Forward response to client                          │
└────┬───────────────────────────────────────────────────┘
     │
     ↓
┌────────────────────────────────────────────────────────┐
│ CLIENT                                                 │
└────────────────────────────────────────────────────────┘

Latency: ~300-500ms
Processes: 3 (router + aistudio + webai)
```

---

# 4. IMPLEMENTATION PLAN

## 4.1 Implementation Phases

### Phase 0: Repository Setup ✅
**Status**: Already done (repository exists)

- [x] Repository structure
- [x] Basic Go modules
- [x] Existing executors (Claude, Codex, Gemini)
- [x] WebSocket relay infrastructure

### Phase 1: Configuration & Infrastructure (Week 1)

**Goal**: Set up unified configuration and service management.

#### Tasks
1. **Update Configuration System**
   - [ ] Create `config.example.yaml` with all provider sections
   - [ ] Update `internal/config/config.go` to parse new structure
   - [ ] Add validation for provider configs
   - [ ] Support environment variable overrides

2. **Service Management Scripts**
   - [ ] `scripts/install.sh` - Main installer
   - [ ] `scripts/install/install-base.sh` - Go dependencies
   - [ ] `scripts/install/install-aistudio.sh` - Python + Playwright
   - [ ] `scripts/install/install-webai.sh` - Python + gpt4free
   - [ ] `scripts/start.sh` - Start all services
   - [ ] `scripts/stop.sh` - Stop all services
   - [ ] `scripts/status.sh` - Check service health
   - [ ] `scripts/logs.sh` - View logs

3. **Health Check System**
   - [ ] `internal/api/handlers/management/health.go`
   - [ ] Health check for router itself
   - [ ] Health check for WebSocket connections (AIstudio)
   - [ ] Health check for HTTP services (WebAI)
   - [ ] Endpoint: `GET /v1/health`

**Deliverables**:
- ✅ Unified configuration system
- ✅ Complete management scripts
- ✅ Health monitoring

**Testing**:
```bash
# Install dependencies
./scripts/install.sh

# Verify config loads
go run cmd/server/main.go --validate-config

# Test health endpoint
curl http://localhost:8317/v1/health
```

---

### Phase 2: LLMux Integration (Week 2)

**Goal**: Port LLMux OAuth providers to Go (in-process execution).

#### Tasks

##### 2.1 LLMux Claude OAuth
1. **Auth Implementation**
   - [ ] `internal/auth/llmux/claude_pro_oauth.go`
     - OAuth 2.0 with PKCE flow
     - Token storage in `~/.cli-proxy-api/`
     - Auto-refresh logic
   - [ ] `internal/auth/llmux/oauth_server.go`
     - Local callback server for OAuth redirect
     - PKCE challenge generation

2. **Executor Implementation**
   - [ ] `internal/runtime/executor/llmux_claude_executor.go`
     - Implement `cliproxyexecutor.Executor` interface
     - Load tokens from auth storage
     - Make requests to `api.anthropic.com`
     - Handle streaming responses

3. **Auth Endpoints**
   - [ ] `GET /v1/auth/llmux/claude/login` - Initiate OAuth
   - [ ] `GET /v1/auth/llmux/claude/callback` - OAuth callback
   - [ ] `GET /v1/auth/llmux/claude/status` - Check auth status
   - [ ] `DELETE /v1/auth/llmux/claude` - Logout

##### 2.2 LLMux ChatGPT OAuth
1. **Auth Implementation**
   - [ ] `internal/auth/llmux/chatgpt_plus_oauth.go`
     - OAuth flow for OpenAI
     - Token management
     - Refresh logic

2. **Executor Implementation**
   - [ ] `internal/runtime/executor/llmux_chatgpt_executor.go`
     - OpenAI API integration
     - Streaming support
     - Error handling

3. **Auth Endpoints**
   - [ ] `GET /v1/auth/llmux/chatgpt/login`
   - [ ] `GET /v1/auth/llmux/chatgpt/callback`
   - [ ] `GET /v1/auth/llmux/chatgpt/status`
   - [ ] `DELETE /v1/auth/llmux/chatgpt`

##### 2.3 Model Routing
- [ ] Update model router to handle `claude-sonnet-4-5`, `gpt-5` patterns
- [ ] Add LLMux providers to routing table
- [ ] Implement failover logic

**Deliverables**:
- ✅ LLMux Claude OAuth working
- ✅ LLMux ChatGPT OAuth working
- ✅ In-process execution (no external service)
- ✅ Auto token refresh

**Testing**:
```bash
# Test LLMux Claude auth
curl http://localhost:8317/v1/auth/llmux/claude/login
# Browser opens, user logs in

# Test request
curl http://localhost:8317/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-sonnet-4-5-20250929",
    "messages": [{"role": "user", "content": "Hello"}]
  }'

# Should work without external service!
```

---

### Phase 3: ctonew Integration (Week 3)

**Goal**: Port ctonew from Deno to Go (in-process execution).

#### Tasks

##### 3.1 Auth Implementation
1. **Clerk JWT Handling**
   - [ ] `internal/auth/ctonew/clerk_jwt.go`
     - Parse JWT structure
     - Extract `rotating_token` from payload
     - Base64 decoding

2. **Clerk API Client**
   - [ ] `internal/auth/ctonew/clerk_client.go`
     - HTTP client for Clerk API
     - Token exchange endpoint
     - Error handling

3. **Token Exchange**
   - [ ] `internal/auth/ctonew/token_exchange.go`
     - Exchange `rotating_token` for new JWT
     - Cache tokens temporarily
     - Handle expiration

##### 3.2 Executor Implementation
- [ ] `internal/runtime/executor/ctonew_executor.go`
  - Implement `cliproxyexecutor.Executor` interface
  - Load Clerk JWT from config/auth storage
  - Extract rotating_token
  - Exchange for new JWT
  - Call EngineLabs API
  - Stream responses

##### 3.3 Configuration & Auth Endpoints
- [ ] Update `config.yaml` schema for ctonew
- [ ] `POST /v1/auth/ctonew` - Save Clerk JWT cookie
- [ ] `GET /v1/auth/ctonew/status` - Check auth status
- [ ] `DELETE /v1/auth/ctonew` - Clear credentials

##### 3.4 Model Routing
- [ ] Add `ctonew-*` pattern matching
- [ ] Add as fallback for `gpt-5` and `claude-*` models

**Deliverables**:
- ✅ ctonew ported from Deno to Go
- ✅ Clerk JWT auth working
- ✅ In-process execution (no Deno service needed)

**Testing**:
```bash
# Provide Clerk JWT
curl -X POST http://localhost:8317/v1/auth/ctonew \
  -H "Content-Type: application/json" \
  -d '{"clerk_jwt_cookie": "eyJhbGc..."}'

# Test request
curl http://localhost:8317/v1/chat/completions \
  -d '{"model": "ctonew-claude-sonnet", "messages": [...]}'

# Verify no Deno process running!
ps aux | grep deno  # Should be empty
```

---

### Phase 4: AIstudio Service Integration (Week 4)

**Goal**: Integrate AIstudio as WebSocket service (browser automation).

#### Tasks

##### 4.1 Reference Submodule
- [ ] Add AIstudio as git submodule:
  ```bash
  git submodule add <original-repo-url> providers/_reference/aistudio
  ```

##### 4.2 Active Service Setup
- [ ] Copy relevant files to `providers/aistudio/`
- [ ] Create `providers/aistudio/main.py` - Service entry point
- [ ] Create `providers/aistudio/ws_client.py` - WebSocket client to router
- [ ] Create `providers/aistudio/browser_manager.py` - Browser pool
- [ ] Create `providers/aistudio/session_manager.py` - Session persistence
- [ ] Create `providers/aistudio/gemini_client.py` - Gemini interaction
- [ ] Create `providers/aistudio/requirements.txt` - Dependencies

##### 4.3 Router Integration
- [ ] Verify `internal/runtime/executor/aistudio_executor.go` exists
- [ ] Update to use new WebSocket message format
- [ ] Add auth management endpoints:
  - [ ] `POST /v1/auth/aistudio/login` - Trigger browser login
  - [ ] `GET /v1/auth/aistudio/status`
  - [ ] `DELETE /v1/auth/aistudio`

##### 4.4 Service Management
- [ ] Update `scripts/start.sh` to start AIstudio service
- [ ] Update `scripts/stop.sh` to stop AIstudio service
- [ ] Update `scripts/status.sh` to check AIstudio health
- [ ] Health check via WebSocket ping/pong

##### 4.5 Model Routing
- [ ] Add `gemini-.*-aistudio$` pattern
- [ ] Route to AIstudio executor

**Deliverables**:
- ✅ AIstudio service running
- ✅ WebSocket relay working
- ✅ Browser automation functional
- ✅ Session persistence

**Testing**:
```bash
# Start services
./scripts/start.sh

# Verify AIstudio service running
./scripts/status.sh
# Output: AIstudio: running (PID: 12346)

# Authenticate
curl -X POST http://localhost:8317/v1/auth/aistudio/login
# Visible browser opens, user logs in manually

# Test request
curl http://localhost:8317/v1/chat/completions \
  -d '{"model": "gemini-2-flash-aistudio", "messages": [...]}'
```

---

### Phase 5: WebAI Service Integration (Week 5) [OPTIONAL]

**Goal**: Integrate WebAI as HTTP service (gpt4free fallback).

#### Tasks

##### 5.1 Reference Submodule
- [ ] Add WebAI as git submodule:
  ```bash
  git submodule add <original-repo-url> providers/_reference/webai
  ```

##### 5.2 Active Service Setup
- [ ] Copy relevant files to `providers/webai/`
- [ ] Create `providers/webai/main.py` - Service entry point
- [ ] Create `providers/webai/http_server.py` - FastAPI server
- [ ] Create `providers/webai/cookie_manager.py` - Cookie handling
- [ ] Create `providers/webai/gemini_web_client.py` - Gemini web API
- [ ] Create `providers/webai/gpt4free_client.py` - gpt4free integration
- [ ] Create `providers/webai/requirements.txt` - Dependencies

##### 5.3 Router Integration
- [ ] Create `internal/runtime/executor/http_proxy_executor.go`
- [ ] Generic HTTP proxy logic
- [ ] Forward requests to `http://localhost:8406`
- [ ] Stream responses back

##### 5.4 Service Management
- [ ] Update scripts to manage WebAI service
- [ ] Health check via `GET /health`

##### 5.5 Configuration
- [ ] Add WebAI to `config.yaml` (disabled by default)
- [ ] Document how to enable it

**Deliverables**:
- ✅ WebAI service (optional)
- ✅ HTTP proxy working
- ✅ gpt4free fallback available

**Testing**:
```bash
# Enable in config
# config.yaml: providers.webai.enabled = true

# Start services
./scripts/start.sh

# Test request
curl http://localhost:8317/v1/chat/completions \
  -d '{"model": "gemini-webai", "messages": [...]}'
```

---

### Phase 6: Documentation & Examples (Week 6)

**Goal**: Comprehensive documentation for users and developers.

#### Tasks

##### 6.1 User Documentation
- [ ] `README.md` - Quick start guide
- [ ] `docs/SETUP.md` - Detailed installation
- [ ] `docs/PROVIDERS.md` - Provider documentation
- [ ] `docs/API.md` - API reference
- [ ] `docs/TROUBLESHOOTING.md` - Common issues
- [ ] `docs/ARCHITECTURE.md` - Architecture overview

##### 6.2 Developer Documentation
- [ ] `docs/CONTRIBUTING.md` - Contribution guide
- [ ] `examples/custom-provider/` - Custom provider example
- [ ] Code comments and docstrings

##### 6.3 Client Examples
- [ ] `examples/clients/curl.sh` - cURL examples
- [ ] `examples/clients/python/example.py` - Python client
- [ ] `examples/clients/javascript/example.js` - JS client

**Deliverables**:
- ✅ Complete documentation
- ✅ Working examples
- ✅ Clear troubleshooting guide

---

## 4.2 Implementation Dependencies

```
Phase 1 (Config & Infrastructure)
    ↓
    ├─→ Phase 2 (LLMux) ─┐
    ├─→ Phase 3 (ctonew) ─┤
    ├─→ Phase 4 (AIstudio)─┼─→ Phase 6 (Docs & Examples)
    └─→ Phase 5 (WebAI) ──┘

Parallel work possible:
- Phase 2, 3, 4, 5 can be done concurrently after Phase 1
- Phase 6 can start anytime, finalized at end
```

---

# 5. TESTING & VALIDATION STRATEGY

## 5.1 Testing Pyramid

```
                    ┌────────────────┐
                    │   E2E Tests    │  (10%)
                    │  (Full flows)  │
                    └────────────────┘
                ┌──────────────────────┐
                │ Integration Tests     │  (30%)
                │ (Executor + Provider) │
                └──────────────────────┘
            ┌─────────────────────────────┐
            │      Unit Tests              │  (60%)
            │ (Individual components)      │
            └─────────────────────────────┘
```

## 5.2 Unit Tests

### What to Test

#### Auth Modules
```go
// internal/auth/llmux/claude_pro_oauth_test.go
func TestClaudeProOAuth(t *testing.T) {
    // Test PKCE challenge generation
    // Test token storage
    // Test token refresh
    // Test error handling
}

// internal/auth/ctonew/clerk_jwt_test.go
func TestClerkJWTParsing(t *testing.T) {
    // Test JWT parsing
    // Test rotating_token extraction
    // Test invalid JWT handling
}
```

#### Executors
```go
// internal/runtime/executor/llmux_claude_executor_test.go
func TestLLMuxClaudeExecutor(t *testing.T) {
    // Test executor interface compliance
    // Test request translation
    // Test response normalization
    // Test streaming
    // Test error handling
}
```

#### Configuration
```go
// internal/config/config_test.go
func TestConfigParsing(t *testing.T) {
    // Test valid config
    // Test invalid config
    // Test defaults
    // Test environment variable overrides
}
```

### Running Unit Tests

```bash
# Run all unit tests
go test ./internal/...

# With coverage
go test -cover ./internal/...

# Specific package
go test ./internal/auth/llmux/
```

---

## 5.3 Integration Tests

### What to Test

#### Router + Executor Integration
```go
// tests/integration/llmux_test.go
func TestLLMuxIntegration(t *testing.T) {
    // Setup: Start router with LLMux executor
    // Test: Send request to /v1/chat/completions
    // Verify: Response format matches OpenAI
    // Verify: Tokens refreshed if needed
    // Teardown: Stop router
}
```

#### Router + Service Integration
```go
// tests/integration/aistudio_test.go
func TestAIStudioIntegration(t *testing.T) {
    // Setup: Start router + AIstudio service
    // Test: Send request with gemini-aistudio model
    // Verify: WebSocket messages sent correctly
    // Verify: Response streamed back
    // Teardown: Stop services
}
```

#### Model Routing
```go
// tests/integration/routing_test.go
func TestModelRouting(t *testing.T) {
    // Test: Request with "claude-sonnet-4-5"
    // Verify: Routed to LLMux Claude

    // Test: Request with "gemini-2-flash-aistudio"
    // Verify: Routed to AIstudio

    // Test: Request with unknown model
    // Verify: Error response
}
```

### Running Integration Tests

```bash
# Run integration tests
go test ./tests/integration/

# With verbose output
go test -v ./tests/integration/

# Specific test
go test -v ./tests/integration/ -run TestLLMuxIntegration
```

---

## 5.4 End-to-End Tests

### What to Test

#### Full User Workflows
```bash
#!/bin/bash
# tests/e2e/test_full_flow.sh

# 1. Install
./scripts/install.sh

# 2. Configure
cp config.example.yaml config.yaml

# 3. Start services
./scripts/start.sh

# 4. Wait for health
sleep 5
curl http://localhost:8317/v1/health

# 5. Test each provider
# LLMux Claude
curl http://localhost:8317/v1/chat/completions \
  -d '{"model": "claude-sonnet-4-5", "messages": [...]}'

# AIstudio
curl http://localhost:8317/v1/chat/completions \
  -d '{"model": "gemini-2-flash-aistudio", "messages": [...]}'

# 6. Stop services
./scripts/stop.sh

# 7. Verify clean shutdown
./scripts/status.sh  # Should show all stopped
```

### Running E2E Tests

```bash
# Run all E2E tests
./tests/e2e/run_all.sh

# Run specific scenario
./tests/e2e/test_full_flow.sh
```

---

## 5.5 Validation Against Intentions

### Validation Checklist

#### Intention 1: Unified Interface
- [ ] All providers accessible via `/v1/chat/completions`
- [ ] All responses in OpenAI format
- [ ] Single endpoint for all models
- [ ] Consistent error handling

**Test**:
```bash
# Test same endpoint for different providers
curl http://localhost:8317/v1/chat/completions \
  -d '{"model": "claude-sonnet-4-5", ...}'  # LLMux

curl http://localhost:8317/v1/chat/completions \
  -d '{"model": "gemini-2-flash-aistudio", ...}'  # AIstudio

curl http://localhost:8317/v1/chat/completions \
  -d '{"model": "ctonew-gpt-5", ...}'  # ctonew

# All should return same response format
```

---

#### Intention 2: Simple Deployment
- [ ] One repository contains all code
- [ ] One install command
- [ ] One start command
- [ ] < 3 processes running
- [ ] Works in < 30 minutes from clone

**Test**:
```bash
# Time the full setup
time {
  git clone --recursive https://github.com/Mike-37/CLIProxyAPI.git
  cd CLIProxyAPI
  ./scripts/install.sh
  ./scripts/start.sh
}
# Should complete in < 30 minutes

# Verify process count
ps aux | grep cli-proxy-api | wc -l  # Router
ps aux | grep "python.*aistudio" | wc -l  # AIstudio
# Total: 2 processes (or 3 if WebAI enabled)
```

---

#### Intention 3: Flexible Authentication
- [ ] OAuth working (LLMux)
- [ ] Cookie-based working (AIstudio)
- [ ] JWT-based working (ctonew)
- [ ] Auto token refresh

**Test**:
```bash
# OAuth
curl http://localhost:8317/v1/auth/llmux/claude/login
# Should open browser, save tokens

# Cookie-based
curl -X POST http://localhost:8317/v1/auth/aistudio/login
# Should open browser, save session

# JWT-based
curl -X POST http://localhost:8317/v1/auth/ctonew \
  -d '{"clerk_jwt_cookie": "..."}'
# Should save credentials

# Auto refresh (wait for token expiry, make request)
# Should transparently refresh and succeed
```

---

#### Intention 4: Smart Routing
- [ ] Model name parsing works
- [ ] Pattern matching works
- [ ] Failover works
- [ ] Priority ordering works

**Test**:
```bash
# Test routing
curl http://localhost:8317/v1beta/models
# Should list all models with their providers

# Test failover (disable primary provider)
# config.yaml: providers.llmux.claude_pro.enabled = false
curl http://localhost:8317/v1/chat/completions \
  -d '{"model": "claude-sonnet-4-5", ...}'
# Should fallback to claude-oauth or ctonew
```

---

#### Intention 5: Extensibility
- [ ] Easy to add new provider
- [ ] Clear executor interface
- [ ] Configuration-driven
- [ ] No core changes needed

**Test**:
```bash
# Add custom provider following docs
# examples/custom-provider/

# Should work without modifying router code
```

---

## 5.6 Performance Benchmarks

### Latency Targets

| Provider Type | Target Latency | Measurement |
|--------------|----------------|-------------|
| LLMux (in-process) | < 200ms | Time to first token |
| ctonew (in-process) | < 300ms | Time to first token |
| AIstudio (WebSocket) | < 2s | Time to first token |
| WebAI (HTTP) | < 500ms | Time to first token |

### Benchmark Tests

```bash
# tests/benchmarks/latency_test.sh

# Test LLMux latency (100 requests)
for i in {1..100}; do
  time curl -s http://localhost:8317/v1/chat/completions \
    -d '{"model": "claude-sonnet-4-5", "messages": [...], "max_tokens": 1}' \
    > /dev/null
done | awk '{sum+=$2; count++} END {print "Average:", sum/count}'

# Should be < 200ms average
```

### Load Testing

```bash
# Use Apache Bench
ab -n 1000 -c 10 -p request.json \
  -T "application/json" \
  http://localhost:8317/v1/chat/completions

# Verify:
# - All requests succeed
# - Average latency within targets
# - No memory leaks
# - No goroutine leaks
```

---

## 5.7 Security Validation

### Security Checklist
- [ ] Auth tokens stored securely (file permissions 600)
- [ ] No tokens in logs
- [ ] No tokens in error messages
- [ ] HTTPS for upstream APIs
- [ ] Input validation on all endpoints
- [ ] No command injection vulnerabilities
- [ ] No path traversal vulnerabilities

**Test**:
```bash
# Check file permissions
ls -la ~/.cli-proxy-api/
# Should show: -rw------- (600)

# Check logs for tokens
grep -r "Bearer" logs/
# Should be empty or redacted

# Test input validation
curl http://localhost:8317/v1/chat/completions \
  -d '{"model": "../../../etc/passwd", ...}'
# Should return validation error, not path traversal
```

---

# 6. SUCCESS METRICS

## 6.1 Functional Success Criteria

### Must Have (MVP) - 100% Required

- [x] **Unified API Endpoint**
  - Single `/v1/chat/completions` endpoint works
  - Accepts OpenAI-compatible requests
  - Returns OpenAI-compatible responses

- [ ] **Provider Support**
  - LLMux Claude Pro OAuth ✅
  - LLMux ChatGPT Plus OAuth ✅
  - ctonew (Clerk JWT) ✅
  - AIstudio (browser automation) ✅
  - At least 4 working providers

- [ ] **Authentication**
  - OAuth flows working (LLMux)
  - Cookie/session persistence (AIstudio)
  - JWT handling (ctonew)
  - Auto token refresh

- [ ] **Model Routing**
  - Pattern matching works
  - Priority ordering works
  - Fallback works on provider failure

- [ ] **Deployment**
  - One command install: `./scripts/install.sh`
  - One command start: `./scripts/start.sh`
  - < 3 processes running
  - < 30 minutes setup time

- [ ] **Health & Monitoring**
  - Health endpoint works: `/v1/health`
  - Service status script: `./scripts/status.sh`
  - Logs accessible: `./scripts/logs.sh`

### Should Have (V1.0) - 80% Required

- [ ] **Reliability**
  - Automatic failover between providers
  - Service auto-restart on failure
  - Graceful shutdown

- [ ] **Observability**
  - Structured logging
  - Request/response logging (optional)
  - Metrics endpoint

- [ ] **Documentation**
  - Complete setup guide
  - Provider documentation
  - API reference
  - Troubleshooting guide

- [ ] **Docker Support**
  - Dockerfile for router
  - docker-compose.yml
  - One command Docker deployment

### Nice to Have (Future) - Optional

- [ ] **Advanced Features**
  - Token usage tracking
  - Rate limiting per provider
  - Load balancing across accounts
  - Admin UI

- [ ] **Storage Backends**
  - PostgreSQL auth storage
  - Git-based auth storage
  - Object storage (S3, etc.)

## 6.2 Performance Success Criteria

| Metric | Target | Measurement |
|--------|--------|-------------|
| **Setup Time** | < 30 min | Time from git clone to first request |
| **Process Count** | 2-3 | `ps aux \| grep -E 'cli-proxy-api\|aistudio\|webai'` |
| **Memory (Router)** | < 100 MB | Idle state |
| **Memory (AIstudio)** | < 500 MB | With 1 browser instance |
| **Latency (LLMux)** | < 200ms | Time to first token (TTFT) |
| **Latency (ctonew)** | < 300ms | TTFT |
| **Latency (AIstudio)** | < 2s | TTFT (browser overhead) |
| **Throughput** | > 100 req/s | Router capacity |
| **Startup Time** | < 10s | All services ready |

## 6.3 User Experience Success Criteria

### First-Time User
**Goal**: Can set up and make first request in < 30 minutes without reading docs.

**Validation**:
- [ ] README.md has quick start in first 20 lines
- [ ] Install script has clear progress indicators
- [ ] Errors have actionable messages
- [ ] Example request in README works copy-paste

**Test**:
```bash
# Give new user just the repo URL
# Time them: git clone → first successful request
# Should be < 30 minutes
```

### Daily User
**Goal**: Start system and use without friction.

**Validation**:
- [ ] One command to start: `./scripts/start.sh`
- [ ] One command to stop: `./scripts/stop.sh`
- [ ] Status check easy: `./scripts/status.sh`
- [ ] No manual token refreshes needed

**Test**:
```bash
# Day 1: Setup
./scripts/install.sh
./scripts/start.sh
# Authenticate once

# Day 2-∞: Daily use
./scripts/start.sh
# Make requests all day
./scripts/stop.sh

# Tokens should auto-refresh, no re-auth needed
```

### Developer
**Goal**: Can debug issues and add providers easily.

**Validation**:
- [ ] Logs are clear and structured
- [ ] Error messages point to solution
- [ ] Documentation has "Adding a Provider" guide
- [ ] Example custom provider works

**Test**:
```bash
# Simulate error: kill AIstudio service
kill -9 <aistudio-pid>

# User checks status
./scripts/status.sh
# Output: AIstudio: not running (crashed)

# User checks logs
./scripts/logs.sh aistudio
# Last line: ERROR: Browser crashed: ...

# User restarts
./scripts/restart.sh aistudio

# Should work again
```

## 6.4 Validation Matrix

| Intention | Success Criteria | How to Validate |
|-----------|------------------|-----------------|
| **Unified Interface** | All providers via one endpoint | Test all models via `/v1/chat/completions` |
| **Simple Deployment** | < 3 processes, < 30 min setup | Time full setup, count processes |
| **Flexible Auth** | OAuth, cookies, JWT all work | Test each auth method |
| **Smart Routing** | Model → provider automatic | Test pattern matching, failover |
| **Extensible** | Add provider without core changes | Implement custom provider from example |
| **Production Ready** | Failover, health checks work | Kill service, verify failover |
| **Developer Friendly** | Clear errors, easy debugging | Simulate errors, check logs/messages |

## 6.5 Release Checklist

Before releasing V1.0, verify:

### Functionality
- [ ] All MVP features working
- [ ] All providers tested
- [ ] Authentication flows tested
- [ ] Model routing tested
- [ ] Failover tested

### Documentation
- [ ] README.md complete
- [ ] SETUP.md complete
- [ ] PROVIDERS.md complete
- [ ] API.md complete
- [ ] TROUBLESHOOTING.md complete

### Testing
- [ ] All unit tests pass
- [ ] All integration tests pass
- [ ] All E2E tests pass
- [ ] Performance benchmarks met
- [ ] Security validation complete

### User Experience
- [ ] First-time setup < 30 min
- [ ] Clear error messages
- [ ] Health checks working
- [ ] Logs accessible and clear

### Code Quality
- [ ] No TODO comments in main branch
- [ ] All code formatted (`gofmt`)
- [ ] All linting passes
- [ ] Dependencies up to date

### Deployment
- [ ] Install script tested on clean system
- [ ] Start/stop scripts tested
- [ ] Docker support working
- [ ] GitHub CI/CD passing

---

# Summary

This specification defines:

1. **What we're building**: A unified AI provider router with optimized architecture
2. **Why we're building it**: To solve fragmentation, access barriers, and deployment complexity
3. **Who it's for**: Developers, power users, researchers, self-hosters
4. **How it works**: Detailed workflows, architecture, and data flows
5. **How to build it**: Phased implementation plan with clear tasks
6. **How to test it**: Comprehensive testing strategy at all levels
7. **How to know it works**: Success metrics and validation criteria

**Key Optimizations from Over-Engineered Version**:
- ✅ Reduced from 5 processes to 2-3
- ✅ Ported LLMux to Go (OAuth in-process)
- ✅ Ported ctonew to Go (Clerk JWT in-process)
- ✅ Kept AIstudio as service (browser required)
- ✅ Made WebAI optional (gpt4free complexity)
- ✅ Simpler deployment and management

**Next Steps**:
1. Review this specification
2. Get approval on architecture decisions
3. Begin Phase 1 implementation
4. Iterate based on feedback

---

**END OF SPECIFICATION**

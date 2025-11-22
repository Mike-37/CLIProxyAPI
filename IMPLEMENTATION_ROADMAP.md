# CLIProxyAPI - Implementation Roadmap

> **Quick Reference**: Track implementation progress and key decisions

---

## 🎯 Project Vision

**One API endpoint → Multiple AI providers → Optimized architecture**

```
BEFORE (Fragmented):
- 5+ separate repos
- 5+ processes running
- Complex deployment
- Manual orchestration

AFTER (Unified):
- 1 repository
- 2-3 processes
- One command install
- Automatic management
```

---

## 📊 Implementation Progress

### Phase 0: Repository Setup ✅
**Status**: Complete (existing code)

- [x] Repository structure
- [x] Go modules
- [x] Existing executors (Claude, Codex, Gemini)
- [x] WebSocket relay infrastructure

---

### Phase 1: Configuration & Infrastructure
**Timeline**: Week 1
**Status**: 🔴 Not Started

#### Tasks
- [ ] **Configuration System**
  - [ ] `config.example.yaml` - Complete provider definitions
  - [ ] `internal/config/config.go` - Parse new structure
  - [ ] Environment variable overrides
  - [ ] Validation logic

- [ ] **Management Scripts**
  - [ ] `scripts/install.sh` - Main installer
  - [ ] `scripts/install/install-base.sh` - Go deps
  - [ ] `scripts/install/install-aistudio.sh` - Python + Playwright
  - [ ] `scripts/install/install-webai.sh` - Python + gpt4free
  - [ ] `scripts/start.sh` - Start all services
  - [ ] `scripts/stop.sh` - Stop all services
  - [ ] `scripts/status.sh` - Check health
  - [ ] `scripts/logs.sh` - View logs

- [ ] **Health System**
  - [ ] `internal/api/handlers/management/health.go`
  - [ ] Router health check
  - [ ] WebSocket service health
  - [ ] HTTP service health
  - [ ] Endpoint: `GET /v1/health`

**Deliverable**: ✅ Complete infrastructure for service management

---

### Phase 2: LLMux Integration (In-Process)
**Timeline**: Week 2
**Status**: 🔴 Not Started

#### 2.1 LLMux Claude OAuth
- [ ] **Auth Implementation**
  - [ ] `internal/auth/llmux/claude_pro_oauth.go`
  - [ ] OAuth 2.0 + PKCE flow
  - [ ] Token storage
  - [ ] Auto-refresh logic
  - [ ] `internal/auth/llmux/oauth_server.go`

- [ ] **Executor**
  - [ ] `internal/runtime/executor/llmux_claude_executor.go`
  - [ ] Implement `cliproxyexecutor.Executor`
  - [ ] API calls to `api.anthropic.com`
  - [ ] Streaming support

- [ ] **Endpoints**
  - [ ] `GET /v1/auth/llmux/claude/login`
  - [ ] `GET /v1/auth/llmux/claude/callback`
  - [ ] `GET /v1/auth/llmux/claude/status`
  - [ ] `DELETE /v1/auth/llmux/claude`

#### 2.2 LLMux ChatGPT OAuth
- [ ] **Auth Implementation**
  - [ ] `internal/auth/llmux/chatgpt_plus_oauth.go`
  - [ ] OAuth flow
  - [ ] Token management

- [ ] **Executor**
  - [ ] `internal/runtime/executor/llmux_chatgpt_executor.go`
  - [ ] API calls to `api.openai.com`
  - [ ] Streaming support

- [ ] **Endpoints**
  - [ ] `GET /v1/auth/llmux/chatgpt/login`
  - [ ] `GET /v1/auth/llmux/chatgpt/callback`
  - [ ] `GET /v1/auth/llmux/chatgpt/status`
  - [ ] `DELETE /v1/auth/llmux/chatgpt`

#### 2.3 Model Routing
- [ ] Update router for `claude-sonnet-4-5`, `gpt-5`
- [ ] Add LLMux to routing table
- [ ] Implement failover logic

**Deliverable**: ✅ LLMux working in-process (no external service)

**Validation**:
```bash
# Should work WITHOUT external service
curl http://localhost:8317/v1/chat/completions \
  -d '{"model": "claude-sonnet-4-5-20250929", "messages": [...]}'
```

---

### Phase 3: ctonew Integration (In-Process)
**Timeline**: Week 3
**Status**: 🔴 Not Started

#### Tasks
- [ ] **Auth Implementation**
  - [ ] `internal/auth/ctonew/clerk_jwt.go`
    - JWT parsing
    - Extract `rotating_token`
    - Base64 decoding
  - [ ] `internal/auth/ctonew/clerk_client.go`
    - HTTP client for Clerk API
    - Token exchange
  - [ ] `internal/auth/ctonew/token_exchange.go`
    - Exchange logic
    - Caching

- [ ] **Executor**
  - [ ] `internal/runtime/executor/ctonew_executor.go`
  - [ ] Implement `clipproxyexecutor.Executor`
  - [ ] Load Clerk JWT
  - [ ] Extract rotating_token
  - [ ] Exchange for new JWT
  - [ ] Call EngineLabs API
  - [ ] Stream responses

- [ ] **Endpoints**
  - [ ] `POST /v1/auth/ctonew` - Save JWT
  - [ ] `GET /v1/auth/ctonew/status`
  - [ ] `DELETE /v1/auth/ctonew`

- [ ] **Model Routing**
  - [ ] Add `ctonew-*` pattern
  - [ ] Add as fallback for `gpt-5`, `claude-*`

**Deliverable**: ✅ ctonew ported from Deno to Go (no Deno service)

**Validation**:
```bash
# Should work WITHOUT Deno service
curl http://localhost:8317/v1/chat/completions \
  -d '{"model": "ctonew-claude-sonnet", "messages": [...]}'

# Verify no Deno process
ps aux | grep deno  # Should be empty
```

---

### Phase 4: AIstudio Service Integration
**Timeline**: Week 4
**Status**: 🔴 Not Started

#### Tasks
- [ ] **Submodule Setup**
  - [ ] Add as submodule: `providers/_reference/aistudio`

- [ ] **Service Files**
  - [ ] `providers/aistudio/main.py`
  - [ ] `providers/aistudio/ws_client.py`
  - [ ] `providers/aistudio/browser_manager.py`
  - [ ] `providers/aistudio/session_manager.py`
  - [ ] `providers/aistudio/gemini_client.py`
  - [ ] `providers/aistudio/requirements.txt`

- [ ] **Router Integration**
  - [ ] Verify `internal/runtime/executor/aistudio_executor.go`
  - [ ] Update WebSocket message format
  - [ ] Auth endpoints:
    - [ ] `POST /v1/auth/aistudio/login`
    - [ ] `GET /v1/auth/aistudio/status`
    - [ ] `DELETE /v1/auth/aistudio`

- [ ] **Service Management**
  - [ ] Update `scripts/start.sh`
  - [ ] Update `scripts/stop.sh`
  - [ ] Update `scripts/status.sh`
  - [ ] Health check via WebSocket

- [ ] **Model Routing**
  - [ ] Add `gemini-.*-aistudio$` pattern

**Deliverable**: ✅ AIstudio service running (WebSocket relay)

**Validation**:
```bash
# Should see browser open
curl -X POST http://localhost:8317/v1/auth/aistudio/login

# Should work via WebSocket relay
curl http://localhost:8317/v1/chat/completions \
  -d '{"model": "gemini-2-flash-aistudio", "messages": [...]}'
```

---

### Phase 5: WebAI Service Integration (OPTIONAL)
**Timeline**: Week 5
**Status**: 🔴 Not Started

#### Tasks
- [ ] **Submodule Setup**
  - [ ] Add as submodule: `providers/_reference/webai`

- [ ] **Service Files**
  - [ ] `providers/webai/main.py`
  - [ ] `providers/webai/http_server.py`
  - [ ] `providers/webai/cookie_manager.py`
  - [ ] `providers/webai/gemini_web_client.py`
  - [ ] `providers/webai/gpt4free_client.py`
  - [ ] `providers/webai/requirements.txt`

- [ ] **Router Integration**
  - [ ] `internal/runtime/executor/http_proxy_executor.go`
  - [ ] Forward to `http://localhost:8406`
  - [ ] Stream responses

- [ ] **Service Management**
  - [ ] Update scripts
  - [ ] Health check via `GET /health`

- [ ] **Configuration**
  - [ ] Add to `config.yaml` (disabled by default)

**Deliverable**: ✅ WebAI service (optional, disabled by default)

---

### Phase 6: Documentation & Examples
**Timeline**: Week 6
**Status**: 🔴 Not Started

#### Tasks
- [ ] **User Docs**
  - [ ] `README.md` - Quick start
  - [ ] `docs/SETUP.md` - Detailed install
  - [ ] `docs/PROVIDERS.md` - Provider docs
  - [ ] `docs/API.md` - API reference
  - [ ] `docs/TROUBLESHOOTING.md` - Common issues
  - [ ] `docs/ARCHITECTURE.md` - Architecture

- [ ] **Developer Docs**
  - [ ] `docs/CONTRIBUTING.md`
  - [ ] `examples/custom-provider/`

- [ ] **Client Examples**
  - [ ] `examples/clients/curl.sh`
  - [ ] `examples/clients/python/example.py`
  - [ ] `examples/clients/javascript/example.js`

**Deliverable**: ✅ Complete documentation

---

## 🏗️ Architecture Summary

### Process Architecture (Final)

```
Production:
├─ cli-proxy-api (PID: 12345)         ← Router (Go binary)
│  ├─ LLMux Claude  (in-process)
│  ├─ LLMux ChatGPT (in-process)
│  └─ ctonew        (in-process)
├─ aistudio (PID: 12346)              ← WebSocket service (Python)
└─ webai (PID: 12347) [OPTIONAL]      ← HTTP service (Python)

Total: 2-3 processes (down from 5+)
```

### Provider Implementation Strategy

| Provider | Type | Why |
|----------|------|-----|
| **LLMux** | In-Process (Go) | OAuth tokens, same as existing Codex/Claude |
| **ctonew** | In-Process (Go) | Simple logic (~70 lines), stateless |
| **AIstudio** | WebSocket Service | Browser automation required |
| **WebAI** | HTTP Service (optional) | Complex gpt4free, Python-specific |

### Request Flow Comparison

#### Before (Over-engineered):
```
Client → Router → HTTP → ctonew service → HTTP → Upstream
                                         (Deno process)
Latency: ~500ms | Processes: 2
```

#### After (Optimized):
```
Client → Router → Direct → Upstream
        (ctonew in-process)
Latency: ~200ms | Processes: 1
```

**Improvement**: 60% faster, 50% fewer processes

---

## ✅ Success Validation

### Functional Requirements

| Requirement | How to Validate | Status |
|-------------|-----------------|--------|
| **Unified API** | All models via `/v1/chat/completions` | ⬜ |
| **Simple Setup** | < 30 min from clone to request | ⬜ |
| **< 3 Processes** | `ps aux \| grep -E 'router\|aistudio\|webai'` | ⬜ |
| **OAuth Works** | LLMux auth flows complete | ⬜ |
| **Browser Works** | AIstudio auth opens browser | ⬜ |
| **JWT Works** | ctonew Clerk JWT exchange | ⬜ |
| **Routing Works** | Model patterns match correctly | ⬜ |
| **Failover Works** | Fallback on provider failure | ⬜ |
| **Health Works** | `/v1/health` returns status | ⬜ |

### Performance Requirements

| Metric | Target | How to Test | Status |
|--------|--------|-------------|--------|
| **Setup Time** | < 30 min | Time full install | ⬜ |
| **Process Count** | 2-3 | Count processes | ⬜ |
| **LLMux Latency** | < 200ms | TTFT benchmark | ⬜ |
| **ctonew Latency** | < 300ms | TTFT benchmark | ⬜ |
| **AIstudio Latency** | < 2s | TTFT benchmark | ⬜ |
| **Memory (Router)** | < 100 MB | Idle state | ⬜ |
| **Throughput** | > 100 req/s | Load test | ⬜ |

### User Experience Requirements

| Requirement | Test | Status |
|-------------|------|--------|
| **Quick Start** | README quick start works copy-paste | ⬜ |
| **Clear Errors** | All errors have actionable messages | ⬜ |
| **Easy Debug** | Logs show clear issue traces | ⬜ |
| **One Command** | `./scripts/start.sh` starts all | ⬜ |

---

## 🧪 Testing Strategy

### Unit Tests (60%)
```bash
# Run all unit tests
go test ./internal/...

# With coverage
go test -cover ./internal/...
```

**Coverage targets**:
- Auth modules: 80%
- Executors: 85%
- Config: 90%
- Routing: 90%

### Integration Tests (30%)
```bash
# Run integration tests
go test ./tests/integration/
```

**Test scenarios**:
- Router + LLMux integration
- Router + AIstudio integration
- Model routing
- Failover logic

### E2E Tests (10%)
```bash
# Run full workflow tests
./tests/e2e/run_all.sh
```

**Test scenarios**:
- Full install → start → request → stop
- Authentication flows
- Multi-provider requests
- Error recovery

---

## 📋 Release Checklist

Before V1.0 release:

### Code
- [ ] All MVP features implemented
- [ ] All tests passing
- [ ] No TODOs in main branch
- [ ] Code formatted (`gofmt`)
- [ ] Linting passing

### Documentation
- [ ] README complete
- [ ] All docs in `docs/` complete
- [ ] API reference accurate
- [ ] Examples working

### Testing
- [ ] Unit tests > 80% coverage
- [ ] Integration tests passing
- [ ] E2E tests passing
- [ ] Performance benchmarks met
- [ ] Security validation complete

### User Experience
- [ ] Setup < 30 min on clean system
- [ ] Error messages clear
- [ ] Health checks working
- [ ] Scripts working

### Deployment
- [ ] Install script tested
- [ ] Docker support working
- [ ] GitHub CI/CD passing

---

## 🎯 Key Decisions

### ✅ Confirmed Decisions

1. **Port LLMux to Go** (in-process)
   - Rationale: OAuth tokens, same pattern as existing
   - Benefit: No external service, faster

2. **Port ctonew to Go** (in-process)
   - Rationale: Simple logic (~70 lines), high value
   - Benefit: No Deno service, -1 process

3. **Keep AIstudio as Service** (WebSocket)
   - Rationale: Browser automation required
   - Trade-off: Worth the complexity

4. **Make WebAI Optional** (HTTP service)
   - Rationale: Complex gpt4free, low priority
   - Trade-off: Keep as service, disable by default

### 🔄 Open Questions

None currently - all major architectural decisions made.

---

## 📞 Quick Commands Reference

```bash
# Installation
git clone --recursive https://github.com/Mike-37/CLIProxyAPI.git
cd CLIProxyAPI
./scripts/install.sh

# Management
./scripts/start.sh      # Start all services
./scripts/stop.sh       # Stop all services
./scripts/restart.sh    # Restart all
./scripts/status.sh     # Check health
./scripts/logs.sh       # View logs

# Development
./scripts/dev/start-router.sh       # Router only
./scripts/dev/start-aistudio.sh     # AIstudio only
./scripts/dev/test-provider.sh      # Test endpoint
./scripts/dev/build.sh              # Build router

# Testing
go test ./internal/...              # Unit tests
go test ./tests/integration/        # Integration tests
./tests/e2e/run_all.sh             # E2E tests
```

---

## 📚 Documentation Index

- **[FINAL_SPECIFICATION.md](docs/FINAL_SPECIFICATION.md)** - Complete spec (this is the master document)
- **[ARCHITECTURE.md](docs/ARCHITECTURE.md)** - Architecture details (TBD)
- **[SETUP.md](docs/SETUP.md)** - Installation guide (TBD)
- **[PROVIDERS.md](docs/PROVIDERS.md)** - Provider docs (TBD)
- **[API.md](docs/API.md)** - API reference (TBD)
- **[TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md)** - Common issues (TBD)
- **[CONTRIBUTING.md](docs/CONTRIBUTING.md)** - Contribution guide (TBD)

---

**Last Updated**: 2025-11-22
**Version**: 2.0 (Optimized Architecture)
**Status**: Design Phase - Ready for Implementation

#!/bin/bash
# ============================================
# CLIProxyAPI - Start All Services
# ============================================
# This script starts the router and all enabled provider services.

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[✓]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[!]${NC} $1"; }
print_error() { echo -e "${RED}[✗]${NC} $1"; }

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
PID_DIR="$ROOT_DIR/pids"
LOG_DIR="$ROOT_DIR/logs"

cd "$ROOT_DIR"

# Create directories
mkdir -p "$PID_DIR" "$LOG_DIR"

# Check if router binary exists
if [ ! -f "bin/cli-proxy-api" ]; then
    print_error "Router binary not found!"
    print_info "Run ./scripts/install.sh first"
    exit 1
fi

# Check for config
if [ ! -f "config.yaml" ]; then
    print_error "config.yaml not found!"
    exit 1
fi

# ============================================
# START ROUTER
# ============================================
print_info "Starting CLIProxyAPI router..."

# Check if already running
if [ -f "$PID_DIR/router.pid" ]; then
    OLD_PID=$(cat "$PID_DIR/router.pid")
    if kill -0 "$OLD_PID" 2>/dev/null; then
        print_warning "Router is already running (PID: $OLD_PID)"
    else
        print_warning "Stale PID file found, removing..."
        rm "$PID_DIR/router.pid"
    fi
fi

# Start router in background
nohup ./bin/cli-proxy-api > "$LOG_DIR/router.log" 2>&1 &
ROUTER_PID=$!
echo $ROUTER_PID > "$PID_DIR/router.pid"
print_success "Router started (PID: $ROUTER_PID)"

# Wait for router to be ready
sleep 2
if ! kill -0 "$ROUTER_PID" 2>/dev/null; then
    print_error "Router failed to start!"
    print_info "Check logs: tail -f $LOG_DIR/router.log"
    exit 1
fi

# Check if router is responding
if command -v curl &> /dev/null; then
    for i in {1..10}; do
        if curl -s http://localhost:8317/v1/health > /dev/null 2>&1; then
            print_success "Router is healthy"
            break
        fi
        if [ $i -eq 10 ]; then
            print_warning "Router may not be ready yet (health check failed)"
        fi
        sleep 1
    done
fi

# ============================================
# START AISTUDIO (if enabled)
# ============================================
AISTUDIO_ENABLED=false
if grep -q "aistudio:" config.yaml && grep -A5 "aistudio:" config.yaml | grep -q "enabled: true"; then
    AISTUDIO_ENABLED=true
fi

if [ "$AISTUDIO_ENABLED" = true ]; then
    print_info "Starting AIstudio service..."

    # Check if already running
    if [ -f "$PID_DIR/aistudio.pid" ]; then
        OLD_PID=$(cat "$PID_DIR/aistudio.pid")
        if kill -0 "$OLD_PID" 2>/dev/null; then
            print_warning "AIstudio is already running (PID: $OLD_PID)"
        else
            rm "$PID_DIR/aistudio.pid"
        fi
    fi

    # Check if service exists
    if [ -f "providers/aistudio/main.py" ]; then
        cd providers/aistudio
        nohup python3 main.py > "$LOG_DIR/aistudio.log" 2>&1 &
        AISTUDIO_PID=$!
        echo $AISTUDIO_PID > "$PID_DIR/aistudio.pid"
        cd "$ROOT_DIR"
        print_success "AIstudio started (PID: $AISTUDIO_PID)"
    else
        print_warning "AIstudio service not found (will be implemented in Phase 4)"
    fi
else
    print_info "AIstudio is disabled, skipping"
fi

# ============================================
# START WEBAI (if enabled)
# ============================================
WEBAI_ENABLED=false
if grep -q "webai:" config.yaml && grep -A5 "webai:" config.yaml | grep -q "enabled: true"; then
    WEBAI_ENABLED=true
fi

if [ "$WEBAI_ENABLED" = true ]; then
    print_info "Starting WebAI service..."

    # Check if already running
    if [ -f "$PID_DIR/webai.pid" ]; then
        OLD_PID=$(cat "$PID_DIR/webai.pid")
        if kill -0 "$OLD_PID" 2>/dev/null; then
            print_warning "WebAI is already running (PID: $OLD_PID)"
        else
            rm "$PID_DIR/webai.pid"
        fi
    fi

    # Check if service exists
    if [ -f "providers/webai/main.py" ]; then
        cd providers/webai
        nohup python3 main.py > "$LOG_DIR/webai.log" 2>&1 &
        WEBAI_PID=$!
        echo $WEBAI_PID > "$PID_DIR/webai.pid"
        cd "$ROOT_DIR"
        print_success "WebAI started (PID: $WEBAI_PID)"
    else
        print_warning "WebAI service not found (will be implemented in Phase 5)"
    fi
else
    print_info "WebAI is disabled, skipping"
fi

# ============================================
# SUMMARY
# ============================================
echo ""
print_success "All services started successfully!"
echo ""
print_info "Services running:"
echo "  - Router:   http://localhost:8317 (PID: $ROUTER_PID)"
[ "$AISTUDIO_ENABLED" = true ] && [ -n "$AISTUDIO_PID" ] && echo "  - AIstudio: WebSocket relay (PID: $AISTUDIO_PID)"
[ "$WEBAI_ENABLED" = true ] && [ -n "$WEBAI_PID" ] && echo "  - WebAI:    http://localhost:8406 (PID: $WEBAI_PID)"
echo ""
print_info "Check status: ./scripts/status.sh"
print_info "View logs:    ./scripts/logs.sh [router|aistudio|webai]"
print_info "Stop services: ./scripts/stop.sh"

#!/bin/bash
# CLIProxyAPI - Service Management (Start)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BINARY="$PROJECT_ROOT/cli-proxy-api"
LOGDIR="$PROJECT_ROOT/logs"
PIDDIR="$PROJECT_ROOT/.pids"

CONFIG="${CONFIG:-$PROJECT_ROOT/config.yaml}"

# Create necessary directories
mkdir -p "$LOGDIR" "$PIDDIR"

echo "🚀 Starting CLIProxyAPI services..."

# Check if binary exists
if [ ! -f "$BINARY" ]; then
    echo "❌ Binary not found: $BINARY"
    echo "Run: ./scripts/install.sh"
    exit 1
fi

# Check if config exists
if [ ! -f "$CONFIG" ]; then
    echo "❌ Config not found: $CONFIG"
    echo "Run: ./scripts/install.sh"
    exit 1
fi

# Start main router
echo "📡 Starting main router..."
if [ -f "$PIDDIR/router.pid" ]; then
    EXISTING_PID=$(cat "$PIDDIR/router.pid")
    if ps -p "$EXISTING_PID" &> /dev/null; then
        echo "⚠️  Router already running (PID: $EXISTING_PID)"
        echo "Run: ./scripts/stop.sh"
        exit 1
    fi
fi

# Build if needed
if [ ! -f "$BINARY" ] || [ "$PROJECT_ROOT/cmd/server/main.go" -nt "$BINARY" ]; then
    echo "🔨 Building CLI Proxy API..."
    cd "$PROJECT_ROOT"
    go build -o cli-proxy-api ./cmd/server
fi

# Start router in background
$BINARY --config "$CONFIG" > "$LOGDIR/router.log" 2>&1 &
ROUTER_PID=$!
echo $ROUTER_PID > "$PIDDIR/router.pid"
sleep 1

# Verify router started
if ! ps -p "$ROUTER_PID" &> /dev/null; then
    echo "❌ Router failed to start"
    echo "Check logs: tail -f $LOGDIR/router.log"
    exit 1
fi
echo "✅ Router started (PID: $ROUTER_PID)"

# Wait for router to be ready
echo "⏳ Waiting for router to be ready..."
for i in {1..30}; do
    if curl -s http://localhost:8317/v1/health &> /dev/null; then
        echo "✅ Router is ready"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "❌ Router did not become ready"
        echo "Check logs: tail -f $LOGDIR/router.log"
        exit 1
    fi
    sleep 1
done

# Start AIstudio if configured and not disabled
if grep -q "aistudio.*enabled.*true" "$CONFIG" 2>/dev/null; then
    AISTUDIO_SCRIPT="$SCRIPT_DIR/services/start-aistudio.sh"
    if [ -f "$AISTUDIO_SCRIPT" ]; then
        echo "📸 Starting AIstudio service..."
        bash "$AISTUDIO_SCRIPT"
    fi
fi

# Start WebAI if configured and enabled
if grep -q "webai.*enabled.*true" "$CONFIG" 2>/dev/null; then
    WEBAI_SCRIPT="$SCRIPT_DIR/services/start-webai.sh"
    if [ -f "$WEBAI_SCRIPT" ]; then
        echo "🌐 Starting WebAI service..."
        bash "$WEBAI_SCRIPT"
    fi
fi

echo ""
echo "✅ All services started successfully"
echo ""
echo "Router PID: $ROUTER_PID"
echo "Config: $CONFIG"
echo "Logs: $LOGDIR/"
echo ""
echo "Test with: curl http://localhost:8317/v1/health"
echo "View logs: tail -f $LOGDIR/router.log"
echo "Stop with: ./scripts/stop.sh"

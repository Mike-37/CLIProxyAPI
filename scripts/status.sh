#!/bin/bash
# CLIProxyAPI - Service Management (Status)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
PIDDIR="$PROJECT_ROOT/.pids"
LOGDIR="$PROJECT_ROOT/logs"

echo "📊 CLIProxyAPI Service Status"
echo "=============================="
echo ""

# Check main router
echo "📡 Main Router:"
if [ -f "$PIDDIR/router.pid" ]; then
    ROUTER_PID=$(cat "$PIDDIR/router.pid")
    if ps -p "$ROUTER_PID" &> /dev/null; then
        echo "   Status: ✅ Running (PID: $ROUTER_PID)"

        # Check health endpoint
        if curl -s http://localhost:8317/v1/health &> /dev/null; then
            echo "   Health: ✅ Healthy"
        else
            echo "   Health: ⚠️  Unreachable"
        fi
    else
        echo "   Status: ❌ Not running (PID file stale)"
    fi
else
    echo "   Status: ❌ Not running"
fi

echo ""
echo "📸 AIstudio Service:"
if [ -f "$PIDDIR/aistudio.pid" ]; then
    AISTUDIO_PID=$(cat "$PIDDIR/aistudio.pid")
    if ps -p "$AISTUDIO_PID" &> /dev/null; then
        echo "   Status: ✅ Running (PID: $AISTUDIO_PID)"
    else
        echo "   Status: ❌ Not running (PID file stale)"
    fi
else
    echo "   Status: ❌ Not configured or running"
fi

echo ""
echo "🌐 WebAI Service:"
if [ -f "$PIDDIR/webai.pid" ]; then
    WEBAI_PID=$(cat "$PIDDIR/webai.pid")
    if ps -p "$WEBAI_PID" &> /dev/null; then
        echo "   Status: ✅ Running (PID: $WEBAI_PID)"
    else
        echo "   Status: ❌ Not running (PID file stale)"
    fi
else
    echo "   Status: ❌ Not configured or running"
fi

echo ""
echo "📁 Logs:"
echo "   Router: $LOGDIR/router.log"
echo "   AIstudio: $LOGDIR/aistudio.log"
echo "   WebAI: $LOGDIR/webai.log"

echo ""
echo "Commands:"
echo "   Start:  ./scripts/start.sh"
echo "   Stop:   ./scripts/stop.sh"
echo "   Logs:   ./scripts/logs.sh"
echo "   Restart: ./scripts/restart.sh"

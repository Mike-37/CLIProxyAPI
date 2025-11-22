#!/bin/bash
# CLIProxyAPI - Service Management (Stop)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
PIDDIR="$PROJECT_ROOT/.pids"

echo "🛑 Stopping CLIProxyAPI services..."

# Stop main router
if [ -f "$PIDDIR/router.pid" ]; then
    ROUTER_PID=$(cat "$PIDDIR/router.pid")
    if ps -p "$ROUTER_PID" &> /dev/null; then
        echo "Stopping router (PID: $ROUTER_PID)..."
        kill "$ROUTER_PID"
        sleep 2
        if ps -p "$ROUTER_PID" &> /dev/null; then
            echo "Force killing router..."
            kill -9 "$ROUTER_PID"
        fi
        rm "$PIDDIR/router.pid"
        echo "✅ Router stopped"
    else
        echo "⚠️  Router not running"
        rm "$PIDDIR/router.pid"
    fi
else
    echo "⚠️  No router PID file found"
fi

# Stop AIstudio if running
if [ -f "$PIDDIR/aistudio.pid" ]; then
    AISTUDIO_PID=$(cat "$PIDDIR/aistudio.pid")
    if ps -p "$AISTUDIO_PID" &> /dev/null; then
        echo "Stopping AIstudio (PID: $AISTUDIO_PID)..."
        kill "$AISTUDIO_PID"
        sleep 1
        if ps -p "$AISTUDIO_PID" &> /dev/null; then
            kill -9 "$AISTUDIO_PID"
        fi
        rm "$PIDDIR/aistudio.pid"
        echo "✅ AIstudio stopped"
    else
        rm "$PIDDIR/aistudio.pid"
    fi
fi

# Stop WebAI if running
if [ -f "$PIDDIR/webai.pid" ]; then
    WEBAI_PID=$(cat "$PIDDIR/webai.pid")
    if ps -p "$WEBAI_PID" &> /dev/null; then
        echo "Stopping WebAI (PID: $WEBAI_PID)..."
        kill "$WEBAI_PID"
        sleep 1
        if ps -p "$WEBAI_PID" &> /dev/null; then
            kill -9 "$WEBAI_PID"
        fi
        rm "$PIDDIR/webai.pid"
        echo "✅ WebAI stopped"
    else
        rm "$PIDDIR/webai.pid"
    fi
fi

echo "✅ All services stopped"

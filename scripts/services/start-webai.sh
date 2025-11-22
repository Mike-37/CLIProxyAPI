#!/bin/bash
# CLIProxyAPI - WebAI Service Startup

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"
WEBAI_DIR="$PROJECT_ROOT/providers/webai"
LOGDIR="$PROJECT_ROOT/logs"
PIDDIR="$PROJECT_ROOT/.pids"

mkdir -p "$LOGDIR" "$PIDDIR"

# Check if WebAI directory exists
if [ ! -d "$WEBAI_DIR" ]; then
    echo "❌ WebAI directory not found: $WEBAI_DIR"
    exit 1
fi

# Check if main.py exists
if [ ! -f "$WEBAI_DIR/main.py" ]; then
    echo "❌ WebAI main.py not found: $WEBAI_DIR/main.py"
    exit 1
fi

# Check if already running
if [ -f "$PIDDIR/webai.pid" ]; then
    EXISTING_PID=$(cat "$PIDDIR/webai.pid")
    if ps -p "$EXISTING_PID" &> /dev/null; then
        echo "⚠️  WebAI already running (PID: $EXISTING_PID)"
        return 0 2>/dev/null || exit 0
    fi
fi

# Start WebAI
python3 "$WEBAI_DIR/main.py" > "$LOGDIR/webai.log" 2>&1 &
WEBAI_PID=$!
echo $WEBAI_PID > "$PIDDIR/webai.pid"

sleep 2
if ps -p "$WEBAI_PID" &> /dev/null; then
    echo "✅ WebAI started (PID: $WEBAI_PID)"
else
    echo "❌ WebAI failed to start"
    exit 1
fi

#!/bin/bash
# CLIProxyAPI - AIstudio Service Startup

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"
AISTUDIO_DIR="$PROJECT_ROOT/providers/aistudio"
LOGDIR="$PROJECT_ROOT/logs"
PIDDIR="$PROJECT_ROOT/.pids"

mkdir -p "$LOGDIR" "$PIDDIR"

# Check if AIstudio directory exists
if [ ! -d "$AISTUDIO_DIR" ]; then
    echo "❌ AIstudio directory not found: $AISTUDIO_DIR"
    exit 1
fi

# Check if main.py exists
if [ ! -f "$AISTUDIO_DIR/main.py" ]; then
    echo "❌ AIstudio main.py not found: $AISTUDIO_DIR/main.py"
    exit 1
fi

# Check if already running
if [ -f "$PIDDIR/aistudio.pid" ]; then
    EXISTING_PID=$(cat "$PIDDIR/aistudio.pid")
    if ps -p "$EXISTING_PID" &> /dev/null; then
        echo "⚠️  AIstudio already running (PID: $EXISTING_PID)"
        return 0 2>/dev/null || exit 0
    fi
fi

# Start AIstudio
python3 "$AISTUDIO_DIR/main.py" > "$LOGDIR/aistudio.log" 2>&1 &
AISTUDIO_PID=$!
echo $AISTUDIO_PID > "$PIDDIR/aistudio.pid"

sleep 2
if ps -p "$AISTUDIO_PID" &> /dev/null; then
    echo "✅ AIstudio started (PID: $AISTUDIO_PID)"
else
    echo "❌ AIstudio failed to start"
    exit 1
fi

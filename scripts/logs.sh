#!/bin/bash
# CLIProxyAPI - Service Management (Logs)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
LOGDIR="$PROJECT_ROOT/logs"

SERVICE="${1:-router}"

case "$SERVICE" in
    router)
        echo "📡 Main Router Logs:"
        if [ -f "$LOGDIR/router.log" ]; then
            tail -f "$LOGDIR/router.log"
        else
            echo "❌ Log file not found: $LOGDIR/router.log"
            echo "Start services first: ./scripts/start.sh"
        fi
        ;;
    aistudio)
        echo "📸 AIstudio Service Logs:"
        if [ -f "$LOGDIR/aistudio.log" ]; then
            tail -f "$LOGDIR/aistudio.log"
        else
            echo "❌ Log file not found: $LOGDIR/aistudio.log"
        fi
        ;;
    webai)
        echo "🌐 WebAI Service Logs:"
        if [ -f "$LOGDIR/webai.log" ]; then
            tail -f "$LOGDIR/webai.log"
        else
            echo "❌ Log file not found: $LOGDIR/webai.log"
        fi
        ;;
    *)
        echo "Usage: $0 [router|aistudio|webai]"
        echo ""
        echo "Available log services:"
        echo "  router    - Main router logs (default)"
        echo "  aistudio  - AIstudio service logs"
        echo "  webai     - WebAI service logs"
        ;;
esac

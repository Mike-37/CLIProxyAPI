#!/bin/bash
# ============================================
# CLIProxyAPI - View Logs
# ============================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_error() { echo -e "${RED}[✗]${NC} $1"; }

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
LOG_DIR="$ROOT_DIR/logs"

# Default service
SERVICE=${1:-router}
LINES=${2:-50}

LOG_FILE="$LOG_DIR/${SERVICE}.log"

if [ ! -f "$LOG_FILE" ]; then
    print_error "Log file not found: $LOG_FILE"
    echo ""
    print_info "Available logs:"
    ls -1 "$LOG_DIR"/*.log 2>/dev/null | xargs -n1 basename | sed 's/\.log$//' || echo "  (none)"
    echo ""
    print_info "Usage: $0 [service] [lines]"
    print_info "  service: router, aistudio, webai (default: router)"
    print_info "  lines:   number of lines to show (default: 50)"
    exit 1
fi

# Check if following
if [ "$3" = "-f" ] || [ "$3" = "--follow" ]; then
    print_info "Following $SERVICE logs (Ctrl+C to stop)..."
    echo ""
    tail -f "$LOG_FILE"
else
    print_info "Last $LINES lines of $SERVICE logs:"
    echo ""
    tail -n "$LINES" "$LOG_FILE"
    echo ""
    print_info "Follow logs: $0 $SERVICE $LINES -f"
fi

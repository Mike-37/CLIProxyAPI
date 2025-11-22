#!/bin/bash
# ============================================
# CLIProxyAPI - Stop All Services
# ============================================

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

cd "$ROOT_DIR"

# Function to stop a service
stop_service() {
    local NAME=$1
    local PID_FILE="$PID_DIR/${NAME}.pid"

    if [ ! -f "$PID_FILE" ]; then
        print_info "$NAME is not running (no PID file)"
        return
    fi

    local PID=$(cat "$PID_FILE")

    if kill -0 "$PID" 2>/dev/null; then
        print_info "Stopping $NAME (PID: $PID)..."
        kill "$PID" 2>/dev/null || true

        # Wait for process to stop (max 10 seconds)
        for i in {1..10}; do
            if ! kill -0 "$PID" 2>/dev/null; then
                print_success "$NAME stopped"
                rm "$PID_FILE"
                return
            fi
            sleep 1
        done

        # Force kill if still running
        if kill -0 "$PID" 2>/dev/null; then
            print_warning "$NAME didn't stop gracefully, force killing..."
            kill -9 "$PID" 2>/dev/null || true
            sleep 1
            if ! kill -0 "$PID" 2>/dev/null; then
                print_success "$NAME force stopped"
            else
                print_error "Failed to stop $NAME"
            fi
        fi
        rm "$PID_FILE"
    else
        print_info "$NAME is not running (stale PID file)"
        rm "$PID_FILE"
    fi
}

print_info "Stopping CLIProxyAPI services..."
echo ""

# Stop in reverse order (services first, then router)
stop_service "webai"
stop_service "aistudio"
stop_service "router"

echo ""
print_success "All services stopped"

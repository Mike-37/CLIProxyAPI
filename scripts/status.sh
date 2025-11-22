#!/bin/bash
# ============================================
# CLIProxyAPI - Check Service Status
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

# Function to check service status
check_service() {
    local NAME=$1
    local PID_FILE="$PID_DIR/${NAME}.pid"
    local CHECK_URL=$2

    if [ ! -f "$PID_FILE" ]; then
        echo -e "  ${NAME}: ${RED}not running${NC} (no PID file)"
        return 1
    fi

    local PID=$(cat "$PID_FILE")

    if kill -0 "$PID" 2>/dev/null; then
        # Check if URL health check is provided
        if [ -n "$CHECK_URL" ] && command -v curl &> /dev/null; then
            if curl -s "$CHECK_URL" > /dev/null 2>&1; then
                echo -e "  ${NAME}: ${GREEN}running${NC} (PID: $PID, ${GREEN}healthy${NC})"
                return 0
            else
                echo -e "  ${NAME}: ${YELLOW}running${NC} (PID: $PID, ${YELLOW}unhealthy${NC})"
                return 2
            fi
        else
            echo -e "  ${NAME}: ${GREEN}running${NC} (PID: $PID)"
            return 0
        fi
    else
        echo -e "  ${NAME}: ${RED}not running${NC} (stale PID)"
        return 1
    fi
}

echo ""
echo -e "${BLUE}===================================${NC}"
echo -e "${BLUE}CLIProxyAPI Service Status${NC}"
echo -e "${BLUE}===================================${NC}"
echo ""

# Check router
check_service "router" "http://localhost:8317/v1/health"
ROUTER_STATUS=$?

# Check AIstudio
check_service "aistudio" ""
AISTUDIO_STATUS=$?

# Check WebAI
check_service "webai" "http://localhost:8406/health"
WEBAI_STATUS=$?

echo ""
echo -e "${BLUE}===================================${NC}"
echo ""

# Summary
if [ $ROUTER_STATUS -eq 0 ]; then
    print_success "Router is healthy"
    if command -v curl &> /dev/null; then
        print_info "API endpoint: http://localhost:8317/v1/chat/completions"
        print_info "Health check: http://localhost:8317/v1/health"
    fi
else
    print_error "Router is not running!"
    print_info "Start with: ./scripts/start.sh"
fi

echo ""

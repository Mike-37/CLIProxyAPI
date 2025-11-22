#!/bin/bash
# ============================================
# CLIProxyAPI - Restart Services
# ============================================

set -e

# Colors
BLUE='\033[0;34m'
NC='\033[0m'

print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Stop all services
print_info "Stopping services..."
"$SCRIPT_DIR/stop.sh"

echo ""
sleep 2

# Start all services
print_info "Starting services..."
"$SCRIPT_DIR/start.sh"

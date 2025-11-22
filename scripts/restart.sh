#!/bin/bash
# CLIProxyAPI - Service Management (Restart)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "🔄 Restarting CLIProxyAPI services..."
echo ""

# Stop services
bash "$SCRIPT_DIR/stop.sh"

echo ""
sleep 2

# Start services
bash "$SCRIPT_DIR/start.sh"

echo ""
echo "✅ Services restarted"

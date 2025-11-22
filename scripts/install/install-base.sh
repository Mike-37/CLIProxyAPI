#!/bin/bash
# CLIProxyAPI - Base Installation (Go dependencies)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"

echo "📦 Checking Go installation..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed"
    echo ""
    echo "Install Go from: https://golang.org/dl/"
    echo "Required version: Go 1.21 or later"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}')
echo "✅ Go found: $GO_VERSION"

# Download Go modules
echo ""
echo "📥 Downloading Go dependencies..."
cd "$PROJECT_ROOT"
go mod download
go mod tidy

echo "✅ Go dependencies installed"

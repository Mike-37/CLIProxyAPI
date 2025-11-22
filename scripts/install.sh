#!/bin/bash
# CLIProxyAPI - Main Installation Script
# Installs all dependencies and prepares the system

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
INSTALL_DIR="$SCRIPT_DIR/install"

echo "🚀 CLIProxyAPI Installation"
echo "================================"
echo "Project Root: $PROJECT_ROOT"
echo ""

# Check if running in a supported environment
if [[ "$OSTYPE" != "linux-gnu"* ]] && [[ "$OSTYPE" != "darwin"* ]]; then
    echo "⚠️  Warning: This script has been tested on Linux and macOS only."
    read -p "Continue? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Phase 1: Install base dependencies (Go)
echo "📦 Phase 1: Installing base dependencies..."
if [ -f "$INSTALL_DIR/install-base.sh" ]; then
    bash "$INSTALL_DIR/install-base.sh"
else
    echo "⚠️  install-base.sh not found, skipping Go installation"
    echo "   Make sure Go 1.21+ is installed"
fi

# Phase 4: Install AIstudio dependencies (optional)
read -p "📸 Install AIstudio (Python + Playwright)? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Installing AIstudio dependencies..."
    if [ -f "$INSTALL_DIR/install-aistudio.sh" ]; then
        bash "$INSTALL_DIR/install-aistudio.sh"
    else
        echo "⚠️  install-aistudio.sh not found"
    fi
fi

# Phase 5: Install WebAI dependencies (optional)
read -p "🌐 Install WebAI (Python + gpt4free)? [not recommended] (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Installing WebAI dependencies..."
    if [ -f "$INSTALL_DIR/install-webai.sh" ]; then
        bash "$INSTALL_DIR/install-webai.sh"
    else
        echo "⚠️  install-webai.sh not found"
    fi
fi

# Setup auth directory
AUTH_DIR="${AUTH_DIR:~/.cli-proxy-api}"
mkdir -p "$AUTH_DIR"
echo "✅ Created auth directory: $AUTH_DIR"

# Build the router
echo ""
echo "🔨 Building CLIProxyAPI router..."
cd "$PROJECT_ROOT"
go build -o cli-proxy-api ./cmd/server || {
    echo "❌ Build failed"
    exit 1
}
echo "✅ Build successful: ./cli-proxy-api"

# Setup config file if missing
if [ ! -f "$PROJECT_ROOT/config.yaml" ]; then
    echo ""
    echo "⚙️  Setting up configuration..."
    cp "$PROJECT_ROOT/config.example.yaml" "$PROJECT_ROOT/config.yaml"
    echo "✅ Created config.yaml (edit this file to configure providers)"
fi

echo ""
echo "================================"
echo "✅ Installation complete!"
echo ""
echo "Next steps:"
echo "1. Edit config.yaml to add your API keys"
echo "2. Run: ./scripts/start.sh"
echo "3. Test with: curl http://localhost:8317/v1/health"
echo ""
echo "For more information, see docs/SETUP.md"

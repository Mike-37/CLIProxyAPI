#!/bin/bash
# CLIProxyAPI - WebAI Service Installation (Python + gpt4free)
# NOTE: This is OPTIONAL and disabled by default

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"
WEBAI_DIR="$PROJECT_ROOT/providers/webai"

echo "⚠️  WebAI Service Installation (Optional)"
echo "Note: WebAI is optional and disabled by default"
echo ""

# Check if Python 3 is installed
if ! command -v python3 &> /dev/null; then
    echo "❌ Python 3 is not installed"
    echo "Install Python 3.8+ before continuing"
    exit 1
fi

PYTHON_VERSION=$(python3 --version | awk '{print $2}')
echo "✅ Python found: $PYTHON_VERSION"

# Create WebAI directory if it doesn't exist
mkdir -p "$WEBAI_DIR"

# Check if requirements.txt exists in WebAI
if [ -f "$WEBAI_DIR/requirements.txt" ]; then
    echo "📥 Installing Python dependencies for WebAI..."
    python3 -m pip install --upgrade pip
    python3 -m pip install -r "$WEBAI_DIR/requirements.txt"

    echo "✅ WebAI dependencies installed"
else
    echo "⚠️  WebAI requirements.txt not found at: $WEBAI_DIR/requirements.txt"
    echo "   You may need to set up WebAI manually"
    echo ""
    echo "Expected structure:"
    echo "   providers/webai/"
    echo "   ├── main.py"
    echo "   ├── requirements.txt"
    echo "   └── *.py"
fi

echo ""
echo "ℹ️  To enable WebAI, edit config.yaml and set:"
echo "   webai-enabled: true"

#!/bin/bash
# CLIProxyAPI - AIstudio Service Installation (Python + Playwright)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"
AISTUDIO_DIR="$PROJECT_ROOT/providers/aistudio"

echo "📸 Installing AIstudio service dependencies..."

# Check if Python 3 is installed
if ! command -v python3 &> /dev/null; then
    echo "❌ Python 3 is not installed"
    echo "Install Python 3.8+ before continuing"
    exit 1
fi

PYTHON_VERSION=$(python3 --version | awk '{print $2}')
echo "✅ Python found: $PYTHON_VERSION"

# Create AIstudio directory if it doesn't exist
mkdir -p "$AISTUDIO_DIR"

# Check if requirements.txt exists in AIstudio
if [ -f "$AISTUDIO_DIR/requirements.txt" ]; then
    echo "📥 Installing Python dependencies for AIstudio..."
    python3 -m pip install --upgrade pip
    python3 -m pip install -r "$AISTUDIO_DIR/requirements.txt"

    # Install Playwright browsers
    echo "🌐 Installing Playwright browsers..."
    python3 -m playwright install chromium firefox

    echo "✅ AIstudio dependencies installed"
else
    echo "⚠️  AIstudio requirements.txt not found at: $AISTUDIO_DIR/requirements.txt"
    echo "   You may need to set up AIstudio manually"
    echo ""
    echo "Expected structure:"
    echo "   providers/aistudio/"
    echo "   ├── main.py"
    echo "   ├── requirements.txt"
    echo "   └── *.py"
fi

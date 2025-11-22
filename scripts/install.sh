#!/bin/bash
# ============================================
# CLIProxyAPI - Main Installation Script
# ============================================
# This script installs all dependencies for the unified router
# and enabled provider services.

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print functions
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[!]${NC} $1"
}

print_error() {
    echo -e "${RED}[✗]${NC} $1"
}

print_header() {
    echo -e "\n${BLUE}===================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}===================================${NC}\n"
}

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$ROOT_DIR"

print_header "CLIProxyAPI Installation"
print_info "Root directory: $ROOT_DIR"

# Check for config file
if [ ! -f "config.yaml" ] && [ ! -f "config.unified.yaml" ]; then
    print_warning "No configuration file found"
    if [ -f "config.example.yaml" ]; then
        print_info "Copying config.example.yaml to config.yaml"
        cp config.example.yaml config.yaml
        print_success "Created config.yaml from example"
    else
        print_error "No config.example.yaml found!"
        exit 1
    fi
fi

# ============================================
# PHASE 1: Install Base Dependencies (Go)
# ============================================
print_header "Phase 1: Installing Base Dependencies"

if command -v go &> /dev/null; then
    GO_VERSION=$(go version | awk '{print $3}')
    print_success "Go is installed: $GO_VERSION"
else
    print_error "Go is not installed!"
    print_info "Please install Go 1.21 or later from https://golang.org/dl/"
    exit 1
fi

# Install Go dependencies
print_info "Installing Go dependencies..."
go mod download
print_success "Go dependencies installed"

# Build router
print_info "Building router..."
mkdir -p bin
go build -o bin/cli-proxy-api cmd/server/main.go
print_success "Router built successfully: bin/cli-proxy-api"

# ============================================
# PHASE 2: Check for Provider-Specific Dependencies
# ============================================
print_header "Phase 2: Checking Provider Dependencies"

# Check config for enabled providers
AISTUDIO_ENABLED=false
WEBAI_ENABLED=false

if [ -f "config.yaml" ]; then
    # Simple grep check (not perfect YAML parsing, but works for our case)
    if grep -q "aistudio:" config.yaml && grep -A5 "aistudio:" config.yaml | grep -q "enabled: true"; then
        AISTUDIO_ENABLED=true
    fi
    if grep -q "webai:" config.yaml && grep -A5 "webai:" config.yaml | grep -q "enabled: true"; then
        WEBAI_ENABLED=true
    fi
fi

# ============================================
# PHASE 3: Install AIstudio Dependencies (if enabled)
# ============================================
if [ "$AISTUDIO_ENABLED" = true ]; then
    print_header "Phase 3: Installing AIstudio Dependencies"

    # Check for Python
    if command -v python3 &> /dev/null; then
        PYTHON_VERSION=$(python3 --version)
        print_success "Python is installed: $PYTHON_VERSION"
    else
        print_error "Python 3 is not installed!"
        print_info "Please install Python 3.8 or later"
        exit 1
    fi

    # Check if pip is available
    if ! command -v pip3 &> /dev/null; then
        print_error "pip3 is not installed!"
        print_info "Please install pip3"
        exit 1
    fi

    # Install Python dependencies for AIstudio
    if [ -f "providers/aistudio/requirements.txt" ]; then
        print_info "Installing AIstudio Python dependencies..."
        pip3 install -r providers/aistudio/requirements.txt
        print_success "AIstudio dependencies installed"
    else
        print_warning "No requirements.txt found for AIstudio"
        print_info "AIstudio will be available when the service is implemented"
    fi

    # Check for playwright browsers
    print_info "Installing Playwright browsers..."
    python3 -m playwright install chromium 2>/dev/null || print_warning "Playwright install skipped (will install when needed)"
else
    print_info "AIstudio is disabled, skipping Python dependencies"
fi

# ============================================
# PHASE 4: Install WebAI Dependencies (if enabled)
# ============================================
if [ "$WEBAI_ENABLED" = true ]; then
    print_header "Phase 4: Installing WebAI Dependencies"

    # WebAI also uses Python
    if [ -f "providers/webai/requirements.txt" ]; then
        print_info "Installing WebAI Python dependencies..."
        pip3 install -r providers/webai/requirements.txt
        print_success "WebAI dependencies installed"
    else
        print_warning "No requirements.txt found for WebAI"
        print_info "WebAI will be available when the service is implemented"
    fi
else
    print_info "WebAI is disabled, skipping dependencies"
fi

# ============================================
# PHASE 5: Create necessary directories
# ============================================
print_header "Phase 5: Creating Directories"

mkdir -p logs
print_success "Created logs directory"

mkdir -p pids
print_success "Created pids directory"

mkdir -p ~/.cli-proxy-api
chmod 700 ~/.cli-proxy-api
print_success "Created auth storage directory: ~/.cli-proxy-api"

# Create provider directories
mkdir -p providers/aistudio/auth_profiles
mkdir -p providers/webai
print_success "Created provider directories"

# ============================================
# PHASE 6: Set script permissions
# ============================================
print_header "Phase 6: Setting Script Permissions"

chmod +x scripts/*.sh 2>/dev/null || true
chmod +x scripts/dev/*.sh 2>/dev/null || true
chmod +x scripts/install/*.sh 2>/dev/null || true
print_success "Script permissions set"

# ============================================
# INSTALLATION COMPLETE
# ============================================
print_header "Installation Complete!"

print_success "CLIProxyAPI is ready to use"
echo ""
print_info "Next steps:"
echo "  1. Configure your providers in config.yaml"
echo "  2. Start the services: ./scripts/start.sh"
echo "  3. Authenticate providers as needed"
echo "  4. Make API requests to http://localhost:8317"
echo ""
print_info "Useful commands:"
echo "  - Start services:  ./scripts/start.sh"
echo "  - Stop services:   ./scripts/stop.sh"
echo "  - Check status:    ./scripts/status.sh"
echo "  - View logs:       ./scripts/logs.sh"
echo ""
print_success "Happy routing!"

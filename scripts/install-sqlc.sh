#!/bin/bash

# install-sqlc.sh - Cross-platform SQLC installation script
# Automatically detects OS and architecture to download the correct SQLC binary

set -euo pipefail

# Configuration
SQLC_VERSION="1.29.0"
INSTALL_DIR="$HOME/.local/bin"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Detect OS and architecture
detect_platform() {
    local os arch

    os=$(uname -s)
    arch=$(uname -m)

    log_info "Detected OS: $os"
    log_info "Detected Architecture: $arch"

    # Map architecture names to SQLC release naming convention
    case "$arch" in
        x86_64)
            SQLC_ARCH="amd64"
            ;;
        aarch64|arm64)
            SQLC_ARCH="arm64"
            ;;
        *)
            log_error "Unsupported architecture: $arch"
            log_error "Supported architectures: x86_64, aarch64, arm64"
            exit 1
            ;;
    esac

    # Map OS names to SQLC release naming convention
    case "$os" in
        Linux)
            SQLC_OS="linux"
            ;;
        Darwin)
            SQLC_OS="darwin"
            ;;
        *)
            log_error "Unsupported OS: $os"
            log_error "Supported OS: Linux, Darwin (macOS)"
            exit 1
            ;;
    esac

    log_info "SQLC Platform: ${SQLC_OS}_${SQLC_ARCH}"
}

# Download and install SQLC
install_sqlc() {
    local url filename

    url="https://github.com/sqlc-dev/sqlc/releases/download/v${SQLC_VERSION}/sqlc_${SQLC_VERSION}_${SQLC_OS}_${SQLC_ARCH}.tar.gz"
    filename="sqlc_${SQLC_VERSION}_${SQLC_OS}_${SQLC_ARCH}.tar.gz"

    log_info "Downloading SQLC v${SQLC_VERSION} from:"
    log_info "$url"

    # Create installation directory
    mkdir -p "$INSTALL_DIR"

    # Download and extract SQLC
    if curl -sSfL "$url" | tar -xzv -C "$INSTALL_DIR" sqlc; then
        log_success "SQLC binary extracted to $INSTALL_DIR/sqlc"
    else
        log_error "Failed to download or extract SQLC"
        log_error "Please check your internet connection and try again"
        exit 1
    fi

    # Make sure it's executable
    chmod +x "$INSTALL_DIR/sqlc"
}

# Verify installation
verify_installation() {
    local version

    if [[ ! -f "$INSTALL_DIR/sqlc" ]]; then
        log_error "SQLC binary not found at $INSTALL_DIR/sqlc"
        exit 1
    fi

    if ! version=$("$INSTALL_DIR/sqlc" version 2>/dev/null); then
        log_error "SQLC binary is not working correctly"
        exit 1
    fi

    log_success "SQLC installation verified!"
    log_success "Version: $version"
    log_success "Location: $INSTALL_DIR/sqlc"
    
    # Check if it's in PATH
    if ! command -v sqlc >/dev/null 2>&1; then
        log_warning "SQLC is not in your PATH"
        log_warning "Add this to your shell profile (.bashrc, .zshrc, etc.):"
        log_warning "export PATH=\"\$HOME/.local/bin:\$PATH\""
    fi
}

# Main execution
main() {
    log_info "Starting SQLC installation..."
    
    detect_platform
    install_sqlc
    verify_installation
    
    log_success "SQLC installation complete!"
}

# Run main function
main "$@"
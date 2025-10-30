#!/bin/sh

# Discourse TUI Client Installer
# Downloads and installs the latest release from GitHub

set -e

# Configuration
REPO="Sudo-Ivan/discourse-tui-client"
BINARY_NAME="discourse-tui"
INSTALL_DIR="/usr/local/bin"
GITHUB_API="https://api.github.com/repos/${REPO}/releases/latest"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

log_success() {
    echo -e "${GREEN}✓${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

log_error() {
    echo -e "${RED}✗${NC} $1" >&2
}

# Detect platform
detect_platform() {
    OS=$(uname -s | tr 'A-Z' 'a-z')
    ARCH=$(uname -m)

    case $OS in
        linux)
            GOOS="linux"
            ;;
        darwin)
            GOOS="darwin"
            ;;
        freebsd)
            GOOS="freebsd"
            ;;
        openbsd)
            GOOS="openbsd"
            ;;
        *)
            log_error "Unsupported OS: $OS"
            exit 1
            ;;
    esac

    case $ARCH in
        x86_64|amd64)
            GOARCH="amd64"
            ;;
        i386|i686)
            GOARCH="386"
            ;;
        arm64|aarch64)
            GOARCH="arm64"
            ;;
        arm*)
            GOARCH="arm"
            ;;
        *)
            log_error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac

    log_info "Detected platform: ${GOOS}/${GOARCH}"
}

# Check if binary is already installed
check_existing_installation() {
    if which $BINARY_NAME >/dev/null 2>&1; then
        EXISTING_PATH=$(which $BINARY_NAME)
        log_info "Found existing installation at: $EXISTING_PATH"

        # Check if it's in /usr/local/bin
        if [ "$EXISTING_PATH" = "/usr/local/bin/$BINARY_NAME" ]; then
            return 0
        else
            log_warning "Binary found at different location: $EXISTING_PATH"
            echo "This script installs to $INSTALL_DIR"
            printf "Continue anyway? (y/N): "
            read REPLY
            echo
            case $REPLY in
                [Yy]*)
                    ;;
                *)
                    exit 0
                    ;;
            esac
        fi
    else
        log_info "No existing installation found"
    fi
}

# Get latest release info from GitHub
get_latest_release() {
    log_info "Fetching latest release information..."

    if ! which curl >/dev/null 2>&1; then
        log_error "curl is required but not installed"
        exit 1
    fi

    RELEASE_JSON=$(curl -s "$GITHUB_API")

    if [ $? -ne 0 ]; then
        log_error "Failed to fetch release information"
        exit 1
    fi

    # Extract version
    VERSION=$(echo "$RELEASE_JSON" | grep '"tag_name"' | sed 's/.*"tag_name"[ ]*:[ ]*"//;s/".*//' | head -1)

    if [ -z "$VERSION" ]; then
        log_error "Could not determine latest version"
        exit 1
    fi

    log_info "Latest version: $VERSION"

    # Find the correct asset for our platform
    ASSET_NAME="${BINARY_NAME}-${GOOS}-${GOARCH}"
    if [ "$GOARCH" = "arm" ]; then
        ASSET_NAME="${ASSET_NAME}-v7"
    fi

    DOWNLOAD_URL=$(echo "$RELEASE_JSON" | grep "browser_download_url.*${ASSET_NAME}" | sed 's/.*"browser_download_url"[ ]*:[ ]*"//;s/".*//' | head -1)

    if [ -z "$DOWNLOAD_URL" ]; then
        log_error "No binary found for platform ${GOOS}/${GOARCH}"
        exit 1
    fi

    log_info "Download URL: $DOWNLOAD_URL"

    SHA256_URL=$(echo "$RELEASE_JSON" | grep "browser_download_url.*${ASSET_NAME}\.sha256" | sed 's/.*"browser_download_url"[ ]*:[ ]*"//;s/".*//' | head -1)

    if [ -n "$SHA256_URL" ]; then
        log_info "SHA256 checksum URL found: $SHA256_URL"
    else
        log_warning "No SHA256 checksum file found in release"
    fi
}

# Download file with progress
download_file() {
    url=$1
    output=$2

    log_info "Downloading $url..."

    if which wget >/dev/null 2>&1; then
        wget -q --show-progress -O "$output" "$url"
    else
        curl -L -o "$output" "$url"
    fi

    if [ $? -ne 0 ]; then
        log_error "Download failed"
        rm -f "$output"
        exit 1
    fi
}

# Verify SHA256 hash
verify_checksum() {
    file=$1
    expected_hash=$2

    if [ -z "$expected_hash" ]; then
        log_warning "No checksum available for verification"
        return 0
    fi

    if ! which sha256sum >/dev/null 2>&1; then
        log_warning "sha256sum not available, skipping verification"
        return 0
    fi

    actual_hash=$(sha256sum "$file" | awk '{print $1}')

    if [ "$actual_hash" != "$expected_hash" ]; then
        log_error "Checksum verification failed!"
        log_error "Expected: $expected_hash"
        log_error "Actual: $actual_hash"
        rm -f "$file"
        exit 1
    fi

    log_success "Checksum verification passed"
}

# Check if we need to update
check_for_update() {
    if [ ! -f "$INSTALL_DIR/$BINARY_NAME" ]; then
        return 0  # Not installed, need to install
    fi

    log_info "Checking for updates..."

    # Get checksum of installed binary
    if which sha256sum >/dev/null 2>&1; then
        INSTALLED_HASH=$(sha256sum "$INSTALL_DIR/$BINARY_NAME" | awk '{print $1}')

        # Try to get expected hash from checksums file
        if [ -n "$SHA256_URL" ]; then
            download_file "$SHA256_URL" "/tmp/checksums.txt"
            EXPECTED_HASH=$(grep "$ASSET_NAME" "/tmp/checksums.txt" | awk '{print $1}')
            rm -f "/tmp/checksums.txt"

            if [ "$INSTALLED_HASH" = "$EXPECTED_HASH" ]; then
                log_success "Already up to date!"
                exit 0
            else
                log_info "Update available"
            fi
        fi
    else
        log_warning "Cannot verify current version (sha256sum not available)"
    fi
}

# Install binary
install_binary() {
    temp_file="/tmp/${BINARY_NAME}"

    # Download the binary
    download_file "$DOWNLOAD_URL" "$temp_file"

    # Verify checksum if available
    if [ -n "$SHA256_URL" ]; then
        download_file "$SHA256_URL" "/tmp/checksum.sha256"
        EXPECTED_HASH=$(cat "/tmp/checksum.sha256" | tr -d '\n')
        verify_checksum "$temp_file" "$EXPECTED_HASH"
        rm -f "/tmp/checksum.sha256"
    fi

    # Make executable
    chmod +x "$temp_file"

    # Install with sudo
    log_info "Installing to $INSTALL_DIR (requires sudo)..."

    if ! sudo mv "$temp_file" "$INSTALL_DIR/$BINARY_NAME"; then
        log_error "Installation failed"
        rm -f "$temp_file"
        exit 1
    fi

    log_success "Installation completed!"
    log_info "Run '$BINARY_NAME --help' to get started"
}

# Main function
main() {
    echo "Discourse TUI Client Installer"
    echo "=============================="

    detect_platform
    check_existing_installation
    get_latest_release
    check_for_update
    install_binary
}

# Run main function
main "$@"

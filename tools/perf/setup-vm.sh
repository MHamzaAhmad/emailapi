#!/bin/bash

# =============================================================================
# VM Setup Script for Performance Testing
# 
# This script sets up a fresh Linux VM with all the tools needed to run
# performance benchmarks against the SimpleEmailAPI.
#
# Usage: curl -sSL <raw-url> | bash
#    or: ./setup-vm.sh
#
# Tested on: Ubuntu 22.04/24.04, Debian 12
# =============================================================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m'

print_step() {
    echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}$1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

# =============================================================================
# Pre-flight Checks
# =============================================================================
check_os() {
    if [[ ! -f /etc/os-release ]]; then
        print_error "This script requires a Linux system with /etc/os-release"
        exit 1
    fi
    
    source /etc/os-release
    
    if [[ "$ID" != "ubuntu" && "$ID" != "debian" ]]; then
        print_warning "This script is optimized for Ubuntu/Debian. Your OS: $ID"
        print_warning "Some commands may need adjustment."
    fi
    
    echo -e "Detected OS: ${BOLD}$PRETTY_NAME${NC}"
}

# =============================================================================
# Install System Dependencies
# =============================================================================
install_system_deps() {
    print_step "Installing System Dependencies"
    
    sudo apt-get update
    sudo apt-get install -y \
        curl \
        wget \
        git \
        vim \
        jq \
        bc \
        unzip \
        gnupg \
        apt-transport-https \
        ca-certificates \
        software-properties-common
    
    print_success "System dependencies installed"
}

# =============================================================================
# Install k6 (HTTP Load Testing)
# =============================================================================
install_k6() {
    print_step "Installing k6"
    
    if command -v k6 &> /dev/null; then
        print_success "k6 already installed: $(k6 version)"
        return
    fi
    
    # Add k6 GPG key and repository
    sudo gpg --no-default-keyring \
        --keyring /usr/share/keyrings/k6-archive-keyring.gpg \
        --keyserver hkp://keyserver.ubuntu.com:80 \
        --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69 2>/dev/null || true
    
    echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | \
        sudo tee /etc/apt/sources.list.d/k6.list > /dev/null
    
    sudo apt-get update
    sudo apt-get install -y k6
    
    print_success "k6 installed: $(k6 version)"
}

# =============================================================================
# Install ghz (gRPC Load Testing)
# =============================================================================
install_ghz() {
    print_step "Installing ghz"
    
    if command -v ghz &> /dev/null; then
        print_success "ghz already installed: $(ghz --version)"
        return
    fi
    
    # Get latest release version
    local GHZ_VERSION
    GHZ_VERSION=$(curl -sSL https://api.github.com/repos/bojand/ghz/releases/latest | jq -r '.tag_name' | sed 's/v//')
    
    if [[ -z "$GHZ_VERSION" || "$GHZ_VERSION" == "null" ]]; then
        # Fallback version
        GHZ_VERSION="0.120.0"
        print_warning "Could not fetch latest version, using fallback: $GHZ_VERSION"
    fi
    
    local ARCH
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64)  ARCH="x86_64" ;;
        aarch64) ARCH="arm64" ;;
        arm64)   ARCH="arm64" ;;
        *)
            print_error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac
    
    local DOWNLOAD_URL="https://github.com/bojand/ghz/releases/download/v${GHZ_VERSION}/ghz-linux-${ARCH}.tar.gz"
    
    echo "Downloading ghz v${GHZ_VERSION} for ${ARCH}..."
    curl -sSL "$DOWNLOAD_URL" -o /tmp/ghz.tar.gz
    
    sudo tar -xzf /tmp/ghz.tar.gz -C /usr/local/bin ghz
    sudo chmod +x /usr/local/bin/ghz
    rm /tmp/ghz.tar.gz
    
    print_success "ghz installed: $(ghz --version)"
}

# =============================================================================
# Install Go (for proto compilation if needed)
# =============================================================================
install_go() {
    print_step "Installing Go"
    
    if command -v go &> /dev/null; then
        print_success "Go already installed: $(go version)"
        return
    fi
    
    local GO_VERSION="1.23.4"
    
    local ARCH
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64)  ARCH="amd64" ;;
        aarch64) ARCH="arm64" ;;
        arm64)   ARCH="arm64" ;;
    esac
    
    local DOWNLOAD_URL="https://go.dev/dl/go${GO_VERSION}.linux-${ARCH}.tar.gz"
    
    echo "Downloading Go ${GO_VERSION}..."
    curl -sSL "$DOWNLOAD_URL" -o /tmp/go.tar.gz
    
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf /tmp/go.tar.gz
    rm /tmp/go.tar.gz
    
    # Add to PATH for current session
    export PATH=$PATH:/usr/local/go/bin
    
    # Add to shell profile
    if ! grep -q '/usr/local/go/bin' ~/.bashrc 2>/dev/null; then
        echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
    fi
    
    print_success "Go installed: $(go version)"
}

# =============================================================================
# Clone Repository
# =============================================================================
clone_repo() {
    print_step "Setting Up Repository"
    
    local REPO_DIR="$HOME/emailapi"
    
    if [[ -d "$REPO_DIR" ]]; then
        print_warning "Repository already exists at $REPO_DIR"
        echo "Pulling latest changes..."
        cd "$REPO_DIR"
        git pull origin main || true
    else
        echo "Cloning repository..."
        git clone https://github.com/MHamzaAhmad/emailapi.git "$REPO_DIR"
        cd "$REPO_DIR"
    fi
    
    print_success "Repository ready at $REPO_DIR"
}

# =============================================================================
# Setup Environment
# =============================================================================
setup_env() {
    print_step "Setting Up Environment"
    
    local PERF_DIR="$HOME/emailapi/tools/perf"
    
    if [[ ! -f "$PERF_DIR/.env" ]]; then
        if [[ -f "$PERF_DIR/config.env" ]]; then
            cp "$PERF_DIR/config.env" "$PERF_DIR/.env"
            print_warning "Created .env from template. Please edit with your credentials:"
            echo -e "  ${BOLD}cd $PERF_DIR && nano .env${NC}"
        fi
    else
        print_success ".env already exists"
    fi
    
    # Make scripts executable
    chmod +x "$PERF_DIR"/*.sh 2>/dev/null || true
    
    print_success "Scripts are executable"
}

# =============================================================================
# Verify Installation
# =============================================================================
verify_installation() {
    print_step "Verifying Installation"
    
    local all_good=true
    
    echo "Checking installed tools..."
    echo ""
    
    if command -v k6 &> /dev/null; then
        echo -e "  k6:  ${GREEN}✓${NC} $(k6 version 2>&1 | head -1)"
    else
        echo -e "  k6:  ${RED}✗ Not found${NC}"
        all_good=false
    fi
    
    if command -v ghz &> /dev/null; then
        echo -e "  ghz: ${GREEN}✓${NC} $(ghz --version 2>&1 | head -1)"
    else
        echo -e "  ghz: ${RED}✗ Not found${NC}"
        all_good=false
    fi
    
    if command -v go &> /dev/null; then
        echo -e "  go:  ${GREEN}✓${NC} $(go version 2>&1)"
    else
        echo -e "  go:  ${YELLOW}○${NC} Not installed (optional)"
    fi
    
    if command -v jq &> /dev/null; then
        echo -e "  jq:  ${GREEN}✓${NC} $(jq --version 2>&1)"
    else
        echo -e "  jq:  ${RED}✗ Not found${NC}"
        all_good=false
    fi
    
    echo ""
    
    if [[ "$all_good" == "true" ]]; then
        print_success "All required tools are installed!"
    else
        print_error "Some tools failed to install. Please check the errors above."
        exit 1
    fi
}

# =============================================================================
# Print Next Steps
# =============================================================================
print_next_steps() {
    echo ""
    echo -e "${GREEN}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║${NC}${BOLD}                   VM Setup Complete!                      ${NC}${GREEN}║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${BOLD}Next Steps:${NC}"
    echo ""
    echo "1. Configure your environment:"
    echo -e "   ${BLUE}cd ~/emailapi/tools/perf${NC}"
    echo -e "   ${BLUE}nano .env${NC}"
    echo ""
    echo "   Required variables:"
    echo "     PERF_API_KEY=\"your-api-key\""
    echo "     PERF_FROM_EMAIL=\"test@yourverified.domain\""
    echo "     PERF_TO_EMAIL=\"loadtest-sink@example.com\""
    echo ""
    echo "2. Run a quick test to verify connectivity:"
    echo -e "   ${BLUE}./quick-test.sh --dry-run${NC}"
    echo ""
    echo "3. Run the full benchmark suite:"
    echo -e "   ${BLUE}./benchmark.sh --dry-run --quick${NC}"
    echo ""
    echo "4. View results:"
    echo -e "   ${BLUE}ls -la results/${NC}"
    echo ""
    echo -e "${YELLOW}Pro tip:${NC} Run tests from this VM against your API server."
    echo "Make sure this VM is in a different region to get realistic network latency."
    echo ""
}

# =============================================================================
# Main
# =============================================================================
main() {
    echo ""
    echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║${NC}${BOLD}        SimpleEmailAPI - Performance VM Setup            ${NC}${BLUE}║${NC}"
    echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    
    check_os
    install_system_deps
    install_k6
    install_ghz
    install_go
    clone_repo
    setup_env
    verify_installation
    print_next_steps
}

main "$@"

#!/bin/bash
#
# install.sh - One-line installer for gitea-robot
#
# This script downloads and installs gitea-robot with MCP integration
# for Claude Code, Opencode, and Codex CLI.
#
# Usage:
#   curl -fsSL "https://git.terraphim.cloud/terraphim/gitea/raw/branch/main/scripts/install.sh" | bash
#   curl -fsSL ... | bash -s -- --prefix /opt/local
#
# Options:
#   --prefix DIR    Install to DIR/bin instead of /usr/local/bin
#   --no-mcp        Skip MCP integration setup
#   --version VER   Install specific version (default: latest)
#   --help          Show this help message
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
REPO_URL="https://git.terraphim.cloud/terraphim/gitea"
DEFAULT_PREFIX="/usr/local"
VERSION="latest"
SKIP_MCP=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --prefix)
            PREFIX="$2"
            shift 2
            ;;
        --no-mcp)
            SKIP_MCP=true
            shift
            ;;
        --version)
            VERSION="$2"
            shift 2
            ;;
        --help)
            echo "Usage: install.sh [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --prefix DIR    Install to DIR/bin (default: /usr/local)"
            echo "  --no-mcp        Skip MCP integration setup"
            echo "  --version VER   Install specific version (default: latest)"
            echo "  --help          Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Run with --help for usage information"
            exit 1
            ;;
    esac
done

# Function to print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_step() {
    echo -e "${CYAN}[STEP]${NC} $1"
}

# Detect OS and architecture
detect_platform() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)

    case "$os" in
        linux)
            OS="linux"
            ;;
        darwin)
            OS="darwin"
            ;;
        *)
            print_error "Unsupported operating system: $os"
            exit 1
            ;;
    esac

    case "$arch" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        arm64|aarch64)
            ARCH="arm64"
            ;;
        *)
            print_error "Unsupported architecture: $arch"
            exit 1
            ;;
    esac

    PLATFORM="${OS}-${ARCH}"
    print_info "Detected platform: $PLATFORM"
}

# Get latest release version
get_latest_version() {
    if [ "$VERSION" != "latest" ]; then
        echo "$VERSION"
        return
    fi

    # Try to get latest version from GitHub API
    local version
    version=$(curl -sL "${REPO_URL}/releases/latest" 2>/dev/null | grep -o 'tag/v[0-9.]*-robot' | head -1 | sed 's/tag\///' || true)

    if [ -z "$version" ]; then
        version="v1.26.0-robot"  # Fallback version
        print_warning "Could not detect latest version, using $version"
    fi

    echo "$version"
}

# Download binary
download_binary() {
    local version=$1
    local binary_name="gitea-robot-${PLATFORM}"
    local download_url="${REPO_URL}/releases/download/${version}/${binary_name}"

    print_step "Downloading gitea-robot ${version} for ${PLATFORM}..."

    # Create temp directory
    TMP_DIR=$(mktemp -d)
    trap "rm -rf $TMP_DIR" EXIT

    # Download
    if command -v wget &> /dev/null; then
        wget -q --show-progress "$download_url" -O "$TMP_DIR/gitea-robot" 2>&1 || {
            print_error "Failed to download from $download_url"
            print_info "Attempting to build from source instead..."
            return 1
        }
    elif command -v curl &> /dev/null; then
        curl -fsSL --progress-bar "$download_url" -o "$TMP_DIR/gitea-robot" 2>&1 || {
            print_error "Failed to download from $download_url"
            print_info "Attempting to build from source instead..."
            return 1
        }
    else
        print_error "Neither wget nor curl found. Please install one of them."
        exit 1
    fi

    chmod +x "$TMP_DIR/gitea-robot"
    GITEA_ROBOT_PATH="$TMP_DIR/gitea-robot"
    print_success "Downloaded gitea-robot"
}

# Build from source
build_from_source() {
    print_step "Building gitea-robot from source..."

    if ! command -v go &> /dev/null; then
        print_error "Go is not installed. Please install Go 1.21 or later."
        exit 1
    fi

    # Clone repo to temp directory
    TMP_DIR=$(mktemp -d)
    trap "rm -rf $TMP_DIR" EXIT

    print_info "Cloning repository..."
    git clone --depth 1 "${REPO_URL}.git" "$TMP_DIR/repo" &>/dev/null

    print_info "Building binary..."
    cd "$TMP_DIR/repo"
    go build -o "$TMP_DIR/gitea-robot" cmd/gitea-robot/main.go

    GITEA_ROBOT_PATH="$TMP_DIR/gitea-robot"
    print_success "Built gitea-robot from source"
}

# Install binary
install_binary() {
    local install_dir="${PREFIX:-$DEFAULT_PREFIX}/bin"

    print_step "Installing gitea-robot to $install_dir..."

    # Create install directory if needed
    if [ ! -d "$install_dir" ]; then
        print_info "Creating directory: $install_dir"
        mkdir -p "$install_dir" 2>/dev/null || {
            print_warning "Cannot create $install_dir (permission denied)"
            install_dir="$HOME/.local/bin"
            print_info "Falling back to: $install_dir"
            mkdir -p "$install_dir"
        }
    fi

    # Check if we need sudo
    if [ ! -w "$install_dir" ]; then
        print_info "Installing with sudo (need write permission to $install_dir)"
        sudo cp "$GITEA_ROBOT_PATH" "$install_dir/gitea-robot"
        sudo chmod +x "$install_dir/gitea-robot"
    else
        cp "$GITEA_ROBOT_PATH" "$install_dir/gitea-robot"
        chmod +x "$install_dir/gitea-robot"
    fi

    INSTALLED_PATH="$install_dir/gitea-robot"
    print_success "Installed gitea-robot to $INSTALLED_PATH"

    # Check if install_dir is in PATH
    if [[ ":$PATH:" != *":$install_dir:"* ]]; then
        print_warning "$install_dir is not in your PATH"
        print_info "Add the following to your shell profile:"
        echo "  export PATH=\"$install_dir:\$PATH\""
    fi
}

# Get Gitea configuration
get_gitea_config() {
    print_step "Configuring Gitea connection..."

    # Gitea URL
    if [ -n "$GITEA_URL" ]; then
        print_info "Using GITEA_URL from environment: $GITEA_URL"
    else
        read -p "Gitea URL [https://git.terraphim.cloud]: " input_url
        GITEA_URL="${input_url:-https://git.terraphim.cloud}"
    fi

    # Gitea Token
    if [ -n "$GITEA_TOKEN" ]; then
        print_success "Found GITEA_TOKEN in environment"
    elif command -v op &> /dev/null; then
        # Try 1Password
        local token
        token=$(op read "op://TerraphimPlatform/gitea-test-token/credential" 2>/dev/null || true)
        if [ -n "$token" ]; then
            GITEA_TOKEN="$token"
            print_success "Retrieved GITEA_TOKEN from 1Password"
        fi
    fi

    if [ -z "$GITEA_TOKEN" ]; then
        print_warning "GITEA_TOKEN not found"
        echo
        echo "You need a Gitea API token to use gitea-robot."
        echo "Create one at: ${GITEA_URL}/user/settings/applications"
        echo
        read -s -p "Enter Gitea Token (input hidden): " GITEA_TOKEN
        echo

        if [ -z "$GITEA_TOKEN" ]; then
            print_warning "No token provided. You'll need to set GITEA_TOKEN manually before using gitea-robot."
        fi
    fi
}

# Setup MCP integration
setup_mcp() {
    if [ "$SKIP_MCP" = true ]; then
        print_info "Skipping MCP integration setup (--no-mcp specified)"
        return
    fi

    print_step "Setting up MCP integration..."

    local setup_any=false

    # Claude Code
    if [ -d "$HOME/.claude" ] || command -v claude &> /dev/null; then
        print_info "Claude Code detected"
        read -p "Configure MCP for Claude Code? [Y/n]: " response
        if [[ ! "$response" =~ ^[Nn]$ ]]; then
            configure_claude_code
            setup_any=true
        fi
    fi

    # Opencode
    if [ -d "$HOME/.config/opencode" ] || command -v opencode &> /dev/null; then
        print_info "Opencode detected"
        read -p "Configure MCP for Opencode? [Y/n]: " response
        if [[ ! "$response" =~ ^[Nn]$ ]]; then
            configure_opencode
            setup_any=true
        fi
    fi

    # Codex CLI
    if [ -d "$HOME/.codex" ] || command -v codex &> /dev/null; then
        print_info "Codex CLI detected"
        read -p "Configure MCP for Codex CLI? [Y/n]: " response
        if [[ ! "$response" =~ ^[Nn]$ ]]; then
            configure_codex
            setup_any=true
        fi
    fi

    if [ "$setup_any" = false ]; then
        print_info "No MCP-compatible agents detected"
        print_info "MCP integration can be set up later using the integration scripts in scripts/"
    fi
}

# Configure Claude Code
configure_claude_code() {
    local config_dir="$HOME/.claude"
    local config_file="$config_dir/settings.json"

    mkdir -p "$config_dir"

    # Read existing or create new
    if [ -f "$config_file" ]; then
        local content
        content=$(cat "$config_file")
    else
        content='{}'
    fi

    # Add MCP server config
    if command -v jq &> /dev/null; then
        echo "$content" | jq \
            --arg path "$INSTALLED_PATH" \
            --arg url "$GITEA_URL" \
            --arg token "$GITEA_TOKEN" \
            '.mcpServers = (.mcpServers // {}) | .mcpServers["gitea-robot"] = {
                "command": $path,
                "args": ["mcp-server"],
                "env": {
                    "GITEA_URL": $url,
                    "GITEA_TOKEN": $token
                }
            }' > "$config_file.tmp"
        mv "$config_file.tmp" "$config_file"
    else
        print_warning "jq not found, cannot automatically configure Claude Code"
        print_info "Please manually add the following to $config_file:"
        cat << EOF
{
  "mcpServers": {
    "gitea-robot": {
      "command": "$INSTALLED_PATH",
      "args": ["mcp-server"],
      "env": {
        "GITEA_URL": "$GITEA_URL",
        "GITEA_TOKEN": "$GITEA_TOKEN"
      }
    }
  }
}
EOF
    fi

    print_success "Configured Claude Code MCP integration"
}

# Configure Opencode
configure_opencode() {
    local config_dir="$HOME/.config/opencode"
    local config_file="$config_dir/opencode.json"

    mkdir -p "$config_dir"

    # Read existing or create new
    if [ -f "$config_file" ]; then
        local content
        content=$(cat "$config_file")
    else
        content='{"$schema": "https://opencode.ai/config.json"}'
    fi

    # Add MCP server config
    if command -v jq &> /dev/null; then
        echo "$content" | jq \
            --arg path "$INSTALLED_PATH" \
            --arg url "$GITEA_URL" \
            --arg token "$GITEA_TOKEN" \
            '.mcp = (.mcp // {}) | .mcp["gitea-robot"] = {
                "type": "local",
                "command": [$path, "mcp-server"],
                "env": {
                    "GITEA_URL": $url,
                    "GITEA_TOKEN": $token
                }
            }' > "$config_file.tmp"
        mv "$config_file.tmp" "$config_file"
    else
        print_warning "jq not found, cannot automatically configure Opencode"
        print_info "Please manually add the following to $config_file:"
        cat << EOF
{
  "\$schema": "https://opencode.ai/config.json",
  "mcp": {
    "gitea-robot": {
      "type": "local",
      "command": ["$INSTALLED_PATH", "mcp-server"],
      "env": {
        "GITEA_URL": "$GITEA_URL",
        "GITEA_TOKEN": "$GITEA_TOKEN"
      }
    }
  }
}
EOF
    fi

    print_success "Configured Opencode MCP integration"
}

# Configure Codex CLI
configure_codex() {
    local config_dir="$HOME/.codex"
    local config_file="$config_dir/config.toml"

    mkdir -p "$config_dir"

    # Create MCP config block
    local mcp_config
    mcp_config=$(cat << EOF

# Gitea Robot MCP Server
[mcp_servers.gitea_robot]
command = "$INSTALLED_PATH"
args = ["mcp-server"]
enabled = true

[mcp_servers.gitea_robot.env]
GITEA_URL = "$GITEA_URL"
GITEA_TOKEN = "$GITEA_TOKEN"
EOF
)

    if [ -f "$config_file" ]; then
        # Check if already configured
        if grep -q "\[mcp_servers.gitea_robot\]" "$config_file"; then
            print_warning "gitea_robot already configured in Codex CLI config"
            print_info "Please manually update $config_file"
        else
            # Append
            echo "$mcp_config" >> "$config_file"
            print_success "Configured Codex CLI MCP integration"
        fi
    else
        # Create new
        cat > "$config_file" << EOF
# Codex CLI Configuration
gitea$ mcp_config
EOF
        print_success "Configured Codex CLI MCP integration"
    fi
}

# Create shell aliases
setup_aliases() {
    print_step "Setting up shell aliases..."

    local shell_rc
    if [ -n "$ZSH_VERSION" ]; then
        shell_rc="$HOME/.zshrc"
    elif [ -n "$BASH_VERSION" ]; then
        shell_rc="$HOME/.bashrc"
    else
        shell_rc="$HOME/.profile"
    fi

    # Check if aliases already exist
    if grep -q "alias gr=" "$shell_rc" 2>/dev/null; then
        print_info "Aliases already exist in $shell_rc"
        return
    fi

    cat >> "$shell_rc" << 'EOF'

# Gitea Robot aliases
alias gr='gitea-robot'
alias gr-triage='gitea-robot triage'
alias gr-ready='gitea-robot ready'
alias gr-graph='gitea-robot graph'
EOF

    print_success "Added aliases to $shell_rc"
    print_info "Run 'source $shell_rc' to use aliases in current session"
}

# Print final summary
print_summary() {
    echo
    echo "=========================================="
    echo "  Installation Complete!"
    echo "=========================================="
    echo
    echo -e "${GREEN}gitea-robot${NC} installed to: ${CYAN}$INSTALLED_PATH${NC}"
    echo
    echo "Environment Variables:"
    echo "  GITEA_URL=$GITEA_URL"
    if [ -n "$GITEA_TOKEN" ]; then
        echo "  GITEA_TOKEN=*** (set)"
    else
        echo "  GITEA_TOKEN=${YELLOW}(not set)${NC}"
    fi
    echo
    echo "Quick Start:"
    echo "  # Get prioritized tasks"
    echo "  gitea-robot triage --owner OWNER --repo REPO"
    echo
    echo "  # Get ready tasks"
    echo "  gitea-robot ready --owner OWNER --repo REPO"
    echo
    echo "  # Start MCP server"
    echo "  gitea-robot mcp-server"
    echo
    echo "Shell Aliases (after restarting shell or running 'source ~/.bashrc'):"
    echo "  gr           -> gitea-robot"
    echo "  gr-triage    -> gitea-robot triage"
    echo "  gr-ready     -> gitea-robot ready"
    echo "  gr-graph     -> gitea-robot graph"
    echo
    echo "Documentation:"
    echo "  README: ${REPO_URL}/blob/main/README.md"
    echo "  Issues: ${REPO_URL}/issues"
    echo

    if [ -n "$GITEA_TOKEN" ]; then
        echo -e "${GREEN}You're all set!${NC} Try running:"
        echo "  gitea-robot triage --owner terraphim --repo gitea --format markdown"
    else
        echo -e "${YELLOW}Next Steps:${NC}"
        echo "  1. Get a Gitea API token from: ${GITEA_URL}/user/settings/applications"
        echo "  2. Set the token: export GITEA_TOKEN=your_token_here"
        echo "  3. Start using gitea-robot!"
    fi
    echo
}

# Main installation flow
main() {
    echo "=========================================="
    echo "  Gitea Robot Installer"
    echo "=========================================="
    echo

    detect_platform

    local version
    version=$(get_latest_version)

    # Try to download, fallback to build
    if ! download_binary "$version"; then
        build_from_source
    fi

    install_binary
    get_gitea_config
    setup_mcp
    setup_aliases
    print_summary
}

# Run main function
main "$@"

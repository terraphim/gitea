#!/bin/bash
#
# integrate_claude_code.sh - Configure Claude Code to use gitea-robot MCP server
#
# This script detects the gitea-robot binary location and configures
# Claude Code to use it as an MCP server for Gitea PageRank workflow.
#
# Usage: ./integrate_claude_code.sh [gitea-robot-path]
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default Gitea URL
DEFAULT_GITEA_URL="https://git.terraphim.cloud"

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

# Function to detect gitea-robot binary
detect_gitea_robot() {
    local custom_path="$1"
    
    # Check custom path first
    if [ -n "$custom_path" ] && [ -x "$custom_path" ]; then
        echo "$custom_path"
        return 0
    fi
    
    # Check common locations
    local paths=(
        "./gitea-robot"
        "$(pwd)/gitea-robot"
        "$HOME/projects/terraphim/gitea/gitea-robot"
        "/usr/local/bin/gitea-robot"
        "/usr/bin/gitea-robot"
        "$(which gitea-robot 2>/dev/null || true)"
    )
    
    for path in "${paths[@]}"; do
        if [ -x "$path" ]; then
            echo "$path"
            return 0
        fi
    done
    
    return 1
}

# Function to get Gitea token from 1Password or environment
get_gitea_token() {
    local token=""
    
    # Check if already set in environment
    if [ -n "$GITEA_TOKEN" ]; then
        echo "$GITEA_TOKEN"
        return 0
    fi
    
    # Try to get from 1Password
    if command -v op &> /dev/null; then
        token=$(op read "op://TerraphimPlatform/gitea-test-token/credential" 2>/dev/null || true)
        if [ -n "$token" ]; then
            echo "$token"
            return 0
        fi
    fi
    
    return 1
}

# Function to create wrapper script for 1Password injection
create_1password_wrapper() {
    local binary_path="$1"
    local gitea_url="$2"
    local wrapper_path="$3"
    
    cat > "$wrapper_path" << 'EOF'
#!/bin/bash
# Gitea Robot MCP Server Wrapper - 1Password Injection
# Generated: DATE_PLACEHOLDER
#
# This wrapper uses 1Password to securely inject GITEA_TOKEN

GITEA_URL="URL_PLACEHOLDER"
export GITEA_URL

exec op run --env-format=dotenv --no-masking -- \
    BINARY_PLACEHOLDER mcp-server
EOF
    
    # Replace placeholders
    sed -i.bak "s|DATE_PLACEHOLDER|$(date '+%Y-%m-%d %H:%M:%S')|g" "$wrapper_path"
    sed -i.bak "s|URL_PLACEHOLDER|$gitea_url|g" "$wrapper_path"
    sed -i.bak "s|BINARY_PLACEHOLDER|$binary_path|g" "$wrapper_path"
    rm -f "${wrapper_path}.bak"
    
    chmod +x "$wrapper_path"
}

# Function to check if 1Password CLI is available and configured
check_1password_available() {
    if ! command -v op &> /dev/null; then
        return 1
    fi
    
    # Check if user is signed in
    if ! op account list &>/dev/null; then
        return 1
    fi
    
    return 0
}

# Main script
echo "=========================================="
echo "Claude Code - gitea-robot MCP Integration"
echo "=========================================="
echo

# Detect gitea-robot binary
print_info "Detecting gitea-robot binary..."
GITEA_ROBOT_PATH=$(detect_gitea_robot "$1")

if [ -z "$GITEA_ROBOT_PATH" ]; then
    print_error "Could not find gitea-robot binary"
    echo
    echo "Please provide the path to the gitea-robot binary:"
    echo "  $0 /path/to/gitea-robot"
    echo
    echo "Or ensure gitea-robot is in your PATH"
    exit 1
fi

print_success "Found gitea-robot at: $GITEA_ROBOT_PATH"

# Verify the binary works
if ! "$GITEA_ROBOT_PATH" --help &>/dev/null; then
    print_error "gitea-robot binary does not appear to be valid"
    exit 1
fi

# Get Gitea configuration
print_info "Configuring Gitea connection..."

# Get Gitea URL
GITEA_URL="${GITEA_URL:-$DEFAULT_GITEA_URL}"
read -p "Gitea URL [$GITEA_URL]: " input_url
GITEA_URL="${input_url:-$GITEA_URL}"

# Ask user for token storage preference
USE_1PASSWORD=false
if check_1password_available; then
    echo
    echo "1Password CLI is available and configured."
    read -p "Use 1Password for secure token injection? [Y/n]: " use_op
    if [[ "$use_op" =~ ^[Nn]$ ]]; then
        USE_1PASSWORD=false
    else
        USE_1PASSWORD=true
        print_success "Will use 1Password for token injection"
    fi
fi

# Get Gitea token if not using 1Password
GITEA_TOKEN=""
if [ "$USE_1PASSWORD" = false ]; then
    GITEA_TOKEN=$(get_gitea_token)
    if [ -z "$GITEA_TOKEN" ]; then
        print_warning "Could not automatically detect GITEA_TOKEN"
        echo
        echo "Please set GITEA_TOKEN in your environment, or enter it now:"
        echo "  export GITEA_TOKEN=your_token_here"
        read -s -p "Gitea Token (input hidden): " GITEA_TOKEN
        echo
        
        if [ -z "$GITEA_TOKEN" ]; then
            print_error "GITEA_TOKEN is required"
            exit 1
        fi
    else
        print_success "Found GITEA_TOKEN"
    fi
fi

# Claude Code configuration
CLAUDE_CONFIG_DIR="${HOME}/.claude"
CLAUDE_SETTINGS="${CLAUDE_CONFIG_DIR}/settings.json"

print_info "Configuring Claude Code MCP server..."

# Create config directory if needed
mkdir -p "$CLAUDE_CONFIG_DIR"

# Read existing settings or create new
if [ -f "$CLAUDE_SETTINGS" ]; then
    print_info "Reading existing Claude Code settings..."
    SETTINGS_CONTENT=$(cat "$CLAUDE_SETTINGS")
else
    print_info "Creating new Claude Code settings file..."
    SETTINGS_CONTENT='{}'
fi

# Create temporary file for jq processing
TEMP_FILE=$(mktemp)
trap "rm -f $TEMP_FILE" EXIT

# Use jq to add/update the MCP server configuration
if command -v jq &> /dev/null; then
    # Use absolute path for the binary
    ABSOLUTE_PATH=$(cd "$(dirname "$GITEA_ROBOT_PATH")" && pwd)/$(basename "$GITEA_ROBOT_PATH")
    
    if [ "$USE_1PASSWORD" = true ]; then
        # Create wrapper script for 1Password injection
        WRAPPER_PATH="${CLAUDE_CONFIG_DIR}/gitea-robot-wrapper.sh"
        create_1password_wrapper "$ABSOLUTE_PATH" "$GITEA_URL" "$WRAPPER_PATH"
        print_success "Created 1Password wrapper script: $WRAPPER_PATH"
        
        echo "$SETTINGS_CONTENT" | jq --arg path "$WRAPPER_PATH" \
            '
            .mcpServers = (.mcpServers // {}) |
            .mcpServers["gitea-robot"] = {
                "command": $path,
                "args": []
            }
        ' > "$TEMP_FILE"
    else
        echo "$SETTINGS_CONTENT" | jq --arg path "$ABSOLUTE_PATH" \
            --arg url "$GITEA_URL" \
            --arg token "$GITEA_TOKEN" \
            '
            .mcpServers = (.mcpServers // {}) |
            .mcpServers["gitea-robot"] = {
                "command": $path,
                "args": ["mcp-server"],
                "env": {
                    "GITEA_URL": $url,
                    "GITEA_TOKEN": $token
                }
            }
        ' > "$TEMP_FILE"
    fi
    
    # Backup existing settings
    if [ -f "$CLAUDE_SETTINGS" ]; then
        cp "$CLAUDE_SETTINGS" "${CLAUDE_SETTINGS}.backup.$(date +%Y%m%d_%H%M%S)"
    fi
    
    # Write new settings
    mv "$TEMP_FILE" "$CLAUDE_SETTINGS"
else
    print_warning "jq not found, using manual JSON construction"
    
    if [ "$USE_1PASSWORD" = true ]; then
        print_error "1Password injection requires jq to create the wrapper script"
        echo "Please install jq: https://jqlang.github.io/jq/download/"
        exit 1
    fi
    
    # Create the MCP server entry
    cat > "$TEMP_FILE" << EOF
{
  "mcpServers": {
    "gitea-robot": {
      "command": "$GITEA_ROBOT_PATH",
      "args": ["mcp-server"],
      "env": {
        "GITEA_URL": "$GITEA_URL",
        "GITEA_TOKEN": "$GITEA_TOKEN"
      }
    }
  }
}
EOF
    
    if [ -f "$CLAUDE_SETTINGS" ]; then
        print_error "Cannot merge with existing settings without jq"
        echo "Please manually add the gitea-robot MCP server to: $CLAUDE_SETTINGS"
        echo
        echo "Add the following to your mcpServers section:"
        cat "$TEMP_FILE"
        exit 1
    else
        mv "$TEMP_FILE" "$CLAUDE_SETTINGS"
    fi
fi

print_success "Claude Code MCP configuration updated"
print_info "Configuration saved to: $CLAUDE_SETTINGS"

echo
echo "=========================================="
echo "Configuration Complete"
echo "=========================================="
echo
echo "The gitea-robot MCP server has been configured for Claude Code."
echo
echo "Environment Variables:"
echo "  GITEA_URL=$GITEA_URL"
if [ "$USE_1PASSWORD" = true ]; then
    echo "  GITEA_TOKEN=<managed by 1Password>"
else
    echo "  GITEA_TOKEN=***"
fi
echo
echo "MCP Server Path: $GITEA_ROBOT_PATH"
echo
echo "Usage Instructions:"
echo "  1. Restart Claude Code to load the new MCP server"
echo "  2. The following tools will be available:"
echo "     - triage    : Get prioritized task list with PageRank scores"
echo "     - ready     : Get unblocked (ready) tasks"
echo "     - graph     : Get dependency graph"
echo "     - add_dep   : Add dependency between issues"
echo
echo "Example queries:"
echo "  - 'What should I work on next in owner/repo?'"
echo "  - 'Show me the dependency graph for owner/repo'"
echo "  - 'Which issues are ready to work on in owner/repo?'"
echo

if [ "$USE_1PASSWORD" = true ]; then
    echo "Security Note: Using 1Password for token injection"
    echo "  Token is retrieved securely from 1Password at runtime"
    echo "  Wrapper script: $WRAPPER_PATH"
    echo
else
    echo "For persistent token storage, add to your shell profile:"
    echo "  export GITEA_URL=\"$GITEA_URL\""
    echo "  export GITEA_TOKEN=\"your_token_here\""
    echo
fi

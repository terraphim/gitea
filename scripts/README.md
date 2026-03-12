# Gitea Robot - Integration Scripts and Installer

This directory contains the one-line installer and integration scripts for gitea-robot, enabling PageRank-powered task management via Gitea's API.

## Quick Start

### One-Line Installer

```bash
curl -fsSL "https://git.terraphim.cloud/terraphim/gitea/raw/branch/main/scripts/install.sh" | bash
```

This installs gitea-robot and optionally configures MCP integration for Claude Code, Opencode, and Codex CLI.

### Manual Setup

```bash
# 1. Install gitea-robot
curl -fsSL ... | bash

# 2. Set environment
export GITEA_URL="https://git.terraphim.cloud"
export GITEA_TOKEN="your_token"

# 3. Use it
gitea-robot triage --owner terraphim --repo gitea
```

## Available Scripts

| Script | Purpose |
|--------|---------|
| `install.sh` | One-line installer with auto-detection |
| `integrate_claude_code.sh` | Configure Claude Code MCP integration |
| `integrate_opencode.sh` | Configure Opencode MCP integration |
| `integrate_codex_cli.sh` | Configure Codex CLI MCP integration |

## Integration Scripts

### integrate_claude_code.sh

Configures **Claude Code** to use the gitea-robot MCP server.

**Configuration Location:** `~/.claude/settings.json`

**Usage:**
```bash
./integrate_claude_code.sh [path/to/gitea-robot]
```

**Features:**
- Auto-detects gitea-robot binary in common locations
- Prompts for Gitea URL (defaults to https://git.terraphim.cloud)
- Automatically retrieves GITEA_TOKEN from 1Password or environment
- Backs up existing settings before modification
- Uses `jq` for safe JSON merging when available

### integrate_opencode.sh

Configures **Opencode** to use the gitea-robot MCP server.

**Configuration Location:** `~/.config/opencode/opencode.json`

**Usage:**
```bash
./integrate_opencode.sh [path/to/gitea-robot]
```

**Features:**
- Verifies Opencode CLI is installed
- Auto-detects gitea-robot binary
- Configures MCP server with `type: "local"`
- Supports environment variables in MCP config
- Backs up existing configuration

### integrate_codex_cli.sh

Configures **Codex CLI** to use the gitea-robot MCP server.

**Configuration Location:** `~/.codex/config.toml`

**Usage:**
```bash
./integrate_codex_cli.sh [path/to/gitea-robot]
```

**Features:**
- Verifies Codex CLI is installed
- Auto-detects gitea-robot binary
- Appends to existing TOML configuration
- Handles new config creation
- Backs up existing configuration

## Prerequisites

1. **gitea-robot binary** - Must be built or downloaded
   ```bash
   # From the gitea project root
   go build -o gitea-robot cmd/gitea-robot/main.go
   ```

2. **GITEA_TOKEN** - API token for Gitea authentication
   - Can be set via environment variable: `export GITEA_TOKEN=your_token`
   - Or stored in 1Password: `op://TerraphimPlatform/gitea-test-token/credential`

3. **jq** (optional) - For safer JSON manipulation in Claude Code and Opencode scripts

## MCP Tools Available

Once configured, the following tools become available in your AI coding assistant:

| Tool | Description |
|------|-------------|
| `triage` | Get prioritized task list with PageRank scores |
| `ready` | Get unblocked (ready) tasks |
| `graph` | Get dependency graph |
| `add_dep` | Add dependency between issues |

## Example Usage

After integration, you can ask your AI assistant:

- "What should I work on next in terraphim/gitea?"
- "Show me the dependency graph for owner/repo"
- "Which issues are ready to work on in myproject/backend?"
- "Add a dependency: issue 5 blocks issue 3 in owner/repo"

## Environment Variables

All scripts configure the following environment variables for the MCP server:

- `GITEA_URL` - The Gitea instance URL (default: https://git.terraphim.cloud)
- `GITEA_TOKEN` - API authentication token

For persistent configuration, add to your shell profile:

```bash
export GITEA_URL="https://git.terraphim.cloud"
export GITEA_TOKEN="your_token_here"
```

## Binary Detection

Scripts search for the gitea-robot binary in the following order:

1. Custom path provided as argument
2. `./gitea-robot` (current directory)
3. `$(pwd)/gitea-robot`
4. `$HOME/projects/terraphim/gitea/gitea-robot`
5. `/usr/local/bin/gitea-robot`
6. `/usr/bin/gitea-robot`
7. `$(which gitea-robot)`

## Troubleshooting

### Script fails to find gitea-robot

Provide the full path as an argument:
```bash
./integrate_claude_code.sh /absolute/path/to/gitea-robot
```

### Token not found

Set it manually before running:
```bash
export GITEA_TOKEN="your_token_here"
./integrate_claude_code.sh
```

### Claude Code settings not updating

The script creates backups before modification. Check for files like:
- `~/.claude/settings.json.backup.YYYYMMDD_HHMMSS`

### Opencode MCP server not showing

Verify with:
```bash
opencode mcp list
```

Restart any running Opencode sessions to load the new MCP server.

### Codex CLI tools not available

Codex CLI caches tools per session. Start a new session or restart the CLI.

## Files Created/Modified

| Assistant | Config File | Backup Pattern |
|-----------|-------------|----------------|
| Claude Code | `~/.claude/settings.json` | `settings.json.backup.YYYYMMDD_HHMMSS` |
| Opencode | `~/.config/opencode/opencode.json` | `opencode.json.backup.YYYYMMDD_HHMMSS` |
| Codex CLI | `~/.codex/config.toml` | `config.toml.backup.YYYYMMDD_HHMMSS` |

## Documentation

- [Complete gitea-robot Documentation](../cmd/gitea-robot/README.md)
- [Gitea PageRank Workflow](../AGENTS.md)
- [gitea-robot Source](../cmd/gitea-robot/main.go)

## External References

- [Claude Code MCP Documentation](https://docs.anthropic.com/en/docs/agents-and-tools/claude-code/mcp)
- [Opencode Documentation](https://opencode.ai/)
- [Codex CLI Documentation](https://github.com/openai/codex)

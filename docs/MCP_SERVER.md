# Gitea Robot MCP Server

Comprehensive documentation for the gitea-robot Model Context Protocol (MCP) server, enabling AI agents to interact with Gitea task management through standardized tools.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Environment Variables](#environment-variables)
- [Available MCP Tools](#available-mcp-tools)
- [JSON-RPC 2.0 Protocol](#json-rpc-20-protocol)
- [Integration Examples](#integration-examples)
- [Troubleshooting](#troubleshooting)

## Overview

The gitea-robot MCP server exposes Gitea Robot AI functionality through the Model Context Protocol (MCP), a standardized interface for AI agent tool integration. The server runs as a stdio-based JSON-RPC 2.0 service that AI agents can connect to for:

- **Task Prioritization**: Get PageRank-scored task recommendations
- **Dependency Management**: View and modify issue dependencies
- **Workflow Automation**: Identify unblocked (ready) tasks automatically

### Architecture

```
AI Agent (Claude, Codex, etc.)
    |
    | JSON-RPC 2.0 over stdio
    v
gitea-robot mcp-server
    |
    | HTTP/JSON API calls
    v
Gitea Robot API (git.terraphim.cloud)
    |
    | Internal queries
    v
Gitea Database & Issue Tracker
```

### Key Features

- **Standardized Protocol**: Uses MCP specification for universal AI agent compatibility
- **PageRank Prioritization**: Tasks ranked by impact on downstream work
- **Dependency Awareness**: Full visibility into issue blocking relationships
- **Zero Configuration**: Works with existing GITEA_URL and GITEA_TOKEN environment variables

## Quick Start

### 1. Build the Binary

```bash
# From the gitea repository root
go build -o gitea-robot cmd/gitea-robot/main.go
```

### 2. Set Environment Variables

```bash
export GITEA_URL="https://git.terraphim.cloud"
export GITEA_TOKEN="your-api-token-here"
```

### 3. Start the MCP Server

```bash
./gitea-robot mcp-server
```

The server will start and listen for JSON-RPC 2.0 messages on stdin, writing responses to stdout.

### 4. Verify Operation

Send a ping request to verify the server is running:

```json
{"jsonrpc": "2.0", "id": 1, "method": "ping"}
```

Expected response:

```json
{"jsonrpc": "2.0", "id": 1, "result": {}}
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `GITEA_URL` | No | `http://localhost:3000` | Base URL of your Gitea instance |
| `GITEA_TOKEN` | Yes | - | API token for authentication |

### Obtaining a GITEA_TOKEN

1. Log into your Gitea instance
2. Go to Settings -> Applications
3. Generate a new token with `repo` and `issue` scopes
4. Copy the token value

### Security Best Practices

- Never commit tokens to version control
- Use environment files or secret management tools
- For local development, consider using 1Password CLI:

```bash
export GITEA_TOKEN=$(op read "op://TerraphimPlatform/gitea-test-token/credential")
```

## Available MCP Tools

The MCP server exposes four tools that map directly to gitea-robot CLI commands:

### 1. `triage` - Get Prioritized Task List

Returns all issues ranked by PageRank score (highest impact first).

**Input Schema:**

```json
{
  "type": "object",
  "properties": {
    "owner": {
      "type": "string",
      "description": "Repository owner"
    },
    "repo": {
      "type": "string",
      "description": "Repository name"
    },
    "format": {
      "type": "string",
      "description": "Output format: json or markdown",
      "default": "json"
    }
  },
  "required": ["owner", "repo"]
}
```

**Example Request:**

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "triage",
    "arguments": {
      "owner": "terraphim",
      "repo": "gitea",
      "format": "json"
    }
  }
}
```

**Example Response:**

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "quick_ref": {
      "total": 42,
      "open": 12,
      "blocked": 3,
      "ready": 5
    },
    "recommendations": [
      {
        "id": 123,
        "index": 1,
        "title": "Implement PageRank algorithm",
        "pagerank": 0.8543,
        "blocked_by": [],
        "blocking": [456, 789]
      }
    ]
  }
}
```

### 2. `ready` - Get Unblocked Tasks

Returns only issues that are ready to work on (not blocked by dependencies).

**Input Schema:**

```json
{
  "type": "object",
  "properties": {
    "owner": {
      "type": "string",
      "description": "Repository owner"
    },
    "repo": {
      "type": "string",
      "description": "Repository name"
    }
  },
  "required": ["owner", "repo"]
}
```

**Example Request:**

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "ready",
    "arguments": {
      "owner": "terraphim",
      "repo": "gitea"
    }
  }
}
```

### 3. `graph` - Get Dependency Graph

Returns the complete dependency graph for a repository.

**Input Schema:**

```json
{
  "type": "object",
  "properties": {
    "owner": {
      "type": "string",
      "description": "Repository owner"
    },
    "repo": {
      "type": "string",
      "description": "Repository name"
    }
  },
  "required": ["owner", "repo"]
}
```

**Example Request:**

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "graph",
    "arguments": {
      "owner": "terraphim",
      "repo": "gitea"
    }
  }
}
```

**Example Response:**

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "nodes": [
      {"id": 1, "title": "Setup project", "status": "closed"},
      {"id": 2, "title": "Implement feature", "status": "open"}
    ],
    "edges": [
      {"from": 1, "to": 2, "type": "blocks"}
    ]
  }
}
```

### 4. `add_dep` - Add Dependency Between Issues

Creates a dependency relationship between two issues.

**Input Schema:**

```json
{
  "type": "object",
  "properties": {
    "owner": {
      "type": "string",
      "description": "Repository owner"
    },
    "repo": {
      "type": "string",
      "description": "Repository name"
    },
    "issue": {
      "type": "integer",
      "description": "Issue ID (the one being blocked)"
    },
    "blocks": {
      "type": "integer",
      "description": "Issue ID that blocks this issue"
    },
    "relates_to": {
      "type": "integer",
      "description": "Issue ID that relates to this issue"
    }
  },
  "required": ["owner", "repo", "issue"]
}
```

**Example Request (blocking dependency):**

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "tools/call",
  "params": {
    "name": "add_dep",
    "arguments": {
      "owner": "terraphim",
      "repo": "gitea",
      "issue": 42,
      "blocks": 10
    }
  }
}
```

**Example Response:**

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "result": "Dependency added successfully\n"
}
```

**Note:** Either `blocks` or `relates_to` must be provided, but not both.

## JSON-RPC 2.0 Protocol

The MCP server implements JSON-RPC 2.0 over stdio with newline-delimited messages.

### Message Format

All messages are single-line JSON objects terminated by a newline character (`\n`).

### Supported Methods

#### `initialize`

Called by the client to initialize the MCP session.

**Request:**

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {},
    "clientInfo": {
      "name": "claude-code",
      "version": "1.0.0"
    }
  }
}
```

**Response:**

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2024-11-05",
    "capabilities": {
      "tools": {}
    },
    "serverInfo": {
      "name": "gitea-robot",
      "version": "1.0.0"
    }
  }
}
```

#### `tools/list`

Returns the list of available tools.

**Request:**

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list"
}
```

**Response:**

Returns a list of all four tools with their input schemas (see [Available MCP Tools](#available-mcp-tools)).

#### `tools/call`

Invokes a specific tool with provided arguments.

**Request:**

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "triage",
    "arguments": {
      "owner": "terraphim",
      "repo": "gitea"
    }
  }
}
```

**Response:**

Returns the tool-specific result or an error object.

#### `ping`

Health check endpoint.

**Request:**

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "ping"
}
```

**Response:**

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "result": {}
}
```

### Error Responses

Error responses follow JSON-RPC 2.0 specification:

```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "error": {
    "code": -32602,
    "message": "Missing required argument: owner"
  }
}
```

**Common Error Codes:**

| Code | Meaning | Description |
|------|---------|-------------|
| -32703 | Parse error | Invalid JSON received |
| -32601 | Method not found | Unknown method or tool name |
| -32602 | Invalid params | Missing or invalid parameters |
| -32603 | Internal error | Server-side error |

## Integration Examples

### Claude Code (Claude Desktop)

Add to your Claude Code configuration (`~/.claude/config.json`):

```json
{
  "mcpServers": {
    "gitea-robot": {
      "command": "/path/to/gitea-robot",
      "args": ["mcp-server"],
      "env": {
        "GITEA_URL": "https://git.terraphim.cloud",
        "GITEA_TOKEN": "your-token-here"
      }
    }
  }
}
```

### Opencode

Add to your Opencode configuration (`~/.opencode/config.json`):

```json
{
  "mcpServers": {
    "gitea-robot": {
      "command": "/path/to/gitea-robot",
      "args": ["mcp-server"],
      "env": {
        "GITEA_URL": "https://git.terraphim.cloud",
        "GITEA_TOKEN": "your-token-here"
      }
    }
  }
}
```

### Codex (OpenAI)

Codex automatically detects MCP servers from common configuration locations. Ensure your configuration is in:

- `~/.codex/config.json`
- `~/.config/codex/config.json`

With the same format as above.

### Generic MCP Client

For any MCP-compatible client, use this connection configuration:

```json
{
  "name": "gitea-robot",
  "transport": "stdio",
  "command": "/path/to/gitea-robot",
  "arguments": ["mcp-server"],
  "environment": {
    "GITEA_URL": "https://git.terraphim.cloud",
    "GITEA_TOKEN": "your-token-here"
  }
}
```

### Manual Testing with Netcat

For debugging, you can manually test the MCP server:

```bash
# Terminal 1: Start the server
export GITEA_URL="https://git.terraphim.cloud"
export GITEA_TOKEN="your-token"
./gitea-robot mcp-server

# Terminal 2: Send JSON-RPC messages
echo '{"jsonrpc":"2.0","id":1,"method":"ping"}' | nc -q 0 localhost stdio
```

## Troubleshooting

### Server Won't Start

**Problem:** `./gitea-robot mcp-server` exits immediately

**Solutions:**

1. Check that `GITEA_TOKEN` is set:
   ```bash
   echo $GITEA_TOKEN
   ```

2. Verify the binary is built:
   ```bash
   go build -o gitea-robot cmd/gitea-robot/main.go
   ```

3. Check for port conflicts (though MCP uses stdio, not TCP ports)

### Authentication Errors

**Problem:** "Error: 401 Unauthorized" in responses

**Solutions:**

1. Verify your token is valid:
   ```bash
   curl -H "Authorization: token $GITEA_TOKEN" \
     "$GITEA_URL/api/v1/user"
   ```

2. Check token scopes (needs `repo` and `issue` access)

3. Ensure token hasn't expired

### JSON-RPC Parse Errors

**Problem:** "Failed to parse JSON" errors

**Solutions:**

1. Ensure messages are single-line JSON
2. Check for proper JSON escaping
3. Verify the newline character (`\n`) is present at message end
4. Use jq to validate JSON:
   ```bash
   echo '{"jsonrpc":"2.0","id":1,"method":"ping"}' | jq .
   ```

### Tool Not Found Errors

**Problem:** "Tool not found: triage"

**Solutions:**

1. Check the tool name spelling (use underscore for `add_dep`)
2. Verify you're using `tools/call` method, not calling the tool name directly
3. List available tools to confirm:
   ```json
   {"jsonrpc":"2.0","id":1,"method":"tools/list"}
   ```

### Missing Required Arguments

**Problem:** "Missing required argument: owner"

**Solutions:**

1. Check the tool's input schema in [Available MCP Tools](#available-mcp-tools)
2. Ensure all required fields are provided in the `arguments` object
3. Verify parameter types (string vs integer)

### Connection Timeouts

**Problem:** Client can't connect to MCP server

**Solutions:**

1. MCP uses stdio, not network ports - ensure client spawns the process correctly
2. Check that the binary path is absolute in configuration
3. Verify environment variables are passed through correctly

### Debug Mode

To see detailed error messages, run the server directly and send requests manually:

```bash
# Terminal 1
export GITEA_URL="https://git.terraphim.cloud"
export GITEA_TOKEN="your-token"
./gitea-robot mcp-server

# Then type requests manually:
{"jsonrpc":"2.0","id":1,"method":"tools/list"}
```

Watch stderr for error messages that don't appear in JSON-RPC responses.

### Getting Help

For additional support:

1. Check the [Gitea Robot CLI Testing Guide](ROBOT_CLI_TESTING.md)
2. Review the [Design Document](../.docs/design-mcp-server-integration.md)
3. Test the underlying API directly:
   ```bash
   curl -H "Authorization: token $GITEA_TOKEN" \
     "$GITEA_URL/api/v1/robot/triage?owner=terraphim&repo=gitea"
   ```

## CLI vs MCP Server

The gitea-robot binary supports both CLI and MCP server modes:

| Mode | Command | Use Case |
|------|---------|----------|
| CLI | `gitea-robot triage --owner X --repo Y` | Manual use, scripts |
| CLI | `gitea-robot ready --owner X --repo Y` | Manual use, scripts |
| CLI | `gitea-robot graph --owner X --repo Y` | Manual use, scripts |
| CLI | `gitea-robot add-dep --owner X --repo Y --issue N --blocks M` | Manual use, scripts |
| MCP | `gitea-robot mcp-server` | AI agent integration |

Both modes use the same environment variables and access the same Gitea Robot API.

## PageRank Explained

The triage tool uses PageRank algorithm to prioritize tasks:

- **Higher score** = More downstream impact
- Tasks that unblock many other tasks get higher priority
- Ready tasks (not blocked) with high PageRank should be tackled first

Example interpretation:

```json
{
  "pagerank": 0.85,
  "blocking": [10, 20, 30]
}
```

This task has a high PageRank (0.85) because it blocks 3 other issues. Completing it will unblock significant downstream work.

## Best Practices

1. **Start with `ready`**: Always check ready tasks first before using `triage`
2. **Use PageRank**: Focus on high PageRank tasks to maximize impact
3. **Keep Dependencies Updated**: Use `add_dep` to maintain accurate dependency chains
4. **Regular Triage**: Run triage daily to stay on top of changing priorities
5. **Graph Visualization**: Use `graph` to understand complex dependency relationships

## License

Copyright 2026 The Gitea Authors. All rights reserved.  
SPDX-License-Identifier: MIT

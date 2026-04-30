# API Reference

This document describes the Web UI REST API exposed by **local-agent** when running in interactive mode (`--interactive`).

The server listens on `http://localhost:5050` by default.

---

## Endpoints

### `GET /`

Serves the Web UI HTML page.

---

### `GET /api/status`

Returns the current agent status.

**Response**

```json
{
  "directory": "/path/to/project",
  "model": "wizardlm2:7b",
  "totalFiles": 42,
  "focusedPath": "main.go"
}
```

| Field | Type | Description |
|---|---|---|
| `directory` | string | Scanned directory path |
| `model` | string | Active LLM model |
| `totalFiles` | number | Number of scanned files |
| `focusedPath` | string | Currently focused file (omitted when not set) |

---

### `GET /api/messages`

Returns the full conversation history.

**Response**

```json
[
  {
    "role": "assistant",
    "content": "Interactive mode started!",
    "timestamp": "2026-04-30T10:00:00Z"
  }
]
```

| Field | Type | Description |
|---|---|---|
| `role` | string | `"user"` or `"assistant"` |
| `content` | string | Message text |
| `timestamp` | string | ISO 8601 timestamp |

---

### `POST /api/chat`

Sends a message or command. Returns the assistant's response.

**Request**

```json
{
  "message": "find security issues in this codebase"
}
```

**Response (success)**

```json
{
  "success": true,
  "message": {
    "role": "assistant",
    "content": "...",
    "timestamp": "2026-04-30T10:00:01Z"
  }
}
```

**Response (error)**

```json
{
  "success": false,
  "error": "Empty message"
}
```

**Built-in commands**

| Command | Description |
|---|---|
| `help` | Show available commands |
| `stats` | Show scan statistics |
| `files` | List all files in scope |
| `model <name>` | Switch to a different LLM model |
| `rescan` | Rescan the directory for changes |
| `focus <path>` | Limit analysis to a single file |
| `focus clear` | Clear file focus |
| `clear` | Clear conversation history |

---

### `POST /api/rescan`

Triggers a directory rescan. Updates the file list without restarting the agent.

**Request body:** none

**Response**

```json
{
  "success": true,
  "message": {
    "role": "assistant",
    "content": "Rescan complete!\n\nFiles found: 42\nFiltered: 5",
    "timestamp": "2026-04-30T10:00:05Z"
  }
}
```

---

### `POST /api/focus`

Sets or clears the focused file. When a focus is active, only that file is included in LLM analysis.

**Request — set focus**

```json
{
  "path": "main.go"
}
```

**Request — clear focus**

```json
{
  "path": ""
}
```

**Response**

```json
{
  "success": true,
  "message": {
    "role": "assistant",
    "content": "Focus set to: main.go",
    "timestamp": "2026-04-30T10:00:03Z"
  }
}
```

---

### `GET /api/progress`

Server-Sent Events (SSE) stream that emits progress messages during LLM analysis.

**Headers returned**

```
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
```

**Events**

Each event is a plain text data line:

```
data: Analyzing main.go...

data: done
```

The stream emits `done` when the current analysis completes, then closes.

---

## CLI Reference

```
Usage: local-agent [flags]

Flags:
  --dir <path>          Directory to analyze (default: .)
  --task <prompt>       Analysis task description
  --focus <file>        Analyze only this file
  --model <name>        LLM model to use (overrides config)
  --host <host:port>    Ollama instance address (default: localhost:11434)
  --config <path>       Path to configuration file
  --interactive         Start interactive mode (terminal UI + web UI)
  --dry-run             List matched files without running analysis
  --no-detect-secrets   Disable secret/sensitive content detection
  --health              Check LLM connectivity
  --list-models         List available LLM models
  --version             Show version
```

**Environment variables**

| Variable | Default | Description |
|---|---|---|
| `AGENT_TOKEN_LIMIT` | `4000` | Max tokens per LLM request |
| `AGENT_CONCURRENT_FILES` | `1` | Number of files analyzed in parallel |

---

## curl Examples

These examples assume `--interactive` is running and the server is listening on `localhost:5050`.

```bash
# Check status
curl http://localhost:5050/api/status

# Send a question
curl -s -X POST http://localhost:5050/api/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "find security issues"}' | jq .

# Switch model
curl -s -X POST http://localhost:5050/api/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "model codellama"}' | jq .

# Trigger a rescan
curl -s -X POST http://localhost:5050/api/rescan | jq .

# Focus on a file
curl -s -X POST http://localhost:5050/api/focus \
  -H "Content-Type: application/json" \
  -d '{"path": "main.go"}' | jq .

# Clear focus
curl -s -X POST http://localhost:5050/api/focus \
  -H "Content-Type: application/json" \
  -d '{"path": ""}' | jq .

# Get conversation history
curl http://localhost:5050/api/messages | jq .

# Stream progress events during an active analysis
curl -N http://localhost:5050/api/progress
```

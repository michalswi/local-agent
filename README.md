<div align="center">

<img src="./img/local-agent.png" alt="logo" width="120">

# ai-powered code analysis tool

[![stars](https://img.shields.io/github/stars/michalswi/local-agent?style=for-the-badge&color=353535)](https://github.com/michalswi/local-agent)
[![forks](https://img.shields.io/github/forks/michalswi/local-agent?style=for-the-badge&color=353535)](https://github.com/michalswi/local-agent/fork)
[![releases](https://img.shields.io/github/v/release/michalswi/local-agent?style=for-the-badge&color=353535)](https://github.com/michalswi/local-agent/releases)

**local-agent** is a Go-based CLI to scan, analyze files, and chat with your codebase using local models (Ollama based)

</div>


## ✨ Features

- 🔍 Smart file scanning
- 💬 Interactive mode (terminal UI + web UI at localhost:5050) with live rescan capability
- ⚡ Concurrent batch processing for large projects
- 🔒 Privacy-first - all processing happens locally
- 🌐 Remote Ollama support via `--host` flag (e.g., `--host 192.168.1.100:11434`)
- 📦 Standalone binary with embedded assets - no external dependencies
- 📊 PCAP file analysis - parse and analyze network traffic captures (.pcap, .pcapng, .cap)
- 📄 PDF file analysis - extract and analyze text from PDF files up to 10MB (requires `AGENT_TOKEN_LIMIT >= 8000`)

## 🚀 Quick Start

**Prerequisites:** Install and configure Ollama (see [Ollama Setup](#-ollama-setup) for recommended settings)

```bash
# Build
make build

# Standard analysis
./local-agent -dir ./myproject -task "find security issues"

# Interactive mode
./local-agent -dir ./myproject --interactive

# Connect to remote Ollama instance
./local-agent -dir ./myproject --host 192.168.1.100:11434 --interactive

# Other commands
./local-agent --health         # Check LLM connection
./local-agent --list-models    # Show available models
```

Once you got familiar with [Ollama Setup](#-ollama-setup) you might try:
```bash
OLLAMA_NUM_PARALLEL=5 ollama serve

AGENT_CONCURRENT_FILES=5 ./local-agent -dir (...)

AGENT_CONCURRENT_FILES=5 ./local-agent --dir (...) --interactive
```

## 📋 Usage

### Standard Mode
```bash
./local-agent -dir . -task "give a one-line description of this file in format <filename>:<description>"

./local-agent -dir <full_path_to_dir> -task <prompt> --model codellama

./local-agent --focus ./cmd/main.go -task "review this file"

# Analyze PCAP files
./local-agent --focus /path/to/capture.pcap -task "summarize network traffic patterns"

# Connect to remote Ollama instance
./local-agent -dir . -task "analyze" --host 192.168.1.100:11434
./local-agent -dir . --interactive --host ollama.example.com:8080
```

### Interactive Mode
```bash
./local-agent -dir . --interactive

./local-agent -dir <full_path_to_dir> --interactive
```

**Web UI:** Opens automatically at http://localhost:5050

**Commands:** `help`, `model <name>`, `rescan`, `stats`, `files`, `focus <path>`, `clear`, `quit`

**Navigation:** `↑/↓` scroll, `Enter` send

**Focus:** `focus <filename>` limits analysis to a single scanned file until you run `focus clear`.


## 🔧 Ollama Setup

```bash
# Install & start
curl https://ollama.ai/install.sh | sh


## Start ollama with recommended settings:
# match OLLAMA_CONTEXT_LENGTH with your AGENT_TOKEN_LIMIT
# match OLLAMA_NUM_PARALLEL with your AGENT_CONCURRENT_FILES
OLLAMA_CONTEXT_LENGTH=32768 OLLAMA_NUM_PARALLEL=10 ollama serve

# Set token limit and concurrent files
# AGENT_TOKEN_LIMIT >= 8000 is required to analyze large files (>100KB), including PDFs
# AGENT_CONCURRENT_FILES controls how many files are sent to Ollama in parallel.
# Each file is sent as a separate LLM request — AGENT_CONCURRENT_FILES=4 means
# 4 requests run simultaneously, not 4 files in one request.
AGENT_TOKEN_LIMIT=30000 AGENT_CONCURRENT_FILES=10 ./local-agent -dir . -task "..."
AGENT_TOKEN_LIMIT=30000 AGENT_CONCURRENT_FILES=10 ./local-agent -dir . --interactive


## OR
# use defaults (context=4096, parallel=1)
# Note: with default AGENT_TOKEN_LIMIT=4000, large files (>100KB) are skipped
ollama serve

# AGENT_TOKEN_LIMIT=4000
# AGENT_CONCURRENT_FILES=1
./local-agent -dir . -task "..."
./local-agent -dir . --interactive
```

## 📁 File Filtering

Default filters in [config/config.go](config/config.go): supports common source files (`.go`, `.js`, `.py`, etc.), configs (`.yaml`, `.json`), and docs (`.pdf`, `.md`, `.txt`). Excludes `node_modules`, `.git`, `.env*`, build artifacts.

See [examples/](examples/) directory for sample configuration files:
- [config.yaml](examples/config.yaml) - Full configuration example with comments
- [.agentignore](examples/.agentignore) - Custom ignore patterns example

Customize in `.agent/config.yaml`:
```yaml
filters:
  respect_gitignore: true
  deny_patterns: ["node_modules/**", "*.log"]
  allow_patterns: ["*.go", "*.js"]  # If set, only these are included
```

### Configuration File

Create `.agent/config.yaml`:

```yaml
agent:
  token_limit: 8000     # Max tokens per request 
                        # (adjust based on OLLAMA_CONTEXT_LENGTH)
  concurrent_files: 10  # Number of concurrent batch requests to Ollama
                        # (adjust based on OLLAMA_NUM_PARALLEL)

llm:
  provider: "ollama"
  endpoint: "http://localhost:11434"
  model: "wizardlm2:7b"   # or codellama, mistral, etc.
  temperature: 0.1

filters:
  respect_gitignore: true
  deny_patterns:
    - "node_modules/**"
    - ".git/**"
    - "*.log"
  allow_patterns:
    - "*.go"
    - "*.js"
    - "*.md"

security:
  detect_secrets: false   # Disabled by default
  skip_binaries: true
  follow_symlinks: false
  max_depth: 20
```

**Increase token limit for local LLMs:**
```yaml
agent:
  token_limit: 32000  # (default is 4000)
  # Verify model's context window size and set accordingly
  # Note: token_limit >= 8000 is required to analyze large files (>100KB)
  # This includes PDF files (up to 10MB) and other large documents
```

**Speed up analysis with concurrent processing:**
```yaml
agent:
  concurrent_files: 10  # Number of concurrent batch requests (default is 1)
  # Set to 1 for sequential processing (OLLAMA_NUM_PARALLEL related)
  # Higher values = faster but more resource usage
```
- **1**: Sequential processing (safest, lowest resource usage)
- **2-4**: Light concurrency (good for laptops or low-resource systems)
- **5-10**: Balanced (recommended for most systems)
- **15-20**: Aggressive (requires powerful machine and Ollama parallel support)

**Adjust temperature for different tasks:**
```yaml
llm:
  temperature: 0.4  # default
```
- **0.0-0.3**: Best for code analysis, security audits, bug finding (deterministic)
- **0.4-0.7**: Good for documentation, explanations, suggestions (balanced)

## 📝 Session Logs

After each run, a session log is saved to `~/.local-agent/` as a timestamped JSON file (e.g., `local-agent-20260424-114914.json`).

Each session log contains:

| Field | Description |
|---|---|
| `timestamp` | When the session started |
| `mode` | `standalone` or `interactive` |
| `directory` | Scanned directory path |
| `task` | The user's prompt/question sent to the LLM |
| `focus` | Focused file path (if `--focus` or `focus <file>` was used) |
| `model` | Ollama model used |
| `tokens_used` | Total tokens consumed |
| `file_tokens` | Per-file token breakdown |
| `duration` | Total processing time |
| `files` | List of files included in the analysis |
| `findings` | Security findings (if secret detection is enabled) |
| `response` | The LLM's full response |
| `scan_summary` | File scan stats (total files, filtered files, size, scan duration) |

> **Note:** Session files persist across reboots in `~/.local-agent/`.

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

## 🚀 Quick Start

**Prerequisites:** Install and configure Ollama (see [Ollama Setup](#-ollama-setup) for recommended settings)

```bash
# Build
make build

# Standard analysis
./local-agent -dir ./myproject -task "find security issues"

# Interactive mode
./local-agent -dir ./myproject --interactive

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
./local-agent -dir . -task "find security issues"

./local-agent -dir <full_path_to_dir> -task "explain the architecture" --model codellama

./local-agent --focus ./cmd/main.go -task "review this file"
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
OLLAMA_CONTEXT_LENGTH=8192 OLLAMA_NUM_PARALLEL=10 ollama serve

# Set token limit and concurrent files
AGENT_TOKEN_LIMIT=8000 AGENT_CONCURRENT_FILES=10 ./local-agent -dir . -task "..."
AGENT_TOKEN_LIMIT=8000 AGENT_CONCURRENT_FILES=10 ./local-agent -dir . --interactive


## OR
# use defaults (context=4096, parallel=1)
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
  temperature: 0.1  # Default - precise, deterministic (code analysis, security)
  # temperature: 0.7  # Creative, varied (documentation, explanations)
```
- **0.0-0.3**: Best for code analysis, security audits, bug finding (deterministic)
- **0.4-0.7**: Good for documentation, explanations, suggestions (balanced)

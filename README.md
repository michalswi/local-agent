# Local Agent - AI-powered code analysis tool

Scan, analyze files, and chat with your codebase using local LLMs (Ollama)

## ✨ Features

- 🔍 Smart file scanning with .gitignore support and automatic batching
- 🛡️ Security-aware - detects and sanitizes secrets/PII before LLM submission
- 💬 Interactive mode with live rescan capability
- ⚡ Concurrent batch processing for large projects
- 💾 Auto-save results with `--view-last` support
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
./local-agent --view-last      # View previous results
./local-agent --health         # Check LLM connection
./local-agent --list-models    # Show available models
```

## 📋 Usage

### Standard Mode
```bash
./local-agent -dir . -task "find security issues"
./local-agent -dir . -task "explain the architecture" --model codellama
```

### Interactive Mode
```bash
./local-agent -dir ./myproject --interactive
```

**Commands:** `help`, `model <name>`, `rescan`, `stats`, `files`, `last`, `clear`, `quit`

**Navigation:** `↑/↓` scroll, `Enter` send

**Examples:**
- "Find all TODO comments"
- "Explain main.go"
- "model codellama" (switch models on the fly)
- "Fix security issues in auth.go"

## 🔧 Ollama Setup

```bash
# Install & start
curl https://ollama.ai/install.sh | sh

# Start with recommended settings for local-agent
# Match OLLAMA_CONTEXT_LENGTH with your AGENT_TOKEN_LIMIT
# Match OLLAMA_NUM_PARALLEL with your AGENT_CONCURRENT_FILES
OLLAMA_CONTEXT_LENGTH=8192 OLLAMA_NUM_PARALLEL=5 ollama serve

# Set token limit and concurrent files without config file
AGENT_TOKEN_LIMIT=8000 AGENT_CONCURRENT_FILES=5 ./local-agent -dir . -task "..."
# Default values if not set:
# AGENT_TOKEN_LIMIT=4000
# AGENT_CONCURRENT_FILES=1

# Or use defaults (context=4096, parallel=1)
ollama serve
```

## 📁 File Filtering

Default filters in [config/config.go](config/config.go): supports common source files (`.go`, `.js`, `.py`, etc.), configs (`.yaml`, `.json`), and docs (`.md`, `.txt`). Excludes `node_modules`, `.git`, `.env*`, build artifacts.

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
  token_limit: 4000     # Max tokens per request (adjust based on OLLAMA_CONTEXT_LENGTH)
  concurrent_files: 10  # Number of concurrent batch requests to Ollama

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
  detect_secrets: false   # Disabled by default (prevents false positives on docs)
  skip_binaries: true
  follow_symlinks: false
  max_depth: 20
```

**Increase token limit for local LLMs:**
```yaml
agent:
  token_limit: 32000  # Default is 24000
  # Verify model's context window size and set accordingly
```

**Speed up analysis with concurrent processing:**
```yaml
agent:
  concurrent_files: 10  # Number of concurrent batch requests (default: 10)
  # Set to 1 for sequential processing
  # Higher values = faster but more resource usage
```
- **1**: Sequential processing (safest, lowest resource usage)
- **5-10**: Balanced (recommended for most systems)
- **15-20**: Aggressive (requires powerful machine and Ollama parallel support)

**Important:** Match Ollama's parallel processing capacity:
```bash
# Enable Ollama to process multiple requests simultaneously
export OLLAMA_NUM_PARALLEL=10  # Match your concurrent_files setting
ollama serve
```
Without `OLLAMA_NUM_PARALLEL`, Ollama processes requests sequentially (queues extras). Effective speedup = **min(concurrent_files, OLLAMA_NUM_PARALLEL)**.

**Adjust temperature for different tasks:**
```yaml
llm:
  temperature: 0.1  # Default - precise, deterministic (code analysis, security)
  # temperature: 0.7  # Creative, varied (documentation, explanations)
  # temperature: 0.0  # Most deterministic (factual extraction, parsing)
```
- **0.0-0.3**: Best for code analysis, security audits, bug finding (deterministic)
- **0.4-0.7**: Good for documentation, explanations, suggestions (balanced)
- **Concurrent processing**: Set `concurrent_files: 4-10` and match with `OLLAMA_NUM_PARALLEL`
- **Large projects**: Auto-batching handles projects exceeding token limit
- **Security**: Secrets/PII automatically detected and sanitized
- **Rescan**: Use `rescan` in interactive mode after code changes
- **View later**: `./local-agent --view-last` or type `last` in interactive mode
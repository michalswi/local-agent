# Local Agent - AI-powered code analysis tool

![](https://img.shields.io/github/stars/michalswi/local-agent)
![](https://img.shields.io/github/last-commit/michalswi/local-agent)
![](https://img.shields.io/github/forks/michalswi/local-agent)
![](https://img.shields.io/github/issues/michalswi/local-agent)

Scan, analyze files, and chat with your codebase using local LLMs

## ✨ Features

- 🔍 **Smart File Scanning** - Respects .gitignore, detects file types, handles large codebases
- 🤖 **LLM Analysis** - Works with Ollama (local) or cloud LLMs for code analysis
- 💬 **Interactive Mode** - Chat with your codebase using a beautiful TUI
- 🔒 **Privacy First** - All scanning happens locally, you control what's sent to LLM
- 📊 **Token Management** - Automatic batching for large projects
- 💾 **Auto-Save Results** - Analysis saved to temp file for easy review


## 🚀 Quick Start

```bash
# make
$ make
Available targets:
  make build         - Build binary for current platform
  make build-macos   - Build binary for macOS (arm64 and amd64)
  make build-linux   - Build binary for Linux (amd64 and arm64)
  make all           - Clean and build for current platform
  make clean         - Remove build artifacts
  make test          - Run tests
  make fmt           - Format code
  make vet           - Run go vet
```

```bash
# Build
make build
or
go build -o local-agent

# Analyze a directory
./local-agent -dir ./myproject -task "find security issues"

# Interactive mode
./local-agent -dir ./myproject --interactive

# View last analysis
./local-agent --view-last
```

## 📋 Usage

### Standard Analysis
```bash
# Basic analysis
./local-agent -dir /path/to/code -task "your task here"

# Examples:
./local-agent -dir . -task "find all TODO comments"
./local-agent -dir ./src -task "list security vulnerabilities"
./local-agent -dir . -task "explain the architecture"

# Use different model
./local-agent -dir . -task "code review" --model codellama

# View results later
./local-agent --view-last
```

### Interactive Mode
```bash
./local-agent -dir ./myproject --interactive
```

**Available commands:**
- `help` - Show help
- `model <name>` - Switch to different model (e.g., `model codellama`)
- `stats` - Scan statistics
- `files` - List scanned files
- `last` - View previous analysis
- `clear` - Clear history
- `quit` - Exit

**Ask questions naturally:**
- "Find all TODO comments"
- "What are the main components?"
- "Explain main.go"
- "List API endpoints"

**Switch models on the fly:**
```
> model codellama
✅ Model switched: wizardlm2:7b → codellama

> find security issues
[analysis with codellama...]

> model mistral
✅ Model switched: codellama → mistral
```

**Navigation:**
- `↑` Scroll up
- `↓` Scroll down
- `Enter` Send message

### Other Commands
```bash
./local-agent -health                    # Check LLM connection
./local-agent -list-models               # Show available models
./local-agent -dry-run -dir .            # Preview files (no analysis)
./local-agent -version                   # Show version
./local-agent --model <name> -dir . -task "analyze"  # Use specific model
```

## ⚙️ Configuration

Create `.agent/config.yaml`:

```yaml
agent:
  token_limit: 32000      # Increase for local LLMs (default: 8000)
  concurrent_files: 10

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
  detect_secrets: true
  skip_binaries: true
```

## 🔧 Setup with Ollama

```bash
# Install Ollama
curl https://ollama.ai/install.sh | sh

# Start Ollama
ollama serve

# Pull models
ollama pull wizardlm2:7b    # Default model
ollama pull codellama        # Code-focused
ollama pull mistral          # Fast general model

# Use default model (wizardlm2:7b)
./local-agent -dir . -task "analyze code"

# Standard mode: specify model with flag
./local-agent -dir . -task "analyze code" --model codellama

# Interactive mode: switch models anytime with 'model <name>' command
./local-agent -dir . --interactive
> model codellama
> analyze this code
```

## 🏗️ Architecture

```
CLI → Scanner → Analyzer → LLM Client
         ↓         ↓
      Filters   Chunker
```

- **Scanner**: Walks directories, filters files
- **Analyzer**: Processes files by size (small/medium/large)
- **LLM Client**: Sends context to LLM, handles responses
- **TUI**: Interactive chat interface (optional)

See [ARCHITECTURE.md](ARCHITECTURE.md) for detailed design.

## 📦 Project Structure

```
local-agent/
├── main.go              # CLI entry point
├── config/              # Configuration
├── filter/              # File filtering
├── analyzer/            # File analysis
├── llm/                 # LLM client
├── tui/                 # Interactive mode
├── types/               # Shared types
└── security/            # Validation
```

## 💡 Tips

**Increase token limit for local LLMs:**
```yaml
agent:
  token_limit: 32000  # Default is 8000
```

**Adjust temperature for different tasks:**
```yaml
llm:
  temperature: 0.1  # Default - precise, deterministic (code analysis, security)
  # temperature: 0.7  # Creative, varied (documentation, explanations)
  # temperature: 0.0  # Most deterministic (factual extraction, parsing)
```
- **0.0-0.3**: Best for code analysis, security audits, bug finding (deterministic)
- **0.4-0.7**: Good for documentation, explanations, suggestions (balanced)
- **0.8-1.0**: Creative tasks, brainstorming (more varied, less predictable)

**Review analysis later:**
```bash
./local-agent -dir . -task "audit"
# ... results displayed ...

# View again later
./local-agent --view-last
# or in interactive mode: type 'last'
```

**Large projects:**
- Agent automatically batches files if they exceed token limit
- Adjust `token_limit` based on your model's context window
- Use filters to focus on specific file types

## 📝 Example Workflows

**Security Audit:**
```bash
./local-agent -dir . -task "find security vulnerabilities"
```

**Documentation:**
```bash
./local-agent -dir ./src -task "generate API documentation"
```

**Code Review:**
```bash
./local-agent --interactive -dir .
> explain the authentication flow
> find potential bugs in auth.go
> suggest improvements
```

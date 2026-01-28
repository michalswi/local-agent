# Local Agent - File Verification Tool

A local agent tool written in Go that verifies and analyzes files on a user's laptop using LLM APIs (Ollama/Claude/Codex-style).

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         CLI Interface                        │
│                         (main.go)                           │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│                     Agent Orchestrator                       │
│  - Coordinates file scanning, filtering, and LLM calls      │
│  - Maintains local state (never sent to LLM)                │
└──┬──────────────┬─────────────────┬────────────────────┬───┘
   │              │                 │                    │
   ▼              ▼                 ▼                    ▼
┌──────┐   ┌──────────┐   ┌─────────────┐   ┌────────────────┐
│ File │   │  Filter  │   │   Analyzer  │   │   LLM Client   │
│ Walk │   │  Engine  │   │   Engine    │   │   (Ollama)     │
└──────┘   └──────────┘   └─────────────┘   └────────────────┘
   │              │                 │                    │
   │              │                 │                    │
   ▼              ▼                 ▼                    ▼
┌──────┐   ┌──────────┐   ┌─────────────┐   ┌────────────────┐
│ Dir  │   │ Rules    │   │ Chunker     │   │ HTTP Client    │
│ Tree │   │ .ignore  │   │ Token Count │   │ JSON Protocol  │
└──────┘   └──────────┘   └─────────────┘   └────────────────┘
```

## Package Structure

```
local-agent/
├── main.go                    # Entry point, CLI handling
├── go.mod                     # Go module definition
├── go.sum                     # Dependency checksums
├── config/
│   ├── config.go             # Configuration loading/management
│   └── rules.go              # Allowlist/denylist rule definitions
├── filter/
│   ├── filter.go             # File filtering logic
│   ├── ignore.go             # .gitignore-style parsing
│   └── matcher.go            # Pattern matching
├── analyzer/
│   ├── analyzer.go           # File analysis orchestration
│   ├── detector.go           # File type/size detection
│   ├── chunker.go            # Content chunking for large files
│   └── summarizer.go         # Summary generation logic
├── llm/
│   ├── client.go             # LLM client interface
│   ├── ollama.go             # Ollama-specific implementation
│   ├── types.go              # Request/response types
│   └── tokenizer.go          # Token counting utilities
├── types/
│   └── types.go              # Shared types and constants
├── security/
│   ├── validator.go          # Path and content validation
│   └── sanitizer.go          # Content sanitization
└── examples/
    ├── .agentignore          # Example ignore rules
    └── config.yaml           # Example configuration
```

## Key Design Principles

### 1. **Privacy First**
- All file scanning and filtering happens locally
- Only explicitly selected content is sent to LLM
- No automatic file uploads or background syncing
- User approval required for directory traversal

### 2. **Stateless LLM Interaction**
- LLM receives only sanitized, approved content
- All context and state managed locally
- Each LLM call is independent
- Local cache for repeat queries (optional)

### 3. **Intelligent Content Selection**
- Small files (<10KB): send full content
- Medium files (10KB-100KB): send with chunking option
- Large files (>100KB): generate summary + selectable chunks
- Binary files: metadata only

### 4. **Security Safeguards**
- Path traversal prevention
- Allowlist/denylist enforcement
- Sensitive file detection (.env, secrets, keys)
- Configurable file size limits
- Content sanitization before LLM calls

## Example Flow: "Verify This Folder"

```
1. User Input:
   $ local-agent verify /path/to/project --task "check for security issues"

2. Configuration Load:
   - Load .agentignore rules (deny: node_modules, .git, *.log)
   - Load allowlist rules (allow: *.go, *.js, *.md)
   - Set token limit (default: 8000 tokens)

3. Directory Traversal:
   - Walk directory tree
   - Apply filter rules at each level
   - Detect file types and sizes
   - Build file inventory (stored locally)

4. File Analysis:
   For each file:
   - Check size: 2KB → small, send full content
   - Check type: .go → text, proceed
   - Read content: "package main..."
   - Token count: 150 tokens → OK
   - Add to batch

5. Content Preparation:
   Batch 1: [file1.go (150 tok), file2.go (200 tok), file3.md (100 tok)]
   Total: 450 tokens + prompt overhead = ~600 tokens
   
   For large file (main.go, 15KB):
   - Generate summary: "Main application entry point, HTTP server..."
   - Create chunks: [chunk1 (lines 1-100), chunk2 (lines 101-200)...]
   - User can request specific chunks

6. LLM Request:
   POST http://localhost:11434/api/chat
   {
     "model": "codellama",
     "messages": [{
       "role": "user",
       "content": "Check these files for security issues:\n\nFile: src/auth.go\n```\n<content>...\n```\n..."
     }],
     "stream": false
   }

7. Response Processing:
   - Receive LLM analysis
   - Parse findings
   - Display to user with file references
   - Offer follow-up options (refine, check specific file, etc.)

8. Follow-up (if needed):
   User: "Show me the details in auth.go lines 45-60"
   - Retrieve chunk from local inventory
   - Send specific chunk to LLM
   - Get detailed analysis
```

## Security & Privacy Safeguards

### Access Control
- **Explicit Directory Approval**: User must specify directories to scan
- **Ignore Rules**: Respect .gitignore, .agentignore patterns
- **Path Validation**: Prevent directory traversal attacks (../)
- **Symlink Handling**: Configurable symlink following with loop detection

### Content Protection
- **Sensitive Pattern Detection**: Scan for API keys, passwords, tokens
- **Binary File Exclusion**: Skip non-text files by default
- **Size Limits**: Reject files exceeding configured limits
- **PII Detection**: Optional scanning for emails, SSNs, etc.

### LLM Communication
- **Local-First**: All processing happens locally
- **Explicit Consent**: User approves what gets sent
- **No Telemetry**: No usage tracking or analytics
- **Network Isolation**: Can run fully offline with local Ollama

### Data Handling
- **No Persistence**: Agent doesn't store file contents
- **Memory Management**: Clear buffers after processing
- **Temp File Cleanup**: Remove any temporary files
- **Audit Logging**: Optional local log of what was sent to LLM

## Configuration Example

```yaml
# .agent/config.yaml
agent:
  max_file_size_bytes: 1048576  # 1MB
  token_limit: 8000
  concurrent_files: 10
  
llm:
  provider: "ollama"
  endpoint: "http://localhost:11434"
  model: "codellama"
  temperature: 0.1
  
filters:
  respect_gitignore: true
  custom_ignore_file: ".agentignore"
  
  deny_patterns:
    - "node_modules/**"
    - ".git/**"
    - "*.log"
    - "*.tmp"
    - ".env*"
    
  allow_patterns:
    - "*.go"
    - "*.js"
    - "*.ts"
    - "*.md"
    - "*.yaml"
    
security:
  detect_secrets: true
  skip_binaries: true
  follow_symlinks: false
  max_depth: 20
  
chunking:
  strategy: "smart"  # smart, lines, tokens
  chunk_size: 1000   # tokens per chunk
  overlap: 100       # token overlap between chunks
```

## Usage Examples

```bash
# Verify entire project
local-agent verify . --task "security audit"

# Analyze specific file
local-agent analyze src/main.go --question "explain this code"

# Interactive mode
local-agent interactive /path/to/project

# Scan with custom config
local-agent verify . --config custom-rules.yaml

# List what would be analyzed (dry-run)
local-agent verify . --dry-run

# Export file inventory
local-agent scan . --output inventory.json
```

## Integration with Ollama

The tool is designed to work seamlessly with Ollama for local LLM inference:

```bash
# Start Ollama
ollama serve

# Pull a code-focused model
ollama pull codellama

# Run agent with Ollama
local-agent verify . --llm ollama --model codellama
```

## Performance Considerations

- **Parallel Processing**: Concurrent file reading and analysis
- **Streaming**: Support for streaming LLM responses
- **Caching**: Optional local cache for repeat queries
- **Incremental Analysis**: Process files in batches
- **Progress Indicators**: Real-time feedback during scanning

## Future Enhancements

- [ ] Multi-model support (Claude, OpenAI, local models)
- [ ] Interactive refinement mode
- [ ] Diff analysis for code changes
- [ ] Project-level insights and trends
- [ ] Plugin system for custom analyzers
- [ ] Web UI for visualization

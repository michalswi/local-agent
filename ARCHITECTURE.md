# Architecture Diagram

## System Flow

```
┌──────────────────────────────────────────────────────────────────┐
│                         USER REQUEST                              │
│          "Verify this folder for security issues"                 │
└────────────────────────────┬─────────────────────────────────────┘
                             │
                             ▼
┌────────────────────────────────────────────────────────────────────┐
│                    CLI INTERFACE (main.go)                         │
│  • Parse flags (--dir, --task, --config, --dry-run)              │
│  • Load configuration                                              │
│  • Initialize components                                           │
└────────────────────────────┬───────────────────────────────────────┘
                             │
                             ▼
┌────────────────────────────────────────────────────────────────────┐
│                 AGENT ORCHESTRATOR                                 │
│  Local State Management (all context kept locally):               │
│  • File inventory                                                  │
│  • Analysis results                                                │
│  • Conversation history                                            │
│  • User preferences                                                │
└───┬──────────────┬─────────────────┬────────────────┬─────────────┘
    │              │                 │                │
    │              │                 │                │
    ▼              ▼                 ▼                ▼
┌────────┐  ┌─────────────┐  ┌──────────────┐  ┌────────────────┐
│ FILTER │  │  ANALYZER   │  │  SECURITY    │  │  LLM CLIENT    │
│ ENGINE │  │  ENGINE     │  │  VALIDATOR   │  │  (Ollama)      │
└────┬───┘  └──────┬──────┘  └──────┬───────┘  └───────┬────────┘
     │             │                 │                  │
     │             │                 │                  │
     ▼             ▼                 ▼                  ▼
┌─────────────────────────────────────────────────────────────┐
│                    DETAILED FLOW                             │
└─────────────────────────────────────────────────────────────┘
```

## Component Interaction Flow

### 1. Directory Scanning Phase

```
User Specifies Directory
    │
    ▼
┌───────────────────────────────────────────┐
│ filepath.Walk() starts traversal          │
│ For each file/directory:                  │
└─────────────────┬─────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────┐
│ Security Validator                               │
│ • ValidatePath() - check for traversal attacks  │
│ • IsPathSafe() - verify within allowed roots    │
└──────────────┬──────────────────────────────────┘
               │ ✅ Path safe
               ▼
┌─────────────────────────────────────────────────┐
│ Filter Engine                                    │
│ 1. Check .gitignore patterns                    │
│ 2. Check .agentignore patterns                  │
│ 3. Check deny_patterns (node_modules, .env)    │
│ 4. Check allow_patterns (*.go, *.js)           │
│ 5. Check sensitive file patterns                │
└──────────────┬──────────────────────────────────┘
               │ ✅ File passes filters
               ▼
┌─────────────────────────────────────────────────┐
│ File Detector                                    │
│ • Detect file type (text/binary/archive)       │
│ • Detect size category (small/medium/large)    │
│ • Check readability                             │
│ • Extract metadata                              │
└──────────────┬──────────────────────────────────┘
               │
               ▼
         [File Inventory]
         Stored Locally
```

### 2. File Analysis Phase

```
For each file in inventory:
    │
    ▼
┌─────────────────────────────────────────────────┐
│ Size-Based Processing Decision                  │
└────┬────────────┬────────────────┬───────────────┘
     │            │                │
     ▼            ▼                ▼
  Small       Medium            Large
  (<10KB)    (10KB-100KB)      (>100KB)
     │            │                │
     ▼            ▼                ▼
┌─────────┐  ┌─────────┐      ┌──────────────┐
│ Read    │  │ Read    │      │ Generate     │
│ Full    │  │ Full    │      │ Summary      │
│ Content │  │ Content │      │              │
│         │  │ +       │      │ +            │
│         │  │ Generate│      │ Create       │
│         │  │ Summary │      │ Chunks       │
└────┬────┘  └────┬────┘      └──────┬───────┘
     │            │                   │
     └────────────┴───────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────┐
│ Security Scanning (if enabled)                  │
│ • ScanForSecrets() - API keys, tokens          │
│ • ScanForPII() - emails, SSNs, phone numbers   │
│ • DetectSensitiveFile() - .env, .key files     │
└──────────────┬──────────────────────────────────┘
               │
               ▼
         [Analyzed Files]
         Stored Locally
```

### 3. LLM Preparation Phase

```
User Task: "Check for security issues"
    │
    ▼
┌─────────────────────────────────────────────────┐
│ Prepare Content for LLM                         │
│                                                  │
│ For each file:                                  │
│   If small/medium → include full content        │
│   If large → include summary + chunk info       │
│   If sensitive → [SKIP]                         │
│                                                  │
│ Calculate total tokens                          │
│ Ensure within token limit (8000 default)       │
└──────────────┬──────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────┐
│ Format as Markdown                              │
│                                                  │
│ === File: src/auth.go ===                       │
│ ```go                                            │
│ package main                                     │
│ func authenticate(token string) bool {          │
│   // hardcoded secret                           │
│   return token == "secret123"                   │
│ }                                                │
│ ```                                              │
│                                                  │
│ === File: src/main.go ===                       │
│ [Large file - 15KB]                             │
│ Summary: Main entry point, HTTP server...      │
│ Available chunks: 15                             │
└──────────────┬──────────────────────────────────┘
               │
               ▼
      [Prepared Content]
      Ready for LLM
```

### 4. LLM Interaction Phase

```
┌─────────────────────────────────────────────────┐
│ Construct LLM Request                           │
│                                                  │
│ System Message:                                 │
│   "You are a code analysis assistant..."       │
│                                                  │
│ User Message:                                   │
│   "Task: Check for security issues              │
│                                                  │
│    [Prepared file content]"                     │
└──────────────┬──────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────┐
│ Send to Ollama (Local LLM)                      │
│                                                  │
│ POST http://localhost:11434/api/chat            │
│ {                                                │
│   "model": "codellama",                         │
│   "messages": [...],                            │
│   "stream": false,                              │
│   "temperature": 0.1                            │
│ }                                                │
│                                                  │
│ ⚡ ONLY SELECTED CONTENT SENT                   │
│ ⚡ NO AUTOMATIC BACKGROUND UPLOADS               │
│ ⚡ USER HAS EXPLICIT CONTROL                     │
└──────────────┬──────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────┐
│ Receive & Parse Response                        │
│                                                  │
│ {                                                │
│   "message": {                                  │
│     "content": "I found 2 issues..."           │
│   },                                            │
│   "model": "codellama",                         │
│   "prompt_eval_count": 450,                     │
│   "eval_count": 280                             │
│ }                                                │
└──────────────┬──────────────────────────────────┘
               │
               ▼
         [Store Response]
         Locally
```

### 5. Result Presentation Phase

```
┌─────────────────────────────────────────────────┐
│ Display Analysis Results                        │
│                                                  │
│ 🎯 Analysis Complete                            │
│    Model: codellama                             │
│    Tokens used: 730                             │
│    Duration: 2.3s                               │
│                                                  │
│ 📝 Response:                                     │
│    [LLM's detailed analysis]                    │
│                                                  │
│ 🔍 Findings:                                     │
│    1. [HIGH] Hardcoded secret in auth.go       │
│       Line 5: token == "secret123"             │
│       Suggestion: Use environment variables     │
│                                                  │
│    2. [MEDIUM] SQL injection risk in db.go     │
│       Line 42: Direct string concatenation     │
│       Suggestion: Use parameterized queries    │
└──────────────┬──────────────────────────────────┘
               │
               ▼
         ┌─────────────┐
         │ Save to File│
         │ (optional)  │
         └─────────────┘
```

## Security & Privacy Flow

```
┌─────────────────────────────────────────────────┐
│ PRIVACY SAFEGUARDS (Applied Throughout)         │
└─────────────────────────────────────────────────┘

1. Path Validation
   ↓
   Check for directory traversal (../)
   Validate within allowed directories
   Block access outside user-approved paths

2. Content Filtering
   ↓
   Skip .env, .key, .pem files
   Detect and skip sensitive patterns
   Respect .gitignore and .agentignore

3. Secret Detection
   ↓
   Scan for API keys, tokens, passwords
   Mark files as sensitive if detected
   Exclude from LLM submission

4. User Control
   ↓
   User explicitly specifies directories
   User provides the analysis task
   User approves what gets analyzed

5. Local Processing
   ↓
   All file scanning happens locally
   All filtering happens locally
   Only approved content sent to LLM

6. No Persistence
   ↓
   Agent doesn't store file contents
   Memory cleared after processing
   Results only saved if user requests

7. LLM Interaction
   ↓
   Stateless calls (no conversation memory on LLM side)
   Only explicitly selected content sent
   Can run fully offline with local Ollama
```

## Key Interfaces

```go
// LLM Client Interface
type Client interface {
    Chat(request *ChatRequest) (*ChatResponse, error)
    IsAvailable() bool
    GetModel() string
}

// File Filter Interface
type Filter struct {
    ShouldInclude(path string, info FileInfo) bool
    IsWithinDepthLimit(depth int) bool
    ShouldFollowSymlink(path string) bool
}

// File Analyzer Interface
type Analyzer struct {
    AnalyzeFile(path string) (*FileInfo, error)
    AnalyzeFiles(paths []string) ([]*FileInfo, []error)
    PrepareForLLM(files []*FileInfo, maxTokens int) string
}

// Security Validator Interface
type Validator struct {
    ValidatePath(path string) error
    ScanForSecrets(content string) []SecurityViolation
    ScanForPII(content string) []SecurityViolation
    SanitizeContent(content string) string
}
```

## Data Structures

```go
// FileInfo - Metadata about a file
type FileInfo struct {
    Path        string
    Size        int64
    Category    FileCategory  // small/medium/large
    Type        FileType      // text/binary/archive
    IsReadable  bool
    IsSensitive bool
    Content     string       // for small files
    Summary     string       // for large files
    Chunks      []FileChunk  // for large files
    TokenCount  int
}

// ScanResult - Result of directory scan
type ScanResult struct {
    RootPath      string
    TotalFiles    int
    FilteredFiles int
    Files         []FileInfo
    Errors        []ScanError
    Duration      time.Duration
}

// AnalysisResponse - LLM response
type AnalysisResponse struct {
    Response    string
    Model       string
    TokensUsed  int
    Duration    time.Duration
    Findings    []Finding
    Suggestions []string
}
```

# local-agent Application Context

<application_context>

## Purpose

`local-agent` is a Go CLI that scans a local directory, sends file contents to a local (or remote) Ollama-compatible chat endpoint, and returns an LLM analysis of those files. It is an orchestrator around Ollama, not a model runtime, and it never sends data anywhere except the configured Ollama endpoint. It has a one-shot standard mode, an interactive mode with both a terminal UI and a companion web UI, and no user account system.

A user provides a directory and a task (e.g. "find security issues"). The agent scans and filters files, detects file type/size/sensitivity, chunks or summarizes large files, sends each file as its own request to the LLM (optionally several files concurrently), and aggregates the per-file responses into one report. Interactive mode keeps a scanned file set in memory and answers follow-up chat messages against it.

## Repository Map

- `main.go`: CLI flags, directory scanning/walking, filtering wiring, batch/concurrency orchestration, standalone-mode output, session log writing, interactive-mode bootstrap.
- `vars.go`: version and ANSI color constants.
- `config/config.go`: `Config` struct (agent/llm/filters/security/chunking), defaults (including env-var overrides), YAML load/save, validation.
- `analyzer/detector.go`: per-file metadata detection (size category, MIME/extension-based type), content extraction for text/PDF/DOC/DOCX/PPTX/PCAP.
- `analyzer/chunker.go`: splits large files into overlapping chunks by lines, tokens, or a "smart" strategy.
- `analyzer/analyzer.go`: orchestrates detection → content read → summarization → chunking → secret-violation flagging per file, and concurrent multi-file analysis.
- `filter/filter.go`, `filter/ignore.go`: `.gitignore`-style and custom allow/deny pattern matching, symlink and max-depth policy.
- `security/validator.go`: path traversal validation and regex-based secret/credential detection (API keys, AWS/GitHub/Slack tokens, private keys, JWTs, passwords, DB connection strings).
- `llm/client.go`: `OllamaClient` — chat requests to `/api/chat`, availability checks against `/api/tags`, model listing, thinking-model detection (Qwen3.5/Gemma4) and reasoning-block extraction, TLS-aware HTTP client construction.
- `llm/tokenizer.go`: rough token estimation used for batching/limits.
- `llm/prompts/system.md`: embedded system prompt controlling analysis tone, brevity rules, and output formatting.
- `sessionlog/sessionlog.go`: writes a JSON session record per run to `~/.local-agent/`.
- `tui/`: Bubble Tea terminal UI (`tui.go`, `interactive.go`, `runner.go`, `styles.go`) for interactive mode.
- `webui/`: companion web UI (`server.go`, `template.go`, `embed.go`, `webstatic/`) served alongside the terminal UI in interactive mode.
- `types/types.go`: shared data types (`FileInfo`, `ScanResult`, `AnalysisRequest/Response`, `Finding`, size/token constants).
- `img/embed.go`: embedded logo asset.

## Runtime Configuration

| Variable / Flag | Default | Meaning |
|---|---|---|
| `--host` | `localhost:11434` | Ollama host (`host:port`); wrapped into `http://host:port` and takes precedence over `OLLAMA_URL` |
| `OLLAMA_URL` | `http://localhost:11434` | Base Ollama endpoint used to seed the default config (`/api/chat`, `/api/tags` are appended per request) |
| `OLLAMA_CA_CERT` | unset | Path to a PEM CA certificate trusted in addition to the system pool, for a self-signed cert on a TLS-terminating Ollama reverse proxy; has no effect unless the endpoint uses `https://` or `OLLAMA_INSECURE_SKIP_VERIFY` is set |
| `OLLAMA_INSECURE_SKIP_VERIFY` | `false` | Disables TLS certificate validation for Ollama connections and ignores `OLLAMA_CA_CERT`. Testing only — a network attacker can impersonate the endpoint undetected |
| `AGENT_TOKEN_LIMIT` | `4000` | Max tokens per file sent to the LLM (raise to `>=8000` to analyze large files, including PDFs) |
| `AGENT_CONCURRENT_FILES` | `1` | Number of files analyzed concurrently (each file is still one independent LLM request) |
| `--config` | unset | Path to a YAML config file; falls back to `.agent/config.yaml`, `.agent/config.yml`, `agent-config.yaml`, `agent-config.yml`, then built-in defaults |
| `--model` | config value | Overrides `llm.model` |
| `--dir` | `.` | Directory to scan/analyze |
| `--focus` | unset | Limit analysis to a single file (relative to `--dir`; adjusts the scan root automatically if the file is outside it) |
| `--ui-port` | `5050` | Web UI port in interactive mode |
| `--https` | unset | PEM file (cert + key) to serve the Web UI over HTTPS |
| `--dry-run` | `false` | List filtered files without analyzing |
| `--no-detect-secrets` | `false` | Disable secret/sensitive content detection for this run |
| `--version`, `--health`, `--list-models`, `--interactive` | — | Print version / check LLM connectivity / list models / start interactive mode, then exit (except `--interactive`) |

Precedence for the Ollama endpoint: built-in default → `OLLAMA_URL` (config default) → YAML `llm.endpoint` → `--host` flag (highest). `llm.ca_cert` / `llm.insecure_skip_verify` follow the same default-from-env, override-via-YAML pattern but have no CLI flag.

All Ollama requests go through a shared `*http.Client` built once in `llm.NewOllamaClient` (see [llm/client.go](llm/client.go)). If the endpoint uses plain HTTP to a non-loopback host, a startup warning is printed since prompts and file contents would cross the network unencrypted and readable/alterable by a network attacker; use `https://` (optionally with `OLLAMA_CA_CERT`) or an encrypted tunnel instead. A second warning is printed when `OLLAMA_INSECURE_SKIP_VERIFY` is enabled.

The application sends non-streaming chat requests (`stream: false`). Thinking mode is used automatically for models identified as Qwen3.5 or Gemma4 (`llm.IsThinkingModel`); the reasoning block is stripped from the visible response and returned separately as `ThinkingContent`.

## Scan and Filter Pipeline

`scanDirectory` (main.go) walks the target directory, honoring `security.max_depth`, `security.follow_symlinks` (with symlink-cycle detection), and `security.SkipBinaries`. For every entry, `security.Validator.ValidatePath` rejects path-traversal attempts, then `filter.Filter.ShouldInclude` applies, in order: `.gitignore` (if `filters.respect_gitignore`), the custom ignore file (default `.agentignore`), `filters.deny_patterns`, then `filters.allow_patterns` (if any allow patterns are set, the file must match one). Included files are handed to `analyzer.AnalyzeFiles`, which detects type/size category (`small` ≤10KB, `medium` ≤100KB, `large` otherwise), optionally flags sensitive paths/secrets (`security.detect_secrets`), and reads content via type-specific extractors (plain text, PDF, DOC, DOCX, PPTX, PCAP). Large files are skipped unless `AGENT_TOKEN_LIMIT >= 8000`, and are chunked (lines/tokens/smart strategy) rather than sent whole.

## Analysis Pipeline

Every readable, in-token-limit file becomes its own batch — the agent always issues one LLM request per file, never combining multiple files into one prompt. `AGENT_CONCURRENT_FILES` (or `agent.concurrent_files` in YAML) controls how many of these per-file requests run in parallel via a worker pool; `1` processes sequentially. Each request calls `OllamaClient.Analyze`, which builds a system+user message pair (system prompt from `llm/prompts/system.md`, user message = task + file content) and posts to `/api/chat`. Per-file responses are reassembled in original order into one report, with failed files reported inline rather than aborting the run; a session JSON record is written to `~/.local-agent/` after standalone runs.

## Interactive Mode

`--interactive` scans once, then starts both a terminal UI (Bubble Tea, `tui/`) and a Web UI server (`webui/server.go`, default `http://localhost:5050`, optionally HTTPS via `--https`) backed by the same `llm.OllamaClient` and in-memory scan result/message history. Supported chat commands (both UIs): `help`, `model <name>` (hot-swaps the LLM client, re-applying the same CA/TLS settings), `rescan` (re-walks the directory live), `stats`, `files`, `focus <path>` / `focus clear`, `clear` (wipes chat history), `quit`. The Web UI additionally exposes a per-session "Session Prompt" (extra instructions appended to every request for the session, not persisted), a "Dir" control to change the working directory at runtime (`/api/changedir`), which triggers an immediate rescan, and a "Clear" button that, after a confirmation prompt, invokes the `clear` command and reloads the message list so the visible chat is wiped immediately (no dedicated endpoint — it reuses `POST /api/chat`). Web UI REST endpoints: `GET /`, `GET /api/status`, `GET /api/messages`, `POST /api/chat`, `POST /api/rescan`, `POST /api/focus`, `POST /api/session-prompt`, `GET /api/progress` (SSE-style live per-file progress), `POST /api/stop` (cancel an in-flight run), `POST /api/changedir` — see [API.md](API.md) for full request/response shapes.

## Security Boundaries

- `security.Validator.ValidatePath` blocks path traversal outside the scanned root.
- `security.Validator` regex-scans file content for common secret patterns (API keys, AWS/GitHub/Slack tokens, private key blocks, JWTs, DB connection strings, password assignments) and flags matching files/lines as findings when `security.detect_secrets` is enabled (disabled by default; `--no-detect-secrets` forces it off for a run).
- Default deny patterns already exclude common secret-bearing paths (`.env*`, `*.key`, `*.pem`, `*.crt`, `node_modules/**`, `.git/**`, build output dirs).
- The app only reads files and calls the configured Ollama endpoint; it does not execute shell commands or write anything outside `~/.local-agent/` (session logs) and files the user explicitly asks it to save.
- No authentication is implemented on the Web UI; it is intended for local/loopback use and must not be exposed on an untrusted network without a trusted authenticated proxy in front of it.

## Product Invariants

- One LLM request per file — never silently merge multiple files into a single prompt.
- Keep the scan → filter → detect → chunk/summarize → analyze pipeline order; don't bypass filtering or path validation for any code path (standalone, interactive terminal, or Web UI).
- Keep the terminal UI and Web UI backed by the same `config.Config` and `llm.OllamaClient` instance/behavior; do not fork divergent LLM-calling logic between them.
- Preserve non-streaming Ollama requests and thinking-block separation (`Response` vs `ThinkingContent`) for Qwen3.5/Gemma4 models.
- Keep `AGENT_TOKEN_LIMIT`/`AGENT_CONCURRENT_FILES`/`OLLAMA_URL`/`OLLAMA_CA_CERT`/`OLLAMA_INSECURE_SKIP_VERIFY` as environment-based defaults that YAML config and CLI flags can still override.
- Session logs and scan results are local artifacts only; no data leaves the machine except to the configured Ollama endpoint.
- Do not add a remote data upload, telemetry, or account system without an explicit product change.

</application_context>

package webui

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"local-agent/analyzer"
	"local-agent/config"
	"local-agent/filter"
	"local-agent/llm"
	"local-agent/sessionlog"
	"local-agent/types"
)

// Server represents the web UI server
type Server struct {
	directory   string
	model       string
	endpoint    string
	scanResult  *types.ScanResult
	focusedPath string
	cfg         *config.Config
	llmClient   *llm.OllamaClient
	messages    []Message
	mu          sync.RWMutex
	progressCh  chan string
	progressMu  sync.Mutex
}

// Message represents a chat message
type Message struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// ChatRequest represents an incoming chat message
type ChatRequest struct {
	Message string `json:"message"`
}

// ChatResponse represents a chat response
type ChatResponse struct {
	Success  bool      `json:"success"`
	Message  *Message  `json:"message,omitempty"`
	Error    string    `json:"error,omitempty"`
	Messages []Message `json:"messages,omitempty"`
}

// StatusResponse represents the current status
type StatusResponse struct {
	Directory   string `json:"directory"`
	Model       string `json:"model"`
	TotalFiles  int    `json:"totalFiles"`
	FocusedPath string `json:"focusedPath,omitempty"`
}

// NewServer creates a new web UI server
func NewServer(directory, model, endpoint string, scanResult *types.ScanResult, cfg *config.Config, llmClient *llm.OllamaClient, focusedPath string) *Server {
	s := &Server{
		directory:   directory,
		model:       model,
		endpoint:    endpoint,
		scanResult:  scanResult,
		focusedPath: focusedPath,
		cfg:         cfg,
		llmClient:   llmClient,
		messages:    make([]Message, 0),
	}

	// Add welcome message
	s.messages = append(s.messages, Message{
		Role: "assistant",
		Content: fmt.Sprintf("🤖 Interactive mode started!\n\nScanned: %s\nFiles found: %d\nModel: %s\n\nToken Limit: %d\nConcurrent Files: %d\nTemperature: %.2f\n\nType your questions or commands.",
			directory, scanResult.TotalFiles, model, cfg.Agent.TokenLimit, cfg.Agent.ConcurrentFiles, cfg.LLM.Temperature),
		Timestamp: time.Now(),
	})

	return s
}

// Start starts the web server
func (s *Server) Start(port int) error {
	// Serve embedded static files
	staticFS, err := fs.Sub(StaticFiles, "webstatic")
	if err != nil {
		log.Printf("Warning: failed to access embedded static files: %v", err)
	} else {
		http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	}

	http.HandleFunc("/", s.handleIndex)
	http.HandleFunc("/api/chat", s.handleChat)
	http.HandleFunc("/api/status", s.handleStatus)
	http.HandleFunc("/api/messages", s.handleMessages)
	http.HandleFunc("/api/rescan", s.handleRescan)
	http.HandleFunc("/api/focus", s.handleFocus)
	http.HandleFunc("/api/progress", s.handleProgress)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("🌐 Web UI available at http://localhost%s\n", addr)
	return http.ListenAndServe(addr, nil)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.New("index").Parse(htmlTemplate))
	tmpl.Execute(w, nil)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := StatusResponse{
		Directory:   s.directory,
		Model:       s.model,
		TotalFiles:  s.scanResult.TotalFiles,
		FocusedPath: s.focusedPath,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.messages)
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request")
		return
	}

	userInput := strings.TrimSpace(req.Message)
	if userInput == "" {
		sendError(w, "Empty message")
		return
	}

	// Add user message
	s.mu.Lock()
	s.messages = append(s.messages, Message{
		Role:      "user",
		Content:   userInput,
		Timestamp: time.Now(),
	})
	s.mu.Unlock()

	// Handle special commands
	if response := s.handleCommand(userInput); response != "" {
		s.mu.Lock()
		msg := Message{
			Role:      "assistant",
			Content:   response,
			Timestamp: time.Now(),
		}
		s.messages = append(s.messages, msg)
		s.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ChatResponse{
			Success: true,
			Message: &msg,
		})
		return
	}

	// Get active files
	activeFiles := s.getActiveFiles()
	if len(activeFiles) == 0 {
		msg := Message{
			Role:      "assistant",
			Content:   "⚠️  No files available for analysis.",
			Timestamp: time.Now(),
		}
		s.mu.Lock()
		s.messages = append(s.messages, msg)
		s.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ChatResponse{
			Success: true,
			Message: &msg,
		})
		return
	}

	// Process question
	s.progressMu.Lock()
	s.progressCh = make(chan string, 100)
	s.progressMu.Unlock()

	analysisResp, err := s.processQuestion(userInput, activeFiles)

	s.progressMu.Lock()
	if s.progressCh != nil {
		close(s.progressCh)
		s.progressCh = nil
	}
	s.progressMu.Unlock()

	if err != nil {
		sendError(w, err.Error())
		return
	}

	msg := Message{
		Role:      "assistant",
		Content:   analysisResp.Response,
		Timestamp: time.Now(),
	}

	s.mu.Lock()
	s.messages = append(s.messages, msg)
	s.mu.Unlock()

	s.saveSession(userInput, analysisResp)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ChatResponse{
		Success: true,
		Message: &msg,
	})
}

func (s *Server) handleRescan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	scanResult, err := s.performRescan()
	if err != nil {
		sendError(w, fmt.Sprintf("Rescan failed: %v", err))
		return
	}

	s.mu.Lock()
	s.scanResult = scanResult
	msg := Message{
		Role:      "assistant",
		Content:   fmt.Sprintf("✅ Rescan complete!\n\nFiles found: %d\nFiltered: %d", scanResult.TotalFiles, scanResult.FilteredFiles),
		Timestamp: time.Now(),
	}
	s.messages = append(s.messages, msg)
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ChatResponse{
		Success: true,
		Message: &msg,
	})
}

func (s *Server) handleFocus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request")
		return
	}

	s.mu.Lock()
	if req.Path == "" {
		s.focusedPath = ""
		msg := Message{
			Role:      "assistant",
			Content:   "🎯 Focus cleared. All files are now active.",
			Timestamp: time.Now(),
		}
		s.messages = append(s.messages, msg)
		s.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ChatResponse{
			Success: true,
			Message: &msg,
		})
	} else {
		s.focusedPath = req.Path
		msg := Message{
			Role:      "assistant",
			Content:   fmt.Sprintf("🎯 Focus set to: %s", req.Path),
			Timestamp: time.Now(),
		}
		s.messages = append(s.messages, msg)
		s.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ChatResponse{
			Success: true,
			Message: &msg,
		})
	}
}

func (s *Server) handleCommand(input string) string {
	lower := strings.ToLower(strings.TrimSpace(input))

	switch {
	case lower == "help":
		return `📚 Available commands:
• help - Show this help message
• clear - Clear conversation history
• model <name> - Switch to a different LLM model
• rescan - Rescan the directory for changes
• focus <path> - Focus on a specific file
• focus clear - Clear file focus
• stats - Show current statistics
• files - List all files in scope`

	case lower == "stats":
		s.mu.RLock()
		defer s.mu.RUnlock()
		activeFiles := s.getActiveFiles()
		return fmt.Sprintf(`📊 Statistics:
• Directory: %s
• Total files scanned: %d
• Active files: %d
• Focused file: %s
• Model: %s`, s.directory, s.scanResult.TotalFiles, len(activeFiles),
			func() string {
				if s.focusedPath != "" {
					return s.focusedPath
				}
				return "none"
			}(), s.model)

	case lower == "files":
		s.mu.RLock()
		defer s.mu.RUnlock()
		activeFiles := s.getActiveFiles()
		var builder strings.Builder
		builder.WriteString(fmt.Sprintf("📁 Files (%d total):\n", len(activeFiles)))
		for _, file := range activeFiles {
			builder.WriteString(fmt.Sprintf("• %s\n", file.RelPath))
		}
		return builder.String()

	case lower == "clear":
		s.mu.Lock()
		// Keep only the welcome message (first message)
		if len(s.messages) > 0 {
			s.messages = s.messages[:1]
		}
		s.mu.Unlock()
		return "🧹 Conversation history cleared."

	case strings.HasPrefix(lower, "model "):
		newModel := strings.TrimSpace(strings.TrimPrefix(lower, "model "))
		if newModel == "" {
			return fmt.Sprintf("⚠️  Please specify a model name.\nCurrent model: %s", s.model)
		}
		s.mu.Lock()
		oldModel := s.model
		s.model = newModel
		s.cfg.LLM.Model = newModel
		s.llmClient = llm.NewOllamaClient(s.cfg.LLM.Endpoint, newModel, s.cfg.LLM.Timeout)
		s.mu.Unlock()
		return fmt.Sprintf("✅ Model switched: %s → %s\n\nYou can now continue asking questions.", oldModel, newModel)

	case lower == "rescan":
		scanResult, err := s.performRescan()
		if err != nil {
			return fmt.Sprintf("❌ Rescan failed: %v", err)
		}
		s.mu.Lock()
		s.scanResult = scanResult
		s.mu.Unlock()
		return fmt.Sprintf("✅ Rescan complete!\n\nFiles found: %d\nFiltered: %d\nTotal size: %s",
			scanResult.TotalFiles, scanResult.FilteredFiles, formatBytes(scanResult.TotalSize))

	case strings.HasPrefix(lower, "focus "):
		parts := strings.SplitN(input, " ", 2)
		if len(parts) != 2 {
			return "❌ Usage: focus <path> or focus clear"
		}
		path := strings.TrimSpace(parts[1])
		if path == "clear" {
			s.mu.Lock()
			s.focusedPath = ""
			s.mu.Unlock()
			return "🎯 Focus cleared. All files are now active."
		}
		s.mu.Lock()
		s.focusedPath = path
		s.mu.Unlock()
		return fmt.Sprintf("🎯 Focus set to: %s", path)
	}

	return "" // Not a command
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func (s *Server) getActiveFiles() []*types.FileInfo {
	if s.scanResult == nil {
		return nil
	}

	if s.focusedPath == "" {
		files := make([]*types.FileInfo, 0, len(s.scanResult.Files))
		for i := range s.scanResult.Files {
			files = append(files, &s.scanResult.Files[i])
		}
		return files
	}

	for i := range s.scanResult.Files {
		if s.scanResult.Files[i].RelPath == s.focusedPath {
			return []*types.FileInfo{&s.scanResult.Files[i]}
		}
	}

	return nil
}

func (s *Server) handleProgress(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	s.progressMu.Lock()
	ch := s.progressCh
	s.progressMu.Unlock()

	if ch == nil {
		fmt.Fprintf(w, "data: done\n\n")
		flusher.Flush()
		return
	}

	ctx := r.Context()
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				fmt.Fprintf(w, "data: done\n\n")
				flusher.Flush()
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-ctx.Done():
			return
		}
	}
}

func (s *Server) processQuestion(question string, files []*types.FileInfo) (*types.AnalysisResponse, error) {
	start := time.Now()
	analyzerEngine := analyzer.NewAnalyzer(s.cfg)

	s.progressMu.Lock()
	progressCh := s.progressCh
	s.progressMu.Unlock()

	sendProgress := func(current, total int, name string) {
		if progressCh != nil {
			select {
			case progressCh <- fmt.Sprintf("Reviewed %d/%d: %s", current, total, name):
			default:
			}
		}
	}

	// Filter to readable files within token limit
	var validFiles []*types.FileInfo
	for _, f := range files {
		if f != nil && f.IsReadable && len(f.Content) > 0 && f.TokenCount <= s.cfg.Agent.TokenLimit {
			validFiles = append(validFiles, f)
		}
	}

	if len(validFiles) == 0 {
		return nil, fmt.Errorf("no valid file content to analyze")
	}

	type fileResult struct {
		idx      int
		name     string
		response string
		tokens   int
		err      error
	}

	processFile := func(idx int, file *types.FileInfo) fileResult {
		content := analyzerEngine.PrepareForLLM([]*types.FileInfo{file}, s.cfg.Agent.TokenLimit)
		if len(content) < 100 {
			return fileResult{idx: idx, name: file.RelPath, err: fmt.Errorf("no valid content")}
		}
		task := fmt.Sprintf("Analyze the file '%s'. %s", file.RelPath, question)
		resp, err := s.llmClient.Analyze(task, content, s.cfg.LLM.Temperature)
		if err != nil {
			return fileResult{idx: idx, name: file.RelPath, err: err}
		}
		return fileResult{idx: idx, name: file.RelPath, response: resp.Response, tokens: resp.TokensUsed}
	}

	results := make([]fileResult, len(validFiles))

	maxConcurrent := s.cfg.Agent.ConcurrentFiles
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}

	if maxConcurrent == 1 || len(validFiles) == 1 {
		for i, file := range validFiles {
			results[i] = processFile(i, file)
			sendProgress(i+1, len(validFiles), file.RelPath)
		}
	} else {
		type job struct {
			idx  int
			file *types.FileInfo
		}
		jobs := make(chan job, len(validFiles))
		resCh := make(chan fileResult, len(validFiles))

		var wg sync.WaitGroup
		for w := 0; w < maxConcurrent; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := range jobs {
					resCh <- processFile(j.idx, j.file)
				}
			}()
		}
		for i, file := range validFiles {
			jobs <- job{i, file}
		}
		close(jobs)
		go func() {
			wg.Wait()
			close(resCh)
		}()
		completed := 0
		for r := range resCh {
			completed++
			sendProgress(completed, len(validFiles), validFiles[r.idx].RelPath)
			results[r.idx] = r
		}
	}

	// Aggregate results in order
	var sb strings.Builder
	fileTokens := make(map[string]int)
	totalTokens := 0
	for _, r := range results {
		if r.err != nil {
			sb.WriteString(fmt.Sprintf("\n=== %s ===\n⚠️  FAILED: %v\n", r.name, r.err))
		} else {
			sb.WriteString(fmt.Sprintf("\n=== %s ===\n%s\n", r.name, strings.TrimSpace(r.response)))
			fileTokens[r.name] = r.tokens
			totalTokens += r.tokens
		}
	}

	return &types.AnalysisResponse{
		Response:   strings.TrimSpace(sb.String()),
		Model:      s.cfg.LLM.Model,
		TokensUsed: totalTokens,
		FileTokens: fileTokens,
		Duration:   time.Since(start),
	}, nil
}

func (s *Server) saveSession(question string, resp *types.AnalysisResponse) {
	if resp == nil {
		return
	}

	record := &sessionlog.Record{
		Timestamp:  time.Now(),
		Mode:       "webui",
		Directory:  s.directory,
		Task:       question,
		Focus:      s.focusedPath,
		Model:      s.model,
		TokensUsed: resp.TokensUsed,
		Duration:   resp.Duration,
		Files:      sessionlog.FilesFromTokens(nil, s.focusedPath),
		Response:   resp.Response,
	}

	if s.scanResult != nil {
		record.ScanSummary = &sessionlog.ScanSummary{
			TotalFiles:    s.scanResult.TotalFiles,
			FilteredFiles: s.scanResult.FilteredFiles,
			TotalSize:     s.scanResult.TotalSize,
			Duration:      s.scanResult.Duration,
		}
	}

	if _, err := sessionlog.Save(record); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save web session JSON: %v\n", err)
	}
}

func (s *Server) performRescan() (*types.ScanResult, error) {
	startTime := time.Now()

	f, err := filter.NewFilter(s.cfg, s.directory)
	if err != nil {
		return nil, err
	}

	analyzerEngine := analyzer.NewAnalyzer(s.cfg)

	result := &types.ScanResult{
		RootPath: s.directory,
		Files:    make([]types.FileInfo, 0),
		Errors:   make([]types.ScanError, 0),
		Summary:  make(map[string]int),
	}

	// Simple file walker
	err = filepath.Walk(s.directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if info.IsDir() {
			return nil
		}

		// Check if file should be included
		if !f.ShouldInclude(path, info) {
			return nil
		}

		// Analyze file
		fileInfo, err := analyzerEngine.AnalyzeFile(path, s.directory)
		if err != nil {
			return nil // Skip errors
		}

		result.Files = append(result.Files, *fileInfo)
		result.TotalFiles++
		result.TotalSize += fileInfo.Size

		// Update summary
		ext := filepath.Ext(path)
		if ext == "" {
			ext = "(no ext)"
		}
		result.Summary[ext]++

		return nil
	})

	if err != nil {
		return nil, err
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

func sendError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(ChatResponse{
		Success: false,
		Error:   message,
	})
}

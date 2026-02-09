package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"local-agent/analyzer"
	"local-agent/config"
	"local-agent/filter"
	"local-agent/llm"
	"local-agent/security"
	"local-agent/types"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// InteractiveModel represents the interactive conversation mode
type InteractiveModel struct {
	// Conversation state
	messages           []Message
	input              textinput.Model
	processing         bool
	processingProgress []string // Progress messages during processing

	// Context
	directory   string
	model       string
	endpoint    string
	scanResult  *types.ScanResult
	focusedPath string
	cfg         *config.Config
	llmClient   *llm.OllamaClient

	// UI state
	width     int
	height    int
	scrollPos int

	// Control
	quitting bool
	err      error
}

// Message represents a conversation message
type Message struct {
	Role      string // "user" or "assistant"
	Content   string
	Timestamp time.Time
}

type processCompleteMsg struct {
	response string
	err      error
}

type processProgressMsg struct {
	message string
}

type rescanCompleteMsg struct {
	scanResult *types.ScanResult
	err        error
}

// NewInteractiveModel creates a new interactive mode model
func NewInteractiveModel(directory, model, endpoint string, scanResult *types.ScanResult, cfg *config.Config, llmClient *llm.OllamaClient, focusedPath string) InteractiveModel {
	ti := textinput.New()
	ti.Placeholder = "Ask a question about your codebase..."
	ti.Focus()
	ti.CharLimit = 500
	ti.Width = 80

	// Add welcome message
	welcome := Message{
		Role: "assistant",
		Content: fmt.Sprintf("🤖 Interactive mode started!\n\nScanned: %s\nFiles found: %d\nModel: %s\n\n🔧 Configuration:\n   Token Limit: %d\n   Concurrent Files: %d\n   Temperature: %.2f\n\nType your questions or commands. Type 'help' for available commands, 'quit' or 'exit' to leave.",
			directory, scanResult.TotalFiles, model, cfg.Agent.TokenLimit, cfg.Agent.ConcurrentFiles, cfg.LLM.Temperature),
		Timestamp: time.Now(),
	}

	return InteractiveModel{
		messages:    []Message{welcome},
		input:       ti,
		directory:   directory,
		model:       model,
		endpoint:    endpoint,
		scanResult:  scanResult,
		focusedPath: focusedPath,
		cfg:         cfg,
		llmClient:   llmClient,
	}
}

func (m InteractiveModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m InteractiveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			m.quitting = true
			m.messages = append(m.messages, Message{
				Role:      "assistant",
				Content:   "👋 Bye!",
				Timestamp: time.Now(),
			})
			return m, tea.Quit

		case tea.KeyEnter:
			if m.processing {
				return m, nil
			}

			userInput := strings.TrimSpace(m.input.Value())
			if userInput == "" {
				return m, nil
			}

			// Handle special commands
			if m.handleCommand(userInput) {
				m.scrollPos = 0
				if m.quitting {
					return m, tea.Quit
				}
				m.input.Reset()
				// Trigger rescan if command was rescan
				if strings.ToLower(userInput) == "rescan" {
					return m, m.performRescan()
				}
				return m, nil
			}

			// Add user message
			m.messages = append(m.messages, Message{
				Role:      "user",
				Content:   userInput,
				Timestamp: time.Now(),
			})

			activeFiles := m.getActiveFiles()
			if len(activeFiles) == 0 {
				m.messages = append(m.messages, Message{
					Role:      "assistant",
					Content:   "⚠️  No files available for analysis. Use 'rescan' or 'focus clear' to reset your selection.",
					Timestamp: time.Now(),
				})
				return m, nil
			}

			m.input.Reset()
			m.processing = true

			// Generate and show processing status immediately
			m.processingProgress = m.generateProcessingStatus(activeFiles)

			// Process the question
			return m, m.processQuestion(userInput, activeFiles)

		case tea.KeyUp:
			m.scrollPos++

		case tea.KeyDown:
			if m.scrollPos > 0 {
				m.scrollPos--
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.Width = msg.Width - 4

	case processCompleteMsg:
		m.processing = false
		m.processingProgress = nil // Clear progress messages
		m.scrollPos = 0            // Reset scroll to show latest message
		if msg.err != nil {
			m.messages = append(m.messages, Message{
				Role:      "assistant",
				Content:   fmt.Sprintf("❌ Error: %v", msg.err),
				Timestamp: time.Now(),
			})
		} else {
			m.messages = append(m.messages, Message{
				Role:      "assistant",
				Content:   msg.response,
				Timestamp: time.Now(),
			})
		}
		return m, nil

	case processProgressMsg:
		// Add progress message if not empty, keep last 15
		if msg.message != "" {
			m.processingProgress = append(m.processingProgress, msg.message)
			if len(m.processingProgress) > 15 {
				m.processingProgress = m.processingProgress[len(m.processingProgress)-15:]
			}
		}
		return m, nil

	case rescanCompleteMsg:
		m.processing = false
		if msg.err != nil {
			m.messages = append(m.messages, Message{
				Role:      "assistant",
				Content:   fmt.Sprintf("❌ Rescan failed: %v", msg.err),
				Timestamp: time.Now(),
			})
		} else {
			m.scanResult = msg.scanResult
			var builder strings.Builder
			builder.WriteString(fmt.Sprintf("✅ Rescan complete!\n\nFiles found: %d\nFiltered: %d\nTotal size: %s", msg.scanResult.TotalFiles, msg.scanResult.FilteredFiles, formatBytes(msg.scanResult.TotalSize)))
			if m.focusedPath != "" && !m.focusedFileAvailable() {
				builder.WriteString(fmt.Sprintf("\n\n🎯 The previously focused file (%s) is no longer available. Reverting to all files.", m.focusedPath))
				m.focusedPath = ""
			}
			m.messages = append(m.messages, Message{
				Role:      "assistant",
				Content:   builder.String(),
				Timestamp: time.Now(),
			})
		}
		return m, nil
	}

	// Update text input
	if !m.processing {
		m.input, cmd = m.input.Update(msg)
	}

	return m, cmd
}

func (m InteractiveModel) View() string {
	if m.quitting {
		return goodbyeStyle.Render("👋 Goodbye!")
	}

	var s strings.Builder

	// Header
	headerText := fmt.Sprintf("🤖 Interactive Mode | %s | Files: %d", m.model, m.scanResult.TotalFiles)
	if m.focusedPath != "" {
		headerText += fmt.Sprintf(" | Focus: %s", m.focusedPath)
	}
	header := headerStyle.Render(headerText)
	s.WriteString(header + "\n\n")

	// Messages area
	messagesHeight := m.height - 8 // Leave room for header, input, and footer
	if messagesHeight < 5 {
		messagesHeight = 5
	}

	messages := m.renderMessages(messagesHeight)
	s.WriteString(messages)

	s.WriteString("\n" + strings.Repeat("─", m.width) + "\n")

	// Input area
	if m.processing {
		s.WriteString(processingStyle.Render("⏳ Processing...") + "\n")
		// Show progress messages
		if len(m.processingProgress) > 0 {
			s.WriteString("\n" + subtleStyle.Render("Progress:") + "\n")
			for _, msg := range m.processingProgress {
				s.WriteString(subtleStyle.Render("  "+msg) + "\n")
			}
			s.WriteString("\n") // Add space after progress messages
		}
	} else {
		s.WriteString(inputLabelStyle.Render("You: ") + m.input.View() + "\n")
	}

	// Footer
	footer := footerStyle.Render("↑/↓ scroll • enter send • ctrl+c quit")
	s.WriteString(footer)

	return s.String()
}

func (m InteractiveModel) renderMessages(maxHeight int) string {
	var lines []string

	for i, msg := range m.messages {
		timestamp := msg.Timestamp.Format("15:04:05")

		if msg.Role == "user" {
			// User message
			header := userHeaderStyle.Render(fmt.Sprintf("[%s] You:", timestamp))
			lines = append(lines, header)

			// Wrap and indent user message
			wrapped := m.wrapMessage(msg.Content, m.width-6)
			for _, line := range strings.Split(wrapped, "\n") {
				lines = append(lines, userMessageStyle.Render("  "+line))
			}
		} else {
			// Assistant message
			header := assistantHeaderStyle.Render(fmt.Sprintf("[%s] Assistant:", timestamp))
			lines = append(lines, header)

			// Parse metadata from response if present
			content := msg.Content
			metadata := ""

			if idx := strings.LastIndex(content, "\n\n---\n"); idx != -1 {
				metadata = strings.TrimSpace(content[idx+5:])
				content = strings.TrimSpace(content[:idx])
			}

			// Wrap and indent assistant message
			wrapped := m.wrapMessage(content, m.width-6)
			for _, line := range strings.Split(wrapped, "\n") {
				renderStyle := assistantMessageStyle
				if isFileHeaderLine(line) {
					renderStyle = fileHeaderStyle
				}
				lines = append(lines, renderStyle.Render("  "+line))
			}

			// Add metadata at the end if present
			if metadata != "" {
				lines = append(lines, "")
				// Don't wrap metadata lines, keep them as-is
				for _, line := range strings.Split(metadata, "\n") {
					lines = append(lines, metadataStyle.Render("  "+line))
				}
			}
		}

		// Add spacing between messages (except after last message)
		if i < len(m.messages)-1 {
			lines = append(lines, "")
			lines = append(lines, subtleStyle.Render(strings.Repeat("─", min(m.width, 80))))
			lines = append(lines, "")
		}
	}

	// Handle scrolling
	totalLines := len(lines)
	if totalLines <= maxHeight {
		return strings.Join(lines, "\n")
	}

	// Show most recent messages (from bottom)
	start := totalLines - maxHeight - m.scrollPos
	if start < 0 {
		start = 0
		m.scrollPos = totalLines - maxHeight
	}
	end := start + maxHeight
	if end > totalLines {
		end = totalLines
	}

	return strings.Join(lines[start:end], "\n")
}

func (m *InteractiveModel) handleCommand(input string) bool {
	lower := strings.ToLower(input)

	switch lower {
	case "quit", "exit", "q":
		m.quitting = true
		m.messages = append(m.messages, Message{
			Role:      "assistant",
			Content:   "👋 Bye!",
			Timestamp: time.Now(),
		})
		return true

	case "help", "h":
		helpMsg := `Available commands:
• help, h - Show this help message
• model <name> - Switch to a different LLM model
• rescan - Re-scan directory for new/changed files
• stats - Show scan statistics
• files - List scanned files
• focus <path> - Analyze only the specified file (use 'focus clear' to reset)
• last - View last saved analysis
• clear - Clear conversation history
• quit, exit, q - Exit interactive mode

You can also ask questions about your codebase, such as:
• "Find all TODO comments"
• "What security issues exist?"
• "Explain the main.go file"
• "List all API endpoints"`

		m.messages = append(m.messages, Message{
			Role:      "assistant",
			Content:   helpMsg,
			Timestamp: time.Now(),
		})
		return true

	case "stats":
		stats := fmt.Sprintf(`📊 Scan Statistics:
• Total files: %d
• Filtered files: %d
• Total size: %s
• Scan duration: %v

File breakdown:`,
			m.scanResult.TotalFiles,
			m.scanResult.FilteredFiles,
			formatBytes(m.scanResult.TotalSize),
			m.scanResult.Duration)

		for key, count := range m.scanResult.Summary {
			stats += fmt.Sprintf("\n  • %s: %d", key, count)
		}

		m.messages = append(m.messages, Message{
			Role:      "assistant",
			Content:   stats,
			Timestamp: time.Now(),
		})
		return true

	case "files":
		var fileList strings.Builder
		fileList.WriteString(fmt.Sprintf("📁 Scanned Files (%d):\n\n", len(m.scanResult.Files)))

		for i, file := range m.scanResult.Files {
			if i >= 50 { // Limit to first 50 files
				fileList.WriteString(fmt.Sprintf("\n... and %d more files", len(m.scanResult.Files)-50))
				break
			}
			fileList.WriteString(fmt.Sprintf("• %s (%s)\n", file.RelPath, formatBytes(file.Size)))
		}

		m.messages = append(m.messages, Message{
			Role:      "assistant",
			Content:   fileList.String(),
			Timestamp: time.Now(),
		})
		return true

	case "last":
		lastFile := filepath.Join(os.TempDir(), "local-agent-last.txt")
		content, err := os.ReadFile(lastFile)
		if err != nil {
			var msg string
			if os.IsNotExist(err) {
				msg = "❌ No previous analysis found.\nRun an analysis first (non-interactive mode) to save results."
			} else {
				msg = fmt.Sprintf("⚠️ Failed to read last analysis: %v", err)
			}
			m.messages = append(m.messages, Message{
				Role:      "assistant",
				Content:   msg,
				Timestamp: time.Now(),
			})
		} else {
			m.messages = append(m.messages, Message{
				Role:      "assistant",
				Content:   string(content),
				Timestamp: time.Now(),
			})
		}
		return true

	case "clear":
		// Keep only the welcome message
		if len(m.messages) > 0 {
			m.messages = m.messages[:1]
		}
		m.scrollPos = 0
		m.messages = append(m.messages, Message{
			Role:      "assistant",
			Content:   "🧹 Conversation history cleared.",
			Timestamp: time.Now(),
		})
		return true

	case "rescan":
		m.processing = true
		m.messages = append(m.messages, Message{
			Role:      "assistant",
			Content:   "🔄 Rescanning directory...",
			Timestamp: time.Now(),
		})
		return true

	default:
		if strings.HasPrefix(lower, "focus") {
			return m.handleFocusCommand(input)
		}

		// Check for model command
		if strings.HasPrefix(lower, "model ") {
			newModel := strings.TrimSpace(strings.TrimPrefix(lower, "model "))
			if newModel == "" {
				m.messages = append(m.messages, Message{
					Role:      "assistant",
					Content:   fmt.Sprintf("⚠️  Please specify a model name.\nCurrent model: %s", m.model),
					Timestamp: time.Now(),
				})
			} else {
				oldModel := m.model
				m.model = newModel
				m.cfg.LLM.Model = newModel
				m.llmClient = llm.NewOllamaClient(m.cfg.LLM.Endpoint, newModel, m.cfg.LLM.Timeout)
				m.messages = append(m.messages, Message{
					Role:      "assistant",
					Content:   fmt.Sprintf("✅ Model switched: %s → %s\n\nYou can now continue asking questions.", oldModel, newModel),
					Timestamp: time.Now(),
				})
			}
			return true
		}
	}

	return false
}

func (m InteractiveModel) processQuestion(question string, files []*types.FileInfo) tea.Cmd {
	return func() tea.Msg {
		// Prepare file context for LLM
		analyzerEngine := analyzer.NewAnalyzer(m.cfg)

		// Process files concurrently
		result, processingInfo, err := m.analyzeBatchesForInteractive(files, question, analyzerEngine)
		if err != nil {
			return processCompleteMsg{
				response: "",
				err:      err,
			}
		}

		// Format response with processing info
		var response strings.Builder

		// Add processing details at the top
		if processingInfo != "" {
			response.WriteString(processingInfo)
			response.WriteString("\n")
		}

		response.WriteString(result.Response)

		// Add metadata with proper spacing
		response.WriteString("\n\n")
		response.WriteString("---\n")

		// Show per-file token usage if available
		if len(result.FileTokens) > 0 {
			response.WriteString("📊 Token usage per file:\n")
			for file, tokens := range result.FileTokens {
				response.WriteString(fmt.Sprintf("   %s: %d tokens\n", file, tokens))
			}
			response.WriteString(fmt.Sprintf("\n⏱️  Total duration: %.2fs", result.Duration.Seconds()))
		} else {
			response.WriteString(fmt.Sprintf("📊 Tokens: %d  •  ⏱️  Duration: %.2fs", result.TokensUsed, result.Duration.Seconds()))
		}

		// Save to temp file for 'last' command
		saveAnalysisToTempFile(result, question)

		return processCompleteMsg{
			response: response.String(),
			err:      nil,
		}
	}
}

// Simple helper to generate progress info that will be shown in processing area
func (m InteractiveModel) generateProcessingStatus(filesPtrs []*types.FileInfo) []string {
	var messages []string
	messages = append(messages, "📦 Processing files individually (one request per file)")

	tokenLimit := m.cfg.Agent.TokenLimit
	processable := 0
	skipped := 0

	for _, file := range filesPtrs {
		if file == nil || !file.IsReadable {
			continue
		}
		if file.TokenCount > tokenLimit {
			messages = append(messages, fmt.Sprintf("⚠️  Skipping %s (%d tokens exceeds limit of %d)",
				file.RelPath, file.TokenCount, tokenLimit))
			skipped++
		} else {
			processable++
		}
	}

	messages = append(messages, fmt.Sprintf("Processing %d files", processable))
	if skipped > 0 {
		messages = append(messages, fmt.Sprintf("%d files skipped (exceeds token limit)", skipped))
	}

	maxConcurrent := m.cfg.Agent.ConcurrentFiles
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}
	if maxConcurrent > 1 && processable > 1 {
		messages = append(messages, fmt.Sprintf("Using %d concurrent workers", maxConcurrent))
	}

	return messages
}

func (m *InteractiveModel) getActiveFiles() []*types.FileInfo {
	if m.scanResult == nil {
		return nil
	}

	if m.focusedPath == "" {
		files := make([]*types.FileInfo, 0, len(m.scanResult.Files))
		for i := range m.scanResult.Files {
			files = append(files, &m.scanResult.Files[i])
		}
		return files
	}

	focusPath := normalizePath(m.focusedPath)
	for i := range m.scanResult.Files {
		if normalizePath(m.scanResult.Files[i].RelPath) == focusPath {
			return []*types.FileInfo{&m.scanResult.Files[i]}
		}
	}

	return nil
}

func (m *InteractiveModel) resolveFocusTarget(query string) (string, string) {
	if m.scanResult == nil {
		return "", "⚠️  No scan data available. Run 'rescan' first."
	}

	normalizedQuery := normalizePath(query)
	for _, file := range m.scanResult.Files {
		if normalizePath(file.RelPath) == normalizedQuery {
			return file.RelPath, ""
		}
	}

	base := strings.ToLower(filepath.Base(normalizedQuery))
	var matches []string
	for _, file := range m.scanResult.Files {
		if strings.ToLower(filepath.Base(file.RelPath)) == base {
			matches = append(matches, file.RelPath)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Sprintf("⚠️  Could not find a file matching %q. Use 'files' to list available files.", query)
	case 1:
		return matches[0], ""
	default:
		return "", fmt.Sprintf("⚠️  Multiple files match %q:\n   %s\nPlease provide a more specific path.", query, strings.Join(matches, "\n   "))
	}
}

func (m *InteractiveModel) handleFocusCommand(input string) bool {
	arg := ""
	if len(input) >= 5 {
		arg = strings.TrimSpace(input[5:])
	}

	if arg == "" {
		var msg string
		if m.focusedPath == "" {
			msg = "🎯 No focused file. All files will be analyzed."
		} else {
			msg = fmt.Sprintf("🎯 Currently focusing on %s. Use 'focus clear' to analyze all files.", m.focusedPath)
		}
		m.messages = append(m.messages, Message{
			Role:      "assistant",
			Content:   msg,
			Timestamp: time.Now(),
		})
		return true
	}

	if strings.EqualFold(arg, "clear") || strings.EqualFold(arg, "all") || strings.EqualFold(arg, "reset") {
		if m.focusedPath == "" {
			m.messages = append(m.messages, Message{
				Role:      "assistant",
				Content:   "🎯 Focus already cleared. Analyzing all files.",
				Timestamp: time.Now(),
			})
		} else {
			cleared := m.focusedPath
			m.focusedPath = ""
			m.messages = append(m.messages, Message{
				Role:      "assistant",
				Content:   fmt.Sprintf("🎯 Focus on %s cleared. Future questions will analyze all files.", cleared),
				Timestamp: time.Now(),
			})
		}
		return true
	}

	matchedPath, errMsg := m.resolveFocusTarget(arg)
	if errMsg != "" {
		m.messages = append(m.messages, Message{
			Role:      "assistant",
			Content:   errMsg,
			Timestamp: time.Now(),
		})
		return true
	}

	m.focusedPath = matchedPath
	m.messages = append(m.messages, Message{
		Role:      "assistant",
		Content:   fmt.Sprintf("🎯 Focus set to %s. Only this file will be analyzed until you run 'focus clear'.", matchedPath),
		Timestamp: time.Now(),
	})
	return true
}

func (m *InteractiveModel) focusedFileAvailable() bool {
	if m.focusedPath == "" || m.scanResult == nil {
		return false
	}
	focusPath := normalizePath(m.focusedPath)
	for _, file := range m.scanResult.Files {
		if normalizePath(file.RelPath) == focusPath {
			return true
		}
	}
	return false
}

func normalizePath(p string) string {
	if p == "" {
		return ""
	}
	clean := filepath.Clean(strings.ReplaceAll(p, "\\", string(filepath.Separator)))
	return filepath.ToSlash(clean)
}

// batchJobInteractive represents a batch processing job for interactive mode
type batchJobInteractive struct {
	batchNum int
	batch    []*types.FileInfo
}

// batchResultInteractive represents the result of processing a batch
type batchResultInteractive struct {
	batchNum int
	response *types.AnalysisResponse
	err      error
}

func (m InteractiveModel) analyzeBatchesForInteractive(files []*types.FileInfo, question string, analyzerEngine *analyzer.Analyzer) (*types.AnalysisResponse, string, error) {
	var processingInfo strings.Builder
	processingInfo.WriteString("📦 Processing files individually (one request per file)\n")

	// Prepare batches (one file per batch)
	batches := m.prepareBatchesForInteractive(files, &processingInfo)
	totalFiles := len(batches)

	if totalFiles == 0 {
		return &types.AnalysisResponse{
			Response: "No files to analyze",
			Model:    m.cfg.LLM.Model,
		}, processingInfo.String(), nil
	}

	processingInfo.WriteString(fmt.Sprintf("   Processing %d files\n", totalFiles))

	// Determine concurrency level
	maxConcurrent := m.cfg.Agent.ConcurrentFiles
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}

	// If only 1 worker or 1 file, process sequentially
	if maxConcurrent == 1 || totalFiles == 1 {
		result, err := m.processSequentiallyForInteractive(batches, question, analyzerEngine)
		return result, processingInfo.String(), err
	}

	processingInfo.WriteString(fmt.Sprintf("   Using %d concurrent workers\n", maxConcurrent))

	// Process batches concurrently
	result, err := m.processConcurrentlyForInteractive(batches, question, analyzerEngine, maxConcurrent)
	return result, processingInfo.String(), err
}

func (m InteractiveModel) prepareBatchesForInteractive(files []*types.FileInfo, info *strings.Builder) [][]*types.FileInfo {
	var batches [][]*types.FileInfo
	tokenLimit := m.cfg.Agent.TokenLimit

	for _, file := range files {
		if file == nil || !file.IsReadable {
			continue
		}

		// Skip files that exceed token limit
		if file.TokenCount > tokenLimit {
			info.WriteString(fmt.Sprintf("   ⚠️  Skipping %s (%d tokens exceeds limit of %d)\n",
				file.RelPath, file.TokenCount, tokenLimit))
			continue
		}

		// Each file is its own batch
		batches = append(batches, []*types.FileInfo{file})
	}

	return batches
}

func (m InteractiveModel) processSequentiallyForInteractive(batches [][]*types.FileInfo, question string, analyzerEngine *analyzer.Analyzer) (*types.AnalysisResponse, error) {
	var allResponses []string
	fileTokens := make(map[string]int)
	var totalDuration time.Duration
	model := ""

	for _, batch := range batches {
		fileName := batch[0].RelPath

		response, err := m.processBatchForInteractive(batch, question, analyzerEngine)
		if err != nil {
			allResponses = append(allResponses, fmt.Sprintf("=== %s ===\n⚠️  FAILED: %v", fileName, err))
		} else {
			// Trim leading/trailing whitespace from response
			cleanResponse := strings.TrimSpace(response.Response)
			allResponses = append(allResponses, fmt.Sprintf("=== %s ===\n%s", fileName, cleanResponse))
			fileTokens[fileName] = response.TokensUsed
			totalDuration += response.Duration
			if model == "" {
				model = response.Model
			}
		}
	}

	return &types.AnalysisResponse{
		Response:   strings.Join(allResponses, "\n\n"),
		Model:      model,
		FileTokens: fileTokens,
		Duration:   totalDuration,
	}, nil
}

func (m InteractiveModel) processConcurrentlyForInteractive(batches [][]*types.FileInfo, question string, analyzerEngine *analyzer.Analyzer, maxConcurrent int) (*types.AnalysisResponse, error) {
	totalFiles := len(batches)

	// Create file name mapping for tracking
	fileNames := make(map[int]string)
	for i, batch := range batches {
		fileNames[i+1] = batch[0].RelPath
	}

	// Create job and result channels
	jobs := make(chan batchJobInteractive, totalFiles)
	results := make(chan batchResultInteractive, totalFiles)

	// Start worker pool
	var wg sync.WaitGroup
	for w := 1; w <= maxConcurrent; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range jobs {
				response, err := m.processBatchForInteractive(job.batch, question, analyzerEngine)
				results <- batchResultInteractive{
					batchNum: job.batchNum,
					response: response,
					err:      err,
				}
			}
		}(w)
	}

	// Send jobs to workers
	for i, batch := range batches {
		jobs <- batchJobInteractive{
			batchNum: i + 1,
			batch:    batch,
		}
	}
	close(jobs)

	// Wait for all workers to finish in background
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	fileResults := make(map[int]*types.AnalysisResponse)
	fileTokens := make(map[string]int)
	var totalDuration time.Duration
	model := ""
	failedFiles := make(map[int]error)

	for result := range results {
		fileName := fileNames[result.batchNum]
		if result.err != nil {
			failedFiles[result.batchNum] = result.err
		} else {
			fileResults[result.batchNum] = result.response
			fileTokens[fileName] = result.response.TokensUsed
			totalDuration += result.response.Duration
			if model == "" {
				model = result.response.Model
			}
		}
	}

	// Aggregate results in order, including failed files
	var allResponses []string
	for i := 1; i <= totalFiles; i++ {
		fileName := fileNames[i]
		if response, ok := fileResults[i]; ok {
			// Successful file
			allResponses = append(allResponses, fmt.Sprintf("\n=== %s ===%s", fileName, response.Response))
		} else if err, failed := failedFiles[i]; failed {
			// Failed file - include error message
			allResponses = append(allResponses, fmt.Sprintf("\n=== %s ===\n⚠️  FAILED: %v", fileName, err))
		}
	}

	// Add summary of failures if any
	var responseText string
	if len(failedFiles) > 0 {
		responseText = fmt.Sprintf("⚠️  Warning: %d of %d files failed (see details below)\n", len(failedFiles), totalFiles)
		responseText += strings.Join(allResponses, "\n")
	} else {
		responseText = strings.Join(allResponses, "\n")
	}

	return &types.AnalysisResponse{
		Response:   responseText,
		Model:      model,
		FileTokens: fileTokens,
		Duration:   totalDuration,
	}, nil
}

func (m InteractiveModel) processBatchForInteractive(batch []*types.FileInfo, question string, analyzerEngine *analyzer.Analyzer) (*types.AnalysisResponse, error) {
	content := analyzerEngine.PrepareForLLM(batch, m.cfg.Agent.TokenLimit)

	// Check if we have any actual content to analyze
	if len(content) < 100 {
		return nil, fmt.Errorf("no valid content to analyze after PrepareForLLM")
	}

	// For single file batches, add filename to question for clarity
	actualQuestion := question
	if len(batch) == 1 && batch[0] != nil {
		actualQuestion = fmt.Sprintf("Analyze the file '%s'. %s", batch[0].RelPath, question)
	}

	return m.llmClient.Analyze(actualQuestion, content, m.cfg.LLM.Temperature)
}

func isFileHeaderLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "===") && strings.HasSuffix(trimmed, "===") && len(trimmed) > 8
}

func (m InteractiveModel) wrapMessage(text string, width int) string {
	if width < 20 {
		width = 20
	}

	// Preserve existing line breaks by processing each line separately
	inputLines := strings.Split(text, "\n")
	var outputLines []string

	for _, inputLine := range inputLines {
		// Empty lines should be preserved
		if strings.TrimSpace(inputLine) == "" {
			outputLines = append(outputLines, "")
			continue
		}

		// Wrap this line
		words := strings.Fields(inputLine)
		if len(words) == 0 {
			outputLines = append(outputLines, "")
			continue
		}

		var currentLine strings.Builder
		for _, word := range words {
			if currentLine.Len() == 0 {
				currentLine.WriteString(word)
			} else if currentLine.Len()+1+len(word) <= width {
				currentLine.WriteString(" " + word)
			} else {
				outputLines = append(outputLines, currentLine.String())
				currentLine.Reset()
				currentLine.WriteString(word)
			}
		}

		if currentLine.Len() > 0 {
			outputLines = append(outputLines, currentLine.String())
		}
	}

	return strings.Join(outputLines, "\n")
}

func saveAnalysisToTempFile(result *types.AnalysisResponse, question string) {
	lastFile := filepath.Join(os.TempDir(), "local-agent-last.txt")

	var content strings.Builder
	content.WriteString("Analysis Results\n")
	content.WriteString(fmt.Sprintf("Generated: %s\n", time.Now().Format(time.RFC3339)))
	content.WriteString(fmt.Sprintf("Model: %s\n", result.Model))
	content.WriteString(fmt.Sprintf("Tokens: %d\n", result.TokensUsed))
	content.WriteString(fmt.Sprintf("Duration: %v\n", result.Duration))
	content.WriteString("\n" + strings.Repeat("=", 80) + "\n\n")
	content.WriteString(fmt.Sprintf("QUESTION:\n%s\n\n", question))
	content.WriteString(strings.Repeat("-", 80) + "\n\n")
	content.WriteString(fmt.Sprintf("RESPONSE:\n%s\n", result.Response))

	// Save to last file (ignore errors)
	os.WriteFile(lastFile, []byte(content.String()), 0644)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (m InteractiveModel) performRescan() tea.Cmd {
	return func() tea.Msg {
		// Import needed here would cause circular dependency, so we duplicate the scan logic
		analyzer := analyzer.NewAnalyzer(m.cfg)
		filter, err := filter.NewFilter(m.cfg, m.directory)
		if err != nil {
			return rescanCompleteMsg{err: err}
		}
		validator := security.NewValidator()

		result := &types.ScanResult{
			RootPath: m.directory,
			Files:    make([]types.FileInfo, 0),
			Errors:   make([]types.ScanError, 0),
			Summary:  make(map[string]int),
		}

		visitedDirs := make(map[string]struct{})
		var filePaths []string

		var walk func(string, int)
		walk = func(current string, depth int) {
			info, err := os.Lstat(current)
			if err != nil {
				return
			}

			if info.Mode()&os.ModeSymlink != 0 {
				if !filter.ShouldFollowSymlink(current) {
					return
				}
				target, err := filepath.EvalSymlinks(current)
				if err != nil {
					return
				}
				targetAbs, _ := filepath.Abs(target)
				if _, seen := visitedDirs[targetAbs]; seen {
					return
				}
				info, err = os.Stat(targetAbs)
				if err != nil {
					return
				}
				current = targetAbs
			}

			if err := validator.ValidatePath(current); err != nil {
				return
			}

			if info.IsDir() {
				if !filter.IsWithinDepthLimit(depth) {
					return
				}
				absDir, _ := filepath.Abs(current)
				visitedDirs[absDir] = struct{}{}
				entries, err := os.ReadDir(current)
				if err != nil {
					return
				}
				for _, entry := range entries {
					childPath := filepath.Join(current, entry.Name())
					walk(childPath, depth+1)
				}
				return
			}

			if !filter.ShouldInclude(current, info) {
				result.FilteredFiles++
				return
			}

			filePaths = append(filePaths, current)
			result.TotalFiles++
			result.TotalSize += info.Size()
		}

		walk(m.directory, 0)

		fileInfos, errors := analyzer.AnalyzeFiles(filePaths, m.directory)
		for i, fileInfo := range fileInfos {
			if errors[i] == nil && fileInfo != nil {
				result.Files = append(result.Files, *fileInfo)
				result.Summary[string(fileInfo.Type)]++
				result.Summary[string(fileInfo.Category)]++
			}
		}

		return rescanCompleteMsg{scanResult: result}
	}
}

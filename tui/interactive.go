package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	messages   []Message
	input      textinput.Model
	processing bool

	// Context
	directory  string
	model      string
	endpoint   string
	scanResult *types.ScanResult
	cfg        *config.Config
	llmClient  *llm.OllamaClient

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

type rescanCompleteMsg struct {
	scanResult *types.ScanResult
	err        error
}

// NewInteractiveModel creates a new interactive mode model
func NewInteractiveModel(directory, model, endpoint string, scanResult *types.ScanResult, cfg *config.Config, llmClient *llm.OllamaClient) InteractiveModel {
	ti := textinput.New()
	ti.Placeholder = "Ask a question about your codebase..."
	ti.Focus()
	ti.CharLimit = 500
	ti.Width = 80

	// Add welcome message
	welcome := Message{
		Role: "assistant",
		Content: fmt.Sprintf("🤖 Interactive mode started!\n\nScanned: %s\nFiles found: %d\nModel: %s\n\nType your questions or commands. Type 'help' for available commands, 'quit' or 'exit' to leave.",
			directory, scanResult.TotalFiles, model),
		Timestamp: time.Now(),
	}

	return InteractiveModel{
		messages:   []Message{welcome},
		input:      ti,
		directory:  directory,
		model:      model,
		endpoint:   endpoint,
		scanResult: scanResult,
		cfg:        cfg,
		llmClient:  llmClient,
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

			m.input.Reset()
			m.processing = true

			// Process the question
			return m, m.processQuestion(userInput)

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
			m.messages = append(m.messages, Message{
				Role:      "assistant",
				Content:   fmt.Sprintf("✅ Rescan complete!\n\nFiles found: %d\nFiltered: %d\nTotal size: %s", msg.scanResult.TotalFiles, msg.scanResult.FilteredFiles, formatBytes(msg.scanResult.TotalSize)),
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
	header := headerStyle.Render(fmt.Sprintf("🤖 Interactive Mode | %s | Files: %d", m.model, m.scanResult.TotalFiles))
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

			if idx := strings.Index(content, "\n\n---\n"); idx != -1 {
				metadata = strings.TrimSpace(content[idx+5:])
				content = strings.TrimSpace(content[:idx])
			}

			// Wrap and indent assistant message
			wrapped := m.wrapMessage(content, m.width-6)
			for _, line := range strings.Split(wrapped, "\n") {
				lines = append(lines, assistantMessageStyle.Render("  "+line))
			}

			// Add metadata at the end if present
			if metadata != "" {
				lines = append(lines, "")
				lines = append(lines, metadataStyle.Render("  "+metadata))
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
		return true

	case "help", "h":
		helpMsg := `Available commands:
• help, h - Show this help message
• model <name> - Switch to a different LLM model
• rescan - Re-scan directory for new/changed files
• stats - Show scan statistics
• files - List scanned files
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

func (m InteractiveModel) processQuestion(question string) tea.Cmd {
	return func() tea.Msg {
		// Prepare file context for LLM
		analyzerEngine := analyzer.NewAnalyzer(m.cfg)

		// Convert []FileInfo to []*FileInfo
		fileInfoPtrs := make([]*types.FileInfo, len(m.scanResult.Files))
		for i := range m.scanResult.Files {
			fileInfoPtrs[i] = &m.scanResult.Files[i]
		}

		// Count readable files and calculate what can fit
		readableCount := 0
		totalTokens := 0
		for _, file := range fileInfoPtrs {
			if file != nil && file.IsReadable && !file.IsSensitive {
				readableCount++
				totalTokens += file.TokenCount
			}
		}

		// Prepare content for LLM
		content := analyzerEngine.PrepareForLLM(fileInfoPtrs, m.cfg.Agent.TokenLimit)

		// Send to LLM
		result, err := m.llmClient.Analyze(question, content, m.cfg.LLM.Temperature)
		if err != nil {
			return processCompleteMsg{
				response: "",
				err:      err,
			}
		}

		// Format response
		var response strings.Builder
		response.WriteString(result.Response)

		// Add metadata on separate lines for better readability
		response.WriteString("\n\n---\n")
		response.WriteString(fmt.Sprintf("📊 Tokens: %d  •  ⏱️  Duration: %.2fs", result.TokensUsed, result.Duration.Seconds()))

		// Warn if files were excluded due to token limits
		if totalTokens > m.cfg.Agent.TokenLimit {
			// Count how many files were actually included
			includedCount := 0
			tempTokens := 0
			for _, file := range fileInfoPtrs {
				if file == nil || !file.IsReadable || file.IsSensitive {
					continue
				}
				if tempTokens+file.TokenCount > m.cfg.Agent.TokenLimit && includedCount > 0 {
					break
				}
				includedCount++
				tempTokens += file.TokenCount
			}

			excluded := readableCount - includedCount
			if excluded > 0 {
				response.WriteString(fmt.Sprintf("\n\n⚠️  Note: %d of %d files excluded (token limit: %d). Consider using filters or increase token_limit in config.",
					excluded, readableCount, m.cfg.Agent.TokenLimit))
			}
		}

		// Save to temp file for 'last' command
		saveAnalysisToTempFile(result, question)

		return processCompleteMsg{
			response: response.String(),
			err:      nil,
		}
	}
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

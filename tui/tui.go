package tui

import (
	"fmt"
	"strings"
	"time"

	"local-agent/types"

	tea "github.com/charmbracelet/bubbletea"
)

// ViewMode represents different screens in the TUI
type ViewMode int

const (
	ViewScanning ViewMode = iota
	ViewResults
	ViewAnalyzing
	ViewFinal
)

// Model represents the main TUI state
type Model struct {
	// View state
	Mode   ViewMode
	Width  int
	Height int

	// Progress tracking
	Progress     float64
	CurrentFile  string
	FilesScanned int
	TotalFiles   int

	// Analysis tracking
	AnalysisProgress []string // Recent progress messages

	// Results
	ScanResult     *types.ScanResult
	AnalysisResult *types.AnalysisResponse

	// UI components
	spinner int
	tick    time.Time

	// Task info
	Task      string
	Directory string
	Model     string
	Endpoint  string

	// Errors
	Errors []string

	// Control
	Quitting bool
}

type scanProgressMsg struct {
	FilesScanned int
	TotalFiles   int
	CurrentFile  string
}

type scanCompleteMsg struct {
	Result *types.ScanResult
}

type analysisProgressMsg struct {
	Message string
}

type analysisCompleteMsg struct {
	Result *types.AnalysisResponse
}

type errorMsg struct {
	Error error
}

type tickMsg time.Time

// New creates a new TUI model
func New(directory, task, model, endpoint string) Model {
	return Model{
		Mode:             ViewScanning,
		Directory:        directory,
		Task:             task,
		Model:            model,
		Endpoint:         endpoint,
		tick:             time.Now(),
		Errors:           make([]string, 0),
		AnalysisProgress: make([]string, 0),
	}
}

func (m Model) Init() tea.Cmd {
	return tickCmd()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.Quitting = true
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height

	case tickMsg:
		m.tick = time.Time(msg)
		m.spinner = (m.spinner + 1) % len(spinnerFrames)
		return m, tickCmd()

	case scanProgressMsg:
		m.FilesScanned = msg.FilesScanned
		m.TotalFiles = msg.TotalFiles
		m.CurrentFile = msg.CurrentFile
		if m.TotalFiles > 0 {
			m.Progress = float64(m.FilesScanned) / float64(m.TotalFiles)
		}

	case scanCompleteMsg:
		m.ScanResult = msg.Result
		m.Mode = ViewResults

	case analysisProgressMsg:
		m.Mode = ViewAnalyzing
		m.AnalysisProgress = append(m.AnalysisProgress, msg.Message)
		// Keep only last 10 messages
		if len(m.AnalysisProgress) > 10 {
			m.AnalysisProgress = m.AnalysisProgress[len(m.AnalysisProgress)-10:]
		}

	case analysisCompleteMsg:
		m.AnalysisResult = msg.Result
		m.Mode = ViewFinal

	case errorMsg:
		m.Errors = append(m.Errors, msg.Error.Error())
	}

	return m, nil
}

func (m Model) View() string {
	if m.Quitting {
		return ""
	}

	switch m.Mode {
	case ViewScanning:
		return m.renderScanningView()
	case ViewResults:
		return m.renderResultsView()
	case ViewAnalyzing:
		return m.renderAnalyzingView()
	case ViewFinal:
		return m.renderFinalView()
	default:
		return "Unknown view"
	}
}

func (m Model) renderScanningView() string {
	var s strings.Builder

	// Header
	s.WriteString(titleStyle.Render("🔍 Local Agent - Scanning"))
	s.WriteString("\n\n")

	// Info
	s.WriteString(labelStyle.Render("Directory: ") + m.Directory + "\n")
	s.WriteString(labelStyle.Render("Task: ") + m.Task + "\n")
	s.WriteString(labelStyle.Render("LLM: ") + m.Model + " @ " + m.Endpoint + "\n\n")

	// Progress
	spinner := spinnerFrames[m.spinner]
	s.WriteString(fmt.Sprintf("%s Scanning files...\n\n", spinner))

	if m.TotalFiles > 0 {
		progress := m.renderProgressBar(m.Progress, 40)
		s.WriteString(progress + "\n")
		s.WriteString(fmt.Sprintf("  %d / %d files scanned\n\n", m.FilesScanned, m.TotalFiles))
	}

	if m.CurrentFile != "" {
		s.WriteString(subtleStyle.Render("Current: "+truncate(m.CurrentFile, 60)) + "\n")
	}

	// Footer
	s.WriteString("\n" + helpStyle.Render("Press q to quit"))

	return s.String()
}

func (m Model) renderResultsView() string {
	var s strings.Builder

	if m.ScanResult == nil {
		return "No scan results available"
	}

	// Header
	s.WriteString(titleStyle.Render("📊 Scan Complete"))
	s.WriteString("\n\n")

	// Statistics
	stats := [][]string{
		{"Total files found:", fmt.Sprintf("%d", m.ScanResult.TotalFiles)},
		{"Filtered files:", fmt.Sprintf("%d", m.ScanResult.FilteredFiles)},
		{"Total size:", formatBytes(m.ScanResult.TotalSize)},
		{"Duration:", m.ScanResult.Duration.String()},
	}

	for _, stat := range stats {
		s.WriteString(labelStyle.Render(stat[0]) + " " + stat[1] + "\n")
	}

	// File breakdown
	if len(m.ScanResult.Summary) > 0 {
		s.WriteString("\n" + sectionStyle.Render("File Breakdown:") + "\n")
		for key, count := range m.ScanResult.Summary {
			s.WriteString(fmt.Sprintf("  • %s: %d\n", key, count))
		}
	}

	// Errors
	if len(m.ScanResult.Errors) > 0 {
		s.WriteString("\n" + errorStyle.Render(fmt.Sprintf("⚠️  Errors: %d", len(m.ScanResult.Errors))) + "\n")
		for i, err := range m.ScanResult.Errors {
			if i < 5 {
				s.WriteString(subtleStyle.Render(fmt.Sprintf("  %s: %s\n", err.Path, err.Error)))
			}
		}
		if len(m.ScanResult.Errors) > 5 {
			s.WriteString(subtleStyle.Render(fmt.Sprintf("  ... and %d more\n", len(m.ScanResult.Errors)-5)))
		}
	}

	// Footer
	s.WriteString("\n" + helpStyle.Render("Press q to quit"))

	return s.String()
}

func (m Model) renderAnalyzingView() string {
	var s strings.Builder

	// Header
	s.WriteString(titleStyle.Render("🔬 Analyzing Files"))
	s.WriteString("\n\n")

	spinner := spinnerFrames[m.spinner]
	s.WriteString(fmt.Sprintf("%s Running LLM analysis...\n\n", spinner))

	s.WriteString(labelStyle.Render("Task: ") + m.Task + "\n")
	s.WriteString(labelStyle.Render("Model: ") + m.Model + "\n\n")

	// Show recent progress messages
	if len(m.AnalysisProgress) > 0 {
		s.WriteString(sectionStyle.Render("Progress:") + "\n")
		for _, msg := range m.AnalysisProgress {
			s.WriteString(subtleStyle.Render("  "+msg) + "\n")
		}
	}

	// Footer
	s.WriteString("\n" + helpStyle.Render("Press q to quit"))

	return s.String()
}

func (m Model) renderFinalView() string {
	var s strings.Builder

	if m.AnalysisResult == nil {
		return "No analysis results available"
	}

	// Header
	s.WriteString(titleStyle.Render("🎯 Analysis Complete"))
	s.WriteString("\n\n")

	// Metadata
	s.WriteString(labelStyle.Render("Total duration: ") + m.AnalysisResult.Duration.String() + "\n")

	// Token usage per file
	if len(m.AnalysisResult.FileTokens) > 0 {
		s.WriteString("\n" + sectionStyle.Render("📊 Token usage per file:") + "\n")
		for file, tokens := range m.AnalysisResult.FileTokens {
			s.WriteString(fmt.Sprintf("   %s: %d tokens\n", file, tokens))
		}
	}
	s.WriteString("\n")

	// Response
	s.WriteString(sectionStyle.Render("📝 Response:") + "\n")
	s.WriteString(wrapText(m.AnalysisResult.Response, 80) + "\n\n")

	// Findings
	if len(m.AnalysisResult.Findings) > 0 {
		s.WriteString(sectionStyle.Render("🔍 Findings:") + "\n")
		for i, finding := range m.AnalysisResult.Findings {
			severityColor := getSeverityStyle(string(finding.Severity))
			s.WriteString(fmt.Sprintf("  %d. %s %s\n", i+1, severityColor.Render("["+string(finding.Severity)+"]"), finding.Description))

			if finding.File != "" {
				location := fmt.Sprintf("File: %s", finding.File)
				if finding.Line > 0 {
					location += fmt.Sprintf(" (Line %d)", finding.Line)
				}
				s.WriteString(subtleStyle.Render("     "+location) + "\n")
			}

			if finding.Suggestion != "" {
				s.WriteString(subtleStyle.Render("     💡 "+finding.Suggestion) + "\n")
			}
			s.WriteString("\n")
		}
	}

	// Footer
	s.WriteString(helpStyle.Render("Analysis complete. Press q to quit"))

	return s.String()
}

func (m Model) renderProgressBar(progress float64, width int) string {
	filled := int(progress * float64(width))
	if filled > width {
		filled = width
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	percent := fmt.Sprintf(" %.0f%%", progress*100)

	return progressStyle.Render(bar) + percent
}

// Commands
func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Public methods to send messages to the model
func SendScanProgress(filesScanned, totalFiles int, currentFile string) tea.Msg {
	return scanProgressMsg{
		FilesScanned: filesScanned,
		TotalFiles:   totalFiles,
		CurrentFile:  currentFile,
	}
}

func SendScanComplete(result *types.ScanResult) tea.Msg {
	return scanCompleteMsg{Result: result}
}

func SendAnalysisProgress(message string) tea.Msg {
	return analysisProgressMsg{Message: message}
}

func SendAnalysisComplete(result *types.AnalysisResponse) tea.Msg {
	return analysisCompleteMsg{Result: result}
}

func SendError(err error) tea.Msg {
	return errorMsg{Error: err}
}

// Utility functions
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func wrapText(text string, width int) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}

	var lines []string
	var currentLine strings.Builder

	for _, word := range words {
		if currentLine.Len() == 0 {
			currentLine.WriteString(word)
		} else if currentLine.Len()+1+len(word) <= width {
			currentLine.WriteString(" " + word)
		} else {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			currentLine.WriteString(word)
		}
	}

	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	return strings.Join(lines, "\n")
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

package main

import (
	"flag"
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
	"local-agent/tui"
	"local-agent/types"

	tea "github.com/charmbracelet/bubbletea"
)

const version = "0.1.0"

func main() {
	// Define CLI flags
	var (
		configPath = flag.String("config", "", "Path to configuration file")
		task       = flag.String("task", "", "Analysis task description")
		directory  = flag.String("dir", ".", "Directory to analyze")
		model      = flag.String("model", "", "LLM model to use (overrides config)")
		dryRun     = flag.Bool("dry-run", false, "List files without analyzing")
		viewLast   = flag.Bool("view-last", false, "View the last saved analysis")

		showVersion = flag.Bool("version", false, "Show version")
		checkHealth = flag.Bool("health", false, "Check LLM connectivity")
		listModels  = flag.Bool("list-models", false, "List available LLM models")
		interactive = flag.Bool("interactive", false, "Start interactive mode")
	)

	flag.Parse()

	// View last analysis
	if *viewLast {
		viewLastAnalysis()
		return
	}

	// Show version
	if *showVersion {
		fmt.Printf("local-agent version %s\n", version)
		return
	}

	// Load configuration
	cfg := config.LoadConfigWithFallback(*configPath)
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid configuration: %v\n", err)
		os.Exit(1)
	}

	// Override model if specified via flag
	if *model != "" {
		cfg.LLM.Model = *model
	}

	// Initialize LLM client
	llmClient := llm.NewOllamaClient(cfg.LLM.Endpoint, cfg.LLM.Model, cfg.LLM.Timeout)

	// Handle health check
	if *checkHealth {
		checkLLMHealth(llmClient)
		return
	}

	// Handle list models
	if *listModels {
		listAvailableModels(llmClient)
		return
	}

	// Validate directory
	absDir, err := filepath.Abs(*directory)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid directory: %v\n", err)
		os.Exit(1)
	}

	if _, err := os.Stat(absDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Directory does not exist: %s\n", absDir)
		os.Exit(1)
	}

	// If interactive mode requested, start the interactive session
	if *interactive {
		startInteractiveMode(absDir, cfg, llmClient)
		return
	}

	// Run agent
	fmt.Printf("🔍 Local Agent v%s\n", version)
	fmt.Printf("📁 Analyzing directory: %s\n", absDir)
	fmt.Printf("🤖 LLM: %s @ %s\n\n", cfg.LLM.Model, cfg.LLM.Endpoint)

	// Scan files
	result, err := scanDirectory(absDir, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to scan directory: %v\n", err)
		os.Exit(1)
	}

	// Display scan results
	displayScanResult(result)

	// If dry-run, stop here
	if *dryRun {
		return
	}

	// Verify task is provided
	if *task == "" {
		fmt.Fprintf(os.Stderr, "\nError: --task is required for analysis\n")
		fmt.Fprintf(os.Stderr, "Example: --task \"check for security issues\"\n")
		os.Exit(1)
	}

	// Perform analysis
	fmt.Printf("\n🔬 Analyzing files with task: %s\n\n", *task)

	analysisResult, err := analyzeFiles(result, *task, cfg, llmClient)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Analysis failed: %v\n", err)
		os.Exit(1)
	}

	// Display analysis results
	displayAnalysisResult(analysisResult)

	// Save analysis to temp file for later viewing
	tempFile, err := saveAnalysisToTemp(analysisResult, *task)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n⚠️  Failed to save to temp file: %v\n", err)
	} else {
		fmt.Printf("\n💾 Results also saved to: %s\n", tempFile)
		fmt.Printf("💡 View anytime with: ./local-agent --view-last\n")
	}
}

func scanDirectory(rootPath string, cfg *config.Config) (*types.ScanResult, error) {
	startTime := time.Now()

	// Initialize components
	fileFilter, err := filter.NewFilter(cfg, rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize filter: %w", err)
	}

	analyzer := analyzer.NewAnalyzer(cfg)
	validator := security.NewValidator()

	result := &types.ScanResult{
		RootPath: rootPath,
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
			result.Errors = append(result.Errors, types.ScanError{Path: current, Error: err.Error(), Time: time.Now()})
			return
		}

		// Follow symlinks when enabled
		if info.Mode()&os.ModeSymlink != 0 {
			if !fileFilter.ShouldFollowSymlink(current) {
				return
			}

			target, err := filepath.EvalSymlinks(current)
			if err != nil {
				result.Errors = append(result.Errors, types.ScanError{Path: current, Error: err.Error(), Time: time.Now()})
				return
			}

			targetAbs, _ := filepath.Abs(target)

			// Avoid cycles
			if _, seen := visitedDirs[targetAbs]; seen {
				return
			}

			info, err = os.Stat(targetAbs)
			if err != nil {
				result.Errors = append(result.Errors, types.ScanError{Path: targetAbs, Error: err.Error(), Time: time.Now()})
				return
			}

			current = targetAbs
		}

		// Validate path traversal
		if err := validator.ValidatePath(current); err != nil {
			return
		}

		if info.IsDir() {
			if !fileFilter.IsWithinDepthLimit(depth) {
				return
			}

			absDir, _ := filepath.Abs(current)
			visitedDirs[absDir] = struct{}{}

			entries, err := os.ReadDir(current)
			if err != nil {
				result.Errors = append(result.Errors, types.ScanError{Path: current, Error: err.Error(), Time: time.Now()})
				return
			}

			for _, entry := range entries {
				childPath := filepath.Join(current, entry.Name())
				walk(childPath, depth+1)
			}
			return
		}

		// Apply filters to files
		if !fileFilter.ShouldInclude(current, info) {
			result.FilteredFiles++
			return
		}

		filePaths = append(filePaths, current)
		result.TotalFiles++
		result.TotalSize += info.Size()
	}

	walk(rootPath, 0)

	// Analyze files
	fileInfos, errors := analyzer.AnalyzeFiles(filePaths, rootPath)

	for i, fileInfo := range fileInfos {
		if errors[i] != nil {
			result.Errors = append(result.Errors, types.ScanError{
				Path:  filePaths[i],
				Error: errors[i].Error(),
				Time:  time.Now(),
			})
			continue
		}

		if fileInfo != nil {
			result.Files = append(result.Files, *fileInfo)

			// Update summary
			result.Summary[string(fileInfo.Type)]++
			result.Summary[string(fileInfo.Category)]++
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

func analyzeFiles(scanResult *types.ScanResult, task string, cfg *config.Config, llmClient *llm.OllamaClient) (*types.AnalysisResponse, error) {
	// Prepare files for LLM
	analyzer := analyzer.NewAnalyzer(cfg)

	// Convert []FileInfo to []*FileInfo
	fileInfoPtrs := make([]*types.FileInfo, len(scanResult.Files))
	for i := range scanResult.Files {
		fileInfoPtrs[i] = &scanResult.Files[i]
	}

	// Check if we need to batch process
	totalTokens := 0
	for _, file := range fileInfoPtrs {
		if file != nil && file.IsReadable && !file.IsSensitive {
			totalTokens += file.TokenCount
		}
	}

	// If total tokens exceed limit, process in batches
	if totalTokens > cfg.Agent.TokenLimit {
		return analyzeBatches(fileInfoPtrs, task, cfg, llmClient, analyzer)
	}

	// Process all files at once
	content := analyzer.PrepareForLLM(fileInfoPtrs, cfg.Agent.TokenLimit)

	// Send to LLM
	response, err := llmClient.Analyze(task, content, cfg.LLM.Temperature)
	if err != nil {
		return nil, fmt.Errorf("LLM analysis failed: %w", err)
	}

	return response, nil
}

func analyzeBatches(files []*types.FileInfo, task string, cfg *config.Config, llmClient *llm.OllamaClient, analyzer *analyzer.Analyzer) (*types.AnalysisResponse, error) {
	var allResponses []string
	var totalTokens int
	var totalDuration time.Duration
	model := ""

	fmt.Printf("\n📦 Processing files in batches (too many files for single analysis)\n")

	currentBatch := make([]*types.FileInfo, 0)
	currentTokens := 0
	batchNum := 1

	for _, file := range files {
		if file == nil || !file.IsReadable {
			continue
		}

		// If adding this file exceeds token limit, process current batch
		if currentTokens+file.TokenCount > cfg.Agent.TokenLimit && len(currentBatch) > 0 {
			fmt.Printf("   Processing batch %d (%d files)...\n", batchNum, len(currentBatch))
			response, err := processBatch(currentBatch, task, cfg, llmClient, analyzer)
			if err != nil {
				fmt.Printf("   ⚠️  Batch %d failed: %v\n", batchNum, err)
			} else {
				allResponses = append(allResponses, fmt.Sprintf("\n=== Batch %d ===%s", batchNum, response.Response))
				totalTokens += response.TokensUsed
				totalDuration += response.Duration
				if model == "" {
					model = response.Model
				}
			}

			// Reset for next batch
			currentBatch = make([]*types.FileInfo, 0)
			currentTokens = 0
			batchNum++
		}

		currentBatch = append(currentBatch, file)
		currentTokens += file.TokenCount
	}

	// Process remaining batch
	if len(currentBatch) > 0 {
		fmt.Printf("   Processing batch %d (%d files)...\n", batchNum, len(currentBatch))
		response, err := processBatch(currentBatch, task, cfg, llmClient, analyzer)
		if err != nil {
			fmt.Printf("   ⚠️  Batch %d failed: %v\n", batchNum, err)
		} else {
			allResponses = append(allResponses, fmt.Sprintf("\n=== Batch %d ===%s", batchNum, response.Response))
			totalTokens += response.TokensUsed
			totalDuration += response.Duration
			if model == "" {
				model = response.Model
			}
		}
	}

	// Aggregate results
	aggregatedResponse := &types.AnalysisResponse{
		Response:   strings.Join(allResponses, "\n"),
		Model:      model,
		TokensUsed: totalTokens,
		Duration:   totalDuration,
	}

	return aggregatedResponse, nil
}

func processBatch(batch []*types.FileInfo, task string, cfg *config.Config, llmClient *llm.OllamaClient, analyzer *analyzer.Analyzer) (*types.AnalysisResponse, error) {
	content := analyzer.PrepareForLLM(batch, cfg.Agent.TokenLimit)
	return llmClient.Analyze(task, content, cfg.LLM.Temperature)
}

func displayScanResult(result *types.ScanResult) {
	fmt.Printf("📊 Scan Results\n")
	fmt.Printf("   Total files found: %d\n", result.TotalFiles)
	fmt.Printf("   Filtered files: %d\n", result.FilteredFiles)
	fmt.Printf("   Total size: %s\n", formatBytes(result.TotalSize))
	fmt.Printf("   Duration: %v\n", result.Duration)

	if len(result.Summary) > 0 {
		fmt.Printf("\n   File breakdown:\n")
		for key, count := range result.Summary {
			fmt.Printf("      %s: %d\n", key, count)
		}
	}

	if len(result.Errors) > 0 {
		fmt.Printf("\n   ⚠️  Errors: %d\n", len(result.Errors))
		for _, e := range result.Errors {
			fmt.Printf("      %s: %s\n", e.Path, e.Error)
		}
	}
}

func displayAnalysisResult(result *types.AnalysisResponse) {
	fmt.Printf("🎯 Analysis Complete\n")
	fmt.Printf("   Model: %s\n", result.Model)
	fmt.Printf("   Tokens used: %d\n", result.TokensUsed)
	fmt.Printf("   Duration: %v\n\n", result.Duration)

	fmt.Printf("📝 Response:\n")
	fmt.Printf("%s\n", result.Response)

	if len(result.Findings) > 0 {
		fmt.Printf("\n🔍 Findings:\n")
		for i, finding := range result.Findings {
			fmt.Printf("   %d. [%s] %s\n", i+1, finding.Severity, finding.Description)
			if finding.File != "" {
				fmt.Printf("      File: %s", finding.File)
				if finding.Line > 0 {
					fmt.Printf(" (Line %d)", finding.Line)
				}
				fmt.Printf("\n")
			}
			if finding.Suggestion != "" {
				fmt.Printf("      Suggestion: %s\n", finding.Suggestion)
			}
		}
	}
}

func checkLLMHealth(client *llm.OllamaClient) {
	fmt.Printf("🏥 Checking LLM health...\n")

	if client.IsAvailable() {
		fmt.Printf("✅ LLM is available at %s\n", client.GetModel())

		// Try to list models
		models, err := client.ListModels()
		if err == nil && len(models) > 0 {
			fmt.Printf("📋 Available models: %d\n", len(models))
			for _, model := range models {
				fmt.Printf("   - %s\n", model)
			}
		}
	} else {
		fmt.Printf("❌ LLM is not available\n")
		fmt.Printf("   Make sure Ollama is running: ollama serve\n")
		os.Exit(1)
	}
}

func listAvailableModels(client *llm.OllamaClient) {
	models, err := client.ListModels()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list models: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📋 Available Models:\n")
	for _, model := range models {
		fmt.Printf("   - %s\n", model)
	}
}

func saveAnalysisToTemp(result *types.AnalysisResponse, task string) (string, error) {
	tempDir := os.TempDir()
	tempFile := filepath.Join(tempDir, fmt.Sprintf("local-agent-analysis-%d.txt", time.Now().Unix()))
	lastFile := filepath.Join(tempDir, "local-agent-last.txt")

	var content strings.Builder
	content.WriteString("Analysis Results\n")
	content.WriteString(fmt.Sprintf("Generated: %s\n", time.Now().Format(time.RFC3339)))
	content.WriteString(fmt.Sprintf("Model: %s\n", result.Model))
	content.WriteString(fmt.Sprintf("Tokens: %d\n", result.TokensUsed))
	content.WriteString(fmt.Sprintf("Duration: %v\n", result.Duration))
	content.WriteString("\n" + strings.Repeat("=", 80) + "\n\n")
	content.WriteString(fmt.Sprintf("TASK:\n%s\n\n", task))
	content.WriteString(strings.Repeat("-", 80) + "\n\n")
	content.WriteString(fmt.Sprintf("RESPONSE:\n%s\n", result.Response))

	contentBytes := []byte(content.String())

	// Save to timestamped file
	if err := os.WriteFile(tempFile, contentBytes, 0644); err != nil {
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}

	// Save to last file for --view-last
	if err := os.WriteFile(lastFile, contentBytes, 0644); err != nil {
		// Don't fail if we can't write last file
		fmt.Fprintf(os.Stderr, "Warning: failed to save last file: %v\n", err)
	}

	return tempFile, nil
}

func viewLastAnalysis() {
	lastFile := filepath.Join(os.TempDir(), "local-agent-last.txt")

	content, err := os.ReadFile(lastFile)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "No previous analysis found.\n")
			fmt.Fprintf(os.Stderr, "Run an analysis first with: ./local-agent -dir <path> -task \"<task>\"\n")
		} else {
			fmt.Fprintf(os.Stderr, "Failed to read last analysis: %v\n", err)
		}
		os.Exit(1)
	}

	fmt.Print(string(content))
}

func displayAnalysisSummary(result *types.AnalysisResponse) {
	fmt.Printf("📊 Summary:\n")
	fmt.Printf("   Model: %s\n", result.Model)
	fmt.Printf("   Tokens: %d\n", result.TokensUsed)
	fmt.Printf("   Duration: %v\n", result.Duration)
	if len(result.Findings) > 0 {
		fmt.Printf("   Findings: %d\n", len(result.Findings))
	}
}

func startInteractiveMode(directory string, cfg *config.Config, llmClient *llm.OllamaClient) {
	fmt.Printf("🔍 Scanning directory: %s\n", directory)
	fmt.Printf("⏳ Please wait...\n\n")

	// Perform initial scan
	scanResult, err := scanDirectory(directory, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to scan directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Scan complete! Found %d files\n", scanResult.TotalFiles)
	fmt.Printf("🚀 Starting interactive mode...\n\n")

	// Start interactive TUI
	m := tui.NewInteractiveModel(directory, cfg.LLM.Model, cfg.LLM.Endpoint, scanResult, cfg, llmClient)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running interactive mode: %v\n", err)
		os.Exit(1)
	}
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

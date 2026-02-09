package main

import (
	"flag"
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
	"local-agent/sessionlog"
	"local-agent/tui"
	"local-agent/types"
	"local-agent/webui"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	version   = "0.1.0"
	ansiGreen = "\033[32m"
	ansiReset = "\033[0m"
)

func main() {
	// Define CLI flags
	var (
		configPath      = flag.String("config", "", "Path to configuration file")
		task            = flag.String("task", "", "Analysis task description")
		directory       = flag.String("dir", ".", "Directory to analyze")
		focusFile       = flag.String("focus", "", "Analyze only this file (relative to --dir; if outside, directory adjusts automatically)")
		model           = flag.String("model", "", "LLM model to use (overrides config)")
		dryRun          = flag.Bool("dry-run", false, "List files without analyzing")
		noDetectSecrets = flag.Bool("no-detect-secrets", false, "Disable secret/sensitive content detection")

		showVersion = flag.Bool("version", false, "Show version")
		checkHealth = flag.Bool("health", false, "Check LLM connectivity")
		listModels  = flag.Bool("list-models", false, "List available LLM models")
		interactive = flag.Bool("interactive", false, "Start interactive mode")
	)

	flag.Parse()

	var dirFlagSet bool
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "dir" {
			dirFlagSet = true
		}
	})

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

	// Override secret detection if disabled via flag
	if *noDetectSecrets {
		cfg.Security.DetectSecrets = false
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

	focusRel := ""
	if *focusFile != "" {
		originalDir := absDir
		var newDir string
		focusRel, newDir, err = resolveFocusPath(absDir, *focusFile, dirFlagSet)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid --focus path: %v\n", err)
			os.Exit(1)
		}
		if newDir != "" {
			absDir = newDir
		}
		if absDir != originalDir {
			fmt.Printf("🎯 Adjusted analysis root to %s based on --focus target.\n", absDir)
		}
	}

	// If interactive mode requested, start the interactive session
	if *interactive {
		startInteractiveMode(absDir, cfg, llmClient, focusRel)
		return
	}

	// Run agent
	fmt.Printf("🔍 Local Agent v%s\n", version)
	fmt.Printf("📁 Analyzing directory: %s\n", absDir)
	fmt.Printf("🤖 LLM: %s @ %s\n", cfg.LLM.Model, cfg.LLM.Endpoint)
	fmt.Printf("⚙️ Configuration:\n")
	fmt.Printf("   Token Limit: %d\n", cfg.Agent.TokenLimit)
	fmt.Printf("   Concurrent Files: %d\n", cfg.Agent.ConcurrentFiles)
	fmt.Printf("   Temperature: %.2f\n\n", cfg.LLM.Temperature)

	// Scan files
	result, err := scanDirectory(absDir, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to scan directory: %v\n", err)
		os.Exit(1)
	}

	if focusRel != "" && !scanResultHasFile(result, focusRel) {
		fmt.Fprintf(os.Stderr, "Focused file %s was not included in the scan (check filters and path).\n", focusRel)
		os.Exit(1)
	}

	// Display scan results
	displayScanResult(result)
	if focusRel != "" {
		fmt.Printf("\n🎯 Focus enabled: %s\nOnly this file will be analyzed.\n", focusRel)
	}

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

	analysisResult, err := analyzeFiles(result, focusRel, *task, cfg, llmClient)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Analysis failed: %v\n", err)
		os.Exit(1)
	}

	// Display analysis results
	displayAnalysisResult(analysisResult)

	// Persist session details as JSON
	saveSessionRecord("standalone", absDir, focusRel, *task, cfg.LLM.Model, result, analysisResult)
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

func analyzeFiles(scanResult *types.ScanResult, focusRel string, task string, cfg *config.Config, llmClient *llm.OllamaClient) (*types.AnalysisResponse, error) {
	// Prepare files for LLM
	analyzer := analyzer.NewAnalyzer(cfg)

	fileInfoPtrs, err := selectFilesForAnalysis(scanResult, focusRel)
	if err != nil {
		return nil, err
	}

	// Always process files individually (one request per file)
	return analyzeBatches(fileInfoPtrs, task, cfg, llmClient, analyzer)
}

func analyzeBatches(files []*types.FileInfo, task string, cfg *config.Config, llmClient *llm.OllamaClient, analyzer *analyzer.Analyzer) (*types.AnalysisResponse, error) {
	fmt.Printf("\n📦 Processing files individually (one request per file)\n")

	// Prepare batches (one file per batch)
	batches := prepareBatches(files, cfg.Agent.TokenLimit)
	totalBatches := len(batches)

	if totalBatches == 0 {
		return &types.AnalysisResponse{
			Response: "No files to analyze",
			Model:    cfg.LLM.Model,
		}, nil
	}

	fmt.Printf("   Processing %d files\n", totalBatches)

	// Determine concurrency level
	maxConcurrent := cfg.Agent.ConcurrentFiles
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}

	// If only 1 worker or 1 file, process sequentially
	if maxConcurrent == 1 || totalBatches == 1 {
		return processSequentially(batches, task, cfg, llmClient, analyzer)
	}

	fmt.Printf("   Using %d concurrent workers\n", maxConcurrent)

	// Process batches concurrently
	return processConcurrently(batches, task, cfg, llmClient, analyzer, maxConcurrent)
}

// prepareBatches creates one batch per file for individual processing
// Filters out files that exceed token limit
func prepareBatches(files []*types.FileInfo, tokenLimit int) [][]*types.FileInfo {
	var batches [][]*types.FileInfo

	for _, file := range files {
		if file == nil || !file.IsReadable {
			continue
		}

		// Skip files that exceed token limit
		if file.TokenCount > tokenLimit {
			fmt.Printf("   ⚠️  Skipping %s (%d tokens exceeds limit of %d)\n",
				file.RelPath, file.TokenCount, tokenLimit)
			continue
		}

		// Each file is its own batch
		batches = append(batches, []*types.FileInfo{file})
	}

	return batches
}

// processSequentially processes files one at a time
func processSequentially(batches [][]*types.FileInfo, task string, cfg *config.Config, llmClient *llm.OllamaClient, analyzer *analyzer.Analyzer) (*types.AnalysisResponse, error) {
	var allResponses []string
	var totalTokens int
	var totalDuration time.Duration
	model := ""

	for i, batch := range batches {
		fileNum := i + 1
		fileName := batch[0].RelPath // Each batch has one file
		fmt.Printf("   Processing file %d/%d: %s\n", fileNum, len(batches), fileName)

		response, err := processBatch(batch, task, cfg, llmClient, analyzer)
		if err != nil {
			fmt.Printf("   ⚠️  File %d (%s) failed: %v\n", fileNum, fileName, err)
			allResponses = append(allResponses, formatFileErrorSection(fileName, err))
		} else {
			allResponses = append(allResponses, formatFileSection(fileName, response.Response))
			totalTokens += response.TokensUsed
			totalDuration += response.Duration
			if model == "" {
				model = response.Model
			}
			fmt.Printf("   ✅ File %d completed\n", fileNum)
		}
	}

	return &types.AnalysisResponse{
		Response:   strings.Join(allResponses, "\n"),
		Model:      model,
		TokensUsed: totalTokens,
		Duration:   totalDuration,
	}, nil
}

// batchJob represents a batch processing job
type batchJob struct {
	batchNum int
	batch    []*types.FileInfo
}

// batchResult represents the result of processing a batch
type batchResult struct {
	batchNum int
	response *types.AnalysisResponse
	err      error
}

// processConcurrently processes batches concurrently using worker pool
func processConcurrently(batches [][]*types.FileInfo, task string, cfg *config.Config, llmClient *llm.OllamaClient, analyzer *analyzer.Analyzer, maxWorkers int) (*types.AnalysisResponse, error) {
	totalFiles := len(batches)

	// Create channels
	jobs := make(chan batchJob, totalFiles)
	results := make(chan batchResult, totalFiles)

	// Build filename map
	fileNames := make(map[int]string)
	for i, batch := range batches {
		fileNames[i+1] = batch[0].RelPath
	}

	// Start worker pool
	var wg sync.WaitGroup
	for w := 1; w <= maxWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range jobs {
				fileName := fileNames[job.batchNum]
				fmt.Printf("   [Worker %d] Processing file %d/%d: %s\n",
					workerID, job.batchNum, totalFiles, fileName)

				response, err := processBatch(job.batch, task, cfg, llmClient, analyzer)
				results <- batchResult{
					batchNum: job.batchNum,
					response: response,
					err:      err,
				}
			}
		}(w)
	}

	// Send jobs to workers
	for i, batch := range batches {
		jobs <- batchJob{
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
	fileTokens := make(map[string]int) // Track tokens per file
	var totalTokens int
	var totalDuration time.Duration
	model := ""
	failedFiles := make(map[int]error)

	for result := range results {
		fileName := fileNames[result.batchNum]
		if result.err != nil {
			fmt.Printf("   ⚠️  File %s failed: %v\n", fileName, result.err)
			failedFiles[result.batchNum] = result.err
		} else {
			fileResults[result.batchNum] = result.response
			fileTokens[fileName] = result.response.TokensUsed
			totalTokens += result.response.TokensUsed
			totalDuration += result.response.Duration
			if model == "" {
				model = result.response.Model
			}
			fmt.Printf("   ✅ File %s completed\n", fileName)
		}
	}

	// Aggregate results in order, including failed files
	var allResponses []string
	for i := 1; i <= totalFiles; i++ {
		fileName := fileNames[i]
		if response, ok := fileResults[i]; ok {
			// Successful file
			allResponses = append(allResponses, formatFileSection(fileName, response.Response))
		} else if err, failed := failedFiles[i]; failed {
			// Failed file - include error message
			allResponses = append(allResponses, formatFileErrorSection(fileName, err))
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
		TokensUsed: totalTokens,
		FileTokens: fileTokens,
		Duration:   totalDuration,
	}, nil
}

func processBatch(batch []*types.FileInfo, task string, cfg *config.Config, llmClient *llm.OllamaClient, analyzer *analyzer.Analyzer) (*types.AnalysisResponse, error) {
	// Show file info being processed
	for _, file := range batch {
		if file != nil {
			fmt.Printf("   [INFO] File: %s, Tokens: %d, Content length: %d bytes, IsReadable: %v\n",
				file.RelPath, file.TokenCount, len(file.Content), file.IsReadable)
		}
	}

	content := analyzer.PrepareForLLM(batch, cfg.Agent.TokenLimit)

	// Check if we have any actual content to analyze
	if len(content) < 100 { // Less than 100 bytes means essentially empty (just headers)
		return nil, fmt.Errorf("no valid content to analyze after PrepareForLLM")
	}

	// For single file batches, add filename to task for clarity
	actualTask := task
	if len(batch) == 1 && batch[0] != nil {
		actualTask = fmt.Sprintf("Analyze the file '%s'. %s", batch[0].RelPath, task)
	}

	return llmClient.Analyze(actualTask, content, cfg.LLM.Temperature)
}

func formatFileSection(fileName, body string) string {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return "\n" + formatFileHeaderLine(fileName)
	}
	return fmt.Sprintf("\n%s\n%s", formatFileHeaderLine(fileName), trimmed)
}

func formatFileErrorSection(fileName string, err error) string {
	return fmt.Sprintf("\n%s\n⚠️  FAILED: %v", formatFileHeaderLine(fileName), err)
}

func formatFileHeaderLine(fileName string) string {
	return fmt.Sprintf("=== %s%s%s ===", ansiGreen, fileName, ansiReset)
}

func selectFilesForAnalysis(scanResult *types.ScanResult, focusRel string) ([]*types.FileInfo, error) {
	if focusRel == "" {
		fileInfoPtrs := make([]*types.FileInfo, len(scanResult.Files))
		for i := range scanResult.Files {
			fileInfoPtrs[i] = &scanResult.Files[i]
		}
		return fileInfoPtrs, nil
	}

	normalizedFocus := normalizeRelPath(focusRel)
	for i := range scanResult.Files {
		if normalizeRelPath(scanResult.Files[i].RelPath) == normalizedFocus {
			return []*types.FileInfo{&scanResult.Files[i]}, nil
		}
	}

	return nil, fmt.Errorf("focused file %s not found in scan results (possibly filtered out)", focusRel)
}

func scanResultHasFile(result *types.ScanResult, relPath string) bool {
	normalized := normalizeRelPath(relPath)
	for _, file := range result.Files {
		if normalizeRelPath(file.RelPath) == normalized {
			return true
		}
	}
	return false
}

func normalizeRelPath(path string) string {
	return filepath.ToSlash(filepath.Clean(path))
}

func resolveFocusPath(rootDir, focusInput string, dirLocked bool) (string, string, error) {
	if focusInput == "" {
		return "", rootDir, nil
	}

	focusPath := focusInput
	if !filepath.IsAbs(focusPath) {
		focusPath = filepath.Join(rootDir, focusPath)
	}

	absFocus, err := filepath.Abs(focusPath)
	if err != nil {
		return "", rootDir, err
	}

	info, err := os.Stat(absFocus)
	if err != nil {
		return "", rootDir, err
	}
	if info.IsDir() {
		return "", rootDir, fmt.Errorf("%s is a directory; provide a file path for --focus", focusInput)
	}

	rel, err := filepath.Rel(rootDir, absFocus)
	if err == nil {
		rel = normalizeRelPath(rel)
		if rel != ".." && !strings.HasPrefix(rel, "../") {
			return rel, rootDir, nil
		}
	}

	if dirLocked {
		return "", rootDir, fmt.Errorf("focus file must be inside the target directory (%s)", rootDir)
	}

	newRoot := filepath.Dir(absFocus)
	relToNewRoot, err := filepath.Rel(newRoot, absFocus)
	if err != nil {
		return "", rootDir, err
	}

	return normalizeRelPath(relToNewRoot), newRoot, nil
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
	fmt.Printf("   Total duration: %v\n", result.Duration)

	if len(result.FileTokens) > 0 {
		fmt.Printf("\n   📊 Token usage per file:\n")
		for file, tokens := range result.FileTokens {
			fmt.Printf("      %s: %d tokens\n", file, tokens)
		}
	}
	fmt.Printf("\n")

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

func saveSessionRecord(mode, directory, focus, task, model string, scanResult *types.ScanResult, analysisResult *types.AnalysisResponse) {
	record := buildSessionRecord(mode, directory, focus, task, model, scanResult, analysisResult)
	if record == nil {
		return
	}

	path, err := sessionlog.Save(record)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save session JSON: %v\n", err)
		return
	}

	fmt.Printf("🗂️ Session saved to: %s\n", path)
}

func buildSessionRecord(mode, directory, focus, task, model string, scanResult *types.ScanResult, analysisResult *types.AnalysisResponse) *sessionlog.Record {
	if analysisResult == nil {
		return nil
	}

	record := &sessionlog.Record{
		Timestamp:  time.Now(),
		Mode:       mode,
		Directory:  directory,
		Task:       task,
		Focus:      focus,
		Model:      model,
		TokensUsed: analysisResult.TokensUsed,
		FileTokens: analysisResult.FileTokens,
		Duration:   analysisResult.Duration,
		Findings:   analysisResult.Findings,
		Response:   analysisResult.Response,
	}

	record.Files = sessionlog.FilesFromTokens(analysisResult.FileTokens, focus)

	if scanResult != nil {
		record.ScanSummary = &sessionlog.ScanSummary{
			TotalFiles:    scanResult.TotalFiles,
			FilteredFiles: scanResult.FilteredFiles,
			TotalSize:     scanResult.TotalSize,
			Duration:      scanResult.Duration,
		}
	}

	return record
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

func startInteractiveMode(directory string, cfg *config.Config, llmClient *llm.OllamaClient, focusRel string) {
	// Perform initial scan silently
	scanResult, err := scanDirectory(directory, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to scan directory: %v\n", err)
		os.Exit(1)
	}

	if focusRel != "" && !scanResultHasFile(scanResult, focusRel) {
		fmt.Fprintf(os.Stderr, "Warning: focused file %s was not included in the scan. Starting without focus.\n", focusRel)
		focusRel = ""
	}

	// Start web server in a goroutine
	webServer := webui.NewServer(directory, cfg.LLM.Model, cfg.LLM.Endpoint, scanResult, cfg, llmClient, focusRel)
	go func() {
		if err := webServer.Start(5050); err != nil {
			fmt.Fprintf(os.Stderr, "Web server error: %v\n", err)
		}
	}()

	// Start interactive TUI
	m := tui.NewInteractiveModel(directory, cfg.LLM.Model, cfg.LLM.Endpoint, scanResult, cfg, llmClient, focusRel)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running interactive mode: %v\n", err)
		os.Exit(1)
	}

	// Print goodbye message after TUI exits
	fmt.Println("👋 Bye!")
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

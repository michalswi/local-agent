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

	tea "github.com/charmbracelet/bubbletea"
)

// Runner manages the TUI and coordinates with backend operations
type Runner struct {
	program *tea.Program
	model   Model
	cfg     *config.Config
	client  *llm.OllamaClient
}

// NewRunner creates a new TUI runner
func NewRunner(directory, task, model, endpoint string, cfg *config.Config, client *llm.OllamaClient) *Runner {
	m := New(directory, task, model, endpoint)
	p := tea.NewProgram(m)

	return &Runner{
		program: p,
		model:   m,
		cfg:     cfg,
		client:  client,
	}
}

// Run starts the TUI and executes the workflow
func (r *Runner) Run() error {
	// Start the TUI in a goroutine
	go func() {
		if _, err := r.program.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
			os.Exit(1)
		}
	}()

	// Give the TUI time to initialize
	time.Sleep(100 * time.Millisecond)

	// Run scanning phase
	scanResult, err := r.runScan()
	if err != nil {
		r.program.Send(SendError(err))
		time.Sleep(2 * time.Second)
		r.program.Quit()
		return err
	}

	// Send scan complete
	r.program.Send(SendScanComplete(scanResult))

	// If no task provided, stop here
	if r.model.Task == "" {
		time.Sleep(2 * time.Second)
		r.program.Quit()
		return nil
	}

	// Wait a moment to show results
	time.Sleep(1 * time.Second)

	// Run analysis phase
	r.program.Send(SendAnalysisProgress("Starting analysis..."))

	analysisResult, err := r.runAnalysis(scanResult)
	if err != nil {
		r.program.Send(SendError(err))
		time.Sleep(2 * time.Second)
		r.program.Quit()
		return err
	}

	// Send analysis complete
	r.program.Send(SendAnalysisComplete(analysisResult))

	// Wait indefinitely for user to quit
	select {}
}

func (r *Runner) runScan() (*types.ScanResult, error) {
	startTime := time.Now()

	// Initialize components
	fileFilter, err := filter.NewFilter(r.cfg, r.model.Directory)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize filter: %w", err)
	}

	analyzerEngine := analyzer.NewAnalyzer(r.cfg)
	validator := security.NewValidator()

	result := &types.ScanResult{
		RootPath: r.model.Directory,
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

		if !fileFilter.ShouldInclude(current, info) {
			result.FilteredFiles++
			r.program.Send(SendScanProgress(result.TotalFiles, -1, fmt.Sprintf("[FILTERED] %s", current)))
			return
		}

		filePaths = append(filePaths, current)
		result.TotalFiles++
		result.TotalSize += info.Size()

		r.program.Send(SendScanProgress(result.TotalFiles, -1, current))
	}

	walk(r.model.Directory, 0)

	// Analyze files
	totalFiles := len(filePaths)
	fileInfos, errors := analyzerEngine.AnalyzeFiles(filePaths, r.model.Directory)

	for i, fileInfo := range fileInfos {
		// Send progress update with file info
		if fileInfo != nil {
			r.program.Send(SendScanProgress(i+1, totalFiles, fmt.Sprintf("[INFO] File: %s, Tokens: %d, IsReadable: %v", filePaths[i], fileInfo.TokenCount, fileInfo.IsReadable)))
		} else {
			r.program.Send(SendScanProgress(i+1, totalFiles, filePaths[i]))
		}

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

func (r *Runner) runAnalysis(scanResult *types.ScanResult) (*types.AnalysisResponse, error) {
	analyzerEngine := analyzer.NewAnalyzer(r.cfg)

	// Convert []FileInfo to []*FileInfo
	fileInfoPtrs := make([]*types.FileInfo, len(scanResult.Files))
	for i := range scanResult.Files {
		fileInfoPtrs[i] = &scanResult.Files[i]
	}

	// Always process files individually (one request per file) with concurrent workers
	return r.analyzeBatches(fileInfoPtrs, analyzerEngine)
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

func (r *Runner) analyzeBatches(files []*types.FileInfo, analyzerEngine *analyzer.Analyzer) (*types.AnalysisResponse, error) {
	r.program.Send(SendAnalysisProgress("📦 Processing files individually (one request per file)"))

	// Prepare batches (one file per batch)
	batches := r.prepareBatches(files)
	totalFiles := len(batches)

	if totalFiles == 0 {
		return &types.AnalysisResponse{
			Response: "No files to analyze",
			Model:    r.cfg.LLM.Model,
		}, nil
	}

	r.program.Send(SendAnalysisProgress(fmt.Sprintf("Processing %d files", totalFiles)))

	// Determine concurrency level
	maxConcurrent := r.cfg.Agent.ConcurrentFiles
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}

	// If only 1 worker or 1 file, process sequentially
	if maxConcurrent == 1 || totalFiles == 1 {
		return r.processSequentially(batches, analyzerEngine)
	}

	r.program.Send(SendAnalysisProgress(fmt.Sprintf("Using %d concurrent workers", maxConcurrent)))

	// Process batches concurrently
	return r.processConcurrently(batches, analyzerEngine, maxConcurrent)
}

func (r *Runner) prepareBatches(files []*types.FileInfo) [][]*types.FileInfo {
	var batches [][]*types.FileInfo
	tokenLimit := r.cfg.Agent.TokenLimit

	for _, file := range files {
		if file == nil || !file.IsReadable {
			continue
		}

		// Skip files that exceed token limit
		if file.TokenCount > tokenLimit {
			r.program.Send(SendAnalysisProgress(fmt.Sprintf("⚠️  Skipping %s (%d tokens exceeds limit of %d)",
				file.RelPath, file.TokenCount, tokenLimit)))
			continue
		}

		// Each file is its own batch
		batches = append(batches, []*types.FileInfo{file})
	}

	return batches
}

func (r *Runner) processSequentially(batches [][]*types.FileInfo, analyzerEngine *analyzer.Analyzer) (*types.AnalysisResponse, error) {
	var allResponses []string
	fileTokens := make(map[string]int)
	var totalDuration time.Duration
	model := ""

	for i, batch := range batches {
		fileNum := i + 1
		fileName := batch[0].RelPath
		r.program.Send(SendAnalysisProgress(fmt.Sprintf("Processing file %d/%d: %s", fileNum, len(batches), fileName)))

		response, err := r.processBatch(batch, analyzerEngine)
		if err != nil {
			r.program.Send(SendAnalysisProgress(fmt.Sprintf("⚠️  File %s failed: %v", fileName, err)))
			allResponses = append(allResponses, formatFileErrorSection(fileName, err))
		} else {
			allResponses = append(allResponses, formatFileSection(fileName, response.Response))
			fileTokens[fileName] = response.TokensUsed
			totalDuration += response.Duration
			if model == "" {
				model = response.Model
			}
			r.program.Send(SendAnalysisProgress(fmt.Sprintf("✅ File %s completed", fileName)))
		}
	}

	return &types.AnalysisResponse{
		Response:   strings.Join(allResponses, "\n"),
		Model:      model,
		FileTokens: fileTokens,
		Duration:   totalDuration,
	}, nil
}

func (r *Runner) processConcurrently(batches [][]*types.FileInfo, analyzerEngine *analyzer.Analyzer, maxConcurrent int) (*types.AnalysisResponse, error) {
	totalFiles := len(batches)

	// Create file name mapping for tracking
	fileNames := make(map[int]string)
	for i, batch := range batches {
		fileNames[i+1] = batch[0].RelPath
	}

	// Create job and result channels
	jobs := make(chan batchJob, totalFiles)
	results := make(chan batchResult, totalFiles)

	// Start worker pool
	var wg sync.WaitGroup
	for w := 1; w <= maxConcurrent; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range jobs {
				fileName := fileNames[job.batchNum]
				r.program.Send(SendAnalysisProgress(fmt.Sprintf("[Worker %d] Processing: %s", workerID, fileName)))

				response, err := r.processBatch(job.batch, analyzerEngine)
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
	fileTokens := make(map[string]int)
	var totalDuration time.Duration
	model := ""
	failedFiles := make(map[int]error)

	for result := range results {
		fileName := fileNames[result.batchNum]
		if result.err != nil {
			r.program.Send(SendAnalysisProgress(fmt.Sprintf("⚠️  File %s failed: %v", fileName, result.err)))
			failedFiles[result.batchNum] = result.err
		} else {
			fileResults[result.batchNum] = result.response
			fileTokens[fileName] = result.response.TokensUsed
			totalDuration += result.response.Duration
			if model == "" {
				model = result.response.Model
			}
			r.program.Send(SendAnalysisProgress(fmt.Sprintf("✅ File %s completed", fileName)))
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
		FileTokens: fileTokens,
		Duration:   totalDuration,
	}, nil
}

func (r *Runner) processBatch(batch []*types.FileInfo, analyzerEngine *analyzer.Analyzer) (*types.AnalysisResponse, error) {
	// Show file info being processed
	for _, file := range batch {
		if file != nil {
			r.program.Send(SendAnalysisProgress(fmt.Sprintf("[INFO] File: %s, Tokens: %d, Content length: %d bytes, IsReadable: %v",
				file.RelPath, file.TokenCount, len(file.Content), file.IsReadable)))
		}
	}

	content := analyzerEngine.PrepareForLLM(batch, r.cfg.Agent.TokenLimit)

	// Check if we have any actual content to analyze
	if len(content) < 100 {
		return nil, fmt.Errorf("no valid content to analyze after PrepareForLLM")
	}

	// For single file batches, add filename to task for clarity
	actualTask := r.model.Task
	if len(batch) == 1 && batch[0] != nil {
		actualTask = fmt.Sprintf("Analyze the file '%s'. %s", batch[0].RelPath, r.model.Task)
	}

	return r.client.Analyze(actualTask, content, r.cfg.LLM.Temperature)
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
	return fmt.Sprintf("=== %s ===", fileName)
}

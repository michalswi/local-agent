package tui

import (
	"fmt"
	"os"
	"path/filepath"
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
		// Send progress update
		r.program.Send(SendScanProgress(i+1, totalFiles, filePaths[i]))

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

	// Prepare content for LLM
	content := analyzerEngine.PrepareForLLM(fileInfoPtrs, r.cfg.Agent.TokenLimit)

	// Send to LLM
	response, err := r.client.Analyze(r.model.Task, content, r.cfg.LLM.Temperature)
	if err != nil {
		return nil, fmt.Errorf("LLM analysis failed: %w", err)
	}

	return response, nil
}

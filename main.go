package main

import (
	"encoding/json"
	"flag"
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
)

const version = "0.1.0"

func main() {
	// Define CLI flags
	var (
		configPath  = flag.String("config", "", "Path to configuration file")
		task        = flag.String("task", "", "Analysis task description")
		directory   = flag.String("dir", ".", "Directory to analyze")
		dryRun      = flag.Bool("dry-run", false, "List files without analyzing")
		outputPath  = flag.String("output", "", "Output file for results")
		showVersion = flag.Bool("version", false, "Show version")
		checkHealth = flag.Bool("health", false, "Check LLM connectivity")
		listModels  = flag.Bool("list-models", false, "List available LLM models")
	)

	flag.Parse()

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
		if *outputPath != "" {
			saveResult(result, *outputPath)
		}
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

	// Save results if output path specified
	if *outputPath != "" {
		saveAnalysisResult(analysisResult, *outputPath)
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

	// Walk directory
	var filePaths []string
	err = filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			result.Errors = append(result.Errors, types.ScanError{
				Path:  path,
				Error: err.Error(),
				Time:  time.Now(),
			})
			return nil
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Validate path
		if err := validator.ValidatePath(path); err != nil {
			return nil
		}

		// Apply filters
		if !fileFilter.ShouldInclude(path, info) {
			result.FilteredFiles++
			return nil
		}

		filePaths = append(filePaths, path)
		result.TotalFiles++
		result.TotalSize += info.Size()

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

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

	content := analyzer.PrepareForLLM(fileInfoPtrs, cfg.Agent.TokenLimit)

	// Send to LLM
	response, err := llmClient.Analyze(task, content, cfg.LLM.Temperature)
	if err != nil {
		return nil, fmt.Errorf("LLM analysis failed: %w", err)
	}

	return response, nil
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

func saveResult(result *types.ScanResult, outputPath string) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal result: %v\n", err)
		return
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write result: %v\n", err)
		return
	}

	fmt.Printf("\n💾 Results saved to: %s\n", outputPath)
}

func saveAnalysisResult(result *types.AnalysisResponse, outputPath string) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal result: %v\n", err)
		return
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write result: %v\n", err)
		return
	}

	fmt.Printf("\n💾 Analysis saved to: %s\n", outputPath)
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

package analyzer

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"local-agent/config"
	"local-agent/types"
)

// Analyzer orchestrates file analysis
type Analyzer struct {
	config   *config.Config
	detector *Detector
	chunker  *Chunker
}

// NewAnalyzer creates a new file analyzer
func NewAnalyzer(cfg *config.Config) *Analyzer {
	return &Analyzer{
		config:   cfg,
		detector: NewDetector(),
		chunker:  NewChunker(&cfg.Chunking),
	}
}

// AnalyzeFile performs complete analysis on a single file
func (a *Analyzer) AnalyzeFile(path string, rootPath string) (*types.FileInfo, error) {
	// Detect file metadata
	info, err := a.detector.DetectFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to detect file: %w", err)
	}

	// Calculate relative path
	relPath, err := filepath.Rel(rootPath, path)
	if err != nil {
		relPath = path
	}
	info.RelPath = relPath

	// Skip if not readable
	if !info.IsReadable {
		return info, nil
	}

	// Skip if too large
	if info.Size > int64(a.config.Agent.MaxFileSizeBytes) {
		return info, nil
	}

	// Read content based on category
	switch info.Category {
	case types.CategorySmall:
		// Read full content
		content, err := a.detector.ReadContent(path, 0)
		if err != nil {
			return info, fmt.Errorf("failed to read content: %w", err)
		}
		info.Content = content
		info.TokenCount = a.chunker.estimateTokens(content)

	case types.CategoryMedium:
		// Read full content but prepare for chunking
		content, err := a.detector.ReadContent(path, 0)
		if err != nil {
			return info, fmt.Errorf("failed to read content: %w", err)
		}
		info.Content = content
		info.TokenCount = a.chunker.estimateTokens(content)

		// Generate summary
		info.Summary = a.generateSummary(info)

	case types.CategoryLarge:
		// Only generate summary and chunks
		info.Summary = a.generateSummary(info)

		chunks, err := a.chunker.ChunkFile(path)
		if err != nil {
			return info, fmt.Errorf("failed to chunk file: %w", err)
		}
		info.Chunks = chunks

		// Calculate total tokens
		totalTokens := 0
		for _, chunk := range chunks {
			totalTokens += chunk.TokenCount
		}
		info.TokenCount = totalTokens
	}

	return info, nil
}

// AnalyzeFiles analyzes multiple files concurrently
func (a *Analyzer) AnalyzeFiles(paths []string, rootPath string) ([]*types.FileInfo, []error) {
	var wg sync.WaitGroup
	results := make([]*types.FileInfo, len(paths))
	errors := make([]error, len(paths))

	// Create semaphore for concurrent limit
	sem := make(chan struct{}, a.config.Agent.ConcurrentFiles)

	for i, path := range paths {
		wg.Add(1)
		go func(idx int, p string) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			info, err := a.AnalyzeFile(p, rootPath)
			results[idx] = info
			errors[idx] = err
		}(i, path)
	}

	wg.Wait()
	return results, errors
}

// generateSummary creates a summary for a file
func (a *Analyzer) generateSummary(info *types.FileInfo) string {
	var parts []string

	// Add file type information
	parts = append(parts, fmt.Sprintf("File: %s", info.RelPath))
	parts = append(parts, fmt.Sprintf("Type: %s", info.Type))
	parts = append(parts, fmt.Sprintf("Size: %s", formatFileSize(info.Size)))

	// Add language/extension
	if info.Extension != "" {
		parts = append(parts, fmt.Sprintf("Extension: %s", info.Extension))
	}

	// Try to detect language
	lang := detectLanguage(info.Extension)
	if lang != "" {
		parts = append(parts, fmt.Sprintf("Language: %s", lang))
	}

	// Add line count if text file
	if info.Type == types.TypeText && info.Content != "" {
		lineCount := strings.Count(info.Content, "\n") + 1
		parts = append(parts, fmt.Sprintf("Lines: %d", lineCount))
	}

	// Add chunk information for large files
	if len(info.Chunks) > 0 {
		parts = append(parts, fmt.Sprintf("Chunks: %d", len(info.Chunks)))
	}

	return strings.Join(parts, " | ")
}

// PrepareForLLM prepares file content for sending to LLM
func (a *Analyzer) PrepareForLLM(files []*types.FileInfo, maxTokens int) string {
	var builder strings.Builder
	currentTokens := 0

	builder.WriteString("Files for analysis:\n\n")

	for _, file := range files {
		if file == nil || !file.IsReadable {
			continue
		}

		// Check token limit
		if currentTokens+file.TokenCount > maxTokens {
			builder.WriteString(fmt.Sprintf("\n[Remaining files omitted due to token limit]\n"))
			break
		}

		// Add file header
		builder.WriteString(fmt.Sprintf("=== File: %s ===\n", file.RelPath))

		if file.IsSensitive {
			builder.WriteString("[SENSITIVE FILE - SKIPPED]\n\n")
			continue
		}

		// Add content based on category
		switch file.Category {
		case types.CategorySmall, types.CategoryMedium:
			if file.Content != "" {
				builder.WriteString(fmt.Sprintf("```%s\n%s\n```\n\n", getLanguageIdentifier(file.Extension), file.Content))
				currentTokens += file.TokenCount
			}

		case types.CategoryLarge:
			builder.WriteString(fmt.Sprintf("[Large file - %s]\n", file.Summary))
			builder.WriteString(fmt.Sprintf("Available chunks: %d\n", len(file.Chunks)))
			builder.WriteString("Use chunk indices to request specific sections.\n\n")
		}
	}

	return builder.String()
}

// Helper functions

func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func detectLanguage(ext string) string {
	languages := map[string]string{
		".go":    "Go",
		".py":    "Python",
		".js":    "JavaScript",
		".ts":    "TypeScript",
		".java":  "Java",
		".c":     "C",
		".cpp":   "C++",
		".h":     "C/C++ Header",
		".rs":    "Rust",
		".rb":    "Ruby",
		".php":   "PHP",
		".swift": "Swift",
		".kt":    "Kotlin",
		".scala": "Scala",
		".sh":    "Shell",
		".sql":   "SQL",
	}

	return languages[ext]
}

func getLanguageIdentifier(ext string) string {
	identifiers := map[string]string{
		".go":   "go",
		".py":   "python",
		".js":   "javascript",
		".ts":   "typescript",
		".java": "java",
		".c":    "c",
		".cpp":  "cpp",
		".rs":   "rust",
		".rb":   "ruby",
		".php":  "php",
		".sh":   "bash",
		".sql":  "sql",
		".md":   "markdown",
		".json": "json",
		".yaml": "yaml",
		".yml":  "yaml",
		".xml":  "xml",
	}

	if id, ok := identifiers[ext]; ok {
		return id
	}
	return ""
}

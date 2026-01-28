package analyzer

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"local-agent/types"
)

// Detector detects file metadata and content
type Detector struct{}

// NewDetector creates a new file detector
func NewDetector() *Detector {
	return &Detector{}
}

// DetectFile analyzes a file and returns its metadata
func (d *Detector) DetectFile(path string) (*types.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	ext := filepath.Ext(path)
	size := info.Size()

	fileInfo := &types.FileInfo{
		Path:      path,
		Size:      size,
		Extension: ext,
		ModTime:   info.ModTime(),
	}

	// Determine category based on size
	if size <= types.SmallFileSizeBytes {
		fileInfo.Category = types.CategorySmall
	} else if size <= types.MediumFileSizeBytes {
		fileInfo.Category = types.CategoryMedium
	} else {
		fileInfo.Category = types.CategoryLarge
	}

	// Detect file type
	fileType, err := d.detectFileType(path, ext)
	if err != nil {
		fileInfo.IsReadable = false
		fileInfo.Type = types.TypeUnknown
		return fileInfo, nil
	}

	fileInfo.Type = fileType
	fileInfo.IsReadable = (fileType == types.TypeText)

	return fileInfo, nil
}

// detectFileType determines the type of a file
func (d *Detector) detectFileType(path, ext string) (types.FileType, error) {
	// Check by extension first
	textExts := []string{
		".txt", ".md", ".go", ".py", ".js", ".ts", ".java", ".c", ".cpp", ".h",
		".rs", ".rb", ".php", ".sh", ".bash", ".zsh", ".yaml", ".yml", ".json",
		".xml", ".html", ".css", ".sql", ".r", ".swift", ".kt", ".scala",
	}

	for _, textExt := range textExts {
		if ext == textExt {
			return types.TypeText, nil
		}
	}

	// Check for binary types
	binaryExts := []string{
		".exe", ".dll", ".so", ".dylib", ".bin", ".o", ".a",
	}

	for _, binExt := range binaryExts {
		if ext == binExt {
			return types.TypeBinary, nil
		}
	}

	// Check for archives
	archiveExts := []string{
		".zip", ".tar", ".gz", ".bz2", ".xz", ".7z", ".rar",
	}

	for _, archExt := range archiveExts {
		if ext == archExt {
			return types.TypeArchive, nil
		}
	}

	// Check for images
	imageExts := []string{
		".jpg", ".jpeg", ".png", ".gif", ".bmp", ".svg", ".ico", ".webp",
	}

	for _, imgExt := range imageExts {
		if ext == imgExt {
			return types.TypeImage, nil
		}
	}

	// Try to detect by content
	file, err := os.Open(path)
	if err != nil {
		return types.TypeUnknown, err
	}
	defer file.Close()

	// Read first 512 bytes
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return types.TypeUnknown, err
	}

	// Check if content is valid UTF-8 text
	if utf8.Valid(buffer[:n]) {
		// Further validate it's text (not binary with valid UTF-8 sequences)
		textCount := 0
		for _, b := range buffer[:n] {
			if b == '\n' || b == '\r' || b == '\t' || (b >= 32 && b < 127) {
				textCount++
			}
		}

		// If more than 90% of characters are text, consider it text
		if n > 0 && float64(textCount)/float64(n) > 0.9 {
			return types.TypeText, nil
		}
	}

	return types.TypeBinary, nil
}

// ReadContent reads the content of a file
func (d *Detector) ReadContent(path string, maxLines int) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var builder strings.Builder
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024) // allow large lines
	lineCount := 0

	for scanner.Scan() {
		if maxLines > 0 && lineCount >= maxLines {
			break
		}

		if lineCount > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(scanner.Text())
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to scan file: %w", err)
	}

	return builder.String(), nil
}

// CountLines counts the number of lines in a file
func (d *Detector) CountLines(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	lineCount := 0

	for scanner.Scan() {
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("failed to scan file: %w", err)
	}

	return lineCount, nil
}

// IsBinary checks if a file appears to be binary
func (d *Detector) IsBinary(path string) (bool, error) {
	fileType, err := d.detectFileType(path, filepath.Ext(path))
	if err != nil {
		return true, err
	}

	return fileType == types.TypeBinary, nil
}

// GetMimeType attempts to determine MIME type of a file
func (d *Detector) GetMimeType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))

	mimeTypes := map[string]string{
		".txt":  "text/plain",
		".md":   "text/markdown",
		".go":   "text/x-go",
		".py":   "text/x-python",
		".js":   "text/javascript",
		".json": "application/json",
		".xml":  "application/xml",
		".yaml": "application/yaml",
		".yml":  "application/yaml",
		".html": "text/html",
		".css":  "text/css",
		".jpg":  "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".pdf":  "application/pdf",
		".zip":  "application/zip",
	}

	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}

	return "application/octet-stream"
}

package llm

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"local-agent/types"
)

//go:embed prompts/system.md
var systemPrompt string

// Client defines the interface for LLM interactions
type Client interface {
	Chat(request *ChatRequest) (*ChatResponse, error)
	IsAvailable() bool
	GetModel() string
}

// ChatRequest represents a request to the LLM
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Stream      bool      `json:"stream"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Think       bool      `json:"think,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role     string `json:"role"` // "user", "assistant", "system"
	Content  string `json:"content"`
	Thinking string `json:"thinking,omitempty"` // populated by Ollama when think:true
}

// ChatResponse represents a response from the LLM
type ChatResponse struct {
	Model     string    `json:"model"`
	Message   Message   `json:"message"`
	CreatedAt time.Time `json:"created_at"`
	Done      bool      `json:"done"`

	// Usage information
	TotalDuration   int64 `json:"total_duration,omitempty"`
	PromptEvalCount int   `json:"prompt_eval_count,omitempty"`
	EvalCount       int   `json:"eval_count,omitempty"`
}

// OllamaClient implements Client for Ollama
type OllamaClient struct {
	endpoint   string
	model      string
	httpClient *http.Client
	timeout    time.Duration
}

// NewOllamaClient creates a new Ollama client
func NewOllamaClient(endpoint, model string, timeout int) *OllamaClient {
	return &OllamaClient{
		endpoint: endpoint,
		model:    model,
		httpClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
		timeout: time.Duration(timeout) * time.Second,
	}
}

// Chat sends a chat request to Ollama
func (c *OllamaClient) Chat(request *ChatRequest) (*ChatResponse, error) {
	return c.ChatWithContext(context.Background(), request)
}

// ChatWithContext sends a chat request to Ollama with cancellation support.
func (c *OllamaClient) ChatWithContext(ctx context.Context, request *ChatRequest) (*ChatResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// Set model if not specified
	if request.Model == "" {
		request.Model = c.model
	}

	// Ensure stream is false (we want complete responses)
	request.Stream = false

	// Marshal request
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/chat", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &chatResp, nil
}

// IsAvailable checks if Ollama is available
func (c *OllamaClient) IsAvailable() bool {
	return c.CheckAvailability() == nil
}

// CheckAvailability verifies Ollama can be reached at the configured endpoint.
func (c *OllamaClient) CheckAvailability() error {
	url := fmt.Sprintf("%s/api/tags", c.endpoint)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create availability request: %w", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach Ollama at %s: %w", c.endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		bodyText := strings.TrimSpace(string(body))
		if bodyText != "" {
			return fmt.Errorf("unexpected response from Ollama at %s: %s (%s)", c.endpoint, resp.Status, bodyText)
		}
		return fmt.Errorf("unexpected response from Ollama at %s: %s", c.endpoint, resp.Status)
	}

	return nil
}

// GetModel returns the model name
func (c *OllamaClient) GetModel() string {
	return c.model
}

// GetEndpoint returns the configured Ollama endpoint.
func (c *OllamaClient) GetEndpoint() string {
	return c.endpoint
}

// IsThinkingModel reports whether the given model name is a reasoning/thinking model
// that generates internal thought blocks before producing a final response.
func IsThinkingModel(model string) bool {
	return isQwen35Model(model) || isGemma4Model(model)
}

func isQwen35Model(model string) bool {
	return strings.Contains(strings.ToLower(model), "qwen3.5")
}

func isGemma4Model(model string) bool {
	return strings.Contains(strings.ToLower(model), "gemma4")
}

// extractThinkingBlock returns only the raw reasoning text (without surrounding tags).
func extractThinkingBlock(content, model string) string {
	if isQwen35Model(model) {
		if start := strings.Index(content, "<think>"); start != -1 {
			if end := strings.Index(content, "</think>"); end != -1 {
				return strings.TrimSpace(content[start+len("<think>") : end])
			}
		}
	}
	if isGemma4Model(model) {
		const openTag = "<|channel>thought\n"
		if start := strings.Index(content, openTag); start != -1 {
			if end := strings.Index(content, "<channel|>"); end != -1 {
				return strings.TrimSpace(content[start+len(openTag) : end])
			}
		}
	}
	return ""
}

// stripThinkingContent removes the internal reasoning block from a model response,
// returning only the final answer.
//
// Qwen3.5: strips <think>\n...\n</think> (and surrounding whitespace)
// Gemma4:  strips <|channel>thought\n...<channel|>
func stripThinkingContent(content, model string) string {
	if isQwen35Model(model) {
		if start := strings.Index(content, "<think>"); start != -1 {
			if end := strings.Index(content, "</think>"); end != -1 {
				return strings.TrimSpace(content[end+len("</think>"):])
			}
		}
	}
	if isGemma4Model(model) {
		if strings.Contains(content, "<|channel>thought\n") {
			if end := strings.Index(content, "<channel|>"); end != -1 {
				return strings.TrimSpace(content[end+len("<channel|>"):])
			}
		}
	}
	return content
}

// AnalyzeThinking is like Analyze but enables thinking/reasoning mode for supported
// models. For Gemma4 it injects <|think|> into the system prompt; for Qwen3.5 the
// model thinks automatically. The reasoning block is stripped from Response and
// returned separately in ThinkingContent.
func (c *OllamaClient) AnalyzeThinking(task string, filesContent string, temperature float64) (*types.AnalysisResponse, error) {
	return c.AnalyzeThinkingWithContext(context.Background(), task, filesContent, temperature)
}

// AnalyzeThinkingWithContext behaves like AnalyzeThinking with request cancellation support.
func (c *OllamaClient) AnalyzeThinkingWithContext(ctx context.Context, task string, filesContent string, temperature float64) (*types.AnalysisResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	startTime := time.Now()

	systemContent := systemPrompt
	if isGemma4Model(c.model) {
		systemContent = "<|think|>\n" + systemContent
	}

	systemMessage := Message{
		Role:    "system",
		Content: systemContent,
	}

	userMessage := Message{
		Role:    "user",
		Content: fmt.Sprintf("**Task:** %s\n\nPlease complete this task based on the following files:\n\n%s", task, filesContent),
	}

	request := &ChatRequest{
		Model:       c.model,
		Messages:    []Message{systemMessage, userMessage},
		Temperature: temperature,
		Think:       true,
	}

	response, err := c.ChatWithContext(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM response: %w", err)
	}

	// Prefer Ollama's dedicated thinking field (returned when think:true);
	// fall back to parsing embedded tags for models that inline them.
	thinkingContent := response.Message.Thinking
	if thinkingContent == "" {
		thinkingContent = extractThinkingBlock(response.Message.Content, c.model)
	}
	finalAnswer := stripThinkingContent(response.Message.Content, c.model)

	return &types.AnalysisResponse{
		Response:        finalAnswer,
		ThinkingContent: thinkingContent,
		Model:           response.Model,
		TokensUsed:      response.PromptEvalCount + response.EvalCount,
		Duration:        time.Since(startTime),
	}, nil
}

// ListModels lists available models in Ollama
func (c *OllamaClient) ListModels() ([]string, error) {
	url := fmt.Sprintf("%s/api/tags", c.endpoint)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	models := make([]string, len(result.Models))
	for i, m := range result.Models {
		models[i] = m.Name
	}

	return models, nil
}

// Analyze sends files for analysis with a specific task
func (c *OllamaClient) Analyze(task string, filesContent string, temperature float64) (*types.AnalysisResponse, error) {
	return c.AnalyzeWithContext(context.Background(), task, filesContent, temperature)
}

// AnalyzeWithContext sends files for analysis with cancellation support.
func (c *OllamaClient) AnalyzeWithContext(ctx context.Context, task string, filesContent string, temperature float64) (*types.AnalysisResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	startTime := time.Now()

	systemMessage := Message{
		Role:    "system",
		Content: systemPrompt,
	}

	userMessage := Message{
		Role:    "user",
		Content: fmt.Sprintf("**Task:** %s\n\nPlease complete this task based on the following files:\n\n%s", task, filesContent),
	}

	request := &ChatRequest{
		Model:       c.model,
		Messages:    []Message{systemMessage, userMessage},
		Temperature: temperature,
	}

	// Send request
	response, err := c.ChatWithContext(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM response: %w", err)
	}

	// Build analysis response
	analysisResp := &types.AnalysisResponse{
		Response:   response.Message.Content,
		Model:      response.Model,
		TokensUsed: response.PromptEvalCount + response.EvalCount,
		Duration:   time.Since(startTime),
	}

	return analysisResp, nil
}

// AnalyzeChunk analyzes a specific file chunk
func (c *OllamaClient) AnalyzeChunk(task string, file *types.FileInfo, chunkIndex int, temperature float64) (*types.AnalysisResponse, error) {
	if chunkIndex < 0 || chunkIndex >= len(file.Chunks) {
		return nil, fmt.Errorf("invalid chunk index: %d", chunkIndex)
	}

	chunk := file.Chunks[chunkIndex]
	content := fmt.Sprintf("File: %s (Lines %d-%d)\n\n```\n%s\n```",
		file.RelPath, chunk.StartLine, chunk.EndLine, chunk.Content)

	return c.Analyze(task, content, temperature)
}

// StreamChat sends a streaming chat request (for future interactive mode)
func (c *OllamaClient) StreamChat(request *ChatRequest, callback func(string) error) error {
	// Set model if not specified
	if request.Model == "" {
		request.Model = c.model
	}

	// Enable streaming
	request.Stream = true

	// Marshal request
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/chat", c.endpoint)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	// Read streaming response
	decoder := json.NewDecoder(resp.Body)
	for {
		var response ChatResponse
		if err := decoder.Decode(&response); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode streaming response: %w", err)
		}

		// Call callback with content
		if err := callback(response.Message.Content); err != nil {
			return err
		}

		if response.Done {
			break
		}
	}

	return nil
}

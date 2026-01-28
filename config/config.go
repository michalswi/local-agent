package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the complete agent configuration
type Config struct {
	Agent    AgentConfig    `yaml:"agent" json:"agent"`
	LLM      LLMConfig      `yaml:"llm" json:"llm"`
	Filters  FilterConfig   `yaml:"filters" json:"filters"`
	Security SecurityConfig `yaml:"security" json:"security"`
	Chunking ChunkingConfig `yaml:"chunking" json:"chunking"`
}

// AgentConfig contains general agent settings
type AgentConfig struct {
	MaxFileSizeBytes int `yaml:"max_file_size_bytes" json:"max_file_size_bytes"`
	TokenLimit       int `yaml:"token_limit" json:"token_limit"`
	ConcurrentFiles  int `yaml:"concurrent_files" json:"concurrent_files"`
}

// LLMConfig contains LLM provider settings
type LLMConfig struct {
	Provider    string  `yaml:"provider" json:"provider"`
	Endpoint    string  `yaml:"endpoint" json:"endpoint"`
	Model       string  `yaml:"model" json:"model"`
	APIKey      string  `yaml:"api_key,omitempty" json:"api_key,omitempty"`
	Temperature float64 `yaml:"temperature" json:"temperature"`
	Timeout     int     `yaml:"timeout" json:"timeout"` // seconds
}

// FilterConfig contains file filtering rules
type FilterConfig struct {
	RespectGitignore bool     `yaml:"respect_gitignore" json:"respect_gitignore"`
	CustomIgnoreFile string   `yaml:"custom_ignore_file" json:"custom_ignore_file"`
	DenyPatterns     []string `yaml:"deny_patterns" json:"deny_patterns"`
	AllowPatterns    []string `yaml:"allow_patterns" json:"allow_patterns"`
}

// SecurityConfig contains security and privacy settings
type SecurityConfig struct {
	DetectSecrets  bool `yaml:"detect_secrets" json:"detect_secrets"`
	SkipBinaries   bool `yaml:"skip_binaries" json:"skip_binaries"`
	FollowSymlinks bool `yaml:"follow_symlinks" json:"follow_symlinks"`
	MaxDepth       int  `yaml:"max_depth" json:"max_depth"`
}

// ChunkingConfig contains file chunking settings
type ChunkingConfig struct {
	Strategy  string `yaml:"strategy" json:"strategy"`     // smart, lines, tokens
	ChunkSize int    `yaml:"chunk_size" json:"chunk_size"` // tokens or lines
	Overlap   int    `yaml:"overlap" json:"overlap"`       // overlap between chunks
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Agent: AgentConfig{
			MaxFileSizeBytes: 1048576, // 1MB
			TokenLimit:       8000,
			ConcurrentFiles:  10,
		},
		LLM: LLMConfig{
			Provider: "ollama",
			Endpoint: "http://localhost:11434",
			Model:    "wizardlm2:7b",
			// Model:       "codellama",
			Temperature: 0.1,
			Timeout:     120,
		},
		Filters: FilterConfig{
			RespectGitignore: true,
			CustomIgnoreFile: ".agentignore",
			DenyPatterns: []string{
				"node_modules/**",
				".git/**",
				"*.log",
				"*.tmp",
				".env*",
				"*.key",
				"*.pem",
				"*.crt",
				"dist/**",
				"build/**",
				"vendor/**",
			},
			AllowPatterns: []string{
				"*.go",
				"*.js",
				"*.ts",
				"*.py",
				"*.java",
				"*.c",
				"*.cpp",
				"*.h",
				"*.md",
				"*.yaml",
				"*.yml",
				"*.json",
				"*.txt",
			},
		},
		Security: SecurityConfig{
			DetectSecrets:  true,
			SkipBinaries:   true,
			FollowSymlinks: false,
			MaxDepth:       20,
		},
		Chunking: ChunkingConfig{
			Strategy:  "smart",
			ChunkSize: 1000,
			Overlap:   100,
		},
	}
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

// LoadConfigWithFallback tries to load config from file, falls back to default
func LoadConfigWithFallback(path string) *Config {
	if path != "" {
		if cfg, err := LoadConfig(path); err == nil {
			return cfg
		}
	}

	// Try to load from standard locations
	standardPaths := []string{
		".agent/config.yaml",
		".agent/config.yml",
		"agent-config.yaml",
		"agent-config.yml",
	}

	for _, p := range standardPaths {
		if cfg, err := LoadConfig(p); err == nil {
			return cfg
		}
	}

	// Return default config
	return DefaultConfig()
}

// Save saves the configuration to a file
func (c *Config) Save(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Agent.MaxFileSizeBytes <= 0 {
		return fmt.Errorf("max_file_size_bytes must be positive")
	}

	if c.Agent.TokenLimit <= 0 {
		return fmt.Errorf("token_limit must be positive")
	}

	if c.Agent.ConcurrentFiles <= 0 {
		return fmt.Errorf("concurrent_files must be positive")
	}

	if c.LLM.Endpoint == "" {
		return fmt.Errorf("llm endpoint is required")
	}

	if c.LLM.Model == "" {
		return fmt.Errorf("llm model is required")
	}

	if c.Security.MaxDepth <= 0 {
		return fmt.Errorf("max_depth must be positive")
	}

	if c.Chunking.ChunkSize <= 0 {
		return fmt.Errorf("chunk_size must be positive")
	}

	return nil
}

// ToJSON converts config to JSON string
func (c *Config) ToJSON() (string, error) {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

package claude

import (
	"fmt"

	"github.com/brettsmith212/orchestrator/internal/adapter"
	"github.com/brettsmith212/orchestrator/internal/adapter/cli"
)

// Default arguments for Claude Code CLI
var defaultArgs = []string{
	"--output-format", "stream-json",
}

// Config holds Claude-specific configuration
type Config struct {
	// BinaryPath is the path to the Claude executable (defaults to "claude")
	BinaryPath string `yaml:"binary_path"`

	// Model specifies which model to use
	Model string `yaml:"model"`

	// MaxTokens specifies the maximum tokens to generate
	MaxTokens int `yaml:"max_tokens"`

	// Args are additional arguments to pass to Claude
	Args []string `yaml:"args"`
}

// New creates a new Claude adapter
func New(id string, config map[string]interface{}) (adapter.Adapter, error) {
	// Parse configuration
	claudeConfig := parseConfig(config)

	// Determine command name
	command := "claude"
	if claudeConfig.BinaryPath != "" {
		command = claudeConfig.BinaryPath
	}

	// Combine default arguments with custom arguments
	args := make([]string, 0, len(defaultArgs)+len(claudeConfig.Args)+4) // Extra space for model and tokens
	args = append(args, defaultArgs...)

	// Add model if specified
	if claudeConfig.Model != "" {
		args = append(args, "--model", claudeConfig.Model)
	}

	// Add max tokens if specified
	if claudeConfig.MaxTokens > 0 {
		args = append(args, "--max-tokens", fmt.Sprintf("%d", claudeConfig.MaxTokens))
	}

	// Add custom arguments
	args = append(args, claudeConfig.Args...)

	// Create and return CLI adapter
	return cli.New(id, command, args), nil
}

// parseConfig converts a generic config map to Claude-specific config
func parseConfig(config map[string]interface{}) *Config {
	cfg := &Config{
		BinaryPath: "",
		Model:      "",
		MaxTokens:  0,
		Args:       []string{},
	}

	// Extract binary path if specified
	if binaryPath, ok := config["binary_path"].(string); ok {
		cfg.BinaryPath = binaryPath
	}

	// Extract model if specified
	if model, ok := config["model"].(string); ok {
		cfg.Model = model
	}

	// Extract max tokens if specified
	if maxTokens, ok := config["max_tokens"].(float64); ok {
		cfg.MaxTokens = int(maxTokens)
	} else if maxTokens, ok := config["max_tokens"].(int); ok {
		cfg.MaxTokens = maxTokens
	}

	// Extract custom arguments if specified
	if args, ok := config["args"].([]interface{}); ok {
		for _, arg := range args {
			if strArg, ok := arg.(string); ok {
				cfg.Args = append(cfg.Args, strArg)
			}
		}
	}

	return cfg
}

// Factory creates a factory function for the Claude adapter
func Factory() adapter.Factory {
	return func(config adapter.Config) (adapter.Adapter, error) {
		if config.Type != "cli" {
			return nil, fmt.Errorf("claude adapter requires cli adapter type, got: %s", config.Type)
		}
		
		return New(config.ID, config.AdapterConfig)
	}
}

// RegisterAdapter registers the Claude adapter in the adapter registry
func RegisterAdapter(registry *adapter.Registry) {
	registry.Register("claude", Factory())
}

// GetTokenUsage parses event payload to track token usage
// This can be enhanced to extract token information from Claude events
func GetTokenUsage(eventPayload []byte) (int, error) {
	// In a real implementation, this would parse the JSON payload
	// to extract token usage information from Claude's output

	// For now, returning a placeholder
	return 0, nil
}
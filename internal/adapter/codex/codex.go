package codex

import (
	"fmt"

	"github.com/brettsmith212/orchestrator/internal/adapter"
	"github.com/brettsmith212/orchestrator/internal/adapter/cli"
)

// Default arguments for Codex CLI
var defaultArgs = []string{
	"run",
	"--output-format", "stream-json",
}

// Config holds Codex-specific configuration
type Config struct {
	// BinaryPath is the path to the Codex executable (defaults to "codex")
	BinaryPath string `yaml:"binary_path"`

	// Model specifies which model to use (if supported by CLI)
	Model string `yaml:"model"`

	// Args are additional arguments to pass to Codex
	Args []string `yaml:"args"`
}

// New creates a new Codex adapter
func New(id string, config map[string]interface{}) (adapter.Adapter, error) {
	// Parse configuration
	codexConfig := parseConfig(config)

	// Determine command name
	command := "codex"
	if codexConfig.BinaryPath != "" {
		command = codexConfig.BinaryPath
	}

	// Combine default arguments with custom arguments
	args := make([]string, 0, len(defaultArgs)+len(codexConfig.Args))
	args = append(args, defaultArgs...)

	// Add model if specified
	if codexConfig.Model != "" {
		args = append(args, "--model", codexConfig.Model)
	}

	// Add custom arguments
	args = append(args, codexConfig.Args...)

	// Create and return CLI adapter
	return cli.New(id, command, args), nil
}

// parseConfig converts a generic config map to Codex-specific config
func parseConfig(config map[string]interface{}) *Config {
	cfg := &Config{
		BinaryPath: "",
		Model:      "",
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

// Factory creates a factory function for the Codex adapter
func Factory() adapter.Factory {
	return func(config adapter.Config) (adapter.Adapter, error) {
		if config.Type != "cli" {
			return nil, fmt.Errorf("codex adapter requires cli adapter type, got: %s", config.Type)
		}
		
		return New(config.ID, config.AdapterConfig)
	}
}

// RegisterAdapter registers the Codex adapter in the adapter registry
func RegisterAdapter(registry *adapter.Registry) {
	registry.Register("codex", Factory())
}
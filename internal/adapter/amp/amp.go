package amp

import (
	"fmt"
	"os/exec"

	"github.com/brettsmith212/orchestrator/internal/adapter"
	"github.com/brettsmith212/orchestrator/internal/adapter/cli"
)

// Default arguments for Amp CLI
var defaultArgs = []string{
	"--json-output", // Enable JSON output format
}

// Config holds Amp-specific configuration
type Config struct {
	// BinaryPath is the path to the Amp executable (defaults to "amp")
	BinaryPath string `yaml:"binary_path"`

	// Args are additional arguments to pass to Amp
	Args []string `yaml:"args"`
}

// New creates a new Amp adapter
func New(id string, config map[string]interface{}) (adapter.Adapter, error) {
	// Parse configuration
	ampConfig := parseConfig(config)

	// Determine command name
	command := "amp"

	// Use specified binary path if provided
	if ampConfig.BinaryPath != "" {
		command = ampConfig.BinaryPath
	} else {
		// Check if amp exists using `which amp`
		whichCmd := exec.Command("which", "amp")
		output, err := whichCmd.Output()
		if err != nil {
			return nil, fmt.Errorf("amp binary not found. Please install it using 'npm install -g @sourcegraph/amp' or specify binary_path in your configuration")
		}
		
		// Trim newline from output and use the full path
		if len(output) > 0 {
			command = string(output[:len(output)-1])
		}
	}

	// Combine default arguments with custom arguments
	args := make([]string, 0, len(defaultArgs)+len(ampConfig.Args))
	args = append(args, defaultArgs...)
	args = append(args, ampConfig.Args...)

	// Create and return CLI adapter
	return cli.New(id, command, args), nil
}

// parseConfig converts a generic config map to Amp-specific config
func parseConfig(config map[string]interface{}) *Config {
	cfg := &Config{
		BinaryPath: "",
		Args:       []string{},
	}

	// Extract binary path if specified
	if binaryPath, ok := config["binary_path"].(string); ok {
		cfg.BinaryPath = binaryPath
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

// Factory creates a factory function for the Amp adapter
func Factory() adapter.Factory {
	return func(config adapter.Config) (adapter.Adapter, error) {
		if config.Type != "cli" {
			return nil, fmt.Errorf("amp adapter requires cli adapter type, got: %s", config.Type)
		}
		
		return New(config.ID, config.AdapterConfig)
	}
}

// RegisterAdapter registers the Amp adapter in the adapter registry
func RegisterAdapter(registry *adapter.Registry) {
	registry.Register("amp", Factory())
}
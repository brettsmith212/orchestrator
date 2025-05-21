package core

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds orchestrator application configuration
type Config struct {
	// WorkingDir is the directory where orchestrator will create git worktrees
	WorkingDir string `yaml:"working_dir"`

	// Agents defines the list of AI coding agents to use
	Agents []AgentConfig `yaml:"agents"`

	// TestCommand is the command to run tests in the repository
	TestCommand string `yaml:"test_command"`

	// TimeoutSeconds is the maximum time to wait for agent responses
	TimeoutSeconds int `yaml:"timeout_seconds"`
}

// AgentConfig defines configuration for a single AI coding agent
type AgentConfig struct {
	// ID is a unique identifier for the agent
	ID string `yaml:"id"`

	// Type is the adapter type ("http" or "cli")
	Type string `yaml:"type"`

	// Config holds adapter-specific configuration
	Config map[string]interface{} `yaml:"config"`
}

// Load reads and parses a YAML configuration file
func Load(path string) (*Config, error) {
	cfg := &Config{}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validateConfig performs validation on the loaded configuration
func validateConfig(cfg *Config) error {
	if cfg.WorkingDir == "" {
		return fmt.Errorf("working_dir is required")
	}

	if len(cfg.Agents) == 0 {
		return fmt.Errorf("at least one agent must be configured")
	}

	for i, agent := range cfg.Agents {
		if agent.ID == "" {
			return fmt.Errorf("agent at index %d is missing ID", i)
		}
		if agent.Type == "" {
			return fmt.Errorf("agent '%s' is missing type", agent.ID)
		}
		if agent.Type != "http" && agent.Type != "cli" {
			return fmt.Errorf("agent '%s' has invalid type '%s', must be 'http' or 'cli'", agent.ID, agent.Type)
		}
	}

	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 300 // Default to 5 minutes if not specified
	}

	return nil
}
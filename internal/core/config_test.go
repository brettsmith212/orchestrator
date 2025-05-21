package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary test config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	configData := `
working_dir: "/tmp/test-dir"
test_command: "go test ./..."
timeout_seconds: 600
agents:
  - id: "test-agent"
    type: "cli"
    config:
      command: "test-command"
      args: ["-a", "-b"]
`

	require.NoError(t, os.WriteFile(configPath, []byte(configData), 0644))

	// Test loading the config
	cfg, err := Load(configPath)
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Verify the loaded config
	assert.Equal(t, "/tmp/test-dir", cfg.WorkingDir)
	assert.Equal(t, "go test ./...", cfg.TestCommand)
	assert.Equal(t, 600, cfg.TimeoutSeconds)

	assert.Len(t, cfg.Agents, 1)
	assert.Equal(t, "test-agent", cfg.Agents[0].ID)
	assert.Equal(t, "cli", cfg.Agents[0].Type)
	assert.NotNil(t, cfg.Agents[0].Config)
}

func TestLoadConfig_InvalidFile(t *testing.T) {
	cfg, err := Load("nonexistent-file.yaml")
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		isValid bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				WorkingDir: "/tmp/test",
				Agents: []AgentConfig{
					{ID: "test", Type: "cli"},
				},
			},
			isValid: true,
		},
		{
			name: "missing working dir",
			cfg: &Config{
				WorkingDir: "",
				Agents: []AgentConfig{
					{ID: "test", Type: "cli"},
				},
			},
			isValid: false,
		},
		{
			name: "no agents",
			cfg: &Config{
				WorkingDir: "/tmp/test",
				Agents:     []AgentConfig{},
			},
			isValid: false,
		},
		{
			name: "agent missing ID",
			cfg: &Config{
				WorkingDir: "/tmp/test",
				Agents: []AgentConfig{
					{ID: "", Type: "cli"},
				},
			},
			isValid: false,
		},
		{
			name: "agent missing type",
			cfg: &Config{
				WorkingDir: "/tmp/test",
				Agents: []AgentConfig{
					{ID: "test", Type: ""},
				},
			},
			isValid: false,
		},
		{
			name: "agent invalid type",
			cfg: &Config{
				WorkingDir: "/tmp/test",
				Agents: []AgentConfig{
					{ID: "test", Type: "invalid"},
				},
			},
			isValid: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateConfig(tc.cfg)
			if tc.isValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
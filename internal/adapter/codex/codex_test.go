package codex

import (
	"testing"

	"github.com/brettsmith212/orchestrator/internal/adapter"
	"github.com/brettsmith212/orchestrator/internal/adapter/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodexAdapter(t *testing.T) {
	// Test creating adapter with default configuration
	codexAdapter, err := New("codex-agent", map[string]interface{}{})
	require.NoError(t, err)

	// Verify it's actually a CLI adapter under the hood
	_, ok := codexAdapter.(*cli.Adapter)
	assert.True(t, ok)

	// Call Shutdown to avoid resource leaks
	defer codexAdapter.Shutdown()
}

func TestCodexFactory(t *testing.T) {
	// Create a factory
	factory := Factory()

	// Create adapter using the factory
	adapterConfig := adapter.Config{
		ID:   "codex-test",
		Type: "cli",
		AdapterConfig: map[string]interface{}{
			"binary_path": "/usr/local/bin/codex",
			"model":      "gpt-4",
			"args": []interface{}{
				"-w", ".",
			},
		},
	}

	codexAdapter, err := factory(adapterConfig)
	require.NoError(t, err)
	defer codexAdapter.Shutdown()

	// Check it's the right type
	_, ok := codexAdapter.(*cli.Adapter)
	assert.True(t, ok)

	// Test error when using wrong adapter type
	wrongConfig := adapter.Config{
		ID:   "codex-test",
		Type: "http", // Not cli
		AdapterConfig: map[string]interface{}{},
	}

	_, err = factory(wrongConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "codex adapter requires cli adapter type")
}

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name           string
		config         map[string]interface{}
		expectedPath   string
		expectedModel  string
		expectedArgs   []string
	}{
		{
			name:           "empty config",
			config:         map[string]interface{}{},
			expectedPath:   "",
			expectedModel:  "",
			expectedArgs:   []string{},
		},
		{
			name: "custom binary path",
			config: map[string]interface{}{
				"binary_path": "/custom/path/to/codex",
			},
			expectedPath:   "/custom/path/to/codex",
			expectedModel:  "",
			expectedArgs:   []string{},
		},
		{
			name: "custom model",
			config: map[string]interface{}{
				"model": "gpt-4",
			},
			expectedPath:   "",
			expectedModel:  "gpt-4",
			expectedArgs:   []string{},
		},
		{
			name: "custom args",
			config: map[string]interface{}{
				"args": []interface{}{
					"-w", ".", "--extra-flag",
				},
			},
			expectedPath:   "",
			expectedModel:  "",
			expectedArgs:   []string{"-w", ".", "--extra-flag"},
		},
		{
			name: "full config",
			config: map[string]interface{}{
				"binary_path": "/path/to/codex",
				"model":      "codex-fast",
				"args": []interface{}{
					"-w", ".", "--verbose",
				},
			},
			expectedPath:   "/path/to/codex",
			expectedModel:  "codex-fast",
			expectedArgs:   []string{"-w", ".", "--verbose"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			config := parseConfig(tc.config)
			assert.Equal(t, tc.expectedPath, config.BinaryPath)
			assert.Equal(t, tc.expectedModel, config.Model)
			assert.Equal(t, tc.expectedArgs, config.Args)
		})
	}
}

func TestRegisterAdapter(t *testing.T) {
	// Create registry
	registry := adapter.NewRegistry()
	
	// Register codex adapter
	RegisterAdapter(registry)
	
	// Check it's registered
	types := registry.RegisteredTypes()
	assert.Contains(t, types, "codex")
	
	// Register the CLI adapter type factory first
	cliFactory := func(config adapter.Config) (adapter.Adapter, error) {
		return cli.New(config.ID, "test-cmd", []string{}), nil
	}
	registry.Register("cli", cliFactory)
	
	// Try to create an adapter from the registry
	config := adapter.Config{
		ID:   "codex-test",
		Type: "cli",
		AdapterConfig: map[string]interface{}{},
	}
	
	codexAdapter, err := registry.Create(config)
	require.NoError(t, err)
	defer codexAdapter.Shutdown()
	
	// Verify it's a CLI adapter
	_, ok := codexAdapter.(*cli.Adapter)
	assert.True(t, ok)
}
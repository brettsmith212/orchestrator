package claude

import (
	"testing"

	"github.com/brettsmith212/orchestrator/internal/adapter"
	"github.com/brettsmith212/orchestrator/internal/adapter/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClaudeAdapter(t *testing.T) {
	// Test creating adapter with default configuration
	claudeAdapter, err := New("claude-agent", map[string]interface{}{})
	require.NoError(t, err)

	// Verify it's actually a CLI adapter under the hood
	_, ok := claudeAdapter.(*cli.Adapter)
	assert.True(t, ok)

	// Call Shutdown to avoid resource leaks
	defer claudeAdapter.Shutdown()
}

func TestClaudeFactory(t *testing.T) {
	// Create a factory
	factory := Factory()

	// Create adapter using the factory
	adapterConfig := adapter.Config{
		ID:   "claude-test",
		Type: "cli",
		AdapterConfig: map[string]interface{}{
			"binary_path": "/usr/local/bin/claude",
			"model":      "claude-3",
			"max_tokens": float64(2000),
			"args": []interface{}{
				"-w", ".",
			},
		},
	}

	claudeAdapter, err := factory(adapterConfig)
	require.NoError(t, err)
	defer claudeAdapter.Shutdown()

	// Check it's the right type
	_, ok := claudeAdapter.(*cli.Adapter)
	assert.True(t, ok)

	// Test error when using wrong adapter type
	wrongConfig := adapter.Config{
		ID:   "claude-test",
		Type: "http", // Not cli
		AdapterConfig: map[string]interface{}{},
	}

	_, err = factory(wrongConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "claude adapter requires cli adapter type")
}

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name           string
		config         map[string]interface{}
		expectedPath   string
		expectedModel  string
		expectedTokens int
		expectedArgs   []string
	}{
		{
			name:           "empty config",
			config:         map[string]interface{}{},
			expectedPath:   "",
			expectedModel:  "",
			expectedTokens: 0,
			expectedArgs:   []string{},
		},
		{
			name: "custom binary path",
			config: map[string]interface{}{
				"binary_path": "/custom/path/to/claude",
			},
			expectedPath:   "/custom/path/to/claude",
			expectedModel:  "",
			expectedTokens: 0,
			expectedArgs:   []string{},
		},
		{
			name: "custom model",
			config: map[string]interface{}{
				"model": "claude-3-opus",
			},
			expectedPath:   "",
			expectedModel:  "claude-3-opus",
			expectedTokens: 0,
			expectedArgs:   []string{},
		},
		{
			name: "max tokens as float",
			config: map[string]interface{}{
				"max_tokens": float64(1000),
			},
			expectedPath:   "",
			expectedModel:  "",
			expectedTokens: 1000,
			expectedArgs:   []string{},
		},
		{
			name: "max tokens as int",
			config: map[string]interface{}{
				"max_tokens": 2000,
			},
			expectedPath:   "",
			expectedModel:  "",
			expectedTokens: 2000,
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
			expectedTokens: 0,
			expectedArgs:   []string{"-w", ".", "--extra-flag"},
		},
		{
			name: "full config",
			config: map[string]interface{}{
				"binary_path": "/path/to/claude",
				"model":      "claude-3-sonnet",
				"max_tokens": float64(4000),
				"args": []interface{}{
					"-w", ".", "--verbose",
				},
			},
			expectedPath:   "/path/to/claude",
			expectedModel:  "claude-3-sonnet",
			expectedTokens: 4000,
			expectedArgs:   []string{"-w", ".", "--verbose"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			config := parseConfig(tc.config)
			assert.Equal(t, tc.expectedPath, config.BinaryPath)
			assert.Equal(t, tc.expectedModel, config.Model)
			assert.Equal(t, tc.expectedTokens, config.MaxTokens)
			assert.Equal(t, tc.expectedArgs, config.Args)
		})
	}
}

func TestRegisterAdapter(t *testing.T) {
	// Create registry
	registry := adapter.NewRegistry()
	
	// Register claude adapter
	RegisterAdapter(registry)
	
	// Check it's registered
	types := registry.RegisteredTypes()
	assert.Contains(t, types, "claude")
	
	// Register the CLI adapter type factory first
	cliFactory := func(config adapter.Config) (adapter.Adapter, error) {
		return cli.New(config.ID, "test-cmd", []string{}), nil
	}
	registry.Register("cli", cliFactory)
	
	// Try to create an adapter from the registry
	config := adapter.Config{
		ID:   "claude-test",
		Type: "cli",
		AdapterConfig: map[string]interface{}{},
	}
	
	claudeAdapter, err := registry.Create(config)
	require.NoError(t, err)
	defer claudeAdapter.Shutdown()
	
	// Verify it's a CLI adapter
	_, ok := claudeAdapter.(*cli.Adapter)
	assert.True(t, ok)
}

func TestGetTokenUsage(t *testing.T) {
	// This is a placeholder test for the token tracking functionality
	// In a real implementation, this would test parsing token usage from events
	tokens, err := GetTokenUsage([]byte(`{"type":"thinking","payload":{"content":"test"}`))
	require.NoError(t, err)
	assert.Equal(t, 0, tokens) // Currently returns placeholder 0
}
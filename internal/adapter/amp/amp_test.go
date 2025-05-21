package amp

import (
	"testing"

	"github.com/brettsmith212/orchestrator/internal/adapter"
	"github.com/brettsmith212/orchestrator/internal/adapter/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAmpAdapter(t *testing.T) {
	// Skip when running in CI or with -short flag
	if testing.Short() {
		t.Skip("Skipping test that requires amp binary")
	}

	// Test creating adapter with default configuration
	ampAdapter, err := New("amp-agent", map[string]interface{}{})
	
	// If amp is not installed, this test will fail but that's expected
	if err != nil && err.Error() == "amp binary not found. Please install it using 'npm install -g @sourcegraph/amp' or specify binary_path in your configuration" {
		t.Skip("Skipping test because amp binary is not installed")
	}

	require.NoError(t, err)

	// Verify it's actually a CLI adapter under the hood
	_, ok := ampAdapter.(*cli.Adapter)
	assert.True(t, ok)

	// Call Shutdown to avoid resource leaks
	defer ampAdapter.Shutdown()
}

func TestAmpFactory(t *testing.T) {
	// Create a factory
	factory := Factory()

	// Create adapter using the factory
	adapterConfig := adapter.Config{
		ID:   "amp-test",
		Type: "cli",
		AdapterConfig: map[string]interface{}{
			"binary_path": "/usr/local/bin/amp",
			"args": []interface{}{
				"-w", ".",
			},
		},
	}

	ampAdapter, err := factory(adapterConfig)
	require.NoError(t, err)
	defer ampAdapter.Shutdown()

	// Check it's the right type
	_, ok := ampAdapter.(*cli.Adapter)
	assert.True(t, ok)

	// Test error when using wrong adapter type
	wrongConfig := adapter.Config{
		ID:   "amp-test",
		Type: "http", // Not cli
		AdapterConfig: map[string]interface{}{},
	}

	_, err = factory(wrongConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amp adapter requires cli adapter type")
}

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name           string
		config         map[string]interface{}
		expectedPath   string
		expectedArgs   []string
	}{
		{
			name:           "empty config",
			config:         map[string]interface{}{},
			expectedPath:   "",
			expectedArgs:   []string{},
		},
		{
			name: "custom binary path",
			config: map[string]interface{}{
				"binary_path": "/custom/path/to/amp",
			},
			expectedPath:   "/custom/path/to/amp",
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
			expectedArgs:   []string{"-w", ".", "--extra-flag"},
		},
		{
			name: "full config",
			config: map[string]interface{}{
				"binary_path": "/path/to/amp",
				"args": []interface{}{
					"-w", ".", "--verbose",
				},
			},
			expectedPath:   "/path/to/amp",
			expectedArgs:   []string{"-w", ".", "--verbose"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			config := parseConfig(tc.config)
			assert.Equal(t, tc.expectedPath, config.BinaryPath)
			assert.Equal(t, tc.expectedArgs, config.Args)
		})
	}
}

func TestRegisterAdapter(t *testing.T) {
	// Create registry
	registry := adapter.NewRegistry()
	
	// Register amp adapter
	RegisterAdapter(registry)
	
	// Check it's registered
	types := registry.RegisteredTypes()
	assert.Contains(t, types, "amp")
	
	// Register the CLI adapter type factory first
	cliFactory := func(config adapter.Config) (adapter.Adapter, error) {
		return cli.New(config.ID, "test-cmd", []string{}), nil
	}
	registry.Register("cli", cliFactory)
	
	// Try to create an adapter from the registry
	config := adapter.Config{
		ID:   "amp-test",
		Type: "cli",
		AdapterConfig: map[string]interface{}{},
	}
	
	ampAdapter, err := registry.Create(config)
	require.NoError(t, err)
	defer ampAdapter.Shutdown()
	
	// Verify it's a CLI adapter
	_, ok := ampAdapter.(*cli.Adapter)
	assert.True(t, ok)
}
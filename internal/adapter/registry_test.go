package adapter

import (
	"context"
	"testing"

	"github.com/brettsmith212/orchestrator/internal/core"
	"github.com/brettsmith212/orchestrator/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAdapter implements Adapter interface for testing
type mockAdapter struct {
	id      string
	adapterType string
	config  map[string]interface{}
	closed  bool
}

// Start implements the Adapter interface
func (m *mockAdapter) Start(ctx context.Context, worktreePath string, prompt string) (<-chan *protocol.Event, error) {
	// Create a channel that returns a single event
	ch := make(chan *protocol.Event, 1)
	event := protocol.NewEvent(protocol.EventTypeComplete, m.id, 1)
	ch <- event
	close(ch)
	return ch, nil
}

// Shutdown implements the Adapter interface
func (m *mockAdapter) Shutdown() error {
	m.closed = true
	return nil
}

// mockFactory creates a factory function for mock adapters
func mockFactory(adapterType string) Factory {
	return func(config Config) (Adapter, error) {
		return &mockAdapter{
			id:      config.ID,
			adapterType: adapterType,
			config:  config.AdapterConfig,
			closed:  false,
		}, nil
	}
}

func TestRegistryBasic(t *testing.T) {
	// Create a new registry
	registry := NewRegistry()
	
	// Register a mock factory
	registry.Register("mock", mockFactory("mock"))
	
	// Verify registered types
	types := registry.RegisteredTypes()
	assert.Len(t, types, 1)
	assert.Contains(t, types, "mock")
	
	// Create an adapter
	adapterConfig := Config{
		ID:            "test-agent",
		Type:          "mock",
		AdapterConfig: map[string]interface{}{
			"key": "value",
		},
	}
	
	adapter, err := registry.Create(adapterConfig)
	require.NoError(t, err)
	
	// Check adapter properties
	mockAdapter, ok := adapter.(*mockAdapter)
	require.True(t, ok)
	assert.Equal(t, "test-agent", mockAdapter.id)
	assert.Equal(t, "mock", mockAdapter.adapterType)
	assert.Equal(t, "value", mockAdapter.config["key"])
}

func TestCreateFromConfig(t *testing.T) {
	// Create a registry with multiple adapters
	registry := NewRegistry()
	registry.Register("http", mockFactory("http"))
	registry.Register("cli", mockFactory("cli"))
	
	// Create a core config
	coreConfig := &core.Config{
		WorkingDir: "/tmp/test",
		Agents: []core.AgentConfig{
			{
				ID:     "agent1",
				Type:   "http",
				Config: map[string]interface{}{
					"api_key": "secret",
				},
			},
			{
				ID:     "agent2",
				Type:   "cli",
				Config: map[string]interface{}{
					"command": "ai-cli",
				},
			},
		},
	}
	
	// Create adapters from config
	adapters, err := registry.CreateFromConfig(coreConfig)
	require.NoError(t, err)
	
	// Verify the adapters
	assert.Len(t, adapters, 2)
	
	// Check first adapter
	agent1, exists := adapters["agent1"]
	require.True(t, exists)
	mockAgent1, ok := agent1.(*mockAdapter)
	require.True(t, ok)
	assert.Equal(t, "agent1", mockAgent1.id)
	assert.Equal(t, "http", mockAgent1.adapterType)
	assert.Equal(t, "secret", mockAgent1.config["api_key"])
	
	// Check second adapter
	agent2, exists := adapters["agent2"]
	require.True(t, exists)
	mockAgent2, ok := agent2.(*mockAdapter)
	require.True(t, ok)
	assert.Equal(t, "agent2", mockAgent2.id)
	assert.Equal(t, "cli", mockAgent2.adapterType)
	assert.Equal(t, "ai-cli", mockAgent2.config["command"])
}

func TestRegistryErrors(t *testing.T) {
	registry := NewRegistry()
	registry.Register("mock", mockFactory("mock"))
	
	// Test creating with unknown adapter type
	_, err := registry.Create(Config{
		ID:   "test",
		Type: "unknown",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no adapter factory registered for type: unknown")
	
	// Test creating from config with unknown adapter type
	coreConfig := &core.Config{
		WorkingDir: "/tmp/test",
		Agents: []core.AgentConfig{
			{
				ID:   "agent1",
				Type: "unknown",
			},
		},
	}
	
	_, err = registry.CreateFromConfig(coreConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create adapter for agent agent1")
}
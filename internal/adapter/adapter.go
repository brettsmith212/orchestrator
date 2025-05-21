package adapter

import (
	"context"

	"github.com/brettsmith212/orchestrator/internal/protocol"
)

// Adapter defines the interface for AI coding agent adapters
// Each implementation provides a way to communicate with a different AI system
type Adapter interface {
	// Start initiates the agent with a given worktree path and prompt
	// It returns a channel that will receive events from the agent
	// The adapter should close this channel when the agent is done or encounters an error
	Start(ctx context.Context, worktreePath string, prompt string) (<-chan *protocol.Event, error)

	// Shutdown gracefully terminates the agent
	// It should be called to clean up resources even if the agent has completed its work
	Shutdown() error
}

// Config represents the common configuration structure for adapters
type Config struct {
	// ID is a unique identifier for the adapter instance
	ID string

	// Type specifies the adapter type ("http", "cli", etc.)
	Type string

	// AdapterConfig contains adapter-specific configuration
	AdapterConfig map[string]interface{}
}

// Factory is a function that creates an adapter instance from a configuration
type Factory func(config Config) (Adapter, error)
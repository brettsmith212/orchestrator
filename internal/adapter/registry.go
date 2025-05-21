package adapter

import (
	"fmt"
	"sync"

	"github.com/brettsmith212/orchestrator/internal/core"
)

// Registry stores adapter factory functions by type
type Registry struct {
	mutex     sync.RWMutex
	factories map[string]Factory
}

// NewRegistry creates a new adapter registry
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]Factory),
	}
}

// Register adds a factory function for an adapter type
func (r *Registry) Register(adapterType string, factory Factory) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	r.factories[adapterType] = factory
}

// Create instantiates an adapter based on the provided configuration
func (r *Registry) Create(config Config) (Adapter, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	factory, exists := r.factories[config.Type]
	if !exists {
		return nil, fmt.Errorf("no adapter factory registered for type: %s", config.Type)
	}
	
	return factory(config)
}

// CreateFromConfig creates adapters from a global configuration
func (r *Registry) CreateFromConfig(cfg *core.Config) (map[string]Adapter, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	adapters := make(map[string]Adapter)
	
	for _, agentCfg := range cfg.Agents {
		// Create adapter configuration
		adapterConfig := Config{
			ID:            agentCfg.ID,
			Type:          agentCfg.Type,
			AdapterConfig: agentCfg.Config,
		}
		
		// Create the adapter
		adapter, err := r.Create(adapterConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create adapter for agent %s: %w", agentCfg.ID, err)
		}
		
		adapters[agentCfg.ID] = adapter
	}
	
	return adapters, nil
}

// RegisteredTypes returns the list of registered adapter types
func (r *Registry) RegisteredTypes() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	types := make([]string, 0, len(r.factories))
	for adapterType := range r.factories {
		types = append(types, adapterType)
	}
	
	return types
}
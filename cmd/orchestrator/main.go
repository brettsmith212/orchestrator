package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/brettsmith212/orchestrator/internal/adapter"
	"github.com/brettsmith212/orchestrator/internal/adapter/amp"
	"github.com/brettsmith212/orchestrator/internal/adapter/claude"
	"github.com/brettsmith212/orchestrator/internal/adapter/cli"
	"github.com/brettsmith212/orchestrator/internal/adapter/codex"
	"github.com/brettsmith212/orchestrator/internal/core"
	"github.com/brettsmith212/orchestrator/internal/gitutil"
	"github.com/brettsmith212/orchestrator/internal/protocol"
)

const defaultConfigPath = "config.yaml"

// Command line flags
var (
	configPath string
	prompt     string
	repoPath   string
	verbose    bool
)

func init() {
	// Define command line flags
	flag.StringVar(&configPath, "config", defaultConfigPath, "Path to configuration file")
	flag.StringVar(&prompt, "prompt", "", "Task prompt for the agents")
	flag.StringVar(&repoPath, "repo", ".", "Path to the git repository")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose output")
}

func main() {
	// Parse command line flags
	flag.Parse()

	// Validate required flags
	if prompt == "" {
		fmt.Println("Error: task prompt is required")
		flag.Usage()
		os.Exit(1)
	}

	// Load configuration
	cfg, err := core.Load(configPath)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nReceived interrupt signal, shutting down...")
		cancel()
	}()

	// Run the orchestrator
	if err := run(ctx, cfg); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg *core.Config) error {
	// Resolve absolute path to repository
	abs, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("failed to resolve repository path: %w", err)
	}

	// Setup git worktree manager
	worktreeManager, err := gitutil.NewWorktreeManager(abs, cfg.WorkingDir)
	if err != nil {
		return fmt.Errorf("failed to create worktree manager: %w", err)
	}
	defer worktreeManager.Cleanup()

	// Setup test runner
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	testRunner := core.NewTestRunner(cfg.TestCommand, timeout)

	// Setup arbitrator
	arbitrator := core.NewArbitrator(testRunner, abs)

	// Run baseline tests
	fmt.Println("Running baseline tests...")
	if err := arbitrator.SetBaselineTestResults(ctx); err != nil {
		return fmt.Errorf("failed to run baseline tests: %w", err)
	}

	// Setup adapter registry
	registry := adapter.NewRegistry()
	registerAdapters(registry)

	// Create adapters based on configuration
	adapters, err := registry.CreateFromConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to create adapters: %w", err)
	}

	// Start agents
	fmt.Printf("Starting %d agents with prompt: %s\n", len(adapters), prompt)
	patchDetails, err := runAgents(ctx, adapters, worktreeManager, prompt)
	if err != nil {
		return fmt.Errorf("error running agents: %w", err)
	}

	// Select best patch
	fmt.Println("Evaluating patches...")
	bestPatch, err := arbitrator.SelectBestPatch(ctx, patchDetails)
	if err != nil {
		return fmt.Errorf("failed to select best patch: %w", err)
	}

	// Display results
	fmt.Println("\n=== Best Patch Selected ===")
	fmt.Println(core.FormatPatchResult(bestPatch))

	// TODO: Apply the patch to the main repository if requested

	return nil
}

// Register all available adapters
func registerAdapters(registry *adapter.Registry) {
	// Register CLI adapters
	registry.Register("cli", adapter.Factory(func(config adapter.Config) (adapter.Adapter, error) {
		switch {
		case config.ID == "amp" || config.AdapterConfig["command"] == "amp":
			// Check for common locations for the binary
			config.AdapterConfig["binary_path"] = findBinary("amp", []string{
				"/opt/homebrew/bin/amp",
				"/usr/local/bin/amp",
			})
			return amp.New(config.ID, config.AdapterConfig)
			
		case config.ID == "codex" || config.AdapterConfig["command"] == "codex":
			// Check for common locations for the binary
			config.AdapterConfig["binary_path"] = findBinary("codex", []string{
				"/opt/homebrew/bin/codex",
				"/usr/local/bin/codex",
			})
			return codex.New(config.ID, config.AdapterConfig)
			
		case config.ID == "claude" || config.AdapterConfig["command"] == "claude":
			// Check for common locations for the binary
			config.AdapterConfig["binary_path"] = findBinary("claude", []string{
				"/opt/homebrew/bin/claude",
				"/usr/local/bin/claude",
			})
			return claude.New(config.ID, config.AdapterConfig)
			
		default:
			// Generic CLI adapter for other command-line tools
			command, _ := config.AdapterConfig["command"].(string)
			if command == "" {
				return nil, fmt.Errorf("missing command for generic CLI adapter")
			}
			
			// Extract arguments
			var cliArgs []string
			if args, ok := config.AdapterConfig["args"].([]interface{}); ok {
				for _, arg := range args {
					if strArg, ok := arg.(string); ok {
						cliArgs = append(cliArgs, strArg)
					}
				}
			}
			
			return cli.New(config.ID, command, cliArgs), nil
		}
	}))

	// Register specific CLI adapter types
	amp.RegisterAdapter(registry)
	codex.RegisterAdapter(registry)
	claude.RegisterAdapter(registry)
	
	// TODO: Register HTTP adapters when implemented
}

// findBinary looks for a binary in PATH and common locations
func findBinary(name string, additionalPaths []string) string {
	// First check if it's in PATH
	path, err := exec.LookPath(name)
	if err == nil {
		return path
	}
	
	// Then check additional locations
	for _, path := range additionalPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	
	// If not found, just return the name and let the system resolve it
	return name
}

// runAgents starts all agents and collects their patches
func runAgents(ctx context.Context, adapters map[string]adapter.Adapter, worktreeManager *gitutil.WorktreeManager, prompt string) (map[string]*core.PatchDetails, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	patchDetails := make(map[string]*core.PatchDetails)

	for agentID, agentAdapter := range adapters {
		wg.Add(1)
		go func(id string, adpt adapter.Adapter) {
			defer wg.Done()

			// Create a worktree for this agent
			worktreePath, err := worktreeManager.CreateWorktree(id, "")
			if err != nil {
				log.Printf("Failed to create worktree for agent %s: %v", id, err)
				return
			}

			// Start the agent
			if verbose {
				fmt.Printf("Starting agent %s in worktree %s\n", id, worktreePath)
			}

			eventCh, err := adpt.Start(ctx, worktreePath, prompt)
			if err != nil {
				log.Printf("Failed to start agent %s: %v", id, err)
				return
			}

			// Collect events
			events := collectEvents(ctx, id, eventCh)

			// Cleanup
			if err := adpt.Shutdown(); err != nil {
				log.Printf("Error shutting down agent %s: %v", id, err)
			}

			// Get the diff
			diff, err := worktreeManager.GetDiff(worktreePath)
			if err != nil {
				log.Printf("Failed to get diff for agent %s: %v", id, err)
				return
			}

			// Store patch details
			mu.Lock()
			patchDetails[id] = &core.PatchDetails{
				WorktreePath: worktreePath,
				Diff:        diff,
				Events:      events,
			}
			mu.Unlock()

			if verbose {
				fmt.Printf("Agent %s completed with %d events and %d bytes of diff\n", 
					id, len(events), len(diff))
			}
		}(agentID, agentAdapter)
	}

	// Wait for all agents to complete
	wg.Wait()
	return patchDetails, nil
}

// collectEvents reads all events from the channel
func collectEvents(ctx context.Context, agentID string, eventCh <-chan *protocol.Event) []*protocol.Event {
	var events []*protocol.Event

	for {
		select {
		case event, ok := <-eventCh:
			if !ok {
				// Channel closed, all events received
				return events
			}
			
			// Only process valid events
			if event != nil {
				events = append(events, event)
				if verbose {
					fmt.Printf("Agent %s: Received %s event\n", agentID, event.Type)
				}
			}

		case <-ctx.Done():
			// Context cancelled, return what we have
			return events
		}
	}
}

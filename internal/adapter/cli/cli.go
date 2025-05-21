package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"

	"github.com/brettsmith212/orchestrator/internal/protocol"
)

// Adapter implements the adapter.Adapter interface for CLI-based AI coding agents
type Adapter struct {
	// ID is the unique identifier for this agent instance
	id string

	// Command is the CLI command to execute
	command string

	// Args are command-line arguments to pass to the command
	args []string

	// mutex protects concurrent access to cmd
	mutex sync.Mutex

	// cmd is the running command process
	cmd *exec.Cmd
}

// New creates a new CLI adapter
func New(id, command string, args []string) *Adapter {
	return &Adapter{
		id:      id,
		command: command,
		args:    args,
	}
}

// Start implements the adapter.Adapter interface
func (a *Adapter) Start(ctx context.Context, worktreePath string, prompt string) (<-chan *protocol.Event, error) {
	// Create output channel for events
	eventCh := make(chan *protocol.Event, 10)

	// Prepare command with worktree path and prompt
	a.mutex.Lock()
	workingArgs := append([]string{}, a.args...)
	
	// Add working directory option if not already specified
	hasWorkingDir := false
	for _, arg := range workingArgs {
		if arg == "-w" || arg == "--worktree" || arg == "--workdir" {
			hasWorkingDir = true
			break
		}
	}
	
	if !hasWorkingDir {
		workingArgs = append(workingArgs, "-w", worktreePath)
	}
	
	// Add prompt as final argument
	workingArgs = append(workingArgs, prompt)
	
	// Create command
	a.cmd = exec.CommandContext(ctx, a.command, workingArgs...)
	
	// Get stdout pipe for reading events
	stdout, err := a.cmd.StdoutPipe()
	if err != nil {
		a.mutex.Unlock()
		close(eventCh)
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	
	// Start the command
	err = a.cmd.Start()
	if err != nil {
		a.mutex.Unlock()
		close(eventCh)
		return nil, fmt.Errorf("failed to start command: %w", err)
	}
	a.mutex.Unlock()

	// Process stdout in a goroutine
	go func() {
		defer close(eventCh)
		
		// Create a scanner for reading lines
		scanner := bufio.NewScanner(stdout)
		seq := 1
		
		// Read one line at a time
		for scanner.Scan() {
			line := scanner.Bytes()
			
			// Parse the line as an event
			event, err := protocol.Unmarshal(line)
			if err != nil {
				// Create an error event if we can't parse the output
				errorEvent := protocol.NewEvent(protocol.EventTypeError, a.id, seq)
				errorPayload := protocol.ErrorPayload{
					Message: fmt.Sprintf("Failed to parse output: %v", err),
					Code:    "parse_error",
				}
				errorEvent, _ = errorEvent.WithPayload(errorPayload)
				eventCh <- errorEvent
				seq++
				continue
			}
			
			// Set agent ID if not present
			if event.AgentID == "" {
				event.AgentID = a.id
			}
			
			// Set sequence number if not present
			if event.SequenceNum == 0 {
				event.SequenceNum = seq
				seq++
			}
			
			// Send the event
			eventCh <- event
		}
		
		// Check for scanner errors
		if err := scanner.Err(); err != nil && err != io.EOF {
			errorEvent := protocol.NewEvent(protocol.EventTypeError, a.id, seq)
			errorPayload := protocol.ErrorPayload{
				Message: fmt.Sprintf("Error reading stdout: %v", err),
				Code:    "io_error",
			}
			errorEvent, _ = errorEvent.WithPayload(errorPayload)
			eventCh <- errorEvent
		}
		
		// Wait for the command to finish
		waitErr := a.cmd.Wait()
		
		// Send error event if command failed (not if it was just canceled)
		if waitErr != nil && ctx.Err() == nil {
			errorEvent := protocol.NewEvent(protocol.EventTypeError, a.id, seq)
			errorPayload := protocol.ErrorPayload{
				Message: fmt.Sprintf("Command failed: %v", waitErr),
				Code:    "command_error",
			}
			errorEvent, _ = errorEvent.WithPayload(errorPayload)
			eventCh <- errorEvent
		}
	}()

	return eventCh, nil
}

// Shutdown implements the adapter.Adapter interface
func (a *Adapter) Shutdown() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	
	if a.cmd != nil && a.cmd.Process != nil {
		// Try to kill the process gracefully
		return a.cmd.Process.Kill()
	}
	
	return nil
}
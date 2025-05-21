package adapter

import (
	"context"
	"testing"
	"time"

	"github.com/brettsmith212/orchestrator/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeAdapter is a simple implementation of the Adapter interface for testing
type fakeAdapter struct {
	id       string
	shutdown bool
}

// Start implements the Adapter interface
func (f *fakeAdapter) Start(ctx context.Context, worktreePath string, prompt string) (<-chan *protocol.Event, error) {
	// Create a channel for events
	eventCh := make(chan *protocol.Event, 10)

	// Send events in a goroutine
	go func() {
		defer close(eventCh) // Always close the channel when done

		// Send a thinking event
		thinkingEvent := protocol.NewEvent(protocol.EventTypeThinking, f.id, 1)
		thinkingPayload := protocol.ThinkingPayload{Content: "Analyzing the prompt..."}
		thinkingEvent, _ = thinkingEvent.WithPayload(thinkingPayload)
		eventCh <- thinkingEvent

		// Simulate some work
		select {
		case <-time.After(100 * time.Millisecond):
			// Send an action event
			actionEvent := protocol.NewEvent(protocol.EventTypeAction, f.id, 2)
			actionPayload := protocol.ActionPayload{
				ActionType: "file_edit",
				FilePath:   "main.go",
				Content:    "package main\n\nfunc main() {}\n",
			}
			actionEvent, _ = actionEvent.WithPayload(actionPayload)
			eventCh <- actionEvent

			// Send a complete event
			completeEvent := protocol.NewEvent(protocol.EventTypeComplete, f.id, 3)
			eventCh <- completeEvent
		case <-ctx.Done():
			// Context was canceled, send an error event
			errorEvent := protocol.NewEvent(protocol.EventTypeError, f.id, 2)
			errorPayload := protocol.ErrorPayload{Message: "Operation canceled"}
			errorEvent, _ = errorEvent.WithPayload(errorPayload)
			eventCh <- errorEvent
		}
	}()

	return eventCh, nil
}

// Shutdown implements the Adapter interface
func (f *fakeAdapter) Shutdown() error {
	f.shutdown = true
	return nil
}

// newFakeAdapter creates a new fake adapter for testing
func newFakeAdapter(id string) *fakeAdapter {
	return &fakeAdapter{id: id}
}

// TestAdapterInterface tests that the fakeAdapter correctly implements the Adapter interface
func TestAdapterInterface(t *testing.T) {
	// Create the fake adapter
	fakeAdapter := newFakeAdapter("test-agent")

	// Assert it implements the interface
	var _ Adapter = fakeAdapter // Compile-time check

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Start the adapter
	eventCh, err := fakeAdapter.Start(ctx, "/tmp/test", "Fix the bug")
	require.NoError(t, err)

	// Collect events
	events := collectEvents(t, eventCh)

	// Verify events
	require.Len(t, events, 3) // Thinking, Action, Complete

	// Check event types and sequence
	assert.Equal(t, protocol.EventTypeThinking, events[0].Type)
	assert.Equal(t, protocol.EventTypeAction, events[1].Type)
	assert.Equal(t, protocol.EventTypeComplete, events[2].Type)

	// Verify agent ID is set correctly
	assert.Equal(t, "test-agent", events[0].AgentID)

	// Verify sequence numbers are correct
	assert.Equal(t, 1, events[0].SequenceNum)
	assert.Equal(t, 2, events[1].SequenceNum)
	assert.Equal(t, 3, events[2].SequenceNum)

	// Test shutdown
	err = fakeAdapter.Shutdown()
	require.NoError(t, err)
	assert.True(t, fakeAdapter.shutdown)
}

// TestAdapterCancel tests that the adapter handles context cancellation
func TestAdapterCancel(t *testing.T) {
	// Create the fake adapter
	fakeAdapter := newFakeAdapter("test-agent")

	// Create a context with immediate cancellation
	ctx, cancel := context.WithCancel(context.Background())
	
	// Start the adapter
	eventCh, err := fakeAdapter.Start(ctx, "/tmp/test", "Fix the bug")
	require.NoError(t, err)

	// Immediately cancel the context
	cancel()

	// Collect events
	events := collectEvents(t, eventCh)

	// Should only get thinking and error events
	require.GreaterOrEqual(t, len(events), 1)
	assert.Equal(t, protocol.EventTypeThinking, events[0].Type)

	// If we have more than one event, the second should be an error
	if len(events) > 1 {
		assert.Equal(t, protocol.EventTypeError, events[1].Type)
	}
}

// collectEvents reads all events from the channel until it's closed
func collectEvents(t *testing.T, eventCh <-chan *protocol.Event) []*protocol.Event {
	var events []*protocol.Event
	
	// Wait for a reasonable time for events
	timeout := time.After(2 * time.Second)
	
	for {
		select {
		case event, ok := <-eventCh:
			if !ok {
				// Channel closed
				return events
			}
			events = append(events, event)
		case <-timeout:
			t.Fatal("Timeout waiting for events")
			return events
		}
	}
}
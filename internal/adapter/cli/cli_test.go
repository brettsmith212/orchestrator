package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/brettsmith212/orchestrator/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLIAdapter(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Create a fake CLI script that outputs ND-JSON events
	testScriptPath := filepath.Join(tempDir, "test-agent.sh")
	testScript := `#!/bin/sh
# Simple test script that outputs ND-JSON events

echo '{"type":"thinking","timestamp":"2023-05-20T10:30:00Z","payload":{"content":"Analyzing the problem..."}}'  
sleep 0.1
echo '{"type":"action","timestamp":"2023-05-20T10:30:01Z","payload":{"action_type":"file_edit","file_path":"test.txt","content":"test content"}}'
sleep 0.1
echo '{"type":"complete","timestamp":"2023-05-20T10:30:02Z"}'
`

	// Write the script
	err := os.WriteFile(testScriptPath, []byte(testScript), 0755)
	require.NoError(t, err, "Failed to write test script")

	// Create a CLI adapter
	adapter := New("test-agent", testScriptPath, []string{})

	// Start the adapter
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	eventCh, err := adapter.Start(ctx, tempDir, "Fix the bug")
	require.NoError(t, err, "Failed to start adapter")

	// Collect events from the channel
	events := []*protocol.Event{}
	for event := range eventCh {
		events = append(events, event)
		if event.Type == protocol.EventTypeComplete {
			break
		}
	}

	// Verify we received the expected events
	require.Len(t, events, 3, "Expected 3 events")

	// Verify event types
	assert.Equal(t, protocol.EventTypeThinking, events[0].Type, "First event should be thinking")
	assert.Equal(t, protocol.EventTypeAction, events[1].Type, "Second event should be action")
	assert.Equal(t, protocol.EventTypeComplete, events[2].Type, "Third event should be complete")

	// Verify agent ID was set for all events
	for i, event := range events {
		assert.Equal(t, "test-agent", event.AgentID, fmt.Sprintf("Event %d missing agent ID", i))
	}

	// Verify sequence numbers were set
	assert.Equal(t, 1, events[0].SequenceNum, "First event should have sequence 1")
	assert.Equal(t, 2, events[1].SequenceNum, "Second event should have sequence 2")
	assert.Equal(t, 3, events[2].SequenceNum, "Third event should have sequence 3")

	// Test shutdown
	err = adapter.Shutdown()
	assert.NoError(t, err, "Shutdown should succeed")
}

func TestCLIAdapter_ProcessError(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a CLI adapter with a non-existent executable
	adapter := New("test-agent", "non-existent-command", []string{})

	// Start the adapter
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Should error when starting
	_, err := adapter.Start(ctx, "/tmp", "Fix the bug")
	require.Error(t, err, "Expected error for non-existent command")

	// Test shutdown (should not error even though never started)
	err = adapter.Shutdown()
	assert.NoError(t, err, "Shutdown should not error")
}

func TestCLIAdapter_ParseError(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Create a fake CLI script that outputs invalid JSON
	testScriptPath := filepath.Join(tempDir, "bad-agent.sh")
	testScript := `#!/bin/sh
# Test script that outputs invalid JSON

echo 'not a valid json'
echo '{"type":"complete","timestamp":"2023-05-20T10:30:02Z"}'
`

	// Write the script
	err := os.WriteFile(testScriptPath, []byte(testScript), 0755)
	require.NoError(t, err, "Failed to write test script")

	// Create a CLI adapter
	adapter := New("test-agent", testScriptPath, []string{})

	// Start the adapter
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	eventCh, err := adapter.Start(ctx, tempDir, "Fix the bug")
	require.NoError(t, err, "Failed to start adapter")

	// Collect events from the channel
	events := []*protocol.Event{}
	for event := range eventCh {
		events = append(events, event)
		if event.Type == protocol.EventTypeComplete {
			break
		}
	}

	// Verify we received the expected events
	require.Len(t, events, 2, "Expected 2 events")

	// First event should be an error
	assert.Equal(t, protocol.EventTypeError, events[0].Type, "First event should be error")
	
	// Extract error message
	errorPayload, err := events[0].UnmarshalErrorPayload()
	require.NoError(t, err, "Should be able to unmarshal error payload")
	assert.Contains(t, errorPayload.Message, "Failed to parse output", "Error should indicate parse failure")

	// Second event should be complete
	assert.Equal(t, protocol.EventTypeComplete, events[1].Type, "Second event should be complete")
}
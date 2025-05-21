package core

import (
	"context"
	"testing"
	"time"

	"github.com/brettsmith212/orchestrator/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWatchdog_MonitorAgent(t *testing.T) {
	watchdog := NewWatchdog(ResourceLimits{
		MaxTokens:   1000,
		MaxDuration: 5 * time.Minute,
	})

	// Monitor a new agent
	watchdog.MonitorAgent("test-agent")

	// Check that the agent is now monitored
	usage := watchdog.GetUsage()
	require.Contains(t, usage, "test-agent", "Agent should be monitored")
	require.NotNil(t, usage["test-agent"], "Agent counter should not be nil")
	assert.Equal(t, "test-agent", usage["test-agent"].AgentID, "Agent ID should match")
	assert.Equal(t, 0, usage["test-agent"].TotalTokens(), "Initial token count should be zero")
}

func TestWatchdog_TrackEvent(t *testing.T) {
	watchdog := NewWatchdog(ResourceLimits{
		MaxTokens:   1000,
		MaxDuration: 5 * time.Minute,
	})

	// Create a test event
	event := protocol.NewEvent(protocol.EventTypeAction, "test-agent", 1)
	event, err := event.WithPayload(map[string]interface{}{
		"token_count": 50,
	})
	require.NoError(t, err, "Failed to create event payload")

	// Track the event
	watchdog.TrackEvent(event)

	// Verify agent is automatically monitored
	usage := watchdog.GetUsage()
	require.Contains(t, usage, "test-agent", "Agent should be monitored")

	// Token count won't increase yet since we're using a mock event format
	// This is expected since the real token extraction is agent-specific
}

func TestWatchdog_CheckLimits(t *testing.T) {
	watchdog := NewWatchdog(ResourceLimits{
		MaxTokens:   100,
		MaxDuration: 50 * time.Millisecond, // Short duration for testing
	})

	// Add an agent
	watchdog.MonitorAgent("test-agent")

	// Manually set token count over the limit
	watchdog.mutex.Lock()
	watchdog.counters["test-agent"].OutputTokens = 150
	watchdog.mutex.Unlock()

	// Check limits
	agentsToStop := watchdog.CheckLimits()
	require.Len(t, agentsToStop, 1, "One agent should exceed limits")
	assert.Equal(t, "test-agent", agentsToStop[0], "Correct agent should be identified")

	// Add another agent that will exceed time limit
	watchdog.MonitorAgent("time-agent")

	// Wait for time limit to exceed
	time.Sleep(100 * time.Millisecond)

	// Check limits again
	agentsToStop = watchdog.CheckLimits()
	require.Len(t, agentsToStop, 2, "Two agents should exceed limits")
	assert.Contains(t, agentsToStop, "test-agent", "Token-exceeding agent should be identified")
	assert.Contains(t, agentsToStop, "time-agent", "Time-exceeding agent should be identified")
}

func TestWatchdog_GetWarningEvents(t *testing.T) {
	watchdog := NewWatchdog(ResourceLimits{
		MaxTokens:   100,
		MaxDuration: 50 * time.Millisecond, // Short duration for testing
	})

	// Add an agent with token count near warning threshold (80%)
	watchdog.MonitorAgent("token-agent")
	watchdog.mutex.Lock()
	watchdog.counters["token-agent"].OutputTokens = 85
	watchdog.mutex.Unlock()

	// Get warnings
	warnings := watchdog.GetWarningEvents()
	require.Len(t, warnings, 1, "One warning should be generated")
	assert.Equal(t, protocol.EventTypeWatchdog, warnings[0].Type, "Warning should have correct type")

	// Check that we don't generate duplicate warnings
	warnings = watchdog.GetWarningEvents()
	assert.Empty(t, warnings, "No duplicate warnings should be generated")

	// Add another agent that will exceed time warning threshold
	watchdog.MonitorAgent("time-agent")

	// Wait for time warning threshold to exceed
	time.Sleep(40 * time.Millisecond) // 80% of 50ms

	// Get warnings again
	warnings = watchdog.GetWarningEvents()
	require.Len(t, warnings, 1, "One warning should be generated")

	// Clean up
	watchdog.StopMonitoring("token-agent")
	watchdog.StopMonitoring("time-agent")
}

func TestWatchdog_RunPeriodicCheck(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping periodic check test in short mode")
	}

	watchdog := NewWatchdog(ResourceLimits{
		MaxTokens:   100,
		MaxDuration: 200 * time.Millisecond, // Short duration for testing
	})

	// Add an agent with token count above limit
	watchdog.MonitorAgent("token-agent")
	watchdog.mutex.Lock()
	watchdog.counters["token-agent"].OutputTokens = 150
	watchdog.mutex.Unlock()

	// Set up channels
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	warningCh := make(chan *protocol.Event, 10)
	terminateCh := make(chan string, 10)

	// Start periodic check
	go watchdog.RunPeriodicCheck(ctx, 50*time.Millisecond, warningCh, terminateCh)

	// Wait for termination signal
	select {
	case agentID := <-terminateCh:
		assert.Equal(t, "token-agent", agentID, "Correct agent should be terminated")
	case <-time.After(300 * time.Millisecond):
		t.Fatal("Timed out waiting for termination signal")
	}

	// Add a time-limit agent and wait for termination
	watchdog.MonitorAgent("time-agent")

	// Should get terminated due to time limit
	select {
	case agentID := <-terminateCh:
		assert.Equal(t, "time-agent", agentID, "Correct agent should be terminated")
	case <-time.After(300 * time.Millisecond):
		t.Fatal("Timed out waiting for termination signal")
	}
}

func TestExtractTokenCount(t *testing.T) {
	// Create test events for different agent types
	events := []*protocol.Event{
		protocol.NewEvent(protocol.EventTypeAction, "claude", 1),
		protocol.NewEvent(protocol.EventTypeAction, "amp", 1),
		protocol.NewEvent(protocol.EventTypeAction, "codex", 1),
		protocol.NewEvent(protocol.EventTypeAction, "unknown-agent", 1),
	}

	// Test that all extractors are called but return 0 for now
	// (since we have placeholder implementations)
	for _, event := range events {
		tokenCount := extractTokenCount(event)
		assert.Equal(t, 0, tokenCount, "Token count should be 0 for placeholder extractor")
	}
}

func TestTokenCounter_Helpers(t *testing.T) {
	// Create a token counter
	counter := &TokenCounter{
		AgentID:      "test-agent",
		InputTokens:  100,
		OutputTokens: 200,
		StartTime:    time.Now().Add(-time.Minute),
		LastActivity: time.Now().Add(-30 * time.Second),
	}

	// Test TotalTokens
	assert.Equal(t, 300, counter.TotalTokens(), "Total tokens should be sum of input and output")

	// Test Duration
	duration := counter.Duration()
	assert.Greater(t, duration, 50*time.Second, "Duration should be approximately 1 minute")
	assert.Less(t, duration, 70*time.Second, "Duration should be approximately 1 minute")

	// Test TimeSinceLastActivity
	lastActivity := counter.TimeSinceLastActivity()
	assert.Greater(t, lastActivity, 20*time.Second, "LastActivity should be approximately 30 seconds")
	assert.Less(t, lastActivity, 40*time.Second, "LastActivity should be approximately 30 seconds")
}
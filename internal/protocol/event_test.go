package protocol

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventMarshalUnmarshal(t *testing.T) {
	// Create a test event
	testTime := time.Date(2023, 5, 20, 10, 30, 0, 0, time.UTC)
	promptPayload := PromptPayload{
		Prompt:       "Fix the bug in main.go",
		ContextFiles: []string{"main.go", "utils.go"},
	}

	payloadData, err := json.Marshal(promptPayload)
	require.NoError(t, err)

	event := &Event{
		Type:        EventTypePrompt,
		Timestamp:   testTime,
		SequenceNum: 1,
		Payload:     payloadData,
	}

	// Test Marshal
	data, err := Marshal(event)
	require.NoError(t, err)

	// Test Unmarshal
	decodedEvent, err := Unmarshal(data)
	require.NoError(t, err)

	// Verify event fields
	assert.Equal(t, event.Type, decodedEvent.Type)
	assert.Equal(t, event.Timestamp.Unix(), decodedEvent.Timestamp.Unix())
	assert.Equal(t, event.SequenceNum, decodedEvent.SequenceNum)

	// Test payload unmarshaling
	decodedPayload, err := decodedEvent.UnmarshalPromptPayload()
	require.NoError(t, err)

	assert.Equal(t, promptPayload.Prompt, decodedPayload.Prompt)
	assert.Equal(t, promptPayload.ContextFiles, decodedPayload.ContextFiles)
}

func TestNewEvent(t *testing.T) {
	event := NewEvent(EventTypeAction, "test-agent", 42)

	assert.Equal(t, EventTypeAction, event.Type)
	assert.Equal(t, "test-agent", event.AgentID)
	assert.Equal(t, 42, event.SequenceNum)
	// Timestamp should be close to now
	assert.WithinDuration(t, time.Now().UTC(), event.Timestamp, 2*time.Second)
}

func TestEventWithPayload(t *testing.T) {
	event := NewEvent(EventTypeAction, "test-agent", 1)

	actionPayload := ActionPayload{
		ActionType: "file_edit",
		FilePath:   "main.go",
		Content:    "package main\n\nfunc main() {}\n",
	}

	event, err := event.WithPayload(actionPayload)
	require.NoError(t, err)

	decodedPayload, err := event.UnmarshalActionPayload()
	require.NoError(t, err)

	assert.Equal(t, actionPayload.ActionType, decodedPayload.ActionType)
	assert.Equal(t, actionPayload.FilePath, decodedPayload.FilePath)
	assert.Equal(t, actionPayload.Content, decodedPayload.Content)
}

func TestNDJSONRoundtrip(t *testing.T) {
	// Create test events
	event1 := NewEvent(EventTypePrompt, "", 1)
	promptPayload := PromptPayload{Prompt: "Fix bug"}
	var err error
	event1, err = event1.WithPayload(promptPayload)
	require.NoError(t, err)

	event2 := NewEvent(EventTypeThinking, "agent1", 1)
	thinkingPayload := ThinkingPayload{Content: "Analyzing code..."}
	event2, err = event2.WithPayload(thinkingPayload)
	require.NoError(t, err)

	event3 := NewEvent(EventTypeAction, "agent1", 2)
	actionPayload := ActionPayload{ActionType: "file_edit", FilePath: "main.go"}
	event3, err = event3.WithPayload(actionPayload)
	require.NoError(t, err)

	// Write to NDJSON
	buf := &bytes.Buffer{}
	err = WriteNDJSON(buf, event1, event2, event3)
	require.NoError(t, err)

	// Read from NDJSON
	events, err := ReadNDJSON(buf.Bytes())
	require.NoError(t, err)

	// Verify
	assert.Len(t, events, 3)

	// Check first event
	assert.Equal(t, EventTypePrompt, events[0].Type)
	assert.Equal(t, "", events[0].AgentID)
	assert.Equal(t, 1, events[0].SequenceNum)

	// Check second event
	assert.Equal(t, EventTypeThinking, events[1].Type)
	assert.Equal(t, "agent1", events[1].AgentID)
	assert.Equal(t, 1, events[1].SequenceNum)

	// Check third event
	assert.Equal(t, EventTypeAction, events[2].Type)
	assert.Equal(t, "agent1", events[2].AgentID)
	assert.Equal(t, 2, events[2].SequenceNum)

	// Verify payload unmarshaling still works
	decodedPrompt, err := events[0].UnmarshalPromptPayload()
	require.NoError(t, err)
	assert.Equal(t, "Fix bug", decodedPrompt.Prompt)

	decodedThinking, err := events[1].UnmarshalThinkingPayload()
	require.NoError(t, err)
	assert.Equal(t, "Analyzing code...", decodedThinking.Content)

	decodedAction, err := events[2].UnmarshalActionPayload()
	require.NoError(t, err)
	assert.Equal(t, "file_edit", decodedAction.ActionType)
	assert.Equal(t, "main.go", decodedAction.FilePath)
}

func TestPayloadTypeChecking(t *testing.T) {
	// Test that trying to unmarshal the wrong payload type returns an error
	event := NewEvent(EventTypeAction, "agent1", 1)
	actionPayload := ActionPayload{ActionType: "file_edit"}
	var err error
	event, err = event.WithPayload(actionPayload)
	require.NoError(t, err)

	// Try to unmarshal as a thinking payload (wrong type)
	_, err = event.UnmarshalThinkingPayload()
	assert.Error(t, err)

	// Correct type should work
	_, err = event.UnmarshalActionPayload()
	assert.NoError(t, err)
}
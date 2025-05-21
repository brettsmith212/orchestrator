package protocol

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

// EventType represents the type of an event in the protocol
type EventType string

// Event types defined by the protocol
const (
	// Events from orchestrator to agent
	EventTypePrompt    EventType = "prompt"     // Initial task prompt
	EventTypeCancel    EventType = "cancel"     // Request to cancel work
	EventTypeWatchdog  EventType = "watchdog"   // Resource limit warning

	// Events from agent to orchestrator
	EventTypeThinking  EventType = "thinking"   // Agent is thinking/planning
	EventTypeAction    EventType = "action"     // Agent performed an action
	EventTypeComplete  EventType = "complete"   // Agent completed the task
	EventTypeError     EventType = "error"      // Agent encountered an error
)

// Event represents a single protocol event in the communication stream
type Event struct {
	// Type defines the event type
	Type EventType `json:"type"`

	// Timestamp when the event was created
	Timestamp time.Time `json:"timestamp"`

	// AgentID identifies which agent generated this event (empty if from orchestrator)
	AgentID string `json:"agent_id,omitempty"`

	// SequenceNum is monotonically increasing for events from the same source
	SequenceNum int `json:"sequence_num,omitempty"`

	// Payload contains event-specific data
	Payload json.RawMessage `json:"payload,omitempty"`
}

// PromptPayload contains data for a prompt event
type PromptPayload struct {
	// Prompt is the task description
	Prompt string `json:"prompt"`

	// ContextFiles are optional files relevant to the task
	ContextFiles []string `json:"context_files,omitempty"`
}

// ThinkingPayload contains data for a thinking event
type ThinkingPayload struct {
	// Content is the thinking/planning text
	Content string `json:"content"`
}

// ActionPayload contains data for an action event
type ActionPayload struct {
	// ActionType defines what kind of action was performed
	ActionType string `json:"action_type"`

	// FilePath is the path of the file being modified (if applicable)
	FilePath string `json:"file_path,omitempty"`

	// Content is the content being added/modified (if applicable)
	Content string `json:"content,omitempty"`

	// Diff is a unified diff representation of the change (if applicable)
	Diff string `json:"diff,omitempty"`
}

// ErrorPayload contains data for an error event
type ErrorPayload struct {
	// Message is the error message
	Message string `json:"message"`

	// Code is an optional error code
	Code string `json:"code,omitempty"`
}

// NewEvent creates a new event with the current timestamp
func NewEvent(eventType EventType, agentID string, sequenceNum int) *Event {
	return &Event{
		Type:        eventType,
		Timestamp:   time.Now().UTC(),
		AgentID:     agentID,
		SequenceNum: sequenceNum,
	}
}

// WithPayload adds a payload to the event
func (e *Event) WithPayload(payload interface{}) (*Event, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}
	e.Payload = json.RawMessage(data)
	return e, nil
}

// Marshal serializes an event to JSON
func Marshal(event *Event) ([]byte, error) {
	return json.Marshal(event)
}

// Unmarshal deserializes an event from JSON
func Unmarshal(data []byte) (*Event, error) {
	event := &Event{}
	if err := json.Unmarshal(data, event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}
	return event, nil
}

// UnmarshalPromptPayload deserializes a prompt payload
func (e *Event) UnmarshalPromptPayload() (*PromptPayload, error) {
	if e.Type != EventTypePrompt {
		return nil, fmt.Errorf("event is not a prompt event")
	}
	var payload PromptPayload
	if err := json.Unmarshal(e.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal prompt payload: %w", err)
	}
	return &payload, nil
}

// UnmarshalThinkingPayload deserializes a thinking payload
func (e *Event) UnmarshalThinkingPayload() (*ThinkingPayload, error) {
	if e.Type != EventTypeThinking {
		return nil, fmt.Errorf("event is not a thinking event")
	}
	var payload ThinkingPayload
	if err := json.Unmarshal(e.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal thinking payload: %w", err)
	}
	return &payload, nil
}

// UnmarshalActionPayload deserializes an action payload
func (e *Event) UnmarshalActionPayload() (*ActionPayload, error) {
	if e.Type != EventTypeAction {
		return nil, fmt.Errorf("event is not an action event")
	}
	var payload ActionPayload
	if err := json.Unmarshal(e.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal action payload: %w", err)
	}
	return &payload, nil
}

// UnmarshalErrorPayload deserializes an error payload
func (e *Event) UnmarshalErrorPayload() (*ErrorPayload, error) {
	if e.Type != EventTypeError {
		return nil, fmt.Errorf("event is not an error event")
	}
	var payload ErrorPayload
	if err := json.Unmarshal(e.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal error payload: %w", err)
	}
	return &payload, nil
}

// WriteNDJSON writes events to the given buffer in ND-JSON format
func WriteNDJSON(buf *bytes.Buffer, events ...*Event) error {
	for _, event := range events {
		data, err := Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal event: %w", err)
		}
		
		// Write the JSON line
		_, err = buf.Write(data)
		if err != nil {
			return fmt.Errorf("failed to write event: %w", err)
		}
		
		// Add newline
		_, err = buf.Write([]byte("\n"))
		if err != nil {
			return fmt.Errorf("failed to write newline: %w", err)
		}
	}
	return nil
}

// ReadNDJSON reads events from the given buffer in ND-JSON format
func ReadNDJSON(data []byte) ([]*Event, error) {
	var events []*Event
	lines := bytes.Split(data, []byte("\n"))
	
	for _, line := range lines {
		// Skip empty lines
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		
		event, err := Unmarshal(line)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal event: %w", err)
		}
		events = append(events, event)
	}
	
	return events, nil
}
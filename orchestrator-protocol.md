# Orchestrator Protocol

## Overview

This document defines the JSON protocol used for communication between the orchestrator and CLI-based AI coding agents. The protocol is designed to be simple, extensible, and compatible with different AI systems using a unified ND-JSON format.

## Protocol Version

**Current Version:** `0.1.0`

## CLI Agent Interface

All agents must support a command-line interface that:
1. Accepts a prompt as input
2. Processes that input in a git repository context
3. Outputs events as ND-JSON (newline-delimited JSON) to stdout

Typical invocation pattern:
```
<agent-command> [global-options] --output-format stream-json [working-directory-options] [prompt-args]
```

## Event Types

All events are encoded as JSON objects with a `type` field that indicates the event type.

### Event Structure

```json
{
  "type": "[event-type]",
  "timestamp": "2023-05-20T10:30:00Z",
  "agent_id": "[agent-identifier]",
  "sequence_num": 1,
  "payload": { ... }
}
```

### Agent Events

Events sent from CLI agents to the orchestrator via stdout:

- `thinking` - Agent is thinking/planning (includes text content)
- `action` - Agent performed an action (file edits, etc.)
- `complete` - Agent completed the task
- `error` - Agent encountered an error

### Orchestrator Events

Events sent from orchestrator to agents via stdin or command arguments:

- `prompt` - Initial task description
- `cancel` - Request to cancel work
- `watchdog` - Resource limit warning

## Versioning Rules

TBD: Version compatibility requirements and rules

## Example Event Streams

TBD: Example interaction sequences

## Error Handling

TBD: Error response codes and recovery mechanisms
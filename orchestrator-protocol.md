# Orchestrator Protocol

## Overview

This document defines the JSON protocol used for communication between the orchestrator and AI coding agents. The protocol is designed to be simple, extensible, and compatible with different AI systems.

## Protocol Version

**Current Version:** `0.1.0`

## Event Types

All events are encoded as JSON objects with a `type` field that indicates the event type.

### TBD: Event Structure

```json
{
  "type": "[event-type]",
  "timestamp": "2023-05-20T10:30:00Z",
  // Additional fields depend on event type
}
```

### TBD: Agent Events

Events sent from agents to the orchestrator.

### TBD: Orchestrator Events

Events sent from the orchestrator to agents.

## Versioning Rules

TBD: Version compatibility requirements and rules

## Example Event Streams

TBD: Example interaction sequences

## Error Handling

TBD: Error response codes and recovery mechanisms
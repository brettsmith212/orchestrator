# AI Coding Agents Compatibility Matrix

This document lists the AI coding agents that are compatible with the orchestrator.

## Installation Instructions

| Agent | Installation Command | Version | Notes |
|-------|---------------------|---------|-------|
| Amp   | `npm install -g @sourcegraph/cody-cli` | | Sourcegraph Cody CLI |
| OpenAI | N/A - HTTP API | | Requires API key |
| aider | `pip install aider-chat` | | |
| | | | |
| | | | |

## Configuration Guide

### Amp (Sourcegraph Cody)

```yaml
agents:
  - id: "amp"
    type: "cli"
    config:
      command: "amp"
      args: ["-w", ".", "--json-output"]
```

### OpenAI API

```yaml
agents:
  - id: "openai"
    type: "http"
    config:
      api_url: "https://api.openai.com/v1/chat/completions"
      model: "gpt-4"
      max_tokens: 2000
```

## Exit Codes

| Agent | Success | Error | Timeout |
|-------|---------|-------|--------|
| Amp   | 0       | 1     | 124    |
| OpenAI | N/A     | N/A   | N/A    |
| aider | 0       | 1     | 124    |
# AI Coding Agents Compatibility Matrix

This document lists the AI coding agents that are compatible with the orchestrator.

## Installation Instructions

| Agent       | Installation Command                       | Version | Notes                     |
| ----------- | ------------------------------------------ | ------- | ------------------------- |
| Amp         | `npm install -g @sourcegraph/amp`          |         | Sourcegraph Cody CLI      |
| Codex       | `npm install -g @openai/codex`             |         | OpenAI Codex CLI          |
| Claude Code | `npm install -g @anthropic-ai/claude-code` |         | Anthropic Claude Code CLI |
|             |                                            |         |                           |
|             |                                            |         |                           |

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

### OpenAI Codex

```yaml
agents:
  - id: "codex"
    type: "cli"
    config:
      command: "codex"
      args: ["run", "--output-format", "stream-json"]
```

### Claude Code

```yaml
agents:
  - id: "claude"
    type: "cli"
    config:
      command: "claude"
      args: ["--output-format", "stream-json"]
```

## Exit Codes

| Agent       | Success | Error | Timeout |
| ----------- | ------- | ----- | ------- |
| Amp         | 0       | 1     | 124     |
| Codex       | 0       | 1     | 124     |
| Claude Code | 0       | 1     | 124     |

# Orchestrator

## Vision

Orchestrator is a tool designed to coordinate multiple AI coding agents to collectively solve programming tasks. By running various CLI-based agents in parallel on the same problem and comparing their solutions, Orchestrator can select the most effective solution based on test results and other criteria.

## Features

- Run multiple AI coding agents in parallel
- Coordinate agent execution and solution evaluation
- Support for CLI-based AI coding agents (Sourcegraph Amp, OpenAI Codex, Claude Code)
- Git-based worktree isolation for each agent
- Test-driven solution evaluation
- Optional TUI for monitoring agent progress

## Usage

```
go run ./cmd/orchestrator
```
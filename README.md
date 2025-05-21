# Orchestrator

## Vision

Orchestrator is a tool designed to coordinate multiple AI coding agents to collectively solve programming tasks. By running various agents in parallel on the same problem and comparing their solutions, Orchestrator can select the most effective solution based on test results and other criteria.

## Features

- Run multiple AI coding agents in parallel
- Coordinate agent execution and solution evaluation
- Support for both HTTP-based and CLI-based AI coding agents
- Git-based worktree isolation for each agent
- Test-driven solution evaluation
- Optional TUI for monitoring agent progress

## Usage

```
go run ./cmd/orchestrator
```
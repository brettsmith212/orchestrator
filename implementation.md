# Implementation Plan

## Project Setup

- [x] Step 1: Initialise repo scaffold

  - **Task**: Create directory layout, empty main, README stub.
  - **Description**: Allows subsequent steps to add code without restructuring.
  - **Files**:
    - `go.mod` – `module github.com/brettsmith212/orchestrator`
    - `README.md` – outline vision
    - `cmd/orchestrator/main.go` – “hello orchestrator”
    - `internal/{core,protocol,adapter,gitutil}/.gitkeep`
  - **Step Dependencies**: none
  - **User Instructions**: `go run ./cmd/orchestrator` should print a greeting.

- [x] Step 2: Add core dependencies & tooling config

  - **Task**: Pin Go version, add test deps, set up `make test`.
  - **Description**: Ensures deterministic builds & CI later.
  - **Files**:
    - `go.mod` / `go.sum` – add `github.com/stretchr/testify`
    - `Makefile` – `test`, `lint`
  - **Dependencies**: Step 1
  - **User Instructions**: `make test` passes (no tests yet).

- [x] Step 3: Establish configuration struct & default file

  - **Task**: Define `Config` (YAML) with basic paths and agent list.
  - **Description**: Central place for flags & future env overrides.
  - **Files**:
    - `internal/core/config.go` – struct + `Load(path)`
    - `config.example.yaml`
    - `internal/core/config_test.go`
  - **Dependencies**: Step 2
  - **User Instructions**: `go test ./...` should still pass.

- [x] Step 4: Stub living documentation
  - **Task**: Add placeholders so docs stay version-controlled.
  - **Description**: Early visibility drives doc discipline.
  - **Files**:
    - `AGENTS.md` – empty table rows
    - `orchestrator-protocol.md` – TBD sections
  - **Dependencies**: Step 1
  - **User Instructions**: Visual check in editor; no code impact.

## Core Interfaces

- [x] Step 5: Define `Event` types & ND-JSON codec

  - **Task**: Create `protocol.Event` struct + `Marshal/Unmarshal`.
  - **Description**: Contracts every component relies on.
  - **Files**:
    - `internal/protocol/event.go`
    - `internal/protocol/event_test.go`
  - **Dependencies**: Step 3
  - **User Instructions**: Tests ensure round-trip JSON fidelity.

- [x] Step 6: Draft `Adapter` interface

  - **Task**: Interface with `Start(ctx, worktree, prompt) (<-chan protocol.Event, error)` & `Shutdown()`.
  - **Description**: Provides compile target for all agents.
  - **Files**:
    - `internal/adapter/adapter.go`
    - `internal/adapter/adapter_test.go` (uses tiny fake)
  - **Dependencies**: Step 5

- [x] Step 7: Implement adapter registry
  - **Task**: Map agent IDs → factory funcs; load via config.
  - **Description**: Decouples Orchestrator from concrete adapters.
  - **Files**:
    - `internal/adapter/registry.go`
    - `internal/adapter/registry_test.go`
  - **Dependencies**: Step 6

## Adapters – CLI-only

- [x] **Step 8: Generic `cliAdapter` implementation**

  - **Task**: Exec external binary, stream `stdout` lines, map to events.
  - **Description**: Single code-path for all agents.
  - **Files**:
    - `internal/adapter/cli/cli.go`
    - `internal/adapter/cli/cli_test.go`
  - **Dependencies**: Step 6

- [x] **Step 9: Sourcegraph Amp CLI adapter**

  - **Task**: Install hint (`npm install -g @sourcegraph/amp`), Config preset (binary name `amp`, args), integration test with fake script.
  - **Files**:
    - `internal/adapter/amp/amp.go`
    - `internal/adapter/amp/amp_test.go`
  - **Dependencies**: Step 8

- [x] **Step 10: OpenAI Codex CLI adapter**

  - **Task**: Install hint (`npm -g @openai/codex`), invoke `codex run … --output-format stream-json`.
  - **Files**:
    - `internal/adapter/codex/codex.go`
    - `internal/adapter/codex/codex_test.go`
  - **Dependencies**: Step 8

- [x] **Step 11: Claude Code CLI adapter**
  - **Task**: Install hint (`npm install -g @anthropic-ai/claude-code`) Invoke `claude-code … --output-format stream-json`, parse events, track token usage.
  - **Files**:
    - `internal/adapter/claude/claude.go`
    - `internal/adapter/claude/claude_test.go`
  - **Dependencies**: Step 8

## Git Work-Tree & Diff Utilities

- [ ] **Step 12: Git work-tree manager**

  - **Files**:
    - `internal/gitutil/worktree.go`
    - `internal/gitutil/worktree_test.go`
  - **Dependencies**: Step 3

- [ ] **Step 13: Unified diff normaliser**
  - **Files**:
    - `internal/gitutil/diff.go`
    - `internal/gitutil/diff_test.go`
  - **Dependencies**: Step 12

## Test Runner & Patch Arbitration

- [ ] **Step 14: Minimal test runner wrapper**

  - **Files**:
    - `internal/core/testrunner.go`
    - `internal/core/testrunner_test.go`
  - **Dependencies**: Step 12

- [ ] **Step 15: Patch selector logic**
  - **Files**:
    - `internal/core/arbitrator.go`
    - `internal/core/arbitrator_test.go`
  - **Dependencies**: Steps 5, 13-14

## Orchestrator Command & Watchdogs

- [ ] **Step 16: Wire up `cmd/orchestrator run` baseline**

  - **Files**:
    - `cmd/orchestrator/main.go` (replace stub)
    - `cmd/orchestrator/main_test.go`
  - **Dependencies**: Steps 7-15

- [ ] **Step 17: Cost & timeout watchdog**
  - **Task**: Parse token counts from agent events; enforce caps.
  - **Files**:
    - `internal/core/watchdog.go`
    - `internal/core/watchdog_test.go`
  - **Dependencies**: Step 16

## Event Stream & Optional TUI

- [ ] **Step 18: ND-JSON streamer utility**

  - **Files**:
    - `internal/core/streamer.go`
    - `internal/core/streamer_test.go`
  - **Dependencies**: Steps 5, 16

- [ ] **Step 19: Bubble Tea TUI MVP**

  - **Files**:
    - `internal/ui/model.go`
    - `internal/ui/view.go`
    - `internal/ui/update.go`
    - `internal/ui/ui_test.go`
  - **Dependencies**: Step 18

- [ ] **Step 20: Non-TTY fallback guard**
  - **Files**:
    - `cmd/orchestrator/tty_guard.go`
    - `cmd/orchestrator/tty_guard_test.go`
  - **Dependencies**: Step 19

## Living Documentation & Examples

- [ ] **Step 23: Populate `AGENTS.md` install matrix**

  - **Files**:
    - `AGENTS.md`
  - **Dependencies**: Steps 9-11

- [ ] **Step 24: Draft `orchestrator-protocol.md` v1**

  - **Files**:
    - `orchestrator-protocol.md`
  - **Dependencies**: Steps 5, 18

- [ ] **Step 25: End-to-end demo script & README update**
  - **Files**:
    - `examples/demo.sh`
    - `README.md`
  - **Dependencies**: Steps 16-24

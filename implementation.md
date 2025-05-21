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

- [ ] Step 6: Draft `Adapter` interface

  - **Task**: Interface with `Start(ctx, worktree, prompt) (<-chan protocol.Event, error)` & `Shutdown()`.
  - **Description**: Provides compile target for all agents.
  - **Files**:
    - `internal/adapter/adapter.go`
    - `internal/adapter/adapter_test.go` (uses tiny fake)
  - **Dependencies**: Step 5

- [ ] Step 7: Implement adapter registry
  - **Task**: Map agent IDs → factory funcs; load via config.
  - **Description**: Decouples Orchestrator from concrete adapters.
  - **Files**:
    - `internal/adapter/registry.go`
    - `internal/adapter/registry_test.go`
  - **Dependencies**: Step 6

## Adapters – HTTP & CLI

- [ ] Step 8: Skeleton `httpAdapter` for OpenAI Codex

  - **Task**: Scaffolding with mocked API call & fake response.
  - **Description**: Exercises registry and event flow early.
  - **Files**:
    - `internal/adapter/codex/http.go`
    - `internal/adapter/codex/http_test.go`
  - **Dependencies**: Steps 6-7
  - **User Instructions**: Unit test uses httptest server; no real keys.

- [ ] Step 9: Generic `cliAdapter` implementation

  - **Task**: Run external binary, stream JSON lines to events.
  - **Description**: One codepath for Amp, Aider, etc.
  - **Files**:
    - `internal/adapter/cli/cli.go`
    - `internal/adapter/cli/cli_test.go`
  - **Dependencies**: Step 6

- [ ] Step 10: Sourcegraph Amp adapter via `cliAdapter`
  - **Task**: Config preset (install hint, args) + integration test w/ fake script.
  - **Description**: Validates cliAdapter flexibility.
  - **Files**:
    - `internal/adapter/amp/amp.go`
    - `internal/adapter/amp/amp_test.go`
  - **Dependencies**: Step 9

## Git Work-Tree & Diff Utilities

- [ ] Step 11: Git work-tree manager

  - **Task**: Create temp worktree, checkout current HEAD, cleanup.
  - **Description**: Isolates each agent’s patch.
  - **Files**:
    - `internal/gitutil/worktree.go`
    - `internal/gitutil/worktree_test.go`
  - **Dependencies**: Step 3

- [ ] Step 12: Unified diff normaliser
  - **Task**: Normalise `git diff` output for comparison/scoring.
  - **Description**: Enables objective patch ranking.
  - **Files**:
    - `internal/gitutil/diff.go`
    - `internal/gitutil/diff_test.go`
  - **Dependencies**: Step 11

## Test Runner & Patch Arbitration

- [ ] Step 13: Minimal test runner wrapper

  - **Task**: Run project’s tests inside a given worktree, capture JSON summary.
  - **Description**: Determines “did patch fix failing tests?”
  - **Files**:
    - `internal/core/testrunner.go`
    - `internal/core/testrunner_test.go`
  - **Dependencies**: Step 11

- [ ] Step 14: Patch selector logic
  - **Task**: Compare events + test results, select best diff.
  - **Description**: Core “arbitrator” responsibility.
  - **Files**:
    - `internal/core/arbitrator.go`
    - `internal/core/arbitrator_test.go`
  - **Dependencies**: Steps 5, 12-13

## Orchestrator Command & Watchdogs

- [ ] Step 15: Wire up `cmd/orchestrator run` baseline

  - **Task**: Parse flags, load config, instantiate registry, fan-out prompt, print ND-JSON.
  - **Description**: End-to-end baseline proving earlier layers work.
  - **Files**:
    - `cmd/orchestrator/main.go` (replace stub)
    - `cmd/orchestrator/main_test.go` (uses exec.Command test)
  - **Dependencies**: Steps 7-14

- [ ] Step 16: Cost & timeout watchdog
  - **Task**: Track tokens/time, cancel overruns, emit warning events.
  - **Description**: Prevent runaway spend; aligns with spec.
  - **Files**:
    - `internal/core/watchdog.go`
    - `internal/core/watchdog_test.go`
  - **Dependencies**: Step 15

## Event Stream & Optional TUI

- [ ] Step 17: ND-JSON streamer utility

  - **Task**: Multiplex adapter event channels → stdout or pipe.
  - **Description**: Guarantees consistent format for CLI & TUI.
  - **Files**:
    - `internal/core/streamer.go`
    - `internal/core/streamer_test.go`
  - **Dependencies**: Step 5, 15

- [ ] Step 18: Bubble Tea TUI MVP

  - **Task**: Render live list of agents & status; flag `--ui`.
  - **Description**: Optional richer UX per spec.
  - **Files**:
    - `internal/ui/model.go`
    - `internal/ui/view.go`
    - `internal/ui/update.go`
    - `internal/ui/ui_test.go`
  - **Dependencies**: Step 17

- [ ] Step 19: Non-TTY fallback guard
  - **Task**: Detect `$TERM=dumb` or piped stdout, disable TUI.
  - **Description**: Mitigates risk #4.
  - **Files**:
    - `cmd/orchestrator/tty_guard.go`
    - `cmd/orchestrator/tty_guard_test.go`
  - **Dependencies**: Step 18

## Packaging & Continuous Delivery

- [ ] Step 20: GoReleaser config

  - **Task**: `.goreleaser.yaml` with six OS/arch targets, signing.
  - **Description**: Cross-platform binaries per risk table.
  - **Files**:
    - `.goreleaser.yaml`
  - **Dependencies**: Step 15

- [ ] Step 21: GitHub Actions CI matrix
  - **Task**: `ci.yml` to lint, test, run GoReleaser on tags.
  - **Description**: Automated builds & checks.
  - **Files**:
    - `.github/workflows/ci.yml`
  - **Dependencies**: Steps 2, 20

## Living Documentation & Examples

- [ ] Step 22: Populate `AGENTS.md` install matrix

  - **Task**: Fill real commands, version pins, exit codes.
  - **Description**: Guides users & CI.
  - **Files**:
    - `AGENTS.md`
  - **Dependencies**: Step 8-10

- [ ] Step 23: Draft `orchestrator-protocol.md` v1

  - **Task**: Document event schemas, versioning rules.
  - **Description**: Locks contract before external integrations.
  - **Files**:
    - `orchestrator-protocol.md`
  - **Dependencies**: Step 5, 17

- [ ] Step 24: End-to-end demo script & README update
  - **Task**: Tutorial using toy repo, includes GIF of TUI.
  - **Description**: Completes Milestone-1 exit criteria.
  - **Files**:
    - `examples/demo.sh`
    - `README.md` (expanded)
  - **Dependencies**: Steps 15-23

## Hardening & Edge-Cases

- [ ] Step 25: Enterprise agent whitelist feature flag

  - **Task**: `--allowed-agents` cli flag + config.
  - **Description**: Satisfies licensing mitigation.
  - **Files**:
    - `internal/core/whitelist.go`
    - `internal/core/whitelist_test.go`
  - **Dependencies**: Steps 3, 15

- [ ] Step 26: Version compatibility tests
  - **Task**: Add CI job comparing latest protocol vs frozen files.
  - **Description**: Enforces risk #5.
  - **Files**:
    - `.github/workflows/compat.yml`
    - `internal/protocol/compat_test.go`
  - **Dependencies**: Step 23

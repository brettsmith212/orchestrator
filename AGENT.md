# Agent Guidelines for Orchestrator

## Implementation Rules
- When following an implementation instruction step, only complete one step at a time unless otherwise told to do so
- After completing a step, ensure all unit tests still pass; if any fail, resolve issues immediately

## Build & Test Commands
- Build: `go build ./cmd/orchestrator`
- Run: `go run ./cmd/orchestrator`
- Test all: `make test` or `go test ./...`
- Test single package: `go test ./internal/protocol`
- Test specific test: `go test -run TestFunctionName ./internal/package`
- Skip long tests: `go test ./... -short`
- Lint: `make lint` (runs go vet and go fmt)

## Code Style Guidelines
- Imports: Group standard library, 3rd party, then internal imports
- Formatting: Use `go fmt` for all files
- Types: Strong typing with proper interfaces
- Error handling: Use `fmt.Errorf("%w", err)` for error wrapping
- Naming: Follow Go conventions (CamelCase for exported, camelCase for internal)
- Tests: Table-driven tests using github.com/stretchr/testify
- Comments: Document public APIs with proper godoc format
- Package structure: domain-driven (core, adapter, protocol, gitutil)

## Project Structure
- `cmd/`: Entry points (orchestrator binary)
- `internal/`: All implementation code (not exposed outside repository)
  - `core/`: Core domain logic and config
    - Config loading
    - Test runner
    - Arbitrator (patch selector)
  - `adapter/`: Agent adapter implementations
    - Generic CLI adapter
    - Amp, Codex, Claude specific adapters
    - Adapter registry
  - `protocol/`: Communication protocols and events
    - Event types and ND-JSON serialization
  - `gitutil/`: Git utilities and worktree management
    - Worktree creation and management
    - Diff normalization and comparison

## Key Design Elements
- **Adapter Pattern**: All AI agents implement the common Adapter interface
- **Event-Based Communication**: Agents communicate via standardized event streams
- **Worktree Isolation**: Each agent works in its own git worktree
- **Test-Driven Evaluation**: Patches are evaluated based on test results
- **Scoring System**: The arbitrator selects the best patch based on multiple factors

## Common Patterns
- Creating agent tests: Use `-short` flag to skip integration tests that call external processes
- CLI adapters: All agent-specific adapters delegate to the common CLI adapter
- Event handling: Always include agent ID and sequence numbers in events
- Context usage: Pass context.Context to functions that may run for extended periods
- Error handling: Wrap errors with contextual information when returning them
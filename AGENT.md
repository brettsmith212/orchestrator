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
  - `adapter/`: Agent adapter implementations
  - `protocol/`: Communication protocols and events
  - `gitutil/`: Git utilities and worktree management
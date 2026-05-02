# Phase 1 Research: CLI Foundation

## RESEARCH COMPLETE

## Objective

Research what is needed to plan Phase 1 well: a Go CLI foundation named `dem` with JSON-only success output, JSON errors on stderr, help/version behavior that does not touch HLTV, and an internal extension pattern for future commands.

## Relevant Current Context

- No application code exists yet.
- Remote repository is `https://github.com/alekseitsvetkov/dem.git`, so the Go module path should be `github.com/alekseitsvetkov/dem`.
- Phase 1 requirements are `CLI-01` through `CLI-04`.
- Phase 1 must not implement HLTV network fetching or parsing; those start in Phase 2.

## Recommended Technical Shape

### Go Module

Initialize a Go module at the repository root:

```text
module github.com/alekseitsvetkov/dem
```

Use Go's conventional CLI layout:

- `cmd/dem/main.go` for the binary entry point.
- `internal/cli` for Cobra command construction and command handlers.
- `internal/output` for success/error JSON encoding.
- `internal/domain` for shared response structs.

This layout keeps future HLTV provider packages out of the CLI command wiring while making `dem events`, `dem results`, and `dem demo <match-id>` easy to add later.

### CLI Framework

Use `github.com/spf13/cobra` for root/subcommand wiring. Cobra is appropriate here because:

- It supports a clean command tree as new features arrive.
- Commands can inject output writers and dependencies in tests.
- Help behavior is local and does not require network access.

### JSON Contract

Successful output should always use the envelope:

```json
{
  "data": {},
  "meta": {}
}
```

Errors should always use the envelope:

```json
{
  "error": {
    "code": "validation_error",
    "message": "human readable summary",
    "details": {}
  }
}
```

Stdout must be reserved for successful JSON. Stderr must be reserved for error JSON and diagnostics.

### Baseline Command

Implement `dem version` first. It can return:

```json
{
  "data": {
    "name": "dem",
    "version": "dev",
    "commit": "unknown",
    "date": "unknown"
  },
  "meta": {}
}
```

Use package variables with default values so later release builds can inject values with `-ldflags`.

### Testing Strategy

Plan 1 should include basic unit tests for output helpers. Plan 2 should add CLI contract tests using injected buffers:

- `dem version` writes valid JSON to stdout and nothing to stderr.
- `dem --help` or `dem help` succeeds without using any network dependency.
- Unknown commands return non-zero and write structured JSON to stderr.
- A test-only command can prove the command registration pattern without modifying existing handlers.

## Key Planning Risks

- Cobra's default error/help behavior may print plain text to stderr unless explicitly controlled. The executor must configure `SilenceUsage`, `SilenceErrors`, and centralized error handling.
- Calling `os.Exit` inside command handlers would make tests and future embedding harder. Return errors instead and let `main` decide the exit code.
- JSON helpers should be generic enough for any future data type without encouraging `map[string]any` everywhere.

## Validation Architecture

The phase can be validated entirely with local commands:

- `go test ./...`
- `go run ./cmd/dem version`
- `go run ./cmd/dem --help`
- `go run ./cmd/dem does-not-exist`

The executor must verify stdout/stderr separation for success and failure commands.

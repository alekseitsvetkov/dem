---
phase: 01-cli-foundation
plan: 02
subsystem: cli
tags: [go, cobra, json, errors, tests]

requires:
  - phase: 01-cli-foundation
    provides: Go module, root command, version command, and success JSON envelope
provides:
  - Structured JSON error envelope helper
  - Centralized CLI execution and exit behavior
  - CLI contract tests for version, help, unknown command, and subcommand extension
affects: [cli-foundation, hltv-provider, events-results, demo-lookup]

tech-stack:
  added: []
  patterns: [json-error-envelope, centralized-execute, command-extension-tests]

key-files:
  created:
    - internal/output/error.go
    - internal/output/error_test.go
    - internal/cli/root_test.go
  modified:
    - cmd/dem/main.go
    - internal/cli/root.go

key-decisions:
  - "Use `command_error` as the stable fallback error code for command execution failures."
  - "Keep Cobra default errors silenced and route process failures through JSON on stderr."

patterns-established:
  - "Errors use a top-level `error` envelope with stable `code`, human-readable `message`, and object-valued `details`."
  - "Only `cmd/dem/main.go` calls `os.Exit`; command packages return errors or exit codes for testability."

requirements-completed: [CLI-01, CLI-02, CLI-03, CLI-04]

duration: unknown
completed: 2026-05-02
---

# Phase 1: CLI Foundation Summary

**Structured JSON error path and CLI contract tests for the `dem` command framework**

## Performance

- **Duration:** unknown
- **Started:** 2026-05-02
- **Completed:** 2026-05-02
- **Tasks:** 3 completed
- **Files modified:** 5 created/modified

## Accomplishments

- Added `internal/output.WriteErrorJSON` with stable JSON error shape.
- Centralized `cli.Execute()` so command failures write JSON to stderr and return non-zero exit codes.
- Added tests for version output, help behavior, unknown command behavior, and future subcommand registration.
- Confirmed via source search that `os.Exit` appears only in `cmd/dem/main.go`.

## Task Commits

Each task was committed as part of the phase implementation commit:

1. **Task 1: Implement structured error JSON helper** - `103e18a`
2. **Task 2: Centralize CLI execution and error handling** - `103e18a`
3. **Task 3: Add CLI contract and extension tests** - `103e18a`

## Files Created/Modified

- `internal/output/error.go` - Structured JSON error envelope helper.
- `internal/output/error_test.go` - Error envelope tests.
- `internal/cli/root.go` - Centralized `Execute()` and root command setup.
- `internal/cli/root_test.go` - CLI contract tests.
- `cmd/dem/main.go` - Entry point using `os.Exit(cli.Execute())`.

## Decisions Made

- Used `command_error` as the initial generic command failure code.
- Kept command tests focused on injected buffers and direct Cobra execution, avoiding real process exits.

## Deviations from Plan

None - plan executed exactly as written.

---

**Total deviations:** 0 auto-fixed.
**Impact on plan:** No scope changes.

## Issues Encountered

- Go tooling is unavailable in this environment, so `go test ./...`, `go run ./cmd/dem version`, `go run ./cmd/dem --help`, and `go run ./cmd/dem does-not-exist` could not run.
- `go.sum` now contains Cobra and transitive dependency checksums, but runtime verification still could not be performed without Go tooling on PATH.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The CLI foundation source and tests are in place for Phase 2 provider work.
- Before marking Phase 1 fully verified, install/enable Go and run:
  - `go mod tidy`
  - `gofmt -w cmd/dem/main.go internal/cli/root.go internal/cli/version.go internal/cli/root_test.go internal/output/json.go internal/output/json_test.go internal/output/error.go internal/output/error_test.go`
  - `go test ./...`
  - `go run ./cmd/dem version`
  - `go run ./cmd/dem --help`
  - `go run ./cmd/dem does-not-exist`

## Self-Check: FAILED

Implementation files exist and content checks passed, but runtime verification is blocked because `go` and `gofmt` are not installed in this environment.

---
*Phase: 01-cli-foundation*
*Completed: 2026-05-02*

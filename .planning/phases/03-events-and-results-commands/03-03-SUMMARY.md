---
phase: 03-events-and-results-commands
plan: 03
subsystem: provider, cli
tags:
  - results-provider
  - results-command
  - cli
requires:
  - 03-02 (EventsProvider + events command)
  - 02-02 (domain models, parsers, errors)
provides:
  - ResultsProvider interface and implementation
  - results CLI command
affects:
  - internal/cli/root.go (command registration)
tech-stack:
  added: []
  patterns:
    - "ResultsProvider mirrors EventsProvider: NewResultsProvider + WithResultsClient/WithResultsURLs"
    - "mapResultsError mirrors mapEventsError: type switch on ProviderError/ParseError"
key-files:
  created:
    - internal/provider/results.go
    - internal/provider/results_test.go
    - internal/cli/results.go
    - internal/cli/results_test.go
  modified:
    - internal/cli/root.go
decisions:
  - "Follow events.go error handling pattern (write to stderr + return original err) over plan template"
  - "ResultsProvider returns interface (not concrete pointer) matching EventsProvider pattern"
  - "--limit default 0 means no truncation, consistent with events command"
metrics:
  duration: "4 minutes"
  completed_date: "2026-05-02"
---

# Phase 3 Plan 3: ResultsProvider + Results CLI Command Summary

**One-liner:** ResultsProvider interface/implementation with limit truncation, results CLI command with `--limit` flag, error type mapping to JSON envelopes, and root CLI wiring.

## Overview

This plan adds `dem results --limit N` functionality mirroring the existing events command. The ResultsProvider wraps Client.Fetch + ParseResults, truncates by limit, and passes typed errors through unmodified. The CLI command validates `--limit` before network access, maps provider/parser errors to JSON error envelopes on stderr, and emits `{data, meta}` JSON on stdout.

## Tasks Completed

| Task | Name                                             | Commit | Files                                                    |
| ---- | ------------------------------------------------ | ------ | -------------------------------------------------------- |
| 1    | Create ResultsProvider with limit truncation      | 9838781 | `internal/provider/results.go`, `internal/provider/results_test.go` |
| 2    | Create results CLI command, wire into root, test | 76d90f9 | `internal/cli/results.go`, `internal/cli/results_test.go`, `internal/cli/root.go` |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed RunE error return pattern in plan template**

- **Found during:** Task 2
- **Issue:** The plan template for `results.go` returned `mapResultsError(errOut, err)` directly from RunE. `mapResultsError` returns the result of `output.WriteErrorJSON`, which returns `nil` on success (JSON encoding writes to io.Writer without error). This would cause RunE to return nil, leading to exit code 0 even when an error occurred.
- **Fix:** Changed to match the established events.go pattern: `_ = mapResultsError(errOut, err)` (write JSON to stderr) then `return err` (propagate original error for non-zero exit code).
- **Files modified:** `internal/cli/results.go`
- **Commit:** 76d90f9

**2. [Rule 1 - Bug] Fixed validation error return pattern in plan template**

- **Found during:** Task 2
- **Issue:** The plan template returned `output.WriteErrorJSON(...)` directly from RunE for validation errors. Same issue -- `WriteErrorJSON` returns nil on success, causing RunE to return nil and Cobra to exit 0 despite the validation error.
- **Fix:** Changed to match events.go pattern: `_ = output.WriteErrorJSON(...)` then `return fmt.Errorf(...)`.
- **Files modified:** `internal/cli/results.go`
- **Commit:** 76d90f9

### No Other Deviations

All other aspects of the plan were executed exactly as written.

## Threat Surface Scan

No new security-relevant surface introduced beyond what was modeled in the plan's threat model. The provider passes errors through unchanged (no new information disclosure). The command validates `--limit >= 0` before network access. Tests use fake transports (no real HTTP).

## Test Results

```
?   	internal/cli		PASS (17 tests)
?   	internal/provider	PASS (7 tests: 4 events + 3 results)
?   	internal/hltv/parser	PASS (cached)
?   	internal/output		PASS (cached)
```

**Note:** `go test ./internal/hltv` fails due to pre-existing `TestFetchSendsUserAgent` using `httptest.NewServer` (sandbox blocks `ListenAndServe`). This is documented in STATE.md and is not caused by this plan.

## Verification Checklist

- [x] `go test ./internal/provider` exits 0
- [x] `go test ./internal/cli` exits 0
- [x] No `httptest.NewServer` in results test files (0 matches)
- [x] No `os.Stdout`/`os.Stderr` in `internal/cli/results.go` (0 matches)
- [x] `provider.NewResultsProvider` appears exactly once in `internal/cli/root.go`
- [x] `internal/provider/results.go` defines `type ResultsProvider interface`
- [x] `internal/provider/results.go` defines `func NewResultsProvider`
- [x] `internal/provider/results_test.go` uses `roundTripFunc`
- [x] `internal/cli/results.go` defines `func newResultsCommand`
- [x] `internal/cli/results.go` uses `--limit` flag
- [x] `internal/cli/results_test.go` defines `type fakeResultsProvider struct`

## Self-Check: PASSED

All created files exist, all commits exist, all acceptance criteria verified.

- `internal/provider/results.go` -- verified exists
- `internal/provider/results_test.go` -- verified exists
- `internal/cli/results.go` -- verified exists
- `internal/cli/results_test.go` -- verified exists
- `internal/cli/root.go` -- verified modified (3 commands registered)
- Commit 9838781 -- verified exists
- Commit 76d90f9 -- verified exists

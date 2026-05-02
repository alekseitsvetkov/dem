---
phase: 04-demo-link-lookup
plan: 02
type: execute
subsystem: demo-command
tags: [provider, cli, demo-link, json-output, validation]
depends_on: ["04-01"]
requires: [parser.ParseDemoLink, hltv.URLs.MatchURL, domain.DemoLink]
provides: [DemoProvider interface, demo CLI command, mapDemoError]
affects: [internal/cli/root.go]
tech-stack:
  added: []
  patterns: [functional-options, unavailable-data-as-success, validation-before-network, type-switch-error-mapping]
key-files:
  created:
    - internal/provider/demo.go
    - internal/provider/demo_test.go
    - internal/cli/demo.go
    - internal/cli/demo_test.go
  modified:
    - internal/cli/root.go
decisions:
  - "D-01: DemoProvider layer wrapping Client.Fetch + parser.ParseDemoLink"
  - "D-02: Functional options constructor (WithDemoClient, WithDemoURLs)"
  - "D-03: Unavailable data -> success with partial DemoLink (DemoURL empty/omitted)"
  - "D-04: Scripts detect availability by checking data.demo_url key in JSON response"
  - "D-05: match-id validated as strictly numeric before any network access"
  - "D-06: strconv.Atoi + positive integer check (err != nil || matchID <= 0)"
  - "D-07: dem demo <match-id> -- single positional arg, zero flags (DisableFlagParsing: true)"
metrics:
  duration: 12min
  completed_date: 2026-05-03
---

# Phase 4 Plan 02: DemoProvider and CLI Command Summary

**One-liner:** DemoProvider with unavailable-data-as-success handling and `dem demo <match-id>` CLI command with strict input validation, JSON output, and complete error mapping.

## Plan Execution

- **Objective:** Create the DemoProvider layer and the `dem demo <match-id>` CLI command with validation, JSON output, and unavailable-demo handling.
- **Tasks:** 2/2 completed

### Task 1: Create DemoProvider with unavailable-data handling

**Commit:** `db6a00c`

Created `internal/provider/demo.go` and `internal/provider/demo_test.go`. The DemoProvider follows the exact pattern established by EventsProvider and ResultsProvider:

- `DemoProvider` interface with `GetDemo(ctx context.Context, matchID int) (domain.DemoLink, error)`
- `demoProvider` struct with `client *hltv.Client` and `urls hltv.URLs`
- Functional options: `WithDemoClient`, `WithDemoURLs`
- `NewDemoProvider` constructor returning the `DemoProvider` interface
- `GetDemo` calls `client.Fetch` then `parser.ParseDemoLink`, catching `unavailable_data` ParseError and converting to success with partial DemoLink

Three tests cover:
- **Success** — fixture with `[data-demo-link]` attribute resolves to absolute URL
- **Unavailable** — fixture without demo returns nil error, empty DemoURL
- **NetworkError** — transport error returns `*hltv.ProviderError` with `network_error` code

### Task 2: Create demo CLI command with validation and tests, wire into root.go

**Commit:** `011f14b`

Created `internal/cli/demo.go`, `internal/cli/demo_test.go`, modified `internal/cli/root.go`.

The CLI command follows the exact pattern from `newEventsCommand` and `newResultsCommand`:

- `newDemoCommand(out, errOut, p)` — constructor accepting `io.Writer` and `DemoProvider`
- `DisableFlagParsing: true` — zero-flag command (D-07)
- `cobra.ExactArgs(1)` — exactly one positional argument
- `strconv.Atoi` + `matchID <= 0` validation before network access (D-05, D-06)
- `output.WriteJSON` for success, `output.WriteErrorJSON` for errors
- `mapDemoError` type switch maps `*hltv.ProviderError` and `*parser.ParseError` to JSON error envelopes

Seven tests cover:
- **Success** — valid match ID with demo (json has `match_id` and `demo_url`)
- **Unavailable** — valid match ID without demo (json has `match_id`, no `demo_url`)
- **ValidationError_NonNumeric** — `"abc"` → stderr has `validation_error`
- **ValidationError_Zero** — `"0"` → stderr has `validation_error`
- **ValidationError_Negative** — `"-5"` → stderr has `validation_error`
- **ProviderError** — fake returns `*hltv.ProviderError{Code: network_error}` → stderr has `network_error`
- **ParseError** — fake returns `*parser.ParseError{Code: parse_error}` → stderr has `parse_error`

Root command wired via `root.AddCommand(newDemoCommand(out, errOut, provider.NewDemoProvider()))`.

## Deviations from Plan

None — plan executed exactly as written. All files created and modified match the plan specification. All 10 tests (3 provider + 7 CLI) pass on first run.

**Note:** The full binary build (`go build ./cmd/dem/`) could not complete due to sandbox restrictions on Go module resolution for the self-referencing module path. This is a pre-existing sandbox limitation documented in STATE.md, not a code issue. Package-level builds (`go build ./internal/cli/` and `go build ./internal/provider/`) succeed.

## Verification

- [x] All provider tests pass (3/3): `go test ./internal/provider/ -run TestGetDemo`
- [x] All CLI tests pass (7/7): `go test ./internal/cli/ -run TestDemoCommand`
- [x] Full test suite passes: `go test ./internal/cli/ ./internal/provider/ ./internal/hltv/parser/ ./internal/output/`
- [x] CLI package compiles: `go build ./internal/cli/`
- [x] Provider package compiles: `go build ./internal/provider/`
- [x] Demo command registered in root.go: `grep "newDemoCommand" internal/cli/root.go`
- [x] No stale `a.demo-link` selectors anywhere in codebase
- [x] Unavailable data returns success: `grep "ErrorCodeUnavailableData" internal/provider/demo.go` shows `return link, nil`
- [x] Validation checks before network: `strconv.Atoi` check in `demo.go` precedes `GetDemo` call
- [x] No live network calls in tests

## Known Stubs

None. All data paths are fully wired — the DemoProvider calls real parser functions, and the CLI command maps real error types to JSON envelopes.

## Threat Flags

None. All threat surface is covered by the plan's threat model:
- T-04-04 (Tampering): Mitigated by `strconv.Atoi` ensuring only integers reach `MatchURL`
- T-04-05 (DoS): Accepted — 15-second timeout in `Client.Fetch`
- T-04-06 (Info Disclosure): Accepted — CLI outputs public HLTV data as JSON

## Self-Check: PASSED

- [x] `internal/provider/demo.go` exists: found
- [x] `internal/provider/demo_test.go` exists: found
- [x] `internal/cli/demo.go` exists: found
- [x] `internal/cli/demo_test.go` exists: found
- [x] `internal/cli/root.go` modified with newDemoCommand: found
- [x] Commit `db6a00c` (Task 1) exists in git log
- [x] Commit `011f14b` (Task 2) exists in git log

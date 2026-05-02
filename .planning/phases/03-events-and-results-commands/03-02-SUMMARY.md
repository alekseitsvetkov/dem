---
phase: 03-events-and-results-commands
plan: 02
subsystem: cli-provider
tags: [go, cobra, cli, events-provider, tier-filtering, testing]

requires:
  - phase: 02-hltv-provider-infrastructure
    provides: HLTV HTTP client, URL helpers, provider error taxonomy
  - phase: 03-01-domain-models-parsers
    provides: Event model with Tier field, ParseEvents parser, fixture HTML with tier data
provides:
  - EventsProvider interface (GetEvents(ctx, tier, limit)) with option-pattern constructor
  - provider-level tier filtering (case-insensitive EqualFold) and limit truncation
  - `dem events` CLI command with --tier (string) and --limit (int) flags
  - JSON error envelope mapping from ProviderError/ParseError to WriteErrorJSON
  - Fake RoundTripper provider tests (no httptest.NewServer)
  - In-memory fake provider for command tests (no fixtures, no HTTP)
affects: [phase-03-results, phase-04-demo-lookup]

tech-stack:
  added: []
  patterns:
    - "Provider middleware: EventsProvider interface wrapping Client.Fetch + parser.ParseEvents + tier filter + limit truncation"
    - "Constructor options pattern (WithEventsClient, WithEventsURLs) matching Phase 2 hltv.WithHTTPClient precedent"
    - "Command error mapping: file-local mapEventsError type-switches on *hltv.ProviderError and *parser.ParseError"

key-files:
  created:
    - internal/provider/events.go
    - internal/provider/events_test.go
    - internal/cli/events.go
    - internal/cli/events_test.go
  modified:
    - internal/cli/root.go

key-decisions:
  - "NewEventsProvider returns the EventsProvider interface (not a concrete pointer) — callers depend on interface only"
  - "mapEventsError uses type switch (not common error interface) since ProviderError and ParseError have distinct Code/Details structures"
  - "Errors are written to stderr via WriteErrorJSON AND returned as non-nil from RunE — enables both structured stderr output and non-zero exit code propagation through Cobra"
  - "--tier is optional (default ''); empty means no filtering at provider level"
  - "--limit defaults to 0 (no limit); provider truncates only when limit > 0 && limit < len(events)"

patterns-established:
  - "Command constructor accepts (out, errOut io.Writer, p provider.EventsProvider) to enable dependency injection in tests"
  - "Flag validation at top of RunE before any provider call (D-09 contract)"
  - "Nil event slice coerced to empty slice for JSON [] not null"
  - "Command tests construct a bare Cobra command (not NewRootCommand) to inject fake providers"

requirements-completed:
  - EVNT-01
  - EVNT-03

duration: 6min
completed: 2026-05-02
---

# Phase 3: Events and Results Commands - Plan 02 Summary

**EventsProvider with tier filtering and limit truncation, `dem events --tier --limit` CLI command wired into root command**

## Performance

- **Duration:** 6 min
- **Started:** 2026-05-02T18:02:00Z
- **Completed:** 2026-05-02T18:08:00Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- Created `EventsProvider` interface with `GetEvents(ctx, tier, limit ([]domain.Event, error)` wrapping `Client.Fetch` + `parser.ParseEvents` + tier filter + limit truncation
- Implemented case-insensitive tier filtering (`strings.EqualFold` with `strings.TrimSpace` on both sides)
- Provider errors pass through unchanged (no remapping at provider boundary per D-03/D-08)
- Constructor follows option pattern (`WithEventsClient`, `WithEventsURLs`) matching Phase 2 conventions
- Created `dem events --tier <string> --limit <int>` CLI command with validation before network access (D-09)
- Error mapping type-switches on `*hltv.ProviderError` and `*parser.ParseError`, writes structured JSON envelopes to stderr
- Wired events command into `NewRootCommand` with default `provider.NewEventsProvider()`
- 4 provider tests pass using fake RoundTripper transport and parser fixture HTML
- 7 CLI command tests pass using in-memory fake provider (no HTTP, no fixtures)

## Task Commits

Each task was committed atomically:

1. **Task 1: Create EventsProvider interface and implementation with tier filtering and limit truncation** - `0475ead` (feat)
2. **Task 2: Create events CLI command with flag validation and error mapping, wire into root command** - `a02fcc3` (feat)

## Files Created/Modified

- `internal/provider/events.go` - EventsProvider interface, eventsProvider struct, NewEventsProvider constructor, GetEvents implementation, filterByTier helper, option types
- `internal/provider/events_test.go` - roundTripFunc transport adapter, 4 tests: success, tier filter, limit, network error
- `internal/cli/events.go` - newEventsCommand constructor, mapEventsError helper with type switch
- `internal/cli/events_test.go` - fakeEventsProvider in-memory stub, 7 tests: success, tier flag, limit flag, validation error, provider error, parse error, nil events
- `internal/cli/root.go` - Added provider import and events command registration in NewRootCommand

## Decisions Made

- **Error handling pattern:** `mapEventsError` writes JSON error to stderr AND the return value from RunE is non-nil. This ensures Cobra propagates the non-zero exit code while the JSON error envelope is already on stderr. The centralized `Execute()` function adds a `command_error` envelope when RunE returns non-nil — acceptable double-write for now.
- **Optional --tier:** When omitted (default ""), all events are returned unfiltered. The plan's research recommendation (Open Question 2) was followed: make `--tier` optional for flexibility.
- **Default limit behavior:** `--limit` defaults to 0, which means no truncation. The provider truncates only when `limit > 0 && limit < len(events)`. This follows the plan research recommendation (Open Question 3).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Command error return value causes nil exit code**
- **Found during:** Task 2 (Running events command tests)
- **Issue:** The plan's RunE code `return output.WriteErrorJSON(...)` writes JSON to stderr but returns nil (success from `json.Encode`). This means Cobra's `Execute()` returns nil, resulting in exit code 0 for error cases. Three tests failed: validation_error, provider_error, parse_error.
- **Fix:** Changed RunE to write JSON to stderr via `_ = WriteErrorJSON(...)` then `return fmt.Errorf(...)` or `return err` for the original error. This propagates a non-nil error to Cobra for non-zero exit code while keeping structured JSON on stderr.
- **Files modified:** internal/cli/events.go
- **Verification:** All 7 command tests pass, including non-zero exit code checks
- **Committed in:** a02fcc3 (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Necessary fix for correct exit code behavior. Command errors now produce both structured JSON on stderr AND non-zero exit code.

## Issues Encountered

- Pre-existing `internal/hltv` tests use `httptest.NewServer` which fails in the sandbox. This is documented in STATE.md as a known blocker and does not affect this plan's deliverables. `go test ./internal/cli`, `go test ./internal/provider`, `go test ./internal/hltv/parser`, and `go test ./internal/output` all pass cleanly.

## Threat Surface Scan

No new threat surface beyond the plan's existing threat model. All mitigations from the threat model were implemented:
- Validation before network access (D-09)
- ProviderError/ParseError type switch extracts only `.Code` and `.Details()` — no raw bodies
- `filterByTier` uses `strings.EqualFold` for case-insensitive comparison
- Fake RoundTripper in tests (no default transport leak)
- `limit >= 0` validation before provider call; provider checks `limit > 0` before slicing

## Known Stubs

None found — all implementations are fully wired with real data flow or test fakes.

## Next Phase Readiness

- `EventsProvider` pattern is established for reuse by `ResultsProvider` in plan 03-03
- Error mapping pattern (`mapEventsError`) can be adapted for `mapResultsError`
- CLI command constructor and test pattern (`newEventsCommand + fake provider`) ready for `newResultsCommand`
- Root command wiring pattern set for `dem results` addition

## Self-Check: PASSED

- `go test ./internal/provider` — PASS (4/4)
- `go test ./internal/cli` — PASS (11/11)
- `go test ./internal/hltv/parser` — PASS (cached)
- `go test ./internal/output` — PASS (cached)
- No `httptest.NewServer` in provider or CLI test files
- No `os.Stdout` or `os.Stderr` in events.go (injected writers only)
- `provider.NewEventsProvider` referenced exactly once in root.go
- 2 atomic commits with correct types (feat, feat)

---
*Phase: 03-events-and-results-commands*
*Completed: 2026-05-02*

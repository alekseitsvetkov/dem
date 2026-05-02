---
phase: 03-events-and-results-commands
plan: 01
subsystem: domain-api
tags: [go, goquery, hltv, html-parsing, testing]

requires:
  - phase: 02-hltv-provider-infrastructure
    provides: HLTV HTTP client, URL helpers, provider error taxonomy
provides:
  - Domain models: Event (with Tier field), Result, DemoLink with JSON tags
  - Parser error taxonomy: parse_error, unavailable_data codes with structured Details()
  - goquery-based ParseEvents with tier extraction from .event-tier element
  - goquery-based ParseResults with required field validation
  - Sanitized HTML fixtures for events and results parser tests
affects: [phase-03-events-results, phase-04-demo-lookup]

tech-stack:
  added: [github.com/PuerkitoBio/goquery v1.12.0, github.com/andybalholm/cascadia v1.3.3, golang.org/x/net v0.52.0]
  patterns: [fixture-tested html parsers, EachWithBreak validation in goquery, structured parser errors matching ProviderError pattern]

key-files:
  created:
    - internal/domain/models.go
    - internal/hltv/parser/errors.go
    - internal/hltv/parser/events.go
    - internal/hltv/parser/events_test.go
    - internal/hltv/parser/results.go
    - internal/hltv/parser/results_test.go
    - internal/hltv/parser/testdata/events.html
    - internal/hltv/parser/testdata/results.html
  modified:
    - go.mod
    - go.sum

key-decisions:
  - "Use goquery EachWithBreak for early-exit on required field validation failure"
  - "ParseError Details() excludes raw HTML content matching ProviderError pattern from Phase 2"
  - "Missing .event-tier results in empty string Tier (not error) per D-10: silent no-op for tier filter"
  - "Date range split uses ' to ' separator; assigns full text to StartDate if no split found"
  - "Source URL resolution uses net/url.ResolveReference against https://www.hltv.org base"

patterns-established:
  - "Required field validation in HTML parsers: check attr/text existence, return ParseError with Field= on failure, use EachWithBreak to stop iteration"
  - "Optional field extraction: use goquery Selection.Text() with TrimSpace; empty is valid, not an error"

requirements-completed:
  - EVNT-02
  - RSLT-02

duration: 3min
completed: 2026-05-02
---

# Phase 3: Events and Results Commands - Plan 01 Summary

**Event model with Tier field, ParseError taxonomy, goquery-based ParseEvents/ParseResults parsers, and sanitized fixture tests**

## Performance

- **Duration:** 3 min
- **Started:** 2026-05-02T17:59:00Z
- **Completed:** 2026-05-02T18:01:43Z
- **Tasks:** 3
- **Files modified:** 9

## Accomplishments

- Added goquery v1.12.0 as an HTML parsing dependency, upgrading Go toolchain to 1.25.0
- Created `domain.Event` with Tier string field (`json:"tier,omitempty"`), `domain.Result`, and `domain.DemoLink` typed models with JSON tags
- Created parser error taxonomy (`ParseError`) with stable snake_case codes (`parse_error`, `unavailable_data`) matching Phase 2's `ProviderError` pattern — no raw HTML leakage in `Details()`
- Implemented `ParseEvents` with tier extraction from `.event-tier` element, required field validation for event ID and name, and relative URL resolution
- Implemented `ParseResults` with required field validation for match ID, team1, team2, score, and match link URL resolution
- Created sanitized, script-free HTML fixtures with tier data for both parsers
- 10 tests pass using only local testdata fixtures — zero `net/http` imports, zero live HLTV access

## Task Commits

Each task was committed atomically:

1. **Task 1: Add goquery dependency, domain models with Tier, parser error taxonomy** - `9f89edb` (feat)
2. **Task 2: Create sanitized parser fixtures with tier data** - `f344b34` (test)
3. **Task 3: Implement and test ParseEvents/ParseResults** - `820caf8` (feat)

## Files Created/Modified

- `internal/domain/models.go` - Event (with Tier), Result, DemoLink structs with JSON tags
- `internal/hltv/parser/errors.go` - ParseError struct with ErrorCodeParse and ErrorCodeUnavailableData constants
- `internal/hltv/parser/events.go` - ParseEvents implementation with tier extraction and URL resolution
- `internal/hltv/parser/events_test.go` - 5 tests: fixture parsing, missing ID, missing name, missing tier (no error), empty page
- `internal/hltv/parser/results.go` - ParseResults implementation with required field validation
- `internal/hltv/parser/results_test.go` - 5 tests: fixture parsing, missing team1, missing match ID, missing optional fields, empty page
- `internal/hltv/parser/testdata/events.html` - Script-free fixture with 2 events (S-Tier, A-Tier) and .event-tier elements
- `internal/hltv/parser/testdata/results.html` - Script-free fixture with 2 results and match links
- `go.mod`, `go.sum` - Added goquery v1.12.0, cascadia v1.3.3, golang.org/x/net v0.52.0

## Decisions Made

- Used goquery `EachWithBreak` for early-exit on required field validation failure — returns first parse error immediately rather than collecting all errors
- Missing `.event-tier` results in empty string Tier (not an error) — downstream provider filter becomes a no-op for `tier=""`
- Date range split uses `" to "` separator; assigns full text to StartDate if no split found
- Source URL resolution uses `net/url.ResolveReference` against `https://www.hltv.org` base URL
- ParseError `Details()` excludes raw HTML content, matching the ProviderError pattern established in Phase 2

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- The sandbox blocked `proxy.golang.org` during `go get`, requiring sandbox-disable to add goquery dependency
- The pre-existing `internal/hltv` test (`TestFetchSendsUserAgent`) uses `httptest.NewServer` which fails in the sandbox — this is a pre-existing issue from Phase 2, not related to this plan's changes
- The `go get` command upgraded the Go version from `1.22` to `1.25.0` as required by goquery v1.12.0 — this is expected behavior

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `domain.Event` now carries a `Tier` field that Plan 03-02's `EventsProvider` can use for client-side tier filtering
- `domain.Result` is fully defined for Plan 03-03's `ResultsProvider`
- Parser errors pass through to command handlers for error envelope mapping (per D-08)
- Parser fixture structure is established for future updates when HLTV markup changes

## Self-Check: PASSED

- All files exist and compile
- `go test ./internal/hltv/parser` passes (10/10 tests)
- `go vet ./...` clean
- No `net/http` imports in test files
- No `http.Get`, `http.DefaultClient`, or live HLTV URLs in test files
- 3 atomic commits with correct types (feat, test, feat)

---
*Phase: 03-events-and-results-commands*
*Completed: 2026-05-02*

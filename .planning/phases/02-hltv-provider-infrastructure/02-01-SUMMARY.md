---
phase: 02-hltv-provider-infrastructure
plan: 01
subsystem: provider
tags: [go, net-http, hltv, testing]

requires:
  - phase: 01-cli-foundation
    provides: CLI/output conventions and Go module foundation
provides:
  - Injectable HLTV HTTP client with timeout and user-agent defaults
  - Centralized HLTV public URL helpers
  - Provider error taxonomy for network and HTTP status failures
affects: [phase-03-events-results, phase-04-demo-lookup]

tech-stack:
  added: []
  patterns: [injectable-http-client, typed-provider-errors, httptest]

key-files:
  created:
    - internal/hltv/errors.go
    - internal/hltv/errors_test.go
    - internal/hltv/urls.go
    - internal/hltv/urls_test.go
    - internal/hltv/client.go
    - internal/hltv/client_test.go
  modified: []

key-decisions:
  - "Use standard net/http with DefaultTimeout = 15 * time.Second."
  - "Use DefaultUserAgent = \"dem/dev\" and set User-Agent on every fetch."
  - "Return typed ProviderError values with stable snake_case codes."
  - "Keep URL construction centralized in internal/hltv."

patterns-established:
  - "Constructor options: NewClient accepts WithHTTPClient and WithUserAgent for tests and future commands."
  - "Provider errors expose Details() maps without raw response bodies."

requirements-completed: [HLTV-01, HLTV-02]

duration: 8min
completed: 2026-05-02
---

# Phase 2: HLTV Provider Infrastructure Summary

**Injectable HLTV HTTP client with timeout/user-agent defaults, URL helpers, and typed provider errors**

## Performance

- **Duration:** 8 min
- **Started:** 2026-05-02T16:12:00Z
- **Completed:** 2026-05-02T16:20:00Z
- **Tasks:** 3
- **Files modified:** 6

## Accomplishments

- Added `internal/hltv.ProviderError` with `network_error` and `http_error` categories plus structured details.
- Added centralized URL helpers for events, results, and match pages with a configurable base URL.
- Added an injectable HTTP client that uses `DefaultTimeout = 15 * time.Second`, sends `User-Agent`, and is covered by `httptest`/fake-transport tests.

## Task Commits

1. **Task 1: Add provider error taxonomy** - `54d1b4e` (feat)
2. **Task 2: Add HLTV URL construction helpers** - `a8c230e` (feat)
3. **Task 3: Implement injectable HTTP page fetcher** - `e066688` (feat)

## Files Created/Modified

- `internal/hltv/errors.go` - Provider error taxonomy and structured details.
- `internal/hltv/errors_test.go` - Provider error detail and unwrap tests.
- `internal/hltv/urls.go` - Public HLTV URL helpers.
- `internal/hltv/urls_test.go` - URL construction tests.
- `internal/hltv/client.go` - Injectable HTTP page fetcher.
- `internal/hltv/client_test.go` - User-agent, fake transport, HTTP status, and network error tests.

## Decisions Made

- Kept retry behavior out of Phase 2 to preserve polite, predictable single-request fetching.
- Used constructor options rather than package-level mutable state so tests and later commands can inject behavior safely.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- The sandbox blocked the default Go build cache under `~/Library/Caches/go-build`; tests passed after rerunning `go test` with approved cache access.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Wave 2 can build parser/domain packages on top of this provider foundation. The HTTP client and URL helpers are available for Phase 3 and Phase 4 command wiring.

## Self-Check: PASSED

- `go test ./internal/hltv` passed.
- `go test ./...` passed.
- `internal/hltv/client_test.go` uses `httptest.NewServer`.
- `internal/hltv/client_test.go` does not call `https://www.hltv.org`.
- `internal/hltv/client.go` contains no retry loop or goroutine-based crawler behavior.

---
*Phase: 02-hltv-provider-infrastructure*
*Completed: 2026-05-02*

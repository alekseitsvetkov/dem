# Phase 2: HLTV Provider Infrastructure - Research

**Date:** 2026-05-02
**Status:** Complete

## Research Question

What do we need to know to plan Phase 2 well: a reusable Go HLTV provider layer with polite HTTP behavior, injectable tests, parser boundaries, fixtures, and distinguishable errors?

## Findings

### Go HTTP Provider Shape

- Use standard `net/http` with a configured `http.Client{Timeout: ...}` for phase scope. The project has no need for a heavier HTTP dependency yet.
- Keep transport injectable. A constructor accepting `*http.Client`, base URL, timeout, and user-agent supports `httptest.Server`, fake `RoundTripper`, and fixture-backed tests.
- Keep URL construction centralized in `internal/hltv`. Commands and parsers should not hand-build HLTV URLs.
- Avoid package-level global clients. Tests become order-dependent when client configuration is mutable global state.

### Polite Fetching Defaults

- Default timeout should be concrete and bounded. Phase context recommends 15 seconds.
- Every request should include a `User-Agent`, with a default like `dem/dev`.
- Do not add automatic retries yet. This avoids accidental aggressive scraping and keeps failure semantics clear for scripts.
- Tests should prove timeout/user-agent/base URL behavior without live network calls.

### Error Boundaries

- Provider/parser errors should be typed so command handlers can map them into Phase 1's JSON error envelope.
- Stable machine-facing categories for Phase 2:
  - `network_error` for request construction, transport, timeout, DNS, and connection failures.
  - `http_error` for non-2xx HTTP responses.
  - `parse_error` for malformed or unexpected HTML.
  - `unavailable_data` for valid pages that lack requested public data.
- Error values should expose structured details such as URL, status code, parser area, and missing field without embedding raw HTML bodies.

### Parser Package and Fixtures

- Put selectors and HTML interpretation in `internal/hltv/parser`, not in `internal/cli` or the provider client.
- Use `goquery` for HTML traversal once parser implementation begins; it matches the project stack and keeps selector code localized.
- Store parser fixtures in `internal/hltv/parser/testdata`. Go tooling treats `testdata` as test-only and keeps fixtures near parser tests.
- Prefer trimmed, sanitized HLTV-like snapshots. Fully synthetic snippets can miss real class nesting; full raw pages add noise and fragility.
- Phase 2 should establish parser functions and fixture coverage for all v1 page types:
  - events listing
  - results listing
  - match page with demo link
  - match page without demo link

### Domain Models

- Domain models should live under `internal/domain` with JSON tags, even before commands expose them. This lets parser tests assert typed fields that later commands can emit directly.
- Stable available fields from requirements:
  - Event: ID, name, date range, location, source URL.
  - Result: match ID, teams, score, event, date, format, source URL.
  - Demo lookup: match ID, match source URL, demo URL when available.
- Parser functions should return typed values plus typed errors, not JSON payloads.

## Recommended Plan Split

### 02-01: HTTP Client and Provider Interfaces

Build `internal/hltv` with configuration, URL helpers, injectable HTTP client/fetcher behavior, and typed provider errors. Verify using `httptest` and fake transports.

### 02-02: Parser Package, Domain Models, and Fixtures

Add `goquery`, `internal/domain` models, parser functions for the three v1 page types, sanitized fixtures, and parser tests proving success plus missing-data behavior.

## Validation Architecture

Plan verification should cover four independent signals:

- `go test ./internal/hltv` proves provider HTTP behavior, configured timeout/user-agent, fake transports, non-live tests, and error taxonomy.
- `go test ./internal/hltv/parser` proves parser fixture coverage for events, results, match-with-demo, and match-without-demo.
- `go test ./...` proves new packages integrate with the existing CLI/output foundation.
- Text checks confirm no command handler imports `goquery` or owns HLTV selectors.

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| HLTV markup changes after fixtures are written. | Keep fixtures trimmed and selector code isolated in `internal/hltv/parser`. |
| Tests accidentally hit the network. | Design provider around injectable clients and parser tests around checked-in fixtures only. |
| Errors collapse into generic command failures. | Return typed provider/parser errors with stable category codes. |
| Phase 2 overbuilds final commands. | Stop at reusable provider/parser/domain boundaries; Phase 3 and Phase 4 add commands. |

## Research Complete

Phase 2 can be planned as two sequential plans: provider transport first, parser/domain fixtures second.

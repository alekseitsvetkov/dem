# Phase 2: HLTV Provider Infrastructure - Context

**Gathered:** 2026-05-02
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase delivers the reusable HLTV access layer that later `dem events`, `dem results`, and `dem demo <match-id>` commands will use. It should add polite HTTP fetching, provider interfaces, parser package boundaries, parser fixtures, and a distinguishable error taxonomy. It does not add the final user-facing events, results, or demo commands; those remain Phase 3 and Phase 4 work.

</domain>

<decisions>
## Implementation Decisions

### HTTP Request Policy
- **D-01:** Use standard `net/http` for HLTV fetching unless planning discovers a strong reason to add another dependency.
- **D-02:** Configure a default request timeout, recommended at 15 seconds, and keep it injectable so tests and future commands can override it.
- **D-03:** Send a clear `User-Agent` on every HLTV request. The default should identify the CLI, for example `dem/dev`, with build/version metadata when available.
- **D-04:** Do not add automatic retries in v1 provider infrastructure. Avoid accidental aggressive scraping; callers can explicitly retry later if the project adds a careful policy.
- **D-05:** Keep request volume conservative. Phase 2 should make single-page fetches testable and reliable, not introduce crawling, concurrency, or background refresh behavior.

### Provider Boundary
- **D-06:** Create an `internal/hltv` package for fetcher/client/provider interfaces and public-page URL construction.
- **D-07:** Provider code should depend on small interfaces so tests can use `httptest`, fake transports, or fixture-backed fetchers without live HLTV access.
- **D-08:** Keep command handlers independent from selectors, raw HTML parsing, and HTTP details.
- **D-09:** Prefer a boundary where the HTTP layer fetches page content and parser packages convert content into typed domain models. Typed provider methods can compose these pieces as Phase 3 and Phase 4 add commands.
- **D-10:** Avoid global clients and package-level mutable state. Constructors should accept configuration such as base URL, timeout, user-agent, and HTTP client/transport.

### Fixture Strategy
- **D-11:** Parser tests must not hit live HLTV. Use checked-in fixtures under parser `testdata`.
- **D-12:** Prefer sanitized real-structure HLTV HTML snapshots over purely synthetic HTML when possible, because selector regressions are the main risk.
- **D-13:** Fixtures should be trimmed to the relevant page regions and stripped of unnecessary scripts, tracking markup, and bulky unrelated content.
- **D-14:** Cover all v1 page types in Phase 2 fixtures: events listing, results listing, and match page/demo-link structure.
- **D-15:** Use small focused fixtures for edge cases such as missing demo link, unexpected empty sections, and markup that parses but lacks required data.

### Error Taxonomy
- **D-16:** Keep error codes stable, snake_case, and compatible with the Phase 1 JSON error envelope.
- **D-17:** Distinguish at least these provider/parser error categories: `network_error`, `http_error`, `parse_error`, and `unavailable_data`.
- **D-18:** Preserve enough structured details for debugging and scripts without leaking excessive page content. Useful details include URL, status code, parser area, and missing field name.
- **D-19:** Treat "match exists but no demo is exposed" as unavailable data, not a generic parse failure. Phase 4 can specialize this into a demo-specific user-facing code if needed.
- **D-20:** Validation errors remain a command-layer concern and should happen before network access, continuing the Phase 1 contract.

### the agent's Discretion
- The user asked for best-practice defaults. Downstream agents may choose exact Go type names, constructor signatures, fixture filenames, and parser helper layout as long as they preserve the decisions above.
- If planning needs an HTML parser dependency, `goquery` is already the recommended stack choice from project conventions and should be added only when parser implementation requires it.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project Scope
- `.planning/PROJECT.md` - Project identity, constraints, top-level decisions, and v1 scope.
- `.planning/REQUIREMENTS.md` - Phase 2 requirements `HLTV-01` through `HLTV-03`.
- `.planning/ROADMAP.md` - Phase 2 boundary, success criteria, and plan split.
- `AGENTS.md` - Project conventions, architecture expectations, and workflow guidance.

### Prior Phase Decisions
- `.planning/phases/01-cli-foundation/01-CONTEXT.md` - Locked CLI command shape, JSON envelope, error envelope, and extensibility boundaries.
- `.planning/phases/01-cli-foundation/01-01-SUMMARY.md` - Root command, JSON output helper, and command registration implementation summary.
- `.planning/phases/01-cli-foundation/01-02-SUMMARY.md` - Structured error handling, help behavior, and CLI contract test summary.

### External Specs
- No external specs or ADRs were referenced during discussion. Requirements are fully captured in project planning docs and decisions above.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/output.Response[T]` and `internal/output.WriteJSON` provide the success JSON envelope that future provider-backed commands should use.
- `internal/output.WriteErrorJSON` provides the structured stderr error envelope that provider/parser errors should map into at command boundaries.
- `internal/cli.NewRootCommand` wires Cobra commands with injectable stdout/stderr writers and is the integration point for future `events`, `results`, and `demo` commands.

### Established Patterns
- The CLI currently uses Cobra with `SilenceUsage` and `SilenceErrors`, allowing command errors to be encoded as JSON rather than mixed with human-oriented usage output.
- Tests already exercise command behavior using injected writers rather than process globals.
- Application code is currently minimal, so Phase 2 can establish `internal/hltv`, `internal/hltv/parser`, and `internal/domain` without fighting existing provider patterns.

### Integration Points
- `cmd/dem/main.go` calls `cli.Execute`; provider-backed commands should eventually be registered through the root command setup rather than adding behavior in `main`.
- `internal/cli` should consume provider interfaces, not concrete HTML parser selectors.
- `internal/output` remains the boundary for JSON and error encoding; provider packages should return typed errors rather than writing output directly.

</code_context>

<specifics>
## Specific Ideas

- The provider layer should be intentionally boring and reliable: one request in, typed result or typed error out.
- Parser fixtures are the safety net against HLTV markup changes; avoid brittle command-level parsing.
- The user prefers best-practice defaults when uncertain.

</specifics>

<deferred>
## Deferred Ideas

- Automatic retries, caching, concurrent crawling, and rate limiting knobs are deferred until real usage requires them.
- User-facing `events`, `results`, and `demo <match-id>` commands remain Phase 3 and Phase 4 work.

</deferred>

---

*Phase: 2-HLTV Provider Infrastructure*
*Context gathered: 2026-05-02*

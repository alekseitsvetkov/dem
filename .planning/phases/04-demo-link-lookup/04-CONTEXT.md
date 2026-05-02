# Phase 4: Demo Link Lookup - Context

**Gathered:** 2026-05-02
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase delivers `dem demo <match-id>` — a Cobra command that takes a numeric HLTV match ID, fetches the match page, parses it for a demo download link, and returns the result as JSON. When a demo link exists, the response includes `demo_url`. When no demo is available, the response omits `demo_url` and returns success (exit code 0). This is the final v1 command and completes the CLI's public command surface.

</domain>

<decisions>
## Implementation Decisions

### Provider Wiring
- **D-01:** Add a DemoProvider layer wrapping `Client.Fetch(ctx, urls.MatchURL(matchID))` + `parser.ParseDemoLink(body, matchURL)` behind an injectable interface. Follows the Phase 3 EventsProvider/ResultsProvider pattern — commands depend on the interface, not on raw infrastructure.
- **D-02:** Provider constructor uses the functional-options pattern (`WithDemoClient`, `WithDemoURLs`) consistent with Phase 2 and Phase 3.

### Unavailable Demo Handling
- **D-03:** When `ParseDemoLink` returns `unavailable_data`, the DemoProvider catches this, extracts the partial `DemoLink` (with `MatchID` and `MatchURL` populated), and returns it as a success with `DemoURL` empty/omitted. The command writes it to stdout with exit code 0.
- **D-04:** Scripts detect demo availability by checking `data.demo_url` in the JSON response — present = demo available, absent/empty = no demo.

### Match ID Validation
- **D-05:** The positional `<match-id>` argument is validated as strictly numeric before any network access. Non-numeric input returns `validation_error` on stderr with exit code non-zero.
- **D-06:** Validation uses `strconv.Atoi` (or equivalent). Only positive integers are valid match IDs.

### Command Shape
- **D-07:** `dem demo <match-id>` takes only the positional match-id argument — no flags. The command is the simplest possible: one input, one JSON output.

### Prior Decisions (from earlier phases, still binding)
- JSON-only stdout with `{data, meta}` envelope (Phase 1 D-05, D-06, D-07).
- Structured error JSON on stderr with snake_case codes (Phase 1 D-09, D-10, D-11).
- Validation before network access (Phase 1 D-12).
- Commands use injectable writers, not globals (Phase 1 D-15).
- Provider middleware pattern — commands depend on provider interfaces (Phase 3 D-01).
- Phase 2 error codes map 1:1 to CLI envelope codes (Phase 3 D-08).
- Constructor options pattern for injectable dependencies (Phase 2/3 D-03).

### Claude's Discretion
- Exact provider method signature and constructor option names.
- DemoLink JSON shape when demo_url is absent (empty string vs field omitted).
- Test strategy details (fake RoundTripper transports, injected writers).
- Exact plan split — ROADMAP.md defines 2 plans: 04-01 (match page fetching/parsing for demo links) and 04-02 (demo command validation, JSON output, unavailable-demo tests), but the planner may adjust based on existing infrastructure.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project Scope
- `.planning/PROJECT.md` — Project identity, constraints, and locked key decisions.
- `.planning/REQUIREMENTS.md` — Phase 4 requirements `DEMO-01` through `DEMO-03`.
- `.planning/ROADMAP.md` — Phase 4 boundary, success criteria, and the 2-plan split (04-01, 04-02).
- `AGENTS.md` — Project conventions and architecture expectations.

### Prior Phase Context
- `.planning/phases/01-cli-foundation/01-CONTEXT.md` — CLI command shape, JSON envelope, error envelope.
- `.planning/phases/03-events-and-results-commands/03-CONTEXT.md` — Provider pattern, error mapping, command registration pattern.

### Prior Phase Implementation (for reuse)
- `.planning/phases/02-hltv-provider-infrastructure/02-02-SUMMARY.md` — ParseDemoLink, DemoLink model, and fixture coverage.
- `.planning/phases/03-events-and-results-commands/03-02-SUMMARY.md` — EventsProvider reference implementation.
- `.planning/phases/03-events-and-results-commands/03-03-SUMMARY.md` — ResultsProvider reference implementation.

### External References
- Cobra documentation — CLI command framework: https://github.com/spf13/cobra

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/hltv/parser/demo.go` — `ParseDemoLink(io.Reader, matchURL string) (domain.DemoLink, error)` — already implemented and fixture-tested with `match-with-demo.html` and `match-without-demo.html`.
- `internal/domain/models.go` — `DemoLink` struct with `MatchID`, `MatchURL`, `DemoURL` fields and JSON tags.
- `internal/hltv/urls.go` — `MatchURL(matchID int) string` — already exists.
- `internal/hltv/client.go` — `Client.Fetch(ctx, url)` — injectable HTTP client.
- `internal/provider/events.go` — `EventsProvider` interface + implementation — canonical reference for DemoProvider design.
- `internal/cli/events.go` — `newEventsCommand` — canonical reference for demo command structure.
- `internal/output/` — `WriteJSON[T]`, `WriteErrorJSON` — output helpers.

### Established Patterns
- Provider constructor with functional options: `NewEventsProvider(opts ...EventsProviderOption)`.
- Command constructor: `newXxxCommand(out io.Writer, errOut io.Writer, provider XxxProvider) *cobra.Command`.
- Error mapping via type switch on `*hltv.ProviderError` / `*parser.ParseError`.
- Fake `http.RoundTripper` for tests (no `httptest.NewServer` — sandbox constraint).

### Integration Points
- `internal/cli/root.go` — `NewRootCommand` wires subcommands via `root.AddCommand(...)`.
- `cmd/dem/main.go` — Calls `cli.Execute()` — no changes needed.

</code_context>

<specifics>
## Specific Ideas

- The `dem demo` command should be the simplest command in the CLI — one positional arg, one JSON output, zero flags.
- Unavailable demo is a normal outcome, not an error — HLTV doesn't always expose demo links, and users should be able to script around that without checking exit codes.
- The user consistently prefers best-practice defaults and pattern consistency across phases.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 4-Demo Link Lookup*
*Context gathered: 2026-05-02*

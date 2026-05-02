# Phase 3: Events and Results Commands - Context

**Gathered:** 2026-05-02
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase delivers `dem events --tier <tier>` and `dem results` Cobra commands that compose Phase 2's HTTP client and fixture-tested parsers behind injectable provider interfaces. Each command fetches the appropriate HLTV page, parses it into typed domain models, applies filtering/limiting at the provider layer, and writes the result as JSON to stdout. The phase does not add the `dem demo <match-id>` command — that remains Phase 4.

</domain>

<decisions>
## Implementation Decisions

### Command Wiring Architecture
- **D-01:** Add a provider middleware layer (`EventsProvider`, `ResultsProvider`) between Cobra commands and Phase 2 infrastructure. Commands depend on provider interfaces, not directly on `Client.Fetch` or parser functions.
- **D-02:** Each provider wraps `Client.Fetch` + a parser function into a single-call method that returns `([]domain.Event, error)` / `([]domain.Result, error)` plus filtering and limiting.
- **D-03:** Provider constructors accept injectable dependencies (`*hltv.Client`, `hltv.URLs`) following the Phase 2 option-pattern precedent. Providers pass through typed Phase 2 errors without remapping at the provider boundary.

### Tier 1 Filtering
- **D-04:** The `EventsProvider` receives the tier string and filters parsed events internally. The command handler does not perform filtering — it just passes the flag value.
- **D-05:** The `--tier` CLI flag is a string type (not int, not enum). This accommodates non-numeric tier labels HLTV may use (e.g., "S") without constraining v1 to a fixed set.

### Limit Flag Behavior
- **D-06:** Truncation is client-side — the full HLTV page is parsed, then the result slice is bounded. HLTV pages have a fixed size, so there is no server-side pagination to exploit.
- **D-07:** The provider interface receives the limit parameter and returns already-truncated data. The command handler passes the flag value through.

### Error Surface Mapping
- **D-08:** Phase 2 error codes map directly to CLI error envelope codes. Users see `network_error`, `http_error`, `parse_error`, and `unavailable_data` as machine-stable snake_case codes. The CLI error envelope wraps them with URL and status context via its `details` field.
- **D-09:** Validation errors (e.g., missing required flags) must fail before any network access, continuing the Phase 1 contract.

### Prior Decisions (from earlier phases, still binding)
- JSON-only stdout with `{data, meta}` envelope (Phase 1 D-05, D-06, D-07).
- Structured error JSON on stderr with snake_case codes (Phase 1 D-09, D-10, D-11).
- Validation before network access (Phase 1 D-12).
- Commands use injectable writers, not globals (Phase 1 D-15).
- Top-level v1 commands: `dem events`, `dem results`, `dem demo <match-id>` (Phase 1 D-02).
- `dem version` is the non-network JSON-contract proof command (Phase 1 D-16, D-17).

### Claude's Discretion
- Provider interface method signatures, constructor option names, and package layout.
- Default limit value when `--limit` is not provided by the user.
- Test strategy details (fake transports, table-driven tests, injected writers).
- Exact plan split across the 3 roadmap plans (03-01 domain/schemas, 03-02 events command, 03-03 results command).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project Scope
- `.planning/PROJECT.md` — Project identity, constraints, and locked key decisions.
- `.planning/REQUIREMENTS.md` — Phase 3 requirements `EVNT-01` through `EVNT-03` and `RSLT-01` through `RSLT-03`.
- `.planning/ROADMAP.md` — Phase 3 boundary, success criteria, and the 3-plan split (03-01, 03-02, 03-03).
- `AGENTS.md` — Project conventions and architecture expectations.

### Prior Phase Context
- `.planning/phases/01-cli-foundation/01-CONTEXT.md` — CLI command shape, JSON envelope, error envelope, and extensibility boundaries.
- `.planning/phases/02-hltv-provider-infrastructure/02-CONTEXT.md` — Provider/parser isolation, error taxonomy, fixture strategy, and polite HTTP policy.

### Prior Phase Implementation
- `.planning/phases/02-hltv-provider-infrastructure/02-01-SUMMARY.md` — Provider errors, URL helpers, and HTTP client implementation summary.
- `.planning/phases/02-hltv-provider-infrastructure/02-02-SUMMARY.md` — Domain models, parser errors, and fixture-tested parsers implementation summary.

### External References
- Cobra documentation/repository — CLI command framework: https://github.com/spf13/cobra

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/output.WriteJSON[T]` and `internal/output.Response[T]` — JSON success envelope (`{data, meta}`). All commands write success output through this.
- `internal/output.WriteErrorJSON` — JSON error envelope on stderr. All commands write errors through this.
- `internal/hltv.Client` — Injectable HTTP client with timeout and user-agent. Use via `WithHTTPClient` option.
- `internal/hltv.URLs` — URL helpers (`EventsURL()`, `ResultsURL()`). Use for constructing fetch URLs.
- `internal/hltv/parser.ParseEvents` — Event parser (accepts `io.Reader`, returns `[]domain.Event`).
- `internal/hltv/parser.ParseResults` — Result parser (accepts `io.Reader`, returns `[]domain.Result`).
- `internal/domain.Event` and `internal/domain.Result` — Typed domain models with JSON tags already matching EVNT-02 and RSLT-02 requirements.

### Established Patterns
- Command constructors follow `newXxxCommand(out io.Writer) *cobra.Command` — injectable stdout writer from Phase 1.
- `cli.Execute()` calls `cli.NewRootCommand(os.Stdout, os.Stderr)` in `cmd/dem/main.go` and registers subcommands via `root.AddCommand(...)`.
- Provider/parser isolation: commands depend on interfaces, tests use fake transports and fixture files, no live HLTV calls in tests.
- Constructor options pattern from Phase 2: `NewClient(opts ...ClientOption)` — reusable for provider constructors.

### Integration Points
- `internal/cli/root.go` — `NewRootCommand` wires subcommands. New commands are registered via `root.AddCommand(...)`.
- `cmd/dem/main.go` — Calls `cli.Execute()` — no changes needed unless build metadata is updated.
- `go.mod` — Already includes `cobra` and `goquery`. No new dependencies expected.

</code_context>

<specifics>
## Specific Ideas

- The user wants the CLI to feel script-friendly and predictable. Commands should produce stable JSON shapes and machine-stable error codes.
- Tier filtering should be provider-internal — users don't need to know how the HTML represents tiers.
- The user chose provider middleware to keep commands thin and testable, consistent with the Phase 2 isolation philosophy.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 3-Events and Results Commands*
*Context gathered: 2026-05-02*

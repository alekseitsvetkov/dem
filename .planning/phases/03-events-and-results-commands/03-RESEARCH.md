# Phase 3: Events and Results Commands - Research

**Researched:** 2026-05-02
**Domain:** Go CLI command composition with Cobra, provider middleware pattern, HLTV event/result data exposure
**Confidence:** MEDIUM

## Summary

Phase 3 composes Phase 2's existing infrastructure (HTTP client, URL helpers, goquery parsers, domain models) behind injectable provider interfaces to deliver `dem events --tier <tier>` and `dem results` Cobra commands. Each command fetches the appropriate HLTV page, parses it into typed domain models, applies filtering/limiting at the provider layer, and writes JSON to stdout via the established `output.WriteJSON[T]` envelope.

**Primary recommendation:** Implement providers in a new `internal/provider` package using the same constructor-options pattern established in Phase 2. The Event domain model requires a `Tier` field addition (plan 03-01) so the `EventsProvider` can perform client-side tier filtering on parsed events. Error mapping is 1:1 from Phase 2 error codes to CLI error envelope codes, requiring no translation layer — just type-assert the error, extract `.Code` and `.Details()`, and pass to `output.WriteErrorJSON`.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| CLI command registration (`dem events`, `dem results`) | CLI entry point | — | Cobra commands registered in `NewRootCommand`; the command handler is the entry point for user interaction |
| Flag validation (`--tier`, `--limit`) | CLI command handler | — | Validation before network access per Phase 1 D-12 contract; must fail before any provider call |
| HLTV page fetching | Provider (backend) | — | Provider wraps `hltv.Client.Fetch` — network I/O is provider-layer concern, not command-layer |
| HTML parsing | Parser (backend) | Provider | Parser already exists in `internal/hltv/parser`; provider delegates to parser functions |
| Tier filtering | Provider (backend) | — | Per D-04: EventsProvider filters parsed events internally; command just passes flag value |
| Limit truncation | Provider (backend) | — | Per D-06: client-side truncation after full parse; provider returns already-truncated slice |
| JSON output encoding | CLI command handler | output package | Commands call `output.WriteJSON[T]` with the provider result; output layer handles serialization |
| Error encoding (stderr) | CLI command handler | output package | Commands type-assert provider/parser errors, extract code+details, call `output.WriteErrorJSON` |

## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Add a provider middleware layer (`EventsProvider`, `ResultsProvider`) between Cobra commands and Phase 2 infrastructure. Commands depend on provider interfaces, not directly on `Client.Fetch` or parser functions.
- **D-02:** Each provider wraps `Client.Fetch` + a parser function into a single-call method that returns `([]domain.Event, error)` / `([]domain.Result, error)` plus filtering and limiting.
- **D-03:** Provider constructors accept injectable dependencies (`*hltv.Client`, `hltv.URLs`) following the Phase 2 option-pattern precedent. Providers pass through typed Phase 2 errors without remapping at the provider boundary.
- **D-04:** The `EventsProvider` receives the tier string and filters parsed events internally. The command handler does not perform filtering — it just passes the flag value.
- **D-05:** The `--tier` CLI flag is a string type (not int, not enum). This accommodates non-numeric tier labels HLTV may use (e.g., "S") without constraining v1 to a fixed set.
- **D-06:** Truncation is client-side — the full HLTV page is parsed, then the result slice is bounded. HLTV pages have a fixed size, so there is no server-side pagination to exploit.
- **D-07:** The provider interface receives the limit parameter and returns already-truncated data. The command handler passes the flag value through.
- **D-08:** Phase 2 error codes map directly to CLI error envelope codes. Users see `network_error`, `http_error`, `parse_error`, and `unavailable_data` as machine-stable snake_case codes. The CLI error envelope wraps them with URL and status context via its `details` field.
- **D-09:** Validation errors (e.g., missing required flags) must fail before any network access, continuing the Phase 1 contract.
- JSON-only stdout with `{data, meta}` envelope (Phase 1 D-05, D-06, D-07).
- Structured error JSON on stderr with snake_case codes (Phase 1 D-09, D-10, D-11).
- Validation before network access (Phase 1 D-12).
- Commands use injectable writers, not globals (Phase 1 D-15).
- Top-level v1 commands: `dem events`, `dem results`, `dem demo <match-id>` (Phase 1 D-02).

### Claude's Discretion

- Provider interface method signatures, constructor option names, and package layout.
- Default limit value when `--limit` is not provided by the user.
- Test strategy details (fake transports, table-driven tests, injected writers).
- Exact plan split across the 3 roadmap plans (03-01 domain/schemas, 03-02 events command, 03-03 results command).

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| EVNT-01 | User can list Tier 1 HLTV events as JSON. | EventsProvider wraps Client.Fetch(URLs.EventsURL()) + ParseEvents + tier filter; command writes result via WriteJSON[[]domain.Event] |
| EVNT-02 | Each event includes stable available fields such as event ID, name, date range, location, and source URL. | `domain.Event` struct already has all required fields with JSON tags; Section: Domain Models |
| EVNT-03 | User can limit the number of events returned. | Provider-level slice truncation after full parse; --limit flag passed through to provider |
| RSLT-01 | User can list completed HLTV match results as JSON. | ResultsProvider wraps Client.Fetch(URLs.ResultsURL()) + ParseResults; command writes result via WriteJSON[[]domain.Result] |
| RSLT-02 | Each result includes stable available fields such as match ID, teams, score, event, date, format, and source URL. | `domain.Result` struct already has all required fields with JSON tags; Section: Domain Models |
| RSLT-03 | User can limit the number of results returned. | Same truncation pattern as EVNT-03; provider-level bounding |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/spf13/cobra | v1.10.1 | CLI command framework | Already in go.mod; Phase 1/2 patterns built on Cobra |
| github.com/PuerkitoBio/goquery | v1.12.0 | HTML parsing (parser layer) | Already in go.mod; Phase 2 parsers depend on it |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| encoding/json | stdlib | JSON serialization | Already used by `output.WriteJSON[T]` and `output.WriteErrorJSON` |
| net/http | stdlib | HTTP transport (via fake RoundTripper in tests) | Already used by `hltv.Client`; fake transports for provider tests |
| net/http/httptest | stdlib | Test HTTP server | Use in provider tests with caution — sandbox may block `ListenAndServe`; prefer fake `RoundTripper` transports as established in Phase 2 plan 02-02 |

### Alternatives Considered

None — the stack is already locked by Phase 1 and Phase 2 decisions. Phase 3 introduces zero new dependencies.

**Installation:** No new packages required. All dependencies are already in `go.mod`.

**Version verification:** [VERIFIED: go.mod] cobra v1.10.1, goquery v1.12.0. Go toolchain: go1.26.2 darwin/arm64 [VERIFIED: go version].

## Architecture Patterns

### System Architecture Diagram

```
User: dem events --tier 1 --limit 5
    │
    ▼
┌──────────────────────┐
│  cmd/dem/main.go     │  os.Exit(cli.Execute())
│  Entry point         │
└────────┬─────────────┘
         │
         ▼
┌──────────────────────┐
│  cli.Execute()       │  NewRootCommand(os.Stdout, os.Stderr)
│  Root command        │  root.Execute()
└────────┬─────────────┘
         │ Cobra dispatches "events"
         ▼
┌──────────────────────┐
│  newEventsCommand()  │  Flag parsing: --tier (string), --limit (int)
│  Cobra RunE handler  │  Validation BEFORE provider call (D-09)
│                      │  Injects EventsProvider interface
└────────┬─────────────┘
         │ Calls provider.GetEvents(ctx, tier, limit)
         ▼
┌──────────────────────┐
│  EventsProvider      │  Wraps: Client.Fetch(URLs.EventsURL())
│  (internal/provider) │  Then:  parser.ParseEvents(body, url)
│                      │  Then:  filter by tier string
│                      │  Then:  truncate to limit
│                      │  Returns: ([]domain.Event, error)
└────────┬─────────────┘
         │ Success path
         ▼
┌──────────────────────┐
│  output.WriteJSON    │  stdout: {"data": [...], "meta": {...}}
│  [](domain.Event)    │
└──────────────────────┘

         │ Error path (any provider/parser error)
         ▼
┌──────────────────────┐
│  output.WriteErrorJSON│  stderr: {"error": {"code": "...", "message": "...", "details": {...}}}
│  (stderr)            │
└──────────────────────┘
         │
         ▼
  os.Exit(1) or return err to cobra (cli.Execute handles exit code)
```

### Recommended Project Structure

```
internal/
├── cli/
│   ├── root.go          # NewRootCommand — wires events + results commands
│   ├── version.go       # Reference: newVersionCommand pattern
│   ├── events.go        # NEW: newEventsCommand(out io.Writer) *cobra.Command
│   └── results.go       # NEW: newResultsCommand(out io.Writer) *cobra.Command
├── provider/             # NEW package
│   ├── events.go        # EventsProvider interface + implementation
│   ├── events_test.go   # EventsProvider tests (fake transport + fixture)
│   ├── results.go       # ResultsProvider interface + implementation
│   └── results_test.go  # ResultsProvider tests
├── domain/
│   └── models.go        # Existing — may need Tier field on Event (plan 03-01)
├── hltv/
│   ├── client.go        # Existing — injectable HTTP client
│   ├── client_test.go   # Existing
│   ├── errors.go        # Existing — ProviderError with network_error/http_error
│   ├── errors_test.go   # Existing
│   ├── urls.go          # Existing — EventsURL(), ResultsURL(), MatchURL()
│   ├── urls_test.go     # Existing
│   └── parser/
│       ├── events.go    # Existing — ParseEvents(io.Reader, sourceURL)
│       ├── results.go   # Existing — ParseResults(io.Reader, sourceURL)
│       └── ...          # Existing tests and fixtures
└── output/
    ├── json.go          # Existing — WriteJSON[T], Response[T]
    ├── error.go         # Existing — WriteErrorJSON, ErrorBody, ErrorResponse
    └── *_test.go        # Existing tests
```

### Pattern 1: Provider Middleware (Constructor Options)

**What:** A provider struct wraps `hltv.Client` + `hltv.URLs` + a parser function. The constructor uses the same functional-options pattern as `hltv.NewClient`.

**When to use:** For every command that needs HLTV data. The command depends on the provider interface, not on `Client.Fetch` directly.

**Example:**

```go
// Source: Phase 2 hltv.NewClient pattern, adapted for providers
package provider

type EventsProvider struct {
    client *hltv.Client
    urls   hltv.URLs
}

type EventsProviderOption func(*EventsProvider)

func NewEventsProvider(opts ...EventsProviderOption) *EventsProvider {
    p := &EventsProvider{
        client: hltv.NewClient(),
        urls:   hltv.NewURLs(""),
    }
    for _, opt := range opts {
        opt(p)
    }
    return p
}

func WithEventsClient(c *hltv.Client) EventsProviderOption {
    return func(p *EventsProvider) { p.client = c }
}

func WithEventsURLs(u hltv.URLs) EventsProviderOption {
    return func(p *EventsProvider) { p.urls = u }
}

// GetEvents fetches, parses, filters by tier, and truncates to limit.
// A limit of 0 means no truncation (return all).
func (p *EventsProvider) GetEvents(ctx context.Context, tier string, limit int) ([]domain.Event, error) {
    body, err := p.client.Fetch(ctx, p.urls.EventsURL())
    if err != nil {
        return nil, err // Pass through ProviderError without remapping (D-03, D-08)
    }

    events, err := parser.ParseEvents(bytes.NewReader(body), p.urls.EventsURL())
    if err != nil {
        return nil, err // Pass through ParseError without remapping
    }

    if tier != "" {
        events = filterByTier(events, tier)
    }
    if limit > 0 && limit < len(events) {
        events = events[:limit]
    }

    return events, nil
}
```

### Pattern 2: Error Mapping (1:1 from Phase 2 to CLI)

**What:** Provider and parser errors carry machine-stable `.Code` strings and `.Details()` maps. Command handlers type-assert the error, extract code and details, and pass them to `output.WriteErrorJSON`.

**When to use:** Every error path in a command handler.

**Example:**

```go
// Source: Phase 1 version command + Phase 2 error types
func (c *eventsCommand) runE(cmd *cobra.Command, args []string) error {
    tier, _ := cmd.Flags().GetString("tier")
    limit, _ := cmd.Flags().GetInt("limit")

    // D-09: Validation before network access
    if tier == "" {
        return output.WriteErrorJSON(c.errOut, "validation_error", "--tier is required", map[string]any{
            "flag": "tier",
        })
    }
    if limit < 0 {
        return output.WriteErrorJSON(c.errOut, "validation_error", "--limit must be >= 0", map[string]any{
            "flag": "limit",
        })
    }

    events, err := c.provider.GetEvents(cmd.Context(), tier, limit)
    if err != nil {
        return mapToErrorEnvelope(c.errOut, err)
    }

    return output.WriteJSON(c.out, events, nil)
}

func mapToErrorEnvelope(w io.Writer, err error) error {
    // ProviderError and ParseError both have Code()/Details() via type-specific methods
    type codedError interface {
        Code() string   // not in current interface — see Findings below
    }
    // ... type-assert and extract code/details, pass to WriteErrorJSON
}
```

### Pattern 3: Command Registration (Phase 1 Reference)

**What:** Each command has a constructor `newXxxCommand(out io.Writer) *cobra.Command`. The root command wires them via `root.AddCommand(...)`.

**When to use:** For every new top-level command.

**Reference:** `internal/cli/version.go` — `newVersionCommand` is the canonical pattern. Events and results commands follow the same shape but additionally accept `errOut io.Writer` and inject a provider.

**Example:**

```go
// Source: internal/cli/version.go (existing pattern)
func newEventsCommand(out io.Writer, errOut io.Writer, provider EventsProvider) *cobra.Command {
    cmd := &cobra.Command{
        Use:   "events",
        Short: "List Tier 1 HLTV events as JSON",
        RunE: func(cmd *cobra.Command, args []string) error {
            tier, _ := cmd.Flags().GetString("tier")
            limit, _ := cmd.Flags().GetInt("limit")
            // ... validation, provider call, output
        },
    }
    cmd.Flags().String("tier", "", "Event tier filter (e.g., \"1\", \"S\")")
    cmd.Flags().Int("limit", 0, "Maximum events to return (0 = no limit)")
    return cmd
}
```

### Anti-Patterns to Avoid

- **Direct `Client.Fetch` in command handlers:** Violates D-01. Commands depend on provider interfaces, not raw infrastructure.
- **Tier filtering in command handler:** Violates D-04. Pass the flag value through; filtering lives in the provider.
- **Remapping errors at provider boundary:** Violates D-03, D-08. Phase 2 error codes pass through unchanged.
- **Network access before flag validation:** Violates D-09. Always validate required flags first.
- **Using `os.Stdout`/`os.Stderr` directly in commands:** Violates Phase 1 D-15. Use injected writers only.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| CLI framework | Custom arg parser | `github.com/spf13/cobra` v1.10.1 | Already integrated; flag binding, help text, error silence |
| JSON serialization | Custom encoder | `encoding/json` (stdlib) | Already used by `output.WriteJSON[T]`; no schema library needed |
| Error wrapping chain | Custom error hierarchy | `errors.As` + typed error structs with `.Details()` | Phase 2 errors already implement the pattern; just type-assert |
| HTTP transport in tests | Real network calls | Fake `http.RoundTripper` | Phase 2 established this; avoids sandbox `ListenAndServe` restrictions |

**Key insight:** Phase 3 is pure composition. Every piece of infrastructure already exists from Phase 1 (CLI, output, error envelopes) and Phase 2 (client, URLs, parsers, domain models, error taxonomy). The new work is wiring them together through provider interfaces.

## Domain Model Gap: Tier Field

### Finding

The current `domain.Event` struct [VERIFIED: internal/domain/models.go] lacks a `Tier` field:

```go
type Event struct {
    ID        string `json:"id"`
    Name      string `json:"name"`
    StartDate string `json:"start_date,omitempty"`
    EndDate   string `json:"end_date,omitempty"`
    Location  string `json:"location,omitempty"`
    SourceURL string `json:"source_url"`
}
```

The current `ParseEvents` function [VERIFIED: internal/hltv/parser/events.go] does not extract tier/event-type information from the HTML. The test fixture [VERIFIED: internal/hltv/parser/testdata/events.html] does not contain tier-related markup.

### Impact

D-04 requires the `EventsProvider` to filter parsed events by tier client-side. Without a `Tier` field on `Event`, the provider has nothing to filter against. Therefore, **Plan 03-01 must add a Tier field to the Event model and update the parser to extract tier/event-type information from the HLTV events page HTML**.

### Tier Representation in HLTV HTML

Research into HLTV's events page structure (via the gigobyte/HLTV TypeScript library [CITED: github.com/gigobyte/HLTV]) reveals:

1. **Event type query parameter:** HLTV supports `?eventType=` as a URL query parameter with values: `MAJOR`, `INTLLAN`, `REGIONALLAN`, `LOCALLAN`, `ONLINE`, `OTHER` [CITED: gigobyte/HLTV src/shared/EventType.ts].
2. **CSS class-based categorization:** On upcoming events pages, events use CSS classes `.big-event`, `.small-event`, `.ongoing-event` to distinguish prominence [CITED: gigobyte/HLTV src/endpoints/getEvents.ts].
3. **The `/events` page:** HLTV's main events listing page likely embeds event type/tier as a visible label or data attribute on each event row. Exact selectors could not be verified because HLTV blocks automated HTTP requests (Cloudflare 403) [ASSUMED].

The recommended approach for Plan 03-01: extend the parser to look for common HLTV tier indicators (event type label text, CSS class, or data attribute) and populate a new `Tier string` field with `json:"tier,omitempty"`. The `--tier` flag value is compared case-insensitively against this field in the provider's filter function.

## Common Pitfalls

### Pitfall 1: Missing Tier Field Causes Silent No-Op Filter

**What goes wrong:** If the Event model lacks a Tier field and the parser doesn't extract tier info, the provider's tier filter becomes a no-op — all events pass through regardless of the `--tier` flag value. The command appears to work but doesn't actually filter.

**Why it happens:** The Event domain model was created in Phase 2 plan 02-02 without anticipating the client-side tier filtering requirement from Phase 3.

**How to avoid:** Plan 03-01 MUST add a Tier field to Event AND update the parser fixtures/selectors to extract tier from the HTML. The provider filter function should be tested with events that have different tier values to confirm filtering works.

**Warning signs:** All `--tier` values produce the same output. Event count doesn't decrease when a restrictive tier is specified.

### Pitfall 2: Validation After Provider Call

**What goes wrong:** Required flag validation (e.g., `--tier` is required) happens inside the provider or after the provider call. This triggers an unnecessary network request before the user sees the validation error.

**Why it happens:** Developers might put flag validation in the provider for "defense in depth" or handle it in the error-return path of the provider.

**How to avoid:** D-09 requires validation before any network access. The command handler validates flags immediately after `cmd.Flags().GetString("tier")` and before calling `provider.GetEvents(...)`. Cobra's `PreRunE` or early `RunE` validation are appropriate.

**Warning signs:** Network error appears before validation error when both `--tier` is missing and HLTV is unreachable.

### Pitfall 3: Sandbox Blocks httptest.NewServer

**What goes wrong:** Provider tests using `httptest.NewServer` panic with "operation not permitted" in the sandboxed environment.

**Why it happens:** The Claude Code sandbox prohibits binding local TCP listeners. `httptest.NewServer` calls `ListenAndServe` internally.

**How to avoid:** Use fake `http.RoundTripper` transports (the pattern established in Phase 2 plan 02-02 for `client_test.go`). The roundTripFunc adapter pattern allows injecting controlled responses without binding a port.

**Warning signs:** Test panics with "listen tcp :0: bind: operation not permitted" or similar.

### Pitfall 4: Limit=0 Ambiguity

**What goes wrong:** If `--limit` defaults to 0 and 0 means "no limit," then an explicit `--limit 0` returns all results — which may surprise users who expect it to return zero results.

**Why it happens:** Go's zero-value for int is 0, and Cobra flag defaults inherit this.

**How to avoid:** Document that `--limit 0` means "return all" (no truncation). In the provider, treat `limit <= 0` as no-op. This is consistent with how many CLI tools handle limit flags. Alternatively, set no default and check if the flag was explicitly set using `cmd.Flags().Changed("limit")`.

**Warning signs:** User reports that `--limit 0` returns all results instead of empty.

## Code Examples

Verified patterns from existing codebase:

### Command Shape (version command reference)

```go
// Source: internal/cli/version.go — canonical newXxxCommand pattern
func newVersionCommand(out io.Writer) *cobra.Command {
    return &cobra.Command{
        Use:   "version",
        Short: "Print dem version information as JSON",
        RunE: func(cmd *cobra.Command, args []string) error {
            return output.WriteJSON(out, versionInfo{...}, nil)
        },
    }
}
```

### JSON Success Envelope

```go
// Source: internal/output/json.go
type Response[T any] struct {
    Data T              `json:"data"`
    Meta map[string]any `json:"meta"`
}

func WriteJSON[T any](w io.Writer, data T, meta map[string]any) error {
    if meta == nil { meta = map[string]any{} }
    return json.NewEncoder(w).Encode(Response[T]{Data: data, Meta: meta})
}
```

### Error Envelope

```go
// Source: internal/output/error.go
type ErrorResponse struct {
    Error ErrorBody `json:"error"`
}
type ErrorBody struct {
    Code    string         `json:"code"`
    Message string         `json:"message"`
    Details map[string]any `json:"details"`
}

func WriteErrorJSON(w io.Writer, code, message string, details map[string]any) error { ... }
```

### Fake HTTP Transport (for tests)

```go
// Source: internal/hltv/client_test.go — sandbox-safe test pattern
type roundTripFunc func(*http.Request) (*http.Response, error)
func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return fn(req) }

// Usage:
transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
    return &http.Response{
        StatusCode: http.StatusOK,
        Body:       io.NopCloser(strings.NewReader(htmlContent)),
        Header:     make(http.Header),
        Request:    req,
    }, nil
})
client := hltv.NewClient(hltv.WithHTTPClient(&http.Client{Transport: transport}))
```

### Provider Error with Details

```go
// Source: internal/hltv/errors.go
type ProviderError struct {
    Code       string
    Message    string
    URL        string
    StatusCode int
    Err        error
}
func (e *ProviderError) Details() map[string]any { ... }  // returns url, status_code
```

## Error Mapping Table

Per D-08, Phase 2 error codes map 1:1 to CLI error envelope codes. The command handler type-asserts the error and extracts the code + details.

| Error Source | Code | Details Keys | CLI Output |
|-------------|------|-------------|------------|
| `*hltv.ProviderError` | `network_error` | `url` | `{"error":{"code":"network_error","message":"...","details":{"url":"..."}}}` |
| `*hltv.ProviderError` | `http_error` | `url`, `status_code` | `{"error":{"code":"http_error","message":"...","details":{"url":"...","status_code":503}}}` |
| `*parser.ParseError` | `parse_error` | `area`, `field` | `{"error":{"code":"parse_error","message":"...","details":{"area":"events","field":"id"}}}` |
| `*parser.ParseError` | `unavailable_data` | `area` | `{"error":{"code":"unavailable_data","message":"...","details":{"area":"demo"}}}` |
| Validation (command handler) | `validation_error` | `flag`, `value` | `{"error":{"code":"validation_error","message":"...","details":{"flag":"tier"}}}` |

**Implementation note:** The current error types use `Code` as an exported field, not a `Code()` method. The command handler accesses `err.Code` directly after type-asserting to `*hltv.ProviderError` or `*parser.ParseError`. No common `CodedError` interface exists — the command handler uses a type switch.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Build, test | Yes | go1.26.2 darwin/arm64 | — |
| cobra | CLI framework | Yes (in go.mod) | v1.10.1 | — |
| goquery | HTML parsing | Yes (in go.mod) | v1.12.0 | — |
| HLTV.org | Live data (runtime only) | Yes (public) | — | Tests use fixtures, not live HLTV |

**Missing dependencies with no fallback:** None. All build and test dependencies are satisfied.

**Test workaround:** `GOCACHE=/Users/base/Documents/dem/.cache/go-build` is required to avoid sandbox cache-blocking issues during `go test ./...`. This is a known environment constraint from Phase 2.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | No authentication in v1 — all pages are public |
| V3 Session Management | No | No sessions in v1 |
| V4 Access Control | No | No authorization — read-only public data |
| V5 Input Validation | Yes | `--tier` string validated for empty; `--limit` validated for non-negative; Phase 4 `match-id` validated as numeric |
| V6 Cryptography | No | No cryptographic operations in this phase |

### Known Threat Patterns for Go CLI with HTTP

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Unvalidated flag values passed to URL construction | Tampering | Validate `--tier` is non-empty before use; `--limit` is non-negative |
| Response body too large (memory exhaustion) | Denial of Service | `http.Client.Timeout` set to 15s via `hltv.DefaultTimeout`; `io.ReadAll` bounded by server response |
| User-agent impersonation | Spoofing | Fixed `dem/dev` user-agent sent on all requests (polite identification) |
| Error details leaking internal state | Information Disclosure | `ProviderError.Details()` returns only `url` and `status_code` — no stack traces or response bodies |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | HLTV events page HTML includes a tier/event-type label on each event row that the parser can extract (exact CSS selector could not be verified due to Cloudflare blocking) | Domain Model Gap: Tier Field | Plan 03-01 parser extension may need to inspect the live page manually to find the correct selector; may require an additional research pass |
| A2 | The `Event.Tier` field should be a free-form string matching HLTV's label (e.g., "S-Tier", "A-Tier", "Major") rather than a normalized enum | Domain Model Gap: Tier Field | If HLTV uses inconsistent labels across events, tier filtering by string match may produce unexpected results |
| A3 | Default limit of 0 (meaning "no limit"/return all) is the correct behavior | Common Pitfalls: Limit=0 | If users expect `--limit 0` to return zero results, the default semantics need to change |

## Open Questions

1. **Exact HTML selector for event tier**
   - What we know: HLTV events page uses `eventType` query parameter server-side (`MAJOR`, `INTLLAN`, etc.) [CITED: gigobyte/HLTV]. Gigobyte library uses CSS classes `.big-event`, `.small-event`, `.ongoing-event` on upcoming events page. Our current parser uses `.event` selector for the events listing page.
   - What's unclear: The exact element/attribute that carries the tier label on the `/events` listing page. HLTV blocks automated requests (Cloudflare).
   - Recommendation: Plan 03-01 should include a manual page-inspection step (open `hltv.org/events` in a browser with DevTools) to identify the tier selector before coding the parser extension. If live inspection is impractical, implement a flexible selector strategy that tries multiple known patterns (text label, CSS class, data attribute) and communicates discovery results back to the planner.

2. **Should `--tier` be required or optional for `dem events`?**
   - What we know: EVNT-01 says "User can list Tier 1 HLTV events as JSON." The name implies Tier 1 is the primary use case. D-05 says tier is a string flag.
   - What's unclear: Should the command require `--tier` (fail validation if absent) or default to showing all events when omitted?
   - Recommendation: Make `--tier` optional. When absent, return all events unfiltered. When present, filter by the provided tier string. This gives users flexibility while still supporting the primary "Tier 1" use case. Mark `--tier` as required in the help text description but don't enforce it programmatically — let the user discover all events if they omit it.

3. **What is the sensible default limit?**
   - What we know: D-06 says limit applies client-side. HLTV pages have fixed-size event/results lists (typically 20-50 items). The Cobra int flag defaults to 0.
   - What's unclear: Should there be a non-zero default (e.g., 25) to prevent accidentally returning 50+ results? Or is 0 (all) acceptable?
   - Recommendation: Default to 0 (no limit) to match the principle of least surprise — users who don't specify a limit get everything available on the page. The page size is naturally bounded. Power users can add `--limit 5` for concise output.

## Sources

### Primary (HIGH confidence)
- `go.mod` — dependency versions verified [VERIFIED]
- `internal/cli/version.go` — canonical command pattern [VERIFIED]
- `internal/cli/root.go` — command registration pattern [VERIFIED]
- `internal/cli/root_test.go` — command test patterns [VERIFIED]
- `internal/output/json.go` — JSON success envelope [VERIFIED]
- `internal/output/error.go` — JSON error envelope [VERIFIED]
- `internal/hltv/client.go` — injectable HTTP client pattern [VERIFIED]
- `internal/hltv/client_test.go` — fake transport test pattern [VERIFIED]
- `internal/hltv/errors.go` — provider error taxonomy and Details() [VERIFIED]
- `internal/hltv/urls.go` — URL helpers (EventsURL, ResultsURL) [VERIFIED]
- `internal/domain/models.go` — Event, Result, DemoLink structs [VERIFIED]
- `internal/hltv/parser/events.go` — ParseEvents function [VERIFIED]
- `internal/hltv/parser/results.go` — ParseResults function [VERIFIED]
- `internal/hltv/parser/errors.go` — parser error taxonomy [VERIFIED]
- `internal/hltv/parser/events_test.go` — parser test patterns [VERIFIED]
- `internal/hltv/parser/results_test.go` — parser test patterns [VERIFIED]
- `internal/hltv/parser/testdata/events.html` — events fixture (no tier data) [VERIFIED]
- `internal/hltv/parser/testdata/results.html` — results fixture [VERIFIED]
- `.planning/phases/03-events-and-results-commands/03-CONTEXT.md` — locked decisions [VERIFIED]
- `.planning/REQUIREMENTS.md` — EVNT-01 through RSLT-03 [VERIFIED]

### Secondary (MEDIUM confidence)
- gigobyte/HLTV TypeScript library — EventType enum and scrapeEvents.ts selectors [CITED: github.com/gigobyte/HLTV]
- gigobyte/HLTV `src/shared/EventType.ts` — MAJOR, INTLLAN, REGIONALLAN, LOCALLLAN, ONLINE, OTHER [VERIFIED via WebFetch]

### Tertiary (LOW confidence)
- HLTV event page HTML structure for tier labels — blocked by Cloudflare; structure inferred from gigobyte/HLTV patterns and community knowledge [ASSUMED]

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — zero new dependencies; all verified in go.mod and codebase
- Architecture: HIGH — patterns are direct composition of existing, verified Phase 1/2 code
- Pitfalls: MEDIUM — domain model gap around Tier field is a verified gap, but the exact HLTV tier selector is unverified
- Error mapping: HIGH — mapping is 1:1 from verified Phase 2 error types to verified Phase 1 output helpers

**Research date:** 2026-05-02
**Valid until:** 2026-06-02 (30 days — HLTV HTML structure may change; parser selectors should be re-verified against live page if more than 30 days pass)

# Phase 4: Demo Link Lookup - Research

**Researched:** 2026-05-02
**Domain:** Go CLI command for HLTV match demo link extraction via HTML parsing
**Confidence:** MEDIUM

## Summary

Phase 4 adds the final v1 command `dem demo <match-id>` -- a zero-flag Cobra subcommand that takes a numeric HLTV match ID, fetches the match page, parses it for a demo download link, and returns the result as JSON. This phase follows the same 4-layer architecture established in Phase 3: CLI command handler delegates to a DemoProvider interface (wrapping Client.Fetch + parser.ParseDemoLink), which returns a typed `domain.DemoLink` value. The command writes it via `output.WriteJSON` in the standard `{data, meta}` envelope.

The critical architectural decision is how to handle unavailable demos: the parser returns BOTH a partial `DemoLink` (with `MatchID` and `MatchURL` populated) AND a `ParseError` with code `unavailable_data`. The DemoProvider catches this specific error case and converts it to a success (nil error), returning the partial `DemoLink`. The command writes it to stdout with exit code 0. Scripts detect demo availability by checking for the presence of `data.demo_url` in the JSON response -- when absent (due to `omitempty` JSON tag), the demo is unavailable.

A complete reference implementation exists in the worktree at `.claude/worktrees/bold-heyrovsky-44a33d/` covering all layers: parser (`internal/hltv/parser/demo.go`), provider (`internal/provider/demo.go`), and CLI command (`internal/cli/demo.go`), plus comprehensive tests at each layer. However, none of these files exist in the current working tree -- the plan must create them.

**Primary recommendation:** Replicate the worktree implementation exactly, creating parser + fixtures first (Plan 04-01), then provider + CLI command + root.go wiring (Plan 04-02), following the established EventsProvider/ResultsProvider patterns for constructor options and error mapping.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Match ID validation (positive integer) | CLI Command Handler | -- | Input validation before any I/O; uses `strconv.Atoi` + `> 0` check |
| HTTP fetch of match page | HLTV Client (infrastructure) | -- | Reuses existing `Client.Fetch` with timeout, user-agent, TLS fingerprinting |
| HTML parsing for demo link | Parser (`internal/hltv/parser`) | -- | goquery-based CSS selector extraction, returns typed `domain.DemoLink` |
| Unavailable demo detection | Parser + Provider | -- | Parser returns `unavailable_data` error; Provider converts to success (D-03) |
| DemoLink JSON output | CLI (`internal/output`) | -- | `output.WriteJSON` handles `{data, meta}` envelope with `omitempty` on DemoURL |
| Error mapping to stderr | CLI Command Handler | -- | Type switch on `*hltv.ProviderError` / `*parser.ParseError`, mirrors `mapEventsError` |
| Command registration | CLI Root (`internal/cli/root.go`) | -- | `root.AddCommand(newDemoCommand(out, errOut, provider.NewDemoProvider()))` |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go | 1.25.0 (go.mod) / 1.26.2 (installed) | Language runtime | Already in project; goquery v1.12.0 requires Go 1.25+ |
| spf13/cobra | v1.10.1 | CLI command framework | Already in project; all existing commands use it |
| PuerkitoBio/goquery | v1.12.0 | HTML parsing with CSS selectors | Already in project; parser package depends on it |
| encoding/json | stdlib | JSON encoding | Already in project; output helpers use it |
| net/http | stdlib | HTTP client | Already in project; `Client.Fetch` builds on it |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| net/url | stdlib | URL resolution (`resolveURL` helper) | Already used in parser/events.go; demo.go reuses same helper |
| strconv | stdlib | String-to-int conversion for match ID | Validation in command handler (D-05/D-06) |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `DisableFlagParsing: true` on demo cmd | `--` separator (`dem demo -- -5`) | `--` separator requires user education and adds friction; `DisableFlagParsing` is simpler for a zero-flag command but prevents `dem demo --help` (users must use `dem help demo` instead) |

**Installation:**
```bash
# No new dependencies needed -- all packages already in go.mod
```

**Version verification:** All package versions confirmed via `go.mod` in the working tree. goquery v1.12.0 is the latest release (2026-03-15) and requires Go 1.25+ [VERIFIED: go.mod + goquery GitHub releases]. Cobra v1.10.1 is the version pinned in go.mod [VERIFIED: go.mod].

## Architecture Patterns

### System Architecture Diagram

```
User: dem demo 77777
         │
         ▼
┌─────────────────────────────────────────────────────┐
│  CLI Layer (internal/cli)                            │
│                                                      │
│  newDemoCommand(stdout, stderr, DemoProvider)        │
│    │                                                 │
│    ├─ Validate: strconv.Atoi("77777") → 77777 (> 0) │
│    │  (exit early on validation_error → stderr)      │
│    │                                                 │
│    ├─ Call: provider.GetDemo(ctx, 77777)             │
│    │       │                                         │
│    │       ├─ Success → WriteJSON(stdout, link)      │
│    │       └─ Error → mapDemoError(stderr, err)      │
│    │            (type switch: ProviderError /         │
│    │             ParseError / internal_error)        │
└────┼─────────────────────────────────────────────────┘
     │
     ▼  DemoProvider interface
┌─────────────────────────────────────────────────────┐
│  Provider Layer (internal/provider)                  │
│                                                      │
│  demoProvider.GetDemo(ctx, 77777)                    │
│    │                                                 │
│    ├─ urls.MatchURL(77777)                           │
│    │  → "https://www.hltv.org/matches/77777/-"       │
│    │                                                 │
│    ├─ client.Fetch(ctx, matchURL)                    │
│    │  ──── HTTP GET to hltv.org ────▶               │
│    │  ◀─── HTML body or ProviderError ────           │
│    │                                                 │
│    ├─ parser.ParseDemoLink(body, matchURL)           │
│    │       │                                         │
│    │       ├─ Demo available → DemoLink{MatchID,     │
│    │       │    MatchURL, DemoURL}, nil              │
│    │       │                                         │
│    │       └─ No demo → DemoLink{MatchID,            │
│    │            MatchURL} + ParseError{               │
│    │            Code: "unavailable_data"}             │
│    │                                                 │
│    └─ If unavailable_data → return (link, nil)       │
│       (convert error to success per D-03)            │
└────┼─────────────────────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────────────────────────┐
│  Parser Layer (internal/hltv/parser)                 │
│                                                      │
│  ParseDemoLink(io.Reader, matchURL)                  │
│    │                                                 │
│    ├─ goquery.NewDocumentFromReader(r)               │
│    │  (fail → ParseError{Code: "parse_error"})       │
│    │                                                 │
│    ├─ doc.Find("a.demo-link")                        │
│    │  (not found → partial DemoLink +                │
│    │   ParseError{Code: "unavailable_data"})          │
│    │                                                 │
│    ├─ sel.First().Attr("href")                       │
│    │  (empty → partial DemoLink +                    │
│    │   ParseError{Code: "unavailable_data"})          │
│    │                                                 │
│    └─ resolveURL(href) → absolute URL                │
│       → DemoLink{MatchID, MatchURL, DemoURL}, nil   │
│                                                      │
│  extractMatchID(matchURL) ← from url.Parse path      │
│  resolveURL(href) ← shared with events.go parser     │
└─────────────────────────────────────────────────────┘
```

### Recommended Project Structure
```
internal/
├── cli/
│   ├── demo.go              # newDemoCommand + mapDemoError (04-02)
│   ├── demo_test.go         # 7 tests: success, unavailable, validation, errors (04-02)
│   └── root.go              # MODIFY: AddCommand(newDemoCommand(...)) (04-02)
├── provider/
│   ├── demo.go              # DemoProvider interface + implementation (04-01)
│   └── demo_test.go         # 3 tests: success, unavailable, network error (04-01)
├── hltv/
│   └── parser/
│       ├── demo.go           # ParseDemoLink + extractMatchID (04-01)
│       ├── demo_test.go      # 4 tests: with-demo, without-demo, invalid HTML, empty body (04-01)
│       └── testdata/
│           ├── match-with-demo.html     # Fixture: <a class="demo-link" href="..."> (04-01)
│           └── match-without-demo.html  # Fixture: no .demo-link element (04-01)
└── domain/
    └── models.go             # EXISTS: DemoLink struct with JSON tags (no changes needed)
```

### Pattern 1: Provider Interface + Functional Options Constructor
**What:** Commands depend on an injectable interface; concrete implementation wraps Client + URLs; constructor accepts functional options for test doubles.
**When to use:** Every provider in this project (EventsProvider, ResultsProvider, DemoProvider).
**Example:**
```go
// Source: worktree internal/provider/demo.go -- mirrors EventsProvider pattern
type DemoProvider interface {
    GetDemo(ctx context.Context, matchID int) (domain.DemoLink, error)
}

type demoProvider struct {
    client *hltv.Client
    urls   hltv.URLs
}

type DemoProviderOption func(*demoProvider)

func NewDemoProvider(opts ...DemoProviderOption) DemoProvider {
    p := &demoProvider{
        client: hltv.NewClient(),
        urls:   hltv.NewURLs(""),
    }
    for _, opt := range opts {
        opt(p)
    }
    return p // returns interface, not concrete pointer
}
```

### Pattern 2: Command Constructor with Injected Writers
**What:** Command factories accept `io.Writer` for stdout and stderr; no globals.
**When to use:** Every command in this project.
**Example:**
```go
// Source: worktree internal/cli/demo.go
func newDemoCommand(out io.Writer, errOut io.Writer, p provider.DemoProvider) *cobra.Command {
    cmd := &cobra.Command{
        Use:                "demo <match-id>",
        Short:              "Get demo download link for an HLTV match",
        Args:               cobra.ExactArgs(1),
        DisableFlagParsing: true, // prevents Cobra from parsing "-5" as a flag
        RunE: func(cmd *cobra.Command, args []string) error {
            // validation before network access
            // provider call
            // output via out/errOut writers
        },
    }
    return cmd
}
```

### Pattern 3: Error Mapping via Type Switch
**What:** Command handler maps provider/parser errors to JSON error envelopes on stderr using a type switch.
**When to use:** Every command that calls a provider (events, results, demo).
**Example:**
```go
// Source: worktree internal/cli/demo.go -- mirrors mapEventsError
func mapDemoError(w io.Writer, err error) error {
    switch e := err.(type) {
    case *hltv.ProviderError:
        return output.WriteErrorJSON(w, e.Code, e.Message, e.Details())
    case *parser.ParseError:
        return output.WriteErrorJSON(w, e.Code, e.Message, e.Details())
    default:
        return output.WriteErrorJSON(w, "internal_error", err.Error(), nil)
    }
}
```

### Pattern 4: Unavailable Data as Success
**What:** Provider catches `unavailable_data` parse errors and converts them to successful returns with a partial domain object.
**When to use:** Demo link lookup where "no demo" is a normal outcome, not an error.
**Example:**
```go
// Source: worktree internal/provider/demo.go -- implements D-03
func (p *demoProvider) GetDemo(ctx context.Context, matchID int) (domain.DemoLink, error) {
    // ... fetch and parse ...
    link, err := parser.ParseDemoLink(bytes.NewReader(body), matchURL)
    if err != nil {
        var parseErr *parser.ParseError
        if errors.As(err, &parseErr) && parseErr.Code == parser.ErrorCodeUnavailableData {
            return link, nil // success with partial DemoLink (no DemoURL)
        }
        return domain.DemoLink{}, err
    }
    return link, nil
}
```

### Anti-Patterns to Avoid
- **Writing JSON manually in command handlers:** Use `output.WriteJSON` and `output.WriteErrorJSON` -- never call `json.NewEncoder` directly in CLI code.
- **Using `httptest.NewServer` in tests:** The sandbox blocks local listeners. Use `roundTripFunc` fake transports instead (already defined in `internal/provider/events_test.go` and `internal/hltv/client_test.go`).
- **Global stdout/stderr:** Commands receive `io.Writer` parameters; tests use `bytes.Buffer`.
- **Validation after network access:** Match ID validation must happen before `provider.GetDemo()` call (D-05/D-06, Phase 1 D-12).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| URL resolution (relative to absolute) | Custom URL joining | `resolveURL(href)` in `parser/events.go` | Already handles `https://www.hltv.org` base + relative href via `net/url.ResolveReference`; demo.go is in same package so it can call directly |
| HTTP fetching | New HTTP client | `hltv.Client.Fetch(ctx, url)` | Already handles timeout, user-agent, TLS fingerprinting, error taxonomy; used by all providers |
| JSON envelope encoding | Manual `json.Marshal` + map construction | `output.WriteJSON[T]` and `output.WriteErrorJSON` | Already handles `{data, meta}` and `{error: {code, message, details}}` envelopes consistently |
| Match ID extraction from URL | Regex or string splitting | `extractMatchID(matchURL)` in `parser/demo.go` | Parses `/matches/{id}/-` pattern using `net/url.Parse` + path splitting; already in the worktree implementation |
| Fake HTTP transport for tests | `httptest.NewServer` (blocked by sandbox) | `roundTripFunc` implementing `http.RoundTripper` | Already defined in `internal/provider/events_test.go` and `internal/hltv/client_test.go`; works without local listeners |

**Key insight:** The demo command is the simplest v1 command -- everything it needs already exists in the codebase (HTTP client, URL helpers, JSON output, error taxonomy, test patterns). The only new code is the goquery CSS selector for `.demo-link` and the orchestration logic in the provider.

## Runtime State Inventory

> This is a greenfield phase (new command, not rename/refactor/migration). No runtime state migration required.
>
> **Stored data:** None -- this phase adds a read-only lookup command. No database or persistent storage is created or modified.
>
> **Live service config:** None -- no external service configuration is needed beyond the existing HLTV fetch infrastructure.
>
> **OS-registered state:** None -- no OS-level registrations (cron, launchd, systemd) are created or modified.
>
> **Secrets and env vars:** None -- the phase requires no new secrets or environment variables.
>
> **Build artifacts:** The `go build` will produce the same `cmd/dem/main.go` binary with the new command linked in. No additional build artifacts.

## Common Pitfalls

### Pitfall 1: `DisableFlagParsing` Breaks Help on the Demo Command
**What goes wrong:** Setting `DisableFlagParsing: true` prevents Cobra from auto-handling `--help` / `-h` on the demo subcommand. Running `dem demo --help` treats `--help` as a positional arg, which fails validation ("match-id must be a positive integer") instead of showing help.
**Why it happens:** Cobra's `DisableFlagParsing` skips ALL flag processing including the built-in help/version flags. This is documented in cobra PR #2061 and issue #2060.
**How to avoid:** Accept this tradeoff. The alternative (`--` separator) is worse UX for a command whose sole positional argument is a number. Users can get help via `dem help demo` or `dem --help`.
**Warning signs:** If users report `dem demo --help` not working, direct them to `dem help demo`.

[CITED: github.com/spf13/cobra/issues/2060, PR #2061]

### Pitfall 2: HLTV CSS Selector May Be Stale
**What goes wrong:** The parser uses `doc.Find("a.demo-link")` to locate the demo download link. HLTV has historically changed their HTML structure for the demo section (gigobyte/HLTV issue #620 reported that the demo link moved to a `data-` attribute after a sponsor-related layout change). The fixture HTML in the worktree uses `<a href="/download/demo/77777" class="demo-link">` which may not match the current live HLTV page structure.
**Why it happens:** Public websites change markup without notice. The `.demo-link` class selector is an assumption based on the fixture.
**How to avoid:** The fixture tests validate parser logic, but live-selector validity is a separate concern. The parser is structured so the selector is a single line change if HLTV changes. Consider a post-implementation smoke test against a known live match ID with a demo (e.g., a recent major final).
**Warning signs:** `dem demo <valid-id>` returns "no demo available" for matches known to have demos; or `dem demo <valid-id>` returns a partial DemoLink when a demo should be present.

[CITED: github.com/gigobyte/HLTV/issues/620]

### Pitfall 3: Double-Return Pattern (Value + Error) Can Surprise Maintainers
**What goes wrong:** `ParseDemoLink` returns BOTH a non-zero `DemoLink` AND a non-nil `ParseError` when no demo is available. A developer who only checks `err != nil` and discards the value loses the MatchID/MatchURL context.
**Why it happens:** This is intentional (the provider needs the partial DemoLink to return success with match metadata), but it violates the conventional Go idiom of "if err != nil, the value is meaningless."
**How to avoid:** Document this explicitly in the `ParseDemoLink` godoc comment. The provider has the `errors.As` check that handles both branches correctly. Tests cover both the error-channel (parser_test.go) and the value-channel (provider_test.go).
**Warning signs:** A future refactor drops the `link` return value on error, causing the unavailable-demo case to lose MatchID/MatchURL in the response.

### Pitfall 4: Match ID 0 Is Invalid but `strconv.Atoi` Returns 0 for Errors
**What goes wrong:** `strconv.Atoi("abc")` returns `(0, error)`. If the validation only checks `err != nil` without also checking `matchID <= 0`, a non-numeric input that parses to 0 would be indistinguishable from a legitimate ID of 0 (which is also invalid).
**Why it happens:** `strconv.Atoi` returns the zero value on error. The check `err != nil || matchID <= 0` correctly catches both cases, but a future maintainer might refactor to just `err != nil`.
**How to avoid:** The worktree implementation uses `if err != nil || matchID <= 0` which is correct. Keep this combined check.
**Warning signs:** Non-numeric input like "abc" passes validation (it shouldn't -- 0 is not a valid HLTV match ID).

### Pitfall 5: `roundTripFunc` Is Defined in Multiple Test Files
**What goes wrong:** The `roundTripFunc` helper type (`func(*http.Request) (*http.Response, error)`) that implements `http.RoundTripper` is defined in both `internal/hltv/client_test.go` (package `hltv`) and `internal/provider/events_test.go` (package `provider`). Since they are in different packages, this is correct -- each package gets its own copy. But a developer might try to import one from the other.
**Why it happens:** Go test helpers in different packages cannot share unexported types. The `roundTripFunc` type must be defined locally in each test package.
**How to avoid:** Copy the `roundTripFunc` definition into `internal/provider/demo_test.go` if it's not already visible (it should be, since `demo_test.go` shares the `provider` package with `events_test.go`).
**Warning signs:** Compilation error "undefined: roundTripFunc" in `demo_test.go`.

[ASSUMED: based on Go test package scoping rules and the existing codebase pattern]

## Code Examples

Verified patterns from the worktree reference implementation:

### ParseDemoLink -- Main Parser Function
```go
// Source: .claude/worktrees/bold-heyrovsky-44a33d/internal/hltv/parser/demo.go
func ParseDemoLink(r io.Reader, matchURL string) (domain.DemoLink, error) {
    doc, err := goquery.NewDocumentFromReader(r)
    if err != nil {
        return domain.DemoLink{}, &ParseError{Code: ErrorCodeParse, Area: "demo", Err: err}
    }

    matchID := extractMatchID(matchURL)

    sel := doc.Find("a.demo-link")
    if sel.Length() == 0 {
        return domain.DemoLink{
            MatchID:  matchID,
            MatchURL: matchURL,
        }, &ParseError{Code: ErrorCodeUnavailableData, Area: "demo", Message: "no demo available for this match"}
    }

    href, exists := sel.First().Attr("href")
    if !exists || strings.TrimSpace(href) == "" {
        return domain.DemoLink{
            MatchID:  matchID,
            MatchURL: matchURL,
        }, &ParseError{Code: ErrorCodeUnavailableData, Area: "demo", Message: "no demo available for this match"}
    }

    demoURL := resolveURL(href) // shared helper from events.go, same package

    return domain.DemoLink{
        MatchID:  matchID,
        MatchURL: matchURL,
        DemoURL:  demoURL,
    }, nil
}
```

### Demo Command with Validation
```go
// Source: .claude/worktrees/bold-heyrovsky-44a33d/internal/cli/demo.go
func newDemoCommand(out io.Writer, errOut io.Writer, p provider.DemoProvider) *cobra.Command {
    cmd := &cobra.Command{
        Use:                "demo <match-id>",
        Short:              "Get demo download link for an HLTV match",
        Args:               cobra.ExactArgs(1),
        DisableFlagParsing: true,
        RunE: func(cmd *cobra.Command, args []string) error {
            matchID, err := strconv.Atoi(args[0])
            if err != nil || matchID <= 0 {
                _ = output.WriteErrorJSON(errOut, "validation_error",
                    "match-id must be a positive integer",
                    map[string]any{"arg": args[0]})
                return fmt.Errorf("invalid match-id: %q", args[0])
            }

            link, err := p.GetDemo(cmd.Context(), matchID)
            if err != nil {
                _ = mapDemoError(errOut, err)
                return err
            }

            return output.WriteJSON(out, link, nil)
        },
    }
    return cmd
}
```

### Fake Provider for CLI Tests
```go
// Source: .claude/worktrees/bold-heyrovsky-44a33d/internal/cli/demo_test.go
type fakeDemoProvider struct {
    link domain.DemoLink
    err  error
}

func (f *fakeDemoProvider) GetDemo(ctx context.Context, matchID int) (domain.DemoLink, error) {
    return f.link, f.err
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `httptest.NewServer` for HTTP tests | `roundTripFunc` fake transport | Phase 2 (sandbox constraint) | Tests cannot bind local listeners; must use `http.RoundTripper` fakes |
| Commands access infrastructure directly | Provider middleware layer | Phase 3 D-01 | Commands depend on provider interfaces, not raw Client/parser types |
| `DisableFlagParsing` with plugin subcommands | Same pattern, now documented behavior | Cobra PR #2061 (2024) | `--help` auto-completion suppressed when flag parsing disabled |

**Deprecated/outdated:**
- `httptest.NewServer`: Blocked by sandbox; all new tests must use `roundTripFunc` fake transports.
- Direct `json.NewEncoder` in command handlers: Use `output.WriteJSON` / `output.WriteErrorJSON` consistently.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | CSS selector `a.demo-link` is valid for current HLTV match pages | Common Pitfalls #2 | Parser returns false "no demo" for matches that have demos; needs fixture update and selector change |
| A2 | `DisableFlagParsing: true` is the correct tradeoff vs. `--` separator | Common Pitfalls #1 | Users cannot run `dem demo --help`; may cause confusion if help behavior is expected |
| A3 | The worktree implementation reflects the correct current HLTV HTML structure | Architecture Patterns | If HLTV has changed, both parser and fixtures need updating; the worktree code is a starting point, not a final answer |
| A4 | `resolveURL` helper in `parser/events.go` handles all relative URL patterns HLTV uses for demo links | Standard Stack | Broken demo URLs if HLTV uses a URL format that `net/url.ResolveReference` doesn't handle correctly |
| A5 | No Phase 1 issues with the existing goquery v1.12.0 / Go 1.25+ requirement for the demo parser (already working in Phase 3 for events/results) | Standard Stack | If Phase 1/2 used a different goquery version, the dependency would need updating |
| A6 | `DemoURL` field with `omitempty` JSON tag is the correct JSON output contract: empty string omitted from JSON | Architecture Patterns | Scripts checking `data.demo_url` would not find the key at all; they'd need to check key existence, not value |

## Open Questions (RESOLVED)

1. **Is the `.demo-link` CSS selector valid for current live HLTV match pages?**
   - **RESOLVED:** No. The user provided actual live HLTV match page HTML on 2026-05-02. The live structure uses `[data-demo-link]` attribute (primary) and `[data-manuel-download]` attribute (fallback). The `a.demo-link` CSS class does NOT exist in the current HLTV markup. Plans use `doc.Find("[data-demo-link]")` as primary selector and `doc.Find("[data-manuel-download]")` as fallback. Fixtures updated accordingly.

2. **Should the demo command support `--help` at the subcommand level?**
   - **RESOLVED:** D-07 from CONTEXT.md locks `DisableFlagParsing: true` — the command takes zero flags. Users use `dem help demo` for help. This is the accepted tradeoff.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go | Compilation, tests | Yes | 1.26.2 | -- |
| goquery | HTML parsing | Yes (in go.mod) | v1.12.0 | -- |
| Cobra | CLI framework | Yes (in go.mod) | v1.10.1 | -- |
| HLTV.org (network) | Live demo lookup | Not tested | -- | Fixture-based tests cover parser logic without network |

**Missing dependencies with no fallback:**
- None -- all build and test dependencies are already in go.mod.

**Missing dependencies with fallback:**
- None.

**Step 2.6 result:** Environment audit complete. No missing dependencies. The phase requires no external services beyond what the existing codebase already interacts with (HLTV.org, accessed via `Client.Fetch`).

## Security Domain

> security_enforcement: true (from .planning/config.json)
> ASVS level: 1

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | No authentication in this phase |
| V3 Session Management | No | No sessions in this phase |
| V4 Access Control | No | No access control in this phase |
| V5 Input Validation | Yes | `strconv.Atoi` + `> 0` check; validation before any network access |
| V6 Cryptography | No | No cryptographic operations in this phase |

### Known Threat Patterns for Go CLI with HTML Parsing

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Malicious HTML input causing parser panic | Denial of Service | goquery handles malformed HTML gracefully via `golang.org/x/net/html`; empty/invalid input returns `ErrorCodeParse` |
| Extremely large match page causing memory exhaustion | Denial of Service | `Client.Fetch` reads entire body via `io.ReadAll` with 15-second timeout; no explicit body size limit, but the 15s timeout limits practical attack window |
| Command injection via match ID | Tampering | `strconv.Atoi` ensures only integers are passed to `MatchURL`; no shell execution |
| URL injection via manipulated demo href | Spoofing | `resolveURL` uses `net/url.ResolveReference` with a hardcoded base URL of `https://www.hltv.org`; arbitrary schemas/hosts in relative hrefs are rejected |

## Sources

### Primary (HIGH confidence)
- `.claude/worktrees/bold-heyrovsky-44a33d/` -- Complete reference implementation of parser, provider, and CLI command [VERIFIED: local filesystem]
- `internal/domain/models.go` -- Existing `DemoLink` struct with JSON tags [VERIFIED: local filesystem]
- `internal/hltv/urls.go` -- Existing `MatchURL(matchID int)` function [VERIFIED: local filesystem]
- `internal/hltv/client.go` -- Existing `Client.Fetch` with timeout, user-agent, TLS fingerprinting [VERIFIED: local filesystem]
- `internal/hltv/parser/events.go` -- Existing `resolveURL` helper and parser patterns [VERIFIED: local filesystem]
- `internal/hltv/parser/errors.go` -- Existing `ParseError` taxonomy with `ErrorCodeParse` and `ErrorCodeUnavailableData` [VERIFIED: local filesystem]
- `internal/output/json.go`, `error.go` -- Existing JSON output helpers [VERIFIED: local filesystem]
- `internal/provider/events.go` -- Canonical provider pattern (EventsProvider) [VERIFIED: local filesystem]
- `internal/cli/events.go` -- Canonical command pattern (newEventsCommand) [VERIFIED: local filesystem]
- `internal/cli/root.go` -- Command registration point [VERIFIED: local filesystem]
- `go.mod` -- Verified package versions: Go 1.25.0, Cobra v1.10.1, goquery v1.12.0 [VERIFIED: local filesystem]

### Secondary (MEDIUM confidence)
- GitHub: spf13/cobra PR #2061 -- `DisableFlagParsing` + `--help` interaction [CITED: github.com/spf13/cobra/pull/2061]
- GitHub: spf13/cobra issue #2060 -- Completion should not complete --help when `DisableFlagParsing==true` [CITED: github.com/spf13/cobra/issues/2060]
- GitHub: gigobyte/HLTV issue #620 -- HLTV demo download link structure changed [CITED: github.com/gigobyte/HLTV/issues/620]
- GitHub: PuerkitoBio/goquery releases -- v1.12.0 requires Go 1.25+ [CITED: github.com/PuerkitoBio/goquery]

### Tertiary (LOW confidence)
- WebSearch: goquery CSS selector syntax -- `.demo-link`, `a.demo-link` are standard CSS selectors [CITED: pkg.go.dev/github.com/PuerkitoBio/goquery]
- WebSearch: Cobra `DisableFlagParsing` plugin patterns [CITED: various blog posts and documentation]

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- All packages already in go.mod; versions verified against local filesystem.
- Architecture: HIGH -- Pattern is identical to Phase 3 EventsProvider/ResultsProvider; complete reference implementation exists in worktree.
- Pitfalls: MEDIUM -- HLTV CSS selector staleness is a real concern (gigobyte/HLTV #620); `DisableFlagParsing` tradeoff is well-understood.
- Test patterns: HIGH -- `roundTripFunc` is already established; all test patterns match existing code.

**Research date:** 2026-05-02
**Valid until:** 2026-05-16 (14 days -- HLTV markup may change; the CSS selector assumption should be validated with a live smoke test during implementation)

**Key insight for the planner:** The worktree at `.claude/worktrees/bold-heyrovsky-44a33d/` contains a complete, tested implementation. The plan can source directly from these files rather than designing from scratch. The 2-plan split from ROADMAP (04-01: parser + fixtures; 04-02: provider + CLI + root wiring) is natural and should be followed. Plan 04-01 must create the testdata fixtures from the worktree, since they are absent from the current working tree despite what 02-02-SUMMARY.md claims about having created them.

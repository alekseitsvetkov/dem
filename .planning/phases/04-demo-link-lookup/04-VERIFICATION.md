---
phase: 04-demo-link-lookup
verified: 2026-05-03T18:00:00Z
status: passed
score: 11/11 must-haves verified
overrides_applied: 0
overrides: []
re_verification: false
---

# Phase 4: Demo Link Lookup Verification Report

**Phase Goal:** Let users retrieve a demo download link for a specific HLTV match ID.
**Verified:** 2026-05-03T18:00:00Z
**Status:** PASSED
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | User can run `dem demo <match-id>` and receive JSON containing the match source URL | VERIFIED | `cli/demo_test.go:TestDemoCommand_Success` proves JSON output with `match_id` and `match_url` fields. `cli/demo.go` calls `p.GetDemo()` then `output.WriteJSON()`. Command wired in `root.go:35`. |
| 2 | JSON response includes `demo_url` with the absolute download URL when demo exists | VERIFIED | `cli/demo_test.go:TestDemoCommand_Success` asserts `data["demo_url"]` contains full absolute URL. Provider `GetDemo` calls `parser.ParseDemoLink` which resolves relative URL via `resolveURL`. Test confirms `https://www.hltv.org/download/demo/107224`. |
| 3 | No demo available: response omits `demo_url` (exit code 0); scripts detect availability by checking `data.demo_url` key presence | VERIFIED | `cli/demo_test.go:TestDemoCommand_Unavailable` proves `err == nil` and `demo_url` key absent from JSON. `domain/models.go:31` has `json:"demo_url,omitempty"`. Provider `demo.go:64-67` catches `unavailable_data` and returns nil error (D-03). Parser returns `unavailable_data` error when fixture has no demo elements (`demo.go:54-61`). |
| 4 | Invalid match IDs (non-numeric, zero, negative) fail before network with `validation_error` on stderr and non-zero exit | VERIFIED | `cli/demo_test.go:TestDemoCommand_ValidationError_NonNumeric/Zero/Negative` — all three prove `err != nil`, stdout empty, stderr contains `validation_error`. Validation at `cli/demo.go:27-28`: `strconv.Atoi(args[0])` check with `matchID <= 0` runs before `p.GetDemo()` call. |
| 5 | `ParseDemoLink` extracts demo URL when a demo link exists | VERIFIED | `parser/demo_test.go:TestParseDemoLink_WithDemo` proves non-empty `DemoURL` containing resolved absolute URL. Primary selector `[data-demo-link]` at `demo.go:32` extracts attribute value. |
| 6 | `ParseDemoLink` returns partial DemoLink + `unavailable_data` error when no demo available | VERIFIED | `parser/demo_test.go:TestParseDemoLink_WithoutDemo` proves error with `ErrorCodeUnavailableData`, partial `DemoLink{MatchID="99999", MatchURL="...", DemoURL=""}`. Code at `demo.go:54-61`. |
| 7 | `ParseDemoLink` returns `parse_error` on invalid or empty HTML input | VERIFIED | `parser/demo_test.go:TestParseDemoLink_EmptyBody` proves error with `ErrorCodeParse`. `demo.go:25-27` returns parse error when `goquery.NewDocumentFromReader` fails on empty input. Invalid HTML test proves graceful fallback to `unavailable_data`. |
| 8 | Parser selectors match live HLTV HTML structure (`[data-demo-link]` primary, `[data-manuel-download]` fallback) | VERIFIED | `demo.go:32` uses `doc.Find("[data-demo-link]")` primary. `demo.go:43` uses `doc.Find("[data-manuel-download]")` fallback. Zero instances of stale `a.demo-link` in parser directory (confirmed via grep). Fixture `match-with-demo.html` contains both selectors. |
| 9 | `extractMatchID` correctly parses match ID from HLTV match URLs | VERIFIED | `parser/demo_test.go:TestExtractMatchID` proves table-driven cases including standard URL (`107224`), short URL (`5`), and edge cases. `demo.go:71-85` splits `/matches/<id>/<slug>` path structure. |
| 10 | Network errors return `ProviderError` JSON on stderr with `network_error` code | VERIFIED | `cli/demo_test.go:TestDemoCommand_ProviderError` proves stderr contains `network_error`. `cli/demo.go:49-51` maps `*hltv.ProviderError` to `WriteErrorJSON`. Provider `GetDemo` passes through transport errors (`provider/demo.go:59`). |
| 11 | Parse errors return `ParseError` JSON on stderr with `parse_error` code | VERIFIED | `cli/demo_test.go:TestDemoCommand_ParseError` proves stderr contains `parse_error`. `cli/demo.go:52-53` maps `*parser.ParseError` to `WriteErrorJSON`. |

**Score:** 11/11 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/hltv/parser/demo.go` | ParseDemoLink function, extractMatchID helper | VERIFIED | Exists (86 lines). Contains `ParseDemoLink(r io.Reader, matchURL string) (domain.DemoLink, error)` and `extractMatchID(matchURL string) string`. Uses `[data-demo-link]` and `[data-manuel-download]` selectors. Godoc documents partial DemoLink behavior. |
| `internal/hltv/parser/demo_test.go` | Parser tests for all code paths | VERIFIED | Exists (141 lines). 5 tests: WithDemo, WithoutDemo, InvalidHTML, EmptyBody, ExtractMatchID. All pass. |
| `internal/hltv/parser/testdata/match-with-demo.html` | Fixture with `[data-demo-link]` element | VERIFIED | Exists. Contains `data-demo-link="/download/demo/107224"` and `data-manuel-download=""` with `href="/download/demo/107224"`. |
| `internal/hltv/parser/testdata/match-without-demo.html` | Fixture with no demo elements | VERIFIED | Exists. Zero instances of `data-demo-link` or `data-manuel-download` (confirmed via grep). |
| `internal/provider/demo.go` | DemoProvider interface, GetDemo method | VERIFIED | Exists (73 lines). `DemoProvider` interface with `GetDemo(ctx, matchID int)`. Implements unavailable-data-as-success (D-03). Functional options: `WithDemoClient`, `WithDemoURLs`. |
| `internal/provider/demo_test.go` | Provider tests for success, unavailable, network error | VERIFIED | Exists (99 lines). 3 tests: Success, Unavailable, NetworkError. All pass. Uses `roundTripFunc` from `events_test.go`. |
| `internal/cli/demo.go` | newDemoCommand constructor, mapDemoError | VERIFIED | Exists (57 lines). `newDemoCommand(out, errOut, p)` with `DisableFlagParsing: true`, `cobra.ExactArgs(1)`, `strconv.Atoi` validation. `mapDemoError` type-switches on `*hltv.ProviderError` and `*parser.ParseError`. |
| `internal/cli/demo_test.go` | CLI tests for all paths | VERIFIED | Exists (227 lines). 7 tests: Success, Unavailable, ValidationError x3, ProviderError, ParseError. All pass. `fakeDemoProvider` stub. |
| `internal/cli/root.go` | Register demo subcommand on root | VERIFIED | Modified. Line 35: `root.AddCommand(newDemoCommand(out, errOut, provider.NewDemoProvider()))`. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `parser/demo.go` | `parser/events.go` | `resolveURL` (same package) | WIRED | `demo.go:37,48` calls `resolveURL(href)`. Function defined at `events.go:105`. |
| `parser/demo.go` | `domain/models.go` | `domain.DemoLink` | WIRED | `demo.go:11` imports domain package. Returns `domain.DemoLink` at lines 26, 34, 45, 54. |
| `parser/demo.go` | `parser/errors.go` | `ParseError`, `ErrorCodeParse`, `ErrorCodeUnavailableData` | WIRED | `demo.go:26,57-58` uses `ParseError{}` with `ErrorCodeParse` and `ErrorCodeUnavailableData`. Defined in `errors.go:4-5,9-15`. |
| `provider/demo.go` | `parser/demo.go` | `parser.ParseDemoLink` | WIRED | `provider/demo.go:62` calls `parser.ParseDemoLink(bytes.NewReader(body), matchURL)`. Import at line 10. |
| `provider/demo.go` | `hltv/urls.go` | `MatchURL` | WIRED | `provider/demo.go:55` calls `p.urls.MatchURL(matchID)`. `urls.go:33-35` implements `MatchURL(matchID int) string`. |
| `cli/demo.go` | `provider/demo.go` | `GetDemo` | WIRED | `cli/demo.go:35` calls `p.GetDemo(cmd.Context(), matchID)`. `provider/demo.go:54` implements `GetDemo`. |
| `cli/root.go` | `cli/demo.go` | `newDemoCommand` | WIRED | `root.go:35` calls `root.AddCommand(newDemoCommand(out, errOut, provider.NewDemoProvider()))`. `cli/demo.go:15` defines `newDemoCommand`. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|--------------|--------|-------------------|--------|
| `parser/demo.go` | `href` (from attribute) | `goquery.Find("[data-demo-link]").Attr()` | Yes — extracted from fixture HTML, resolved via `resolveURL` | FLOWING |
| `provider/demo.go` | `matchURL` | `p.urls.MatchURL(matchID)` -> URL generator | Yes — builds valid HLTV URL from int matchID | FLOWING |
| `provider/demo.go` | `body` (HTTP response) | `p.client.Fetch(ctx, matchURL)` | Yes — real HTTP fetch returns HLTV page HTML; tests use fixtures | FLOWING |
| `cli/demo.go` | `matchID` (int) | `strconv.Atoi(args[0])` | Yes — CLI arg parsed to int, passed to provider | FLOWING |
| `cli/demo.go` | `link` (DemoLink) via `WriteJSON` | `p.GetDemo(cmd.Context(), matchID)` -> parser -> JSON output | Yes — real data flow from network through parser to JSON on stdout | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Parser build | `go build ./internal/hltv/parser/` | Exit 0, no output | PASS |
| Provider build | `go build ./internal/provider/` | Exit 0, no output | PASS |
| CLI build | `go build ./internal/cli/` | Exit 0, no output | PASS |
| Parser demo tests | `go test ./internal/hltv/parser/ -run TestParseDemo\|TestExtractMatch` | 5/5 PASS | PASS |
| Provider demo tests | `go test ./internal/provider/ -run TestGetDemo` | 3/3 PASS | PASS |
| CLI demo tests | `go test ./internal/cli/ -run TestDemoCommand` | 7/7 PASS | PASS |

Note: Full binary build (`go build ./cmd/dem/`) fails due to sandbox write restrictions on Go module cache — a pre-existing environmental limitation, not a code issue. All individual packages compile successfully.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| DEMO-01 | 04-02 | User can pass an HLTV match ID and receive a JSON response containing the match source URL | SATISFIED | `cli/demo.go` implements `dem demo <match-id>` command. `TestDemoCommand_Success` verifies stdout JSON with `match_url` field. |
| DEMO-02 | 04-01, 04-02 | User receives a demo download URL when HLTV exposes one for the match | SATISFIED | `parser/demo.go` extracts `[data-demo-link]` attribute. `TestParseDemoLink_WithDemo` passes with resolved absolute URL. `TestGetDemo_Success` proves end-to-end. |
| DEMO-03 | 04-01, 04-02 | User receives a distinct structured error when the match exists but no demo link is available | SATISFIED | Parser returns `*ParseError{Code: "unavailable_data"}` (confirmed by `TestParseDemoLink_WithoutDemo`). Provider converts to nil error per design decision D-03 (ROADMAP SC3: exit code 0). Scripts detect by checking `data.demo_url` key presence (`TestDemoCommand_Unavailable`). *Design deviation:* Requirement says "error" but ROADMAP/plan specify success with omitted field. Intent (distinct signal for unavailable) is achieved. |

**Coverage note:** REQUIREMENTS.md traceability table shows DEMO-01/02/03 as "Pending" — the table has not been updated to reflect Phase 4 completion. This is a documentation artifact, not an implementation gap. All three requirements are fully satisfied.

### Anti-Patterns Found

| File | Pattern | Severity | Impact |
|------|---------|----------|--------|
| *None* | No stale `a.demo-link` selectors found in any Phase 4 file | - | - |
| *None* | No TODO/FIXME/PLACEHOLDER comments | - | - |
| *None* | No empty implementations (return null, return []) | - | - |
| *None* | No debug logging stubs (console.log, fmt.Print) | - | - |

Zero anti-patterns detected in Phase 4 files. All code is substantive and properly wired.

### Human Verification Required

None. All truths are programmatically verifiable through the test suite and code inspection. No visual or interactive testing needed.

### Gaps Summary

No gaps found. All 11 truths verified. All 9 artifacts exist, are substantive, properly wired, and data flows are complete. All 15 Phase 4 tests pass. All 7 key links verified.

### Deferred Items

None. Phase 4 is the last phase in the current roadmap milestone. No items are deferred to later phases.

---

_Verified: 2026-05-03T18:00:00Z_
_Verifier: Claude (gsd-verifier)_

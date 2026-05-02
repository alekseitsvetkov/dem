---
status: testing
phase: 02-hltv-provider-infrastructure
source: 
  - 02-01-SUMMARY.md
  - 02-02-SUMMARY.md
started: 2026-05-02T19:35:00Z
updated: 2026-05-02T19:36:00Z
---

## Current Test

number: 2
name: URL Construction Helpers
expected: |
  NewURLs produces correct EventsURL, ResultsURL, and MatchURL
  from both the default base URL and custom base URLs.
  Trailing slashes on base URLs are trimmed.
awaiting: user response

## Tests

### 1. ProviderError Taxonomy
expected: ProviderError supports network_error/http_error codes, Details() with url/status_code, error interface, and Unwrap.
result: pass

### 2. URL Construction Helpers
expected: NewURLs produces correct EventsURL, ResultsURL, MatchURL values from default and custom base URLs.
result: [pending]

### 3. HTTP Client Construction
expected: NewClient creates client with 15s timeout and dem/dev user-agent. WithHTTPClient and WithUserAgent options inject alternatives.
result: [pending]

### 4. HTTP Client Fetch
expected: Client.Fetch sends User-Agent header, returns body on 2xx, returns ProviderError with http_error code on non-2xx, returns ProviderError with network_error code on transport failure.
result: [pending]

### 5. ParseError Taxonomy
expected: ParseError supports parse_error and unavailable_data codes, Details() with area and field, error interface, and Unwrap.
result: [pending]

### 6. Domain Models
expected: Event, Result, DemoLink types have correct fields with JSON tags. Source URLs resolve through parser-local helper.
result: [pending]

### 7. Events Parser
expected: ParseEvents reads HTML fixture with .event elements, returns typed Event slice with ID, name, dates, location, and source URL. Missing data returns parse_error.
result: [pending]

### 8. Results Parser
expected: ParseResults reads HTML fixture with .result-con elements, returns typed Result slice with match ID, teams, score, event, date, format, source URL. Missing data returns parse_error.
result: [pending]

### 9. Demo Link Parser
expected: ParseDemoLink reads match HTML, returns DemoLink with demo URL when .demo-link or a[href*="/download/demo/"] present. Returns unavailable_data when no demo link found. Missing match-id returns parse_error.
result: [pending]

## Summary

total: 9
passed: 1
issues: 0
pending: 8
skipped: 0
blocked: 0

## Gaps

[none yet]

---
status: complete
phase: 01-cli-foundation
source:
  - .planning/phases/01-cli-foundation/01-01-SUMMARY.md
  - .planning/phases/01-cli-foundation/01-02-SUMMARY.md
started: 2026-05-02T15:47:54Z
updated: 2026-05-02T15:50:12Z
---

## Current Test

[testing complete]

## Tests

### 1. Version Command JSON
expected: Running `go run ./cmd/dem version` exits 0 and prints valid JSON on stdout with top-level `data` and `meta`.
result: pass
observed: |
  {"data":{"name":"dem","version":"dev","commit":"unknown","date":"unknown"},"meta":{}}

### 2. Help Command Is Local
expected: Running `go run ./cmd/dem --help` exits 0, prints help text, and does not require any HLTV network access.
result: pass
observed: |
  Fetch HLTV events, results, and match demo links as JSON

### 3. Unknown Command Emits JSON Error
expected: Running `go run ./cmd/dem does-not-exist` exits non-zero and writes JSON to stderr with top-level `error` and a machine-readable `code`.
result: pass
observed: |
  {"error":{"code":"command_error","message":"unknown command \"does-not-exist\" for \"dem\"","details":{}}}
  exit status 1

### 4. CLI Contract Tests
expected: Running `go test ./...` passes tests for JSON success envelopes, JSON error envelopes, help behavior, unknown command behavior, and subcommand registration.
result: pass
observed: |
  ?   	github.com/alekseitsvetkov/dem/cmd/dem	[no test files]
  ok  	github.com/alekseitsvetkov/dem/internal/cli	(cached)
  ok  	github.com/alekseitsvetkov/dem/internal/output	(cached)

## Summary

total: 4
passed: 4
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

[none - all Phase 1 UAT checks passed]

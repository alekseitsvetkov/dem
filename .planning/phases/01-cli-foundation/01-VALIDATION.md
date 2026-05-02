---
phase: 1
slug: cli-foundation
status: verified
nyquist_compliant: true
wave_0_complete: true
created: 2026-05-02
---

# Phase 1 - Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none - Go standard tooling |
| **Quick run command** | `go test ./internal/output ./internal/cli` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | < 10 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/output ./internal/cli`
- **After every plan wave:** Run `go test ./...`
- **Before `$gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** < 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 1-01-01 | 01 | 1 | CLI-01, CLI-02, CLI-04 | T-1-03 | Entry point delegates to `cli.Execute`; no command handler exits the process. | unit + UAT | `go test ./...`; `go run ./cmd/dem --help` | yes | green |
| 1-01-02 | 01 | 1 | CLI-01, CLI-02, CLI-04 | T-1-01, T-1-05, T-1-06 | Root command is local-only; version command writes JSON envelope. | unit + UAT | `go test ./internal/cli`; `go run ./cmd/dem version`; `go run ./cmd/dem --help` | yes | green |
| 1-01-03 | 01 | 1 | CLI-01, CLI-04 | T-1-01 | `WriteJSON` emits top-level `data` and `meta`; nil meta encodes as `{}`. | unit | `go test ./internal/output` | yes | green |
| 1-02-01 | 02 | 2 | CLI-03 | T-1-02, T-1-04 | `WriteErrorJSON` emits top-level `error`, stable `code`, human message, and object-valued `details`. | unit | `go test ./internal/output` | yes | green |
| 1-02-02 | 02 | 2 | CLI-01, CLI-02, CLI-03, CLI-04 | T-1-02, T-1-03 | CLI execution centralizes non-zero failures and writes JSON errors through `WriteErrorJSON`. | unit + UAT | `go test ./internal/cli`; `go run ./cmd/dem does-not-exist` | yes | green |
| 1-02-03 | 02 | 2 | CLI-01, CLI-02, CLI-03, CLI-04 | T-1-01, T-1-02, T-1-05, T-1-06 | Contract tests cover version output, help behavior, unknown command behavior, and subcommand extension. | unit | `go test ./internal/cli` | yes | green |

*Status: pending - green - red - flaky*

---

## Requirement Coverage

| Requirement | Automated Coverage | UAT Coverage | Status |
|-------------|--------------------|--------------|--------|
| CLI-01 | `TestVersionWritesJSONEnvelope`, `TestWriteJSONWritesEnvelope`, `go test ./...` | `go run ./cmd/dem version` | covered |
| CLI-02 | `TestHelpDoesNotWriteError`, `go test ./...` | `go run ./cmd/dem --help` | covered |
| CLI-03 | `TestUnknownCommandReturnsError`, `TestWriteErrorJSONWritesEnvelope`, `go test ./...` | `go run ./cmd/dem does-not-exist` | covered |
| CLI-04 | `TestAdditionalCommandCanBeRegistered`, `go test ./...` | Covered by extensibility test outcome | covered |

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements.

---

## Manual-Only Verifications

All phase behaviors have automated verification.

---

## Validation Audit 2026-05-02

| Metric | Count |
|--------|-------|
| Requirements audited | 4 |
| Covered | 4 |
| Partial | 0 |
| Missing | 0 |
| Manual-only | 0 |

Evidence:

- `.planning/phases/01-cli-foundation/01-UAT.md` records successful user terminal output for all Phase 1 checks.
- `internal/cli/root_test.go` covers CLI command behavior and extension registration.
- `internal/output/json_test.go` covers success JSON envelopes.
- `internal/output/error_test.go` covers structured error JSON envelopes.

---

## Validation Sign-Off

- [x] All tasks have automated verify commands or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all missing references
- [x] No watch-mode flags
- [x] Feedback latency < 10 seconds
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-05-02

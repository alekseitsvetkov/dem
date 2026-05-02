---
phase: 1
slug: cli-foundation
status: verified
threats_open: 0
asvs_level: 1
created: 2026-05-02
---

# Phase 1 - Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| Terminal stdout | Successful CLI command output consumed by users and scripts. | JSON success payloads; low sensitivity in Phase 1. |
| Terminal stderr | Failed CLI command output consumed by users and scripts. | JSON error payloads; low sensitivity in Phase 1. |
| Process boundary | `main` converts CLI execution result into process exit code. | Integer exit status only. |
| Command registration | Future commands attach to the root Cobra command. | Command metadata and handlers; no external input beyond CLI args in Phase 1. |

---

## Threat Register

| Threat ID | Category | Component | Disposition | Mitigation | Status |
|-----------|----------|-----------|-------------|------------|--------|
| T-1-01 | Tampering | Success output contract | mitigate | `internal/output.WriteJSON` writes all success payloads in a `{data, meta}` JSON envelope; `dem version` uses this helper. UAT confirmed JSON output. | closed |
| T-1-02 | Tampering | Error output contract | mitigate | `internal/output.WriteErrorJSON` writes failures in a top-level `error` envelope; `internal/cli.Execute` routes command errors to stderr. UAT confirmed unknown command emits JSON error. | closed |
| T-1-03 | Repudiation | Process exit behavior | mitigate | `os.Exit` is isolated to `cmd/dem/main.go`; command handlers return errors or status through `cli.Execute`, keeping behavior testable. Source scan found `os.Exit` only in `cmd/dem/main.go`. | closed |
| T-1-04 | Information Disclosure | Error details | mitigate | Error details are explicit maps and nil details encode as `{}`; implementation does not dump environment variables or raw structured state. | closed |
| T-1-05 | Denial of Service | Help/version commands | mitigate | Phase 1 contains no `net/http` or HLTV provider code; help/version are local-only. UAT confirmed help runs without network-facing behavior. | closed |
| T-1-06 | Elevation of Privilege | Future command extension | mitigate | `NewRootCommand` exposes explicit Cobra command registration; tests confirm additional commands can be added without mutating existing handlers. | closed |

*Status: open - closed*
*Disposition: mitigate (implementation required) - accept (documented risk) - transfer (third-party)*

---

## Accepted Risks Log

No accepted risks.

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-05-02 | 6 | 6 | 0 | Codex |

---

## Evidence

- `cmd/dem/main.go` calls `os.Exit(cli.Execute())`.
- `internal/cli/root.go` sets `SilenceUsage: true` and `SilenceErrors: true`.
- `internal/cli/root.go` writes command failures through `output.WriteErrorJSON`.
- `internal/output/json.go` defines `Response` and `WriteJSON`.
- `internal/output/error.go` defines `ErrorResponse` and `WriteErrorJSON`.
- `internal/cli/root_test.go` covers version output, help behavior, unknown command behavior, and command extension registration.
- `.planning/phases/01-cli-foundation/01-UAT.md` records successful user terminal verification.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-05-02

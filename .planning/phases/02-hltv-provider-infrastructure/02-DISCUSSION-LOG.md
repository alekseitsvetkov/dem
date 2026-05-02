# Phase 2: HLTV Provider Infrastructure - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-05-02
**Phase:** 2-HLTV Provider Infrastructure
**Areas discussed:** HTTP request policy, Provider boundary, Fixture strategy, Error taxonomy

---

## HTTP Request Policy

| Option | Description | Selected |
|--------|-------------|----------|
| Conservative defaults | Standard `net/http`, configured timeout, clear user-agent, no automatic retries, low request volume. | ✓ |
| Convenience retries | Add automatic retries/backoff in v1 provider layer. | |
| Defer policy | Leave timeout/user-agent/retry behavior mostly to future command work. | |

**User's choice:** best practices
**Notes:** Selected conservative defaults to keep HLTV access polite, testable, and script-friendly.

---

## Provider Boundary

| Option | Description | Selected |
|--------|-------------|----------|
| Small injectable interfaces | HTTP fetcher/client interfaces plus parser composition; no global mutable client state. | ✓ |
| Direct command fetching | Let command handlers fetch and parse pages directly. | |
| Full typed provider only | Build all typed event/result/demo provider methods immediately in Phase 2. | |

**User's choice:** best practices
**Notes:** Selected a reusable internal boundary that keeps command handlers free of HTTP and selector details while not overbuilding final command behavior early.

---

## Fixture Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Sanitized real-structure fixtures | Checked-in trimmed HLTV-like HTML under `testdata`, covering all v1 page types and edge cases. | ✓ |
| Synthetic-only fixtures | Minimal hand-written HTML snippets for parser tests. | |
| Live parser tests | Fetch current HLTV pages during tests. | |

**User's choice:** best practices
**Notes:** Selected sanitized real-structure fixtures because HLTV markup drift is the main parser risk and tests must not depend on live network access.

---

## Error Taxonomy

| Option | Description | Selected |
|--------|-------------|----------|
| Stable provider/parser categories | Use `network_error`, `http_error`, `parse_error`, and `unavailable_data`, with structured details. | ✓ |
| Single provider error | Collapse all HLTV-layer failures into one generic error code. | |
| Message-only distinction | Keep one code and rely on human-readable messages for details. | |

**User's choice:** best practices
**Notes:** Selected stable machine-readable categories that map cleanly into the Phase 1 error envelope.

---

## the agent's Discretion

- The user deferred Phase 2 implementation details to best-practice defaults.
- Downstream agents may choose exact Go type names, constructor signatures, fixture filenames, and parser helper layout as long as the decisions in `02-CONTEXT.md` are preserved.

## Deferred Ideas

- Automatic retries, caching, concurrent crawling, and rate limiting knobs are deferred until real usage requires them.
- User-facing `events`, `results`, and `demo <match-id>` commands remain Phase 3 and Phase 4 work.

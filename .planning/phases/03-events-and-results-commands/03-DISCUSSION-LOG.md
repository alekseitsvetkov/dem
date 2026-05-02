# Phase 3: Events and Results Commands - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-05-02
**Phase:** 3-Events and Results Commands
**Areas discussed:** Command wiring architecture, Tier 1 filtering, Limit flag behavior, Error surface mapping

---

## Command Wiring Architecture

| Option | Description | Selected |
|--------|-------------|----------|
| Provider middleware layer | Create EventsProvider/ResultsProvider wrapping Client.Fetch + parsers behind injectable interfaces | ✓ |
| Direct composition | Commands directly call Client.Fetch + parser functions | |

**User's choice:** Provider middleware layer (Recommended)
**Notes:** Provider design details deferred to best-practice defaults — injectable constructor, passthrough typed errors, single-call per provider method.

---

## Tier 1 Filtering

| Option | Description | Selected |
|--------|-------------|----------|
| Filter client-side after parse | Parse all events, then filter to matching tier | |
| Filter at provider level | Provider receives tier parameter and filters internally | ✓ |
| Defer filtering to v2 | Skip --tier in v1, filter with jq | |

**User's choice:** Filter at provider level

| Option | Description | Selected |
|--------|-------------|----------|
| String flag | --tier accepts any string value (flexible for "S" tier) | ✓ |
| Integer flag | --tier accepts int only (1, 2, 3) | |
| Enum string | --tier accepts fixed set: "1", "2", "3", "S" | |

**User's choice:** String flag (Recommended)
**Notes:** String type offers future flexibility without constraining v1 to a fixed set.

---

## Limit Flag Behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Client-side truncation | Parse all results, then truncate slice to limit | ✓ |
| Truncate in provider layer | Provider applies limit after filtering | |

**User's choice:** Client-side truncation (Recommended)

| Option | Description | Selected |
|--------|-------------|----------|
| Provider handles limit | Provider interface receives limit, returns bounded data | ✓ |
| Command handler truncates | Command handler applies limit to provider result | |

**User's choice:** Provider handles limit
**Notes:** Combined with the provider middleware architecture — limit flows through the provider interface.

---

## Error Surface Mapping

| Option | Description | Selected |
|--------|-------------|----------|
| Map codes directly | Phase 2 codes (network_error, http_error, parse_error, unavailable_data) map 1:1 to CLI envelope codes | ✓ |
| Simplify to two codes | Map all provider errors to fetch_error, all parser errors to parse_error | |
| Pass through with wrapping | Provider errors carry Phase 2 code/message/details directly | |

**User's choice:** Map codes directly (Recommended)
**Notes:** Scripts get full transparency into the failure type.

---

## Claude's Discretion

- Provider interface method signatures, constructor option names, and package layout.
- Default limit value when `--limit` is not provided.
- Test strategy details (fake transports, table-driven tests, injected writers).
- Exact plan split across the 3 roadmap plans (03-01, 03-02, 03-03).

## Deferred Ideas

None — discussion stayed within phase scope.

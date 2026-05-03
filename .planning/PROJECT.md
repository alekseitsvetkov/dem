# HLTV CLI

## What This Is

HLTV CLI is an extensible Go command-line tool for retrieving Counter-Strike data from HLTV.org and emitting JSON for scripts, automations, and downstream tools. v1.0 ships three commands: `dem events`, `dem results`, and `dem demo <match-id>` — covering Tier 1 event listings, completed match results, and demo link lookup by HLTV match ID.

The project is easy to expand with new commands and providers without rewriting the core CLI contract.

## Core Value

Users can reliably fetch HLTV event, result, and demo-link data as stable JSON from a script-friendly CLI.

## Current State

**Shipped:** v1.0 (2026-05-03) — 4 phases, 9 plans, 3,136 LOC Go
**Current milestone:** v1.1 — Microservice Platform
**Commands:** `dem events --tier 1`, `dem results --limit N`, `dem demo <match-id>`
**Tech stack:** Go 1.25+, Cobra v1.10.1, goquery v1.12.0
**Test coverage:** 14 parser tests, 10 provider tests, 10 CLI tests — all fixture-based, no network dependency
**Known issues:** Phase 3 provider fixture tests need updates; Phase 2 UAT incomplete (1/9 passed)

## Requirements

### Validated

- ✓ CLI written in Go with extensible command architecture — v1.0
- ✓ CLI emits JSON only on stdout for successful commands — v1.0
- ✓ User can list Tier 1 HLTV events — v1.0 (`dem events`)
- ✓ User can list completed match results from HLTV — v1.0 (`dem results`)
- ✓ User can request a demo download link by HLTV match ID — v1.0 (`dem demo`)
- ✓ Network and parsing failures return structured errors and non-zero exit codes — v1.0
- ✓ HLTV fetching and parsing are isolated behind replaceable internal interfaces — v1.0

### Active

- [ ] Monorepo with microservices architecture (Go, Docker Compose)
- [ ] Infrastructure platform: NATS, Minio, Postgres, Redis
- [ ] Tournament polling service — daily Tier 1 check, publish new match demos to download queue
- [ ] Demo download service — consume download jobs, fetch .dem files from HLTV, store in Minio
- [ ] Demo parsing service — consume parse jobs, run demoinfocs-golang, extract game events to Postgres

### Out of Scope

- Live match tracking — v1 focuses on completed and static HLTV data.
- Downloading demo files — v1 returns the demo link only.
- Team/date demo search — v1 accepts a direct HLTV match ID for reliability.
- Non-JSON terminal tables — the CLI is intended for automation-first usage.
- User accounts, authentication, or private HLTV access — v1 uses public HLTV pages only.

## Context

Shipped v1.0 with 3,136 LOC Go. Architecture: Cobra command tree → Provider interfaces → HLTV Client + Parser → goquery HTML extraction. JSON stdout contract with `{data, meta}` envelope; structured errors on stderr with snake_case codes. All tests use fixture HTML and `roundTripFunc` fake transports (sandbox-safe). HLTV selectors validated against live markup during Phase 4 planning.

## Constraints

- **Language**: Go
- **Output contract**: JSON only — stdout must remain machine-readable on success.
- **Input contract**: Demo lookup accepts an HLTV match ID.
- **Data source**: Public HLTV pages — no official API dependency.
- **Extensibility**: Commands and HLTV parsing are modular — new features don't require rewrites.
- **Politeness**: HTTP fetching uses timeouts, clear user-agent, and conservative request volume.

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Build in Go | User selected Go; supports strong CLI binaries and maintainable internal packages. | ✓ Good — cobra + goquery worked well |
| JSON-only output | User selected JSON; supports automation and downstream tools. | ✓ Good — `{data, meta}` envelope consistent across all commands |
| Accept HLTV match ID for demo command | User selected match ID; avoids ambiguous search/disambiguation for v1. | ✓ Good — simple, reliable input contract |
| Keep HLTV access behind provider/parser interfaces | HLTV markup can change; isolation keeps future fixes scoped. | ✓ Good — Phase 2-4 proved pattern works across 3 commands |
| Provider middleware layer between commands and infrastructure | Commands depend on injectable interfaces, not raw Client/parser. | ✓ Good — Phase 3 D-01, all commands follow this |
| Functional options pattern for constructors | Injectable dependencies for test doubles. | ✓ Good — `WithDemoClient`, `WithDemoURLs` across all providers |
| Live HLTV selectors over stale CSS classes | User provided actual HLTV HTML; `[data-demo-link]` + `[data-manuel-download]` confirmed correct. | ✓ Good — validated against live markup |
| `DisableFlagParsing: true` for zero-flag commands | Prevents Cobra from parsing negative match IDs as flags; help via `dem help demo`. | ✓ Good — accepted tradeoff |
| `roundTripFunc` fake transports for HTTP tests | Sandbox blocks `httptest.NewServer`; `http.RoundTripper` fakes work without local listeners. | ✓ Good — all HTTP tests sandbox-safe |

## Evolution

This document evolves at phase transitions and milestone boundaries.

---
*Last updated: 2026-05-03 — v1.1 milestone started*

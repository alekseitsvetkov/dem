# HLTV CLI

## What This Is

HLTV CLI is an extensible Go command-line tool and microservice platform for retrieving Counter-Strike data from HLTV.org. It emits JSON for scripts, automations, and downstream tools, and automatically acquires and parses CS2 demo files at scale.

v1.0 ships the CLI: `dem events`, `dem results`, `dem demo <match-id>` — covering Tier 1 event listings, completed match results, and demo link lookup.

v1.1 adds an automated pipeline: Poller discovers matches → Downloader streams demos to MinIO → Parser extracts game events to Postgres. The system is idempotent end-to-end.

## Core Value

Users can reliably fetch HLTV data as stable JSON from a script-friendly CLI, and automatically acquire and parse CS2 demo files into a queryable PostgreSQL database.

## Current State

**Shipped:** v1.0 (2026-05-03) + v1.1 (2026-05-06) — 6 phases, 16 plans, ~16,000 LOC Go
**Tech stack:** Go 1.26+, Cobra v1.10.1, goquery v1.12.0, demoinfocs-golang v5, NATS JetStream, PostgreSQL 17, MinIO, pgxpool, uTLS, cloudscraper (Python)
**Infrastructure:** Docker Compose (NATS, PostgreSQL, MinIO, Redis), single `go.mod` monorepo
**Test coverage:** 14 parser tests, 10 provider tests, 10 CLI tests, 9 infra package tests
**Known issues:** Phase 3 provider fixture tests need updates; downloader runs natively (Docker Desktop VM IP blocked by R2 CDN)

## Requirements

### Validated

- ✓ CLI written in Go with extensible command architecture — v1.0
- ✓ CLI emits JSON only on stdout for successful commands — v1.0
- ✓ User can list Tier 1 HLTV events — v1.0 (`dem events`)
- ✓ User can list completed match results from HLTV — v1.0 (`dem results`)
- ✓ User can request a demo download link by HLTV match ID — v1.0 (`dem demo`)
- ✓ Network and parsing failures return structured errors and non-zero exit codes — v1.0
- ✓ HLTV fetching and parsing are isolated behind replaceable internal interfaces — v1.0
- ✓ Monorepo with microservices architecture (Go, Docker Compose) — v1.1
- ✓ Infrastructure platform: NATS, Minio, Postgres, Redis — v1.1
- ✓ Tournament polling service — daily Tier 1 check, publish new match demos to download queue — v1.1
- ✓ Demo download service — consume download jobs, fetch .dem files from HLTV, store in Minio — v1.1
- ✓ Demo parsing service — consume parse jobs, run demoinfocs-golang, extract game events to Postgres — v1.1

### Out of Scope

- Live match tracking — v1 focuses on completed and static HLTV data.
- Team/date demo search — v1 accepts a direct HLTV match ID for reliability.
- Non-JSON terminal tables — the CLI is intended for automation-first usage.
- User accounts, authentication, or private HLTV access — v1 uses public HLTV pages only.
- Web UI / dashboard — Connect Metabase/Superset to Postgres instead.
- Live CSTV+ match parsing — Fundamentally different architecture, separate project.
- Prometheus/Grafana metrics — Deferred to v2.
- Query API for parsed data — Deferred to v2.
- Grenade analytics (spatial, heatmaps) — Deferred to v2.

## Context

Shipped v1.0 + v1.1 with ~16,000 LOC Go across 6 phases. Architecture: Cobra command tree → Provider interfaces → HLTV Client + Parser → goquery HTML extraction (v1.0), plus NATS JetStream pub/sub → cloudscraper download → MinIO storage → demoinfocs parsing → pgxpool Postgres inserts (v1.1). JSON stdout contract with `{data, meta}` envelope for CLI; `log/slog` structured logging for services. All tests use fixture HTML and `roundTripFunc` fake transports (sandbox-safe). Pipeline proven end-to-end with 616 kills, 2,396 damage events across 5 matches. Demo files auto-deleted from MinIO after successful parse.

## Constraints

- **Language**: Go
- **Output contract**: CLI: JSON only — stdout must remain machine-readable. Services: slog JSON.
- **Input contract**: Demo lookup accepts an HLTV match ID.
- **Data source**: Public HLTV pages — no official API dependency.
- **Extensibility**: Commands and HLTV parsing are modular — new features don't require rewrites.
- **Politeness**: HTTP fetching uses timeouts, clear user-agent, conservative request volume, 2s delay between match page fetches.
- **Idempotency**: All database writes use `INSERT ... ON CONFLICT DO NOTHING` with deterministic IDs.
- **Memory safety**: Streaming everywhere — no `io.ReadAll` on demo files. `defer p.Close()` mandatory on every demoinfocs parser.

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Build in Go | User selected Go; supports strong CLI binaries and maintainable internal packages. | ✓ Good — cobra + goquery worked well |
| JSON-only output | User selected JSON; supports automation and downstream tools. | ✓ Good — `{data, meta}` envelope consistent across all commands |
| Accept HLTV match ID for demo command | User selected match ID; avoids ambiguous search/disambiguation for v1. | ✓ Good — simple, reliable input contract |
| Keep HLTV access behind provider/parser interfaces | HLTV markup can change; isolation keeps future fixes scoped. | ✓ Good — pattern proven across all commands |
| Functional options pattern for constructors | Injectable dependencies for test doubles. | ✓ Good — carried from v1.0 through v1.1 services |
| `roundTripFunc` fake transports for HTTP tests | Sandbox-safe HTTP testing. | ✓ Good — all HTTP tests sandbox-safe |
| Single `go.mod` monorepo | Services compile alongside CLI; no `go.work` complexity. | ✓ Good — `go build ./cmd/...` works for all 4 entrypoints |
| NATS JetStream for service decoupling | WorkQueue retention, durable pull consumers, at-least-once delivery. | ✓ Good — poller→downloader→parser chain works reliably |
| pgxpool.Pool with MaxConns=20 | Connection pooling created once at startup. | ✓ Good — no connection leaks under parsing load |
| `INSERT ... ON CONFLICT DO NOTHING` | Deterministic event IDs for idempotent writes. | ✓ Good — re-parsing produces zero duplicate rows |
| Python cloudscraper for Cloudflare bypass | Cloudflare Turnstile on demo download page requires JS evaluation. | ✓ Good — subprocess approach works. Native Go was blocked. |
| `defer msg.Ack()` with conditional pattern | Ack only on success, NakWithDelay on failure — never both. | ✓ Good — prevents lost jobs on error paths |
| 7z/unar for RAR extraction | HLTV demos come as .rar archives containing .dem files. | ✓ Good — extracted .dem parses via demoinfocs |
| MinIO object deletion after parse | Demo files re-downloadable; storage is expensive (250-800MB each). | ✓ Good — MinIO stays clean after successful parsing |
| Downloader runs natively (not Docker) | Docker Desktop VM IP blocked by R2 CDN. Native uses real Mac IP. | ⚠ Revisit — acceptable for v1.1, should containerize in v2 |

## Evolution

This document evolves at phase transitions and milestone boundaries.

---
*Last updated: 2026-05-06 — v1.1 milestone shipped*

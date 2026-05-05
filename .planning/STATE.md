---
gsd_state_version: 1.0
milestone: v1.1
milestone_name: Microservice Platform
status: complete
stopped_at: v1.1 milestone shipped 2026-05-06
last_updated: "2026-05-06T02:30:00.000Z"
last_activity: 2026-05-06 — v1.1 milestone completed, all phases verified, UAT passed 9/9
progress:
  total_phases: 2
  completed_phases: 2
  total_plans: 7
  completed_plans: 7
  percent: 100
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-05-06)

**Core value:** Users can reliably fetch HLTV data as stable JSON from a script-friendly CLI, and automatically acquire and parse CS2 demo files into a queryable PostgreSQL database.
**Current focus:** Planning next milestone

## Current Position

**v1.1 Microservice Platform — SHIPPED**

Phase 5: Infrastructure Foundation — complete (3/3 plans)
Phase 6: Pipeline Services — complete (4/4 plans)
UAT: 9/9 passed

Progress: ██████████ 100%

## Performance Metrics

**Velocity (v1.0 baseline):**
- Total plans completed (v1.0): 9
- Total execution time (v1.0): ~30 min

**Velocity (v1.1):**
- Total plans completed (v1.1): 7
- Total execution time (v1.1): ~3 days

## Accumulated Context

### Key Architectural Decisions

- Single `go.mod` monorepo — services added as `cmd/<service>/` entrypoints alongside `cmd/dem/`.
- NATS JetStream for service decoupling — Pub/Sub with WorkQueue retention, at-least-once delivery, durable pull consumers.
- `pgxpool` not `pgx.Conn` — pool created once at startup, passed to all handlers, MaxConns 20.
- Streaming everywhere — no `io.ReadAll` on demos (50-200MB). HTTP body -> MinIO -> demoinfocs streamed via `io.Reader`.
- Idempotent Postgres inserts from day one — `INSERT ... ON CONFLICT DO NOTHING` with deterministic event IDs.
- `defer msg.Ack()` mandatory in every NATS handler — conditional pattern: Ack on success, NakWithDelay on failure, never both.
- `defer p.Close()` mandatory on every demoinfocs parser — prevents 250MB C-memory leaks per unclosed parser.
- Python cloudscraper for Cloudflare Challenge bypass — HLTV download page requires JS evaluation.
- 7z/unar for RAR extraction — HLTV demos come as .rar archives.
- MinIO auto-cleanup after successful parse — saves 250-800MB per demo.
- Downloader runs natively on macOS — Docker Desktop VM IP blocked by R2 CDN.

### Decisions from v1.1 (now in PROJECT.md Key Decisions)

Full decision log: .planning/PROJECT.md

## Blockers/Concerns

- Downloader native-only constraint (Docker Desktop VM IP blocked by R2 CDN) — revisit in v2.
- MapName is empty from demoinfocs-golang v5 — revisit when library updates.
- Phase 3 provider fixture tests need updates (carried from v1.0).

## Deferred Items

| Category | Item | Status | Deferred At |
|----------|------|--------|-------------|
| Feature | Grenade analytics (clustering, popularity by map/type/team) | Deferred to v2 | v1.1 initial |
| Feature | PostGIS, pgvector, ClickHouse for spatial/vector analytics | Deferred to v2 | v1.1 initial |
| Feature | Prometheus/Grafana metrics pipeline | Deferred to v2 | v1.1 initial |
| Feature | Query API for parsed data (`dem match <id>` returns parsed stats) | Deferred to v2 | v1.1 initial |
| Discovery | Team/date match search | Still deferred | v1.0 initialization |

## Session Continuity

Last session: 2026-05-06
Stopped at: v1.1 milestone shipped — all phases complete, UAT passed
Resume file: None

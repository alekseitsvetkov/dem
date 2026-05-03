---
gsd_state_version: 1.0
milestone: v1.1
milestone_name: Microservice Platform
status: planning
stopped_at: Roadmap created — Phase 5 and 6 defined
last_updated: "2026-05-03T21:26:00.000Z"
last_activity: 2026-05-03 — v1.1 roadmap created
progress:
  total_phases: 2
  completed_phases: 0
  total_plans: 0
  completed_plans: 0
  percent: 0
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-05-03)

**Core value:** Users can reliably fetch HLTV event, result, and demo-link data as stable JSON from a script-friendly CLI, and automatically acquire and parse CS2 demo files at scale.
**Current focus:** Phase 5: Infrastructure Foundation

## Current Position

Phase: 5 of 6 (Infrastructure Foundation)
Plan: 0 of TBD in current phase (roadmap created, plans not yet broken down)
Status: Ready to plan
Last activity: 2026-05-03 — v1.1 roadmap created (Phase 5 and Phase 6)

Progress: [░░░░░░░░░░] 0%

## Performance Metrics

**Velocity (v1.0 baseline):**
- Total plans completed (v1.0): 9
- Total execution time (v1.0): ~30 min

**By Phase (v1.1):**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| - | - | - | - |

*Updated after each plan completion*

## Accumulated Context

### Key Architectural Decisions (v1.1 Research)

- Single `go.mod` monorepo — services added as `cmd/<service>/` entrypoints alongside `cmd/dem/`.
- NATS JetStream for service decoupling — Pub/Sub with WorkQueue retention, at-least-once delivery, durable pull consumers.
- `pgxpool` not `pgx.Conn` — pool created once at startup, passed to all handlers, MaxConns 10-25.
- Streaming everywhere — no `io.ReadAll` on demos (50-200MB). HTTP body -> MinIO -> demoinfocs streamed via `io.Reader`.
- Idempotent Postgres inserts from day one — `INSERT ... ON CONFLICT DO NOTHING` with deterministic event IDs.
- `defer msg.Ack()` mandatory in every NATS handler — non-negotiable to prevent infinite redelivery loops.
- `defer p.Close()` mandatory on every demoinfocs parser — prevents 250MB C-memory leaks per unclosed parser.
- Existing v1.0 code (`internal/hltv`, `internal/provider`, `internal/cli`) is read-only library code — never modified.
- Functional options pattern carried from v1.0 — all external dependencies (NATS, Minio, Postgres, Redis) injectable.
- Structured logging via `log/slog` (stdlib) — no `{data, meta}` JSON envelopes in service output.

### Legacy Decisions (from v1.0)

- Build in Go — selected by user.
- JSON-only output contract for CLI.
- HLTV fetching behind provider/parser interfaces — portable to services.
- `roundTripFunc` fake transports for HTTP tests.
- Live HLTV selectors validated against actual markup.

### Blockers/Concerns

- NATS stream creation order is critical — streams must exist before any publisher starts (silent data loss pitfall).
- demoinfocs-golang event handler lifecycle needs empirical testing during Phase 6 parser implementation.
- HLTV CDN download endpoint behavior unknown — needs testing during Downloader implementation.

## Deferred Items

| Category | Item | Status | Deferred At |
|----------|------|--------|-------------|
| Feature | Grenade analytics (clustering, popularity by map/type/team) | Deferred to v2 | v1.1 start |
| Feature | PostGIS, pgvector, ClickHouse for spatial/vector analytics | Deferred to v2 | v1.1 start |
| Discovery | Team/date match search (from v1.0) | Still deferred | v1.0 initialization |
| Downloads | Direct demo file download CLI (from v1.0) | Now in-scope as service | v1.0 initialization |

## Session Continuity

Last session: 2026-05-03 21:26
Stopped at: v1.1 roadmap created — Phase 5 and 6 defined, ready for `/gsd-plan-phase 5`
Resume file: None

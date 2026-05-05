---
gsd_state_version: 1.0
milestone: v1.1
milestone_name: Microservice Platform
status: planning
stopped_at: Phase 5 plans created — 3 plans in 2 waves, ready for execute-phase
last_updated: "2026-05-03T21:45:00.000Z"
last_activity: 2026-05-03 — Phase 5 planning complete, 3 PLAN.md files created
progress:
  total_phases: 2
  completed_phases: 0
  total_plans: 3
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
Plan: 3 plans created, 0 executed
Status: Plans ready — waiting for execute-phase
Last activity: 2026-05-03 — Phase 5 plans created (05-01: scaffolding, 05-02: docker+migrations, 05-03: shared pkgs)

Progress: [░░░░░░░░░░] 0%

## Performance Metrics

**Velocity (v1.0 baseline):**
- Total plans completed (v1.0): 9
- Total execution time (v1.0): ~30 min

**By Phase (v1.1):**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 5 | 3 | 0/3 | - |
| 6 | 0 | 0/0 | - |

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

### Phase 5 Decisions (from planning)

- D-01: Standard Go project layout — `cmd/poller/`, `cmd/downloader/`, `cmd/parser/` alongside existing `cmd/dem/`. Shared infra at `pkg/natsutil/`, `pkg/minio/`, `pkg/postgres/`. New domain types in `internal/domain/game_events.go`. Existing `internal/` code is read-only.
- D-02: Single root `go.mod` — all services and the existing CLI compile from one module. No `go.work`, no `replace` directives.
- D-03: Migration tool: `golang-migrate/migrate` v4 with pgx driver. UP/DOWN migrations in `sql/migrations/` with numbered prefixes.
- D-04: Six tables: `matches`, `rounds`, `kill_events`, `damage_events`, `players`, `match_players`. All write paths use `INSERT ... ON CONFLICT DO NOTHING` with deterministic event IDs. Primary keys include `match_id` for partition-ready design.
- D-05: Separate JetStream streams per queue: `DEM_DOWNLOAD` for `dem.download.jobs`, `DEM_PARSE` for `dem.parse.jobs`. Each with WorkQueue retention, `MaxDeliver: 3`, explicit Ack, and durable pull consumers.
- D-06: Streams are created programmatically at service startup via `js.AddStream()`. Startup fails fast if a required stream is missing. No manual `nats stream create` commands.
- D-07: Thin connection/pool helpers with functional options. `pkg/natsutil/` wraps `nats.Connect` + `jetstream.New()`. `pkg/minio/` wraps `minio.New()`. `pkg/postgres/` wraps `pgxpool.New()`. Each returns a concrete client/pool — services import and use the underlying library directly for operations.
- D-08: Postgres connections use `pgxpool.Pool` (not `pgx.Conn`), created once at startup, with explicit `MaxConns: 10-25`. Pass pool to all handlers.

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

Last session: 2026-05-03 21:45
Stopped at: Phase 5 plans created — 05-01 (scaffolding), 05-02 (docker+migrations), 05-03 (shared pkgs)
Resume file: None

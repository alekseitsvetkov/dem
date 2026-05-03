# Phase 5: Infrastructure Foundation — Discussion Log

**Phase:** 05-Infrastructure Foundation
**Discussed:** 2026-05-03
**Mode:** discuss (default)

## Areas Discussed

### 1. Monorepo Layout
**Question:** How should the monorepo be organized?
**User:** "best practice"
**Decision:** Standard Go project layout — `cmd/poller/`, `cmd/downloader/`, `cmd/parser/` alongside existing `cmd/dem/`. Shared infra at `pkg/natsutil/`, `pkg/minio/`, `pkg/postgres/`. New domain types in `internal/domain/game_events.go`. Existing `internal/` code is read-only. Single root `go.mod`.

### 2. Database Schema Design
**Question:** Migration tool — golang-migrate/migrate or raw SQL?
**User:** "best practice"
**Decision:** `golang-migrate/migrate` v4 with pgx driver. UP/DOWN migrations in `sql/migrations/`. Six tables: `matches`, `rounds`, `kill_events`, `damage_events`, `players`, `match_players`. All write paths idempotent (`ON CONFLICT DO NOTHING`).

### 3. NATS Configuration
**Question:** Single stream or separate streams per queue?
**User:** "best practice"
**Decision:** Separate JetStream streams per queue — `DEM_DOWNLOAD` for `dem.download.jobs`, `DEM_PARSE` for `dem.parse.jobs`. Each with WorkQueue retention, `MaxDeliver: 3`, durable pull consumers. Streams created programmatically at startup.

### 4. Shared Package APIs
**Question:** Thin wrappers or full abstraction layers?
**User:** "best practice"
**Decision:** Thin connection/pool helpers with functional options. Services use underlying libraries directly for operations. `pkg/natsutil/` wraps connect, `pkg/minio/` wraps client creation, `pkg/postgres/` wraps `pgxpool.New()`. Postgres uses `pgxpool.Pool` with `MaxConns: 10-25`.

## Claude's Discretion Items
- Exact `docker-compose.yml` structure (service names, ports, volumes, health checks)
- Exact table column types and indexes
- Makefile targets and build commands
- `pkg/` function and type names
- Docker image tags for infrastructure services

## Deferred Ideas
None — all in-scope discussion. Deferred items (analytics, observability) already in REQUIREMENTS.md v2.

---
*Discussion logged: 2026-05-03*

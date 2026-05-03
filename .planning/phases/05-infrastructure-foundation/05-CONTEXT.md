# Phase 5: Infrastructure Foundation - Context

**Gathered:** 2026-05-03
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase delivers the platform infrastructure that all pipeline services run on. It produces a working `docker-compose up` that brings up NATS (with JetStream), PostgreSQL 17, MinIO, and Redis — all with health checks. It creates the monorepo scaffolding (new `cmd/` entrypoints, `pkg/` shared packages) alongside the existing v1.0 CLI. It writes and migrates the full v1.1 database schema. It creates programmatic NATS stream provisioning.

No pipeline services are implemented in this phase — just the foundation they require. Phase 6 (Pipeline Services) builds the poller, downloader, and parser on top of this foundation.
</domain>

<decisions>
## Implementation Decisions

### Monorepo Layout
- **D-01:** Standard Go project layout: `cmd/poller/`, `cmd/downloader/`, `cmd/parser/` alongside existing `cmd/dem/`. Shared infrastructure packages at `pkg/natsutil/`, `pkg/minio/`, `pkg/postgres/`. New domain types in `internal/domain/game_events.go`. Existing `internal/` code is read-only.
- **D-02:** Single root `go.mod` — all services and the existing CLI compile from one module. No `go.work`, no `replace` directives.

### Database Schema
- **D-03:** Migration tool: `golang-migrate/migrate` v4 with pgx driver. UP/DOWN migrations in `sql/migrations/` with numbered prefixes.
- **D-04:** Six tables: `matches`, `rounds`, `kill_events`, `damage_events`, `players`, `match_players`. All write paths use `INSERT ... ON CONFLICT DO NOTHING` with deterministic event IDs. Primary keys include `match_id` for partition-ready design.

### NATS Configuration
- **D-05:** Separate JetStream streams per queue: `DEM_DOWNLOAD` for `dem.download.jobs`, `DEM_PARSE` for `dem.parse.jobs`. Each with WorkQueue retention, `MaxDeliver: 3`, explicit Ack, and durable pull consumers.
- **D-06:** Streams are created programmatically at service startup via `js.AddStream()`. Startup fails fast if a required stream is missing. No manual `nats stream create` commands.

### Shared Package APIs
- **D-07:** Thin connection/pool helpers with functional options. `pkg/natsutil/` wraps `nats.Connect` + `jetstream.New()`. `pkg/minio/` wraps `minio.New()`. `pkg/postgres/` wraps `pgxpool.New()`. Each returns a concrete client/pool — services import and use the underlying library directly for operations.
- **D-08:** Postgres connections use `pgxpool.Pool` (not `pgx.Conn`), created once at startup, with explicit `MaxConns: 10-25`. Pass pool to all handlers.

### Cross-Cutting (carried from research, binding)
- Docker Compose as the orchestration layer — no Kubernetes.
- `log/slog` for all service logging — no `{data, meta}` JSON envelopes from v1.0 CLI in service output.
- All external dependencies injectable via functional options.
- Existing v1.0 code (`internal/hltv`, `internal/provider`, `internal/cli`, `internal/domain/models.go`) is NOT modified.
- Streaming patterns (no `io.ReadAll` on large files) — but actual streaming code lives in Phase 6 services.

### Claude's Discretion
- Exact `docker-compose.yml` structure — service names, port mappings, volume mounts, health check commands.
- Exact table column types and index definitions — planner derives from research schema recommendations.
- Makefile targets and build commands.
- `pkg/` function and type names — planner chooses idiomatic Go names.
- Docker image tags for NATS, PostgreSQL, MinIO, Redis.
</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project Scope
- `.planning/PROJECT.md` — Project identity, constraints, v1.0 shipped state, v1.1 goals.
- `.planning/REQUIREMENTS.md` — Phase 5 requirements INFR-01 through INFR-06.
- `.planning/ROADMAP.md` — Phase 5 boundary, success criteria (5 items), depends on Phase 4.

### Research (comprehensive, MUST read)
- `.planning/research/SUMMARY.md` — Executive summary, phase recommendations, pitfall-to-phase mapping.
- `.planning/research/STACK.md` — Library versions (nats.go v1.51.0, pgx v5.9.2, minio-go v7, go-redis v9, golang-migrate v4), alternatives rejected, integration points.
- `.planning/research/ARCHITECTURE.md` — Monorepo layout, service boundaries, data flow, NATS subject design, Postgres schema design.
- `.planning/research/PITFALLS.md` — Critical pitfalls 1-5 with prevention strategies (stream creation order, msg.Ack, streaming vs buffering, pgxpool vs pgx.Conn, idempotency).

### v1.0 Context (patterns and conventions)
- `.planning/phases/01-cli-foundation/01-CONTEXT.md` — CLI command shape, JSON envelope, error envelope, injectable writers.
- `.planning/phases/03-events-and-results-commands/03-CONTEXT.md` — Provider pattern, functional options, error mapping pattern.

### Project Conventions
- `AGENTS.md` — Project-specific guidelines and architecture expectations (if exists).

### External References
- NATS JetStream + Go: https://github.com/nats-io/nats.go
- pgx v5: https://pkg.go.dev/github.com/jackc/pgx/v5
- golang-migrate: https://github.com/golang-migrate/migrate
- minio-go v7: https://pkg.go.dev/github.com/minio/minio-go/v7
- Docker Compose: https://docs.docker.com/compose/
</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/hltv/client.go` — `Client.Fetch(ctx, url)` — injectable HTTP client with timeout, user-agent, TLS config. Will be used by Phase 6 poller and downloader.
- `internal/hltv/parser/` — `ParseEvents`, `ParseResults`, `ParseDemoLink` — all used by Phase 6 poller for match discovery.
- `internal/domain/models.go` — Existing `Event`, `Result`, `DemoLink` types with JSON tags. Phase 6 poller depends on these.
- `internal/output/` — `WriteJSON[T]`, `WriteErrorJSON` — CLI output helpers (not used by services, which use slog).

### Established Patterns
- **Functional options constructors**: `NewXxxProvider(opts ...XxxProviderOption)` — used for all providers in v1.0. Carried forward to all Phase 5 shared packages (`NewNATSConn`, `NewMinioClient`, `NewPostgresPool`).
- **Interface-based dependency injection**: Commands depend on interfaces, concrete implementations injected at construction. Carried forward: services will depend on NATS/Mino/Postgres interfaces where testability matters.
- **`roundTripFunc` fake transports**: Used in v1.0 HTTP tests to avoid `httptest.NewServer` (sandbox constraint). Pattern: `type roundTripFunc func(*http.Request) (*http.Response, error)` implementing `http.RoundTripper`.
- **Go module conventions**: Single `go.mod`, standard `cmd/` and `internal/` layout.

### Integration Points
- `cmd/dem/main.go` — Existing CLI entrypoint. New `cmd/poller/main.go`, `cmd/downloader/main.go`, `cmd/parser/main.go` are peers.
- `go.mod` — Add new dependencies: `nats.go`, `pgx/v5`, `minio-go/v7`, `go-redis/v9`, `golang-migrate/v4`.
- Root `Makefile` or `Taskfile` — Add build targets for new services.
</code_context>

<specifics>
## Specific Ideas

- The user consistently defers to "best practice" — Go community standards, library author recommendations, and production-tested patterns should be followed without deviation.
- The monorepo should feel like a natural extension of the existing v1.0 project, not a separate codebase bolted on.
- Phase 5 is infrastructure-only — it should produce a clean `docker-compose up` and `go build ./cmd/...` experience with nothing else. No business logic.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope. All deferred items (grenade analytics, Prometheus, Grafana, query API) are already recorded in REQUIREMENTS.md v2 section.
</deferred>

---

*Phase: 05-Infrastructure Foundation*
*Context gathered: 2026-05-03*

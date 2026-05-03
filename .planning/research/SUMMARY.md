# Project Research Summary

**Project:** HLTV CLI v1.1 -- Microservice Platform
**Domain:** CS2 demo pipeline -- tournament polling, demo download, demo parsing
**Researched:** 2026-05-03
**Confidence:** HIGH

## Executive Summary

This project extends an existing Go CLI tool (v1.0, already at 3,136 LOC) into an automated microservice platform for CS2 demo acquisition and parsing. The system polls HLTV.org for newly completed Tier 1 tournament matches, downloads 50-200 MB `.dem.gz` replay files, parses them into structured game events (kills, rounds, grenades, bomb events), and stores everything in a queryable PostgreSQL database. This is the same problem space occupied by Leetify, scope.gg, and HLTV.org's own infrastructure -- but built as a self-hosted, open-source Go platform rather than a commercial SaaS.

The recommended approach is a **NATS JetStream-backed event-driven pipeline** with three decoupled Go microservices (tournament poller, demo downloader, demo parser) running alongside Docker Compose-managed infrastructure. This architecture is opinionated: NATS decouples services entirely so each can scale, fail, and recover independently. Unlike the industry-common monolithic cron script pattern (HLTVDemoDownloader, hltv-utility-api), the JetStream work-queue approach provides at-least-once delivery, automatic retry, and dead-letter queuing -- turning a brittle one-shot script into a resilient platform. Every decision flows from the constraint that HLTV is a publicly scraped website, not an API: polling must be polite (default daily, minimum 6 hours), demo links expire, and demos cannot be re-downloaded from source after the fact.

The critical risks are: (1) publishing NATS messages before creating backing JetStream streams causes silent data loss with zero errors, (2) missing `msg.Ack()` in consumer handlers triggers infinite message redelivery loops, (3) buffering 100MB+ demo files in memory instead of streaming causes OOM kills, and (4) at-least-once delivery without idempotent Postgres inserts produces duplicate game events that silently corrupt analytical queries. Each risk has a well-defined prevention strategy documented in PITFALLS.md and must be addressed in the phase where it first appears -- never retrofitted.

## Key Findings

### Recommended Stack

The entire platform builds on a single Go 1.25.0 module (the existing `go.mod`). New microservices are added as additional entrypoints under a unified monorepo structure. Infrastructure runs in Docker Compose (NATS 2.12 with JetStream, PostgreSQL 17, MinIO, Redis 8); Go services run natively on the host during development for fast iteration.

**Core technologies:**
- **Go 1.25.0** (already locked in go.mod): Brings `testing/synctest` GA for concurrent test support, `log/slog` structured logging in stdlib, and container-aware GOMAXPROCS. No reason to change.
- **NATS 2.12 with JetStream**: Cloud-native message broker chosen over RabbitMQ (Erlang ops burden) and Kafka (JVM, overkill for 3 services). JetStream provides at-least-once delivery, durable consumers, stream replay, and dead-letter queuing -- exactly what a demo pipeline needs. Single Go binary, no ZooKeeper.
- **PostgreSQL 17 with pgx v5.9.2**: Relational store for parsed game events. Chosen over NoSQL because CS2 event data is inherently relational (players in matches, rounds in matches, kills per round). pgx chosen over `database/sql` + `lib/pq` because it speaks PostgreSQL wire protocol directly (2-5x faster) and exposes PG-specific features (COPY protocol, LISTEN/NOTIFY, native type mapping).
- **MinIO with minio-go v7**: Object storage for `.dem.gz` binary files. Self-hosted avoids cloud vendor lock-in, works identically in dev and production. S3 API compatibility means switching to AWS S3 requires only a config change.
- **Redis 8 with go-redis v9**: Lightweight deduplication cache for tracking processed match IDs. TTL-based auto-expiry prevents unbounded growth.
- **demoinfocs-golang v5.2.0**: The only production-grade Go CS2 demo parser. Used by HLTV.org, Leetify, scope.gg. Provides 70+ typed game events, automatic Source 1/2 detection. Parses ~25 minutes of gameplay per second, uses ~250 MB RAM per demo.
- **log/slog (Go stdlib)**: Structured logging chosen over zerolog and zap. Ships with Go 1.21+, has a handler interface for extensibility, and adds no dependency. The project's throughput (hundreds of events per day, not millions per second) does not justify zerolog's allocation optimization.

**What was explicitly rejected:**
- GORM (ORM): generates inefficient queries for complex CS2 event relationships. Raw SQL with pgx gives full control.
- gRPC: adds protobuf compilation complexity. NATS pub/sub handles fire-and-forget pipeline messages more naturally.
- Kubernetes: massive operational overhead for a 3-service platform. Docker Compose is the right tool for this scale.
- `lib/pq`: in maintenance mode. pgx is the community standard for new Go projects.
- `database/sql` interface layer over pgx: strips PostgreSQL-specific features (COPY protocol, LISTEN/NOTIFY, native types).

### Expected Features

**Must have (P1 -- table stakes, the platform doesn't function without these):**
- **Scheduled Tier 1 tournament polling** -- Cron-driven daily check against HLTV events/results pages. Reuses v1.0 `internal/hltv` parsers (`ParseEvents`, `ParseResults`, `ParseDemoLink`). Deduplicates against already-seen match IDs.
- **Demo download from HLTV to MinIO** -- HTTP download of `.dem.gz` files (50-200 MB), streaming directly to MinIO without writing to local disk. Retries on transient failures.
- **Demo parsing with demoinfocs-golang into Postgres** -- Core data product. Parses downloaded demos into kills, rounds, grenades, bomb events, and player data. Batch-inserts per round to manage write volume (~1,200 rows per match for kills alone).
- **Docker Compose orchestration** -- Single `docker-compose up` starts NATS, Postgres, MinIO, Redis, and all three services with correct startup ordering and health checks.
- **Pipeline-wide idempotency** -- `INSERT ... ON CONFLICT DO NOTHING` on all Postgres write paths. Deterministic event IDs. Re-parsing the same demo produces identical data with no duplicates.
- **Structured logging with correlation IDs** -- Every log line tagged with `match_id` and `job_id` for cross-service traceability. Uses `log/slog` exclusively; no `{data, meta}` CLI JSON envelopes in service output.

**Should have (P2 -- add after core pipeline is stable):**
- **Programmatic query API** -- `dem match <id>` or thin HTTP API to retrieve parsed match stats as JSON. Extends the v1.0 JSON-contract philosophy.
- **Grenade trajectory storage** -- Full per-tick trajectory points in Postgres for future spatial analysis (heatmaps, team throw patterns).
- **Prometheus metrics + Grafana dashboard** -- Pipeline throughput, queue depth, error rates, parse latency.

**Defer (v2+ or separate projects):**
- Web UI / dashboard (users point Metabase/Superset at the Postgres database instead)
- Live CSTV+ match parsing (fundamentally different architecture, separate project)
- Multi-account demo downloading (not needed for HLTV-hosted public demos)
- Multiple parsing engines for cross-validation (demoinfocs-golang v5 is production-proven at HLTV.org scale)
- Aggressively fast polling intervals (< 6 hours: risks IP blocking, provides no real benefit for Tier 1)

### Architecture Approach

Three new microservices around the existing v1.0 CLI, connected by NATS JetStream work queues. The existing CLI code (`internal/hltv`, `internal/provider`, `internal/domain`) is preserved unchanged -- services import it as library code but never modify it.

**Major components:**
1. **Tournament Poller Service** -- Cron-driven: fetches Tier 1 HLTV events via existing parsers, discovers matches with demos, publishes download jobs to `dem.download.jobs`. Stateless (uses Redis/Postgres for dedup). First service in the pipeline.
2. **Demo Downloader Service** -- NATS consumer: receives download jobs, streams `.dem.gz` from HLTV CDN directly to MinIO (no local disk write), publishes parse jobs to `dem.parse.jobs`. Stateless (MinIO stores files).
3. **Demo Parser Service** -- NATS consumer: receives parse jobs, streams `.dem.gz` from MinIO into demoinfocs-golang, registers event handlers for 12+ game event types, batch-inserts into Postgres per round. Stateless (Postgres stores data). The most complex service.
4. **Infrastructure (NATS, PostgreSQL, MinIO, Redis)** -- All in Docker Compose. NATS JetStream DEM_JOBS stream with WorkQueue retention, at-least-once delivery (MaxDeliver: 3), and durable pull consumers. PostgreSQL schema has 6 tables: matches, rounds, kill_events, damage_events, players, match_players.

**Data flow:** HLTV.org -> Poller -> NATS `dem.download.jobs` -> Downloader -> MinIO + NATS `dem.parse.jobs` -> Parser -> PostgreSQL

**Key architectural decisions:**
- **Single `go.mod` monorepo** (not multi-module): avoids Docker build context issues with `replace` directives, simplifies dependency management for 3 services + CLI.
- **Services as `cmd/` entrypoints** (alongside existing `cmd/dem/`): keeps binary entrypoints co-located under one module. Each service has its own `internal/` packages for service-specific concerns.
- **Streaming, no temp files**: Demos are never written to disk. HTTP response body streams directly to MinIO; MinIO response body streams directly to demoinfocs-golang.
- **Config via environment variables** (12-factor app style): Docker Compose sets env vars. Viper loads with defaults. No config files in containers.
- **Provider/Tier Isolation** (carried from v1.0): External dependencies (NATS, MinIO, Postgres) are injected via functional options. Every service constructor accepts options.
- **No v1.0 code modified**: The CLI path (`cmd/dem/` through `internal/cli/` through `internal/provider/` through `internal/hltv/`) is completely untouched.

### Critical Pitfalls

These are the top 5 pitfalls that cause rewrites, data loss, or multi-day debugging sessions:

1. **JetStream messages published without a backing stream -- silent data loss** -- The NATS Go client's `Publish` returns success even if no stream exists for the subject; the server silently drops the message. Prevention: create all streams programmatically at service startup before any publisher starts. Add startup health check that verifies expected streams exist via `js.Stream()`.

2. **Missing `msg.Ack()` in consumer handlers -- infinite redelivery loops** -- After `AckWait` expires (default 30s), unacknowledged messages are redelivered indefinitely. Prevention: `defer msg.Ack()` at the TOP of every handler (non-negotiable). Set `MaxDeliver: 5` on every consumer. Use `msg.NakWithDelay()` for transient failures.

3. **Buffering 100MB+ demo files in memory -- OOM kills** -- `io.ReadAll(resp.Body)` on a 200 MB demo plus 250 MB parser overhead crashes the process. Prevention: stream HTTP response body directly to MinIO `PutObject` via `io.Reader`. Stream MinIO `GetObject` directly to `demoinfocs.NewParser(reader)`. Never call `io.ReadAll()` or `os.ReadFile()` on a demo.

4. **Using `pgx.Conn` instead of `pgxpool` -- connection exhaustion** -- `pgx.Conn` is not concurrency-safe. Creating `pgx.Connect()` per request hits `max_connections` under load. Prevention: create `pgxpool.Pool` once at startup with `MaxConns: 10-25` (override the default of 4). Pass the pool to all handlers.

5. **At-least-once delivery without idempotency -- duplicate game events** -- NATS redelivers messages after crashes or `AckWait` expiry. Without idempotent writes, duplicate rows corrupt analytical queries. Prevention: deterministic event IDs, `INSERT ... ON CONFLICT (match_id, event_id) DO NOTHING`, and a `demo_processing` state table for atomic claim-before-work.

## Implications for Roadmap

Based on combined research, the dependency chain forces a specific phase ordering. Each service depends on infrastructure and shared packages, never on another service (NATS decouples them). But the message flow creates a natural build sequence.

### Phase 1: Infrastructure Foundation

**Rationale:** Nothing runs without NATS, Postgres, MinIO, and Redis. The database schema must exist before the parser can insert data. The monorepo structure decision must be locked before any service scaffolding. NATS stream configuration must happen before any publisher runs (Pitfall 1). This phase eliminates the "Docker build context" pitfall (Pitfall 3) and the `pgx.Conn` vs `pgxpool` pitfall (Pitfall 5) upfront.

**Delivers:**
- `docker-compose.yml` with NATS, PostgreSQL, MinIO, Redis -- all with health checks
- Database schema migrations (`sql/migrations/`) for matches, rounds, kills, damage events, players, match_players
- `pkg/natsutil/` -- NATS connection helpers, stream config, subject schemas
- `pkg/minio/` -- MinIO client factory with connection pooling
- `pkg/postgres/` -- pgxpool setup with explicit connection limits (MaxConns: 10-25)
- Monorepo scaffolding: `cmd/poller/`, `cmd/downloader/`, `cmd/parser/` entrypoints
- `internal/domain/game_events.go` -- new domain types (MatchMetadata, KillEvent, RoundInfo, DamageEvent)
- Makefile with per-service build targets

**Addresses:** Docker Compose orchestration (P1), pipeline idempotency schema (P1)
**Avoids:** Pitfall 1 (stream creation order), Pitfall 3 (Docker build with single go.mod), Pitfall 5 (pgxpool not pgx.Conn)

**Research flag:** LOW -- well-documented infrastructure setup. Docker Compose patterns are mature. pgxpool configuration is standard. No novel research needed.

### Phase 2: Pipeline Services

**Rationale:** Each service is independent (NATS-decoupled), but Poller must be built first because it produces the messages that Downloader consumes, and Downloader produces the messages that Parser consumes. While building Downloader, you can manually publish test messages to `dem.download.jobs` using `nats pub`. While building Parser, you can manually place demo files in MinIO and publish test parse jobs. This parallelizes work while respecting the dependency chain. The poller reuses existing `internal/hltv` parsers (no rewrite needed) -- the main new work is NATS publishing.

**Sub-phases:**

**2a. Poller Service** (`cmd/poller/`) -- produces messages, nothing consumes them yet
- Cron-style Tier 1 event polling (configurable interval, default daily, minimum 6h guard)
- Match discovery via existing `internal/hltv` parsers (ParseEvents, ParseResults, ParseDemoLink)
- NATS job publishing to `dem.download.jobs` (after stream creation verification)
- Redis-based deduplication (SETNX with match ID, 30-day TTL)
- Structured logging with match_id correlation

**2b. Downloader Service** (`cmd/downloader/`) -- consumes poller messages, produces parse jobs
- NATS JetStream pull consumer on `dem.download.jobs` with `defer msg.Ack()`
- Streaming HTTP download from HLTV CDN (no `io.ReadAll`)
- Direct stream to MinIO PutObject (no local temp files, Pipeline 2: Streaming Upload)
- Publish parse jobs to `dem.parse.jobs` on success
- Retry with `NakWithDelay()` on transient failures (Pipeline 3: JetStream Work Queue)

**2c. Parser Service** (`cmd/parser/`) -- consumes downloader messages, writes to Postgres
- NATS JetStream pull consumer on `dem.parse.jobs` with `defer msg.Ack()`
- Stream from MinIO GetObject directly into demoinfocs-golang (no disk, no buffering)
- 12+ event handlers (MatchStart, RoundStart/End, Kill, PlayerHurt, WeaponFire, Bomb events, GrenadeProjectileThrow, PlayerConnect, TeamSideSwitch)
- Batch insert per round into Postgres via pgxpool (40-50 kills per round batched)
- Idempotent writes: `INSERT ... ON CONFLICT DO NOTHING` with deterministic event IDs
- `defer p.Close()` on every parser instance (250 MB C-memory leak per unclosed parser)
- `AckWait` tuned for long-running parses (>30 minutes for long demos)

**Addresses:** Tournament polling (P1), demo download (P1), demo parsing (P1), idempotency (P1), structured logging (P1), error resilience (P1)
**Avoids:** Pitfall 2 (defer msg.Ack everywhere), Pitfall 4 (streaming, no buffering), Pitfall 6 (idempotent inserts from day one), Pitfall 7 (defer p.Close for every parser)

**Research flag:** MEDIUM-HIGH -- demoinfocs-golang v5 event handler lifecycle (available data at each event fire time, thread safety of `p.GameState()` during handlers), Postgres batch insert throughput profiling (round-level vs N-events batching), NATS `AckWait` tuning for multi-minute parse jobs, memory profiling under concurrent parse load (2-3 demos simultaneously at 250 MB each). Also: the Downloader needs its own HTTP client for `.dem.gz` CDN downloads (different endpoint and headers than the existing `internal/hltv` Client). Strong candidate for `/gsd-research-phase` during planning.

### Phase 3: Iteration and Polish

**Rationale:** These features depend on a working pipeline with data in Postgres. The query API reads parsed data; grenade trajectories need the parser output path stable; Prometheus metrics need observable pipeline throughput to be meaningful. Delaying these until the core pipeline is validated prevents premature optimization.

**Delivers:**
- Programmatic query API (CLI command `dem match <id>` or HTTP endpoint) for parsed match stats, round timelines, player performance
- Full grenade trajectory storage (schema exists from Phase 1, write path added here)
- Prometheus metrics: job latency histograms, success/failure rates, queue depth gauges
- Configurable polling interval with 6-hour minimum guard (currently hardcoded daily)
- Grafana dashboard (importable JSON)

**Addresses:** Programmatic query API (P2), grenade trajectory storage (P2), Prometheus metrics (P2), configurable polling interval (P3)

**Research flag:** LOW -- Prometheus Go client is well-documented. Query API is thin over Postgres views. Grenade trajectory storage follows the same batch-insert pattern as kill events.

### Phase Ordering Rationale

- **Infrastructure before services** is non-negotiable: Docker Compose, NATS streams, Postgres schema, and shared libraries must exist before any service can start.
- **Poller before Downloader before Parser** follows the message flow: Poller produces download jobs, Downloader consumes and produces parse jobs, Parser consumes parse jobs. But NATS decouples them so each can be tested independently with manual message publishing.
- **Domain models before service implementation**: `internal/domain/game_events.go` must exist before service code references new types.
- **Pipeline idempotency comes with the parser, not after**: designing deterministic event IDs and `ON CONFLICT DO NOTHING` from the start avoids costly data migration later.
- **Streaming (no temp files) comes with the downloader, not after**: every line of download and parse code must pipe `io.Reader` from the first implementation.

### Research Flags

**Phases likely needing deeper research during planning:**
- **Phase 2 (Pipeline Services):** demoinfocs-golang v5 handler lifecycle, Postgres batch insert throughput profiling, NATS `AckWait` tuning for multi-minute parse jobs, memory profiling under concurrent parse load, HLTV CDN download endpoint behavior (range request support, rate limits, appropriate HTTP client config). Strong candidate for `/gsd-research-phase`.

**Phases with standard patterns (skip research-phase):**
- **Phase 1 (Infrastructure Foundation):** All components use well-documented, mature patterns. No novel engineering required.
- **Phase 3 (Iteration and Polish):** Standard additions to a working platform.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All library versions verified against official sources (pkg.go.dev, GitHub releases, project go.mod). No version conflicts. All Go version requirements compatible with 1.25.0. |
| Features | HIGH | Feature landscape validated against 6 competitor/adjacent projects. P1/P2/P3 prioritization grounded in dependency analysis and user value assessment. |
| Architecture | HIGH | Component boundaries, data flow, NATS subject design, database schema, and Docker Compose layout all validated against current documentation. Build order enforced by dependency graph. |
| Pitfalls | HIGH | Every critical pitfall sourced from GitHub issues, maintainer discussions, or production incident reports (nats.go #1793, nats-server #3639, jackc/pgx #1989, minio/minio #21611, helpwave/services #25). Prevention strategies and recovery steps documented. |

**Overall confidence:** HIGH -- all primary recommendations backed by official documentation and production experience from referenced projects. No inferences or single-source findings drive critical decisions.

### Gaps to Address

- **HLTV CDN download endpoint behavior:** The existing v1.0 `ParseDemoLink` extracts a demo URL, but the actual HTTP download endpoint for `.dem.gz` files needs testing during Phase 2. Does it support range requests? Rate limits? Does the existing browser-impersonation TLS config work or is a different configuration needed?
- **demoinfocs event handler timing:** Which events fire during `ParseToEnd()` and in what order? Is `p.GameState()` safe to call from within event handlers? Needs empirical testing during parser implementation.
- **Postgres write throughput under batch insert:** The recommendation is round-level batch insert. Needs profiling to confirm it is sufficient and that `pgx.CopyFrom` bulk loading is not required for MVP scale.
- **NATS `AckWait` for long parse jobs:** The default 30-second `AckWait` will expire during a 60-minute demo parse. The recommended approach (extended `AckWait` vs `msg.InProgress()`) needs empirical testing under concurrent load.
- **MinIO presigned URL clock skew:** The 403 error pitfall from clock skew between Docker containers needs validation in the local Docker Compose setup.

## Sources

### Primary (HIGH confidence)
- [nats-io/nats.go v1.51.0](https://github.com/nats-io/nats.go/releases) -- Go 1.25 requirement, JetStream API, PushConsumer
- [jackc/pgx v5.9.2](https://pkg.go.dev/github.com/jackc/pgx/v5) -- PostgreSQL driver, pgxpool, Go 1.25+ requirement
- [markus-wa/demoinfocs-golang v5.2.0](https://pkg.go.dev/github.com/markus-wa/demoinfocs-golang/v5) -- 70+ typed game events, CS2 support
- [minio/minio-go v7](https://pkg.go.dev/github.com/minio/minio-go/v7) -- S3-compatible client
- [redis/go-redis v9.18.0](https://redis.io/docs/latest/integrate/go-redis/) -- Official Redis client
- [golang-migrate/migrate v4.19.1](https://github.com/golang-migrate/migrate) -- Database migrations, pgx v5 driver
- [spf13/viper v1.21.0](https://pkg.go.dev/github.com/spf13/viper) -- Configuration management
- [Go 1.25 release notes](https://go.dev/doc/go1.25) -- synctest GA, Green Tea GC, container-aware GOMAXPROCS
- [demoinfocs-golang DeepWiki](https://deepwiki.com/markus-wa/demoinfocs-golang) -- Architecture, event list, benchmarks
- Existing codebase: `/Users/base/Documents/dem/go.mod`, `internal/hltv/`, `internal/provider/`, `internal/domain/`, `internal/cli/`

### Secondary (MEDIUM confidence)
- [HLTVDemoDownloader](https://github.com/hooolius/HLTVDemoDownloader) -- Canonical demo download workflow
- [hltv-utility-api](https://github.com/hx-w/hltv-utility-api) -- Closest reference architecture (Go + demoinfocs v2)
- [hltv-match-predictor](https://github.com/ratx64/hltv-match-predictor) -- Dual-cron schedule pattern
- [OpenDota Architecture Blog](http://blog.opendota.com/2016/05/15/architecture) -- Proven pipeline pattern
- [Geqo Observer](https://github.com/geqo/cs2-observer) -- State-of-the-art CS2 event-driven microservice
- [go-coffeeshop](https://github.com/thangchung/go-coffeeshop) -- Go microservice with Docker Compose hybrid pattern
- [NATS JetStream + Go patterns](https://memphis.dev/blog/building-distributed-systems-with-nats-jetstream-and-golang/)

### Pitfall-Specific Sources
- nats.go #1793: Ack after consumer delete silent failure
- nats-server #3639: WorkQueuePolicy overlapping subjects
- jackc/pgx #1989: pgxpool creation pattern
- minio/minio #21611: aws-chunked 16 MiB limit
- helpwave/services #25: go mod replace breaks Docker build
- quantmHQ/quantm #210: pgx.Conn concurrency safety bug

---
*Research completed: 2026-05-03*
*Ready for roadmap: yes*

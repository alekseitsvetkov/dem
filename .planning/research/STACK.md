# Stack Research

**Domain:** Go microservice monorepo — HLTV data platform (CLI + services)
**Researched:** 2026-05-03
**Confidence:** HIGH

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Go | 1.25.0 (already in go.mod) | Language runtime | Already locked; Go 1.25 shipped Aug 2025 with `testing/synctest` GA, container-aware GOMAXPROCS, `encoding/json/v2` experimental, DWARF 5 debug info, and 10-40% GC reduction (Green Tea GC experimental). All new libraries target Go 1.25+. |
| NATS Server | 2.12 (Docker) | Message broker | Cloud-native messaging with JetStream persistence. Single-binary deployment, no ZooKeeper dependency. JetStream provides at-least-once delivery, durable consumers, and stream replay — ideal for download/parse job queues. |
| PostgreSQL | 17 (Docker) | Relational database | Stores parsed game events (kills, rounds, bomb events, player stats). Chosen over NoSQL because CS2 event data is relational (players in matches, rounds in matches, kills per round). |
| MinIO | latest (Docker) | S3-compatible object storage | Stores .dem files (50-200MB each). S3 API is the industry standard for blob storage. Self-hosted MinIO avoids cloud vendor lock-in and works identically in dev and production. |
| Redis | 8 (Docker) | Cache / deduplication | Tracks processed match IDs to avoid re-downloading/re-parsing. Lightweight key-value store with TTL support for cache expiry. Version 8 is current major release. |

### Go Client Libraries

| Library | Module Path | Version | Purpose | Why This One |
|---------|------------|---------|---------|--------------|
| NATS Go Client | `github.com/nats-io/nats.go` | v1.51.0 | NATS pub/sub + JetStream | Official client. v1.51.0 (April 2026) includes JetStream API, durable consumer support, `PushConsumer`, atomic batch publishing. JetStream is built into the same library — no separate import. Supports Go 1.25. |
| pgx | `github.com/jackc/pgx/v5` | v5.9.2 | PostgreSQL driver + connection pool | De facto standard Go Postgres driver. v5.9.2 (April 2026) patched SQL injection via placeholder confusion (GHSA-j88v-2chj-qfwx). v5.9.0 added PostgreSQL protocol 3.2, SCRAM-SHA-256-PLUS, OAuth. pgxpool provides built-in connection pooling. Requires Go 1.25 — compatible with our go.mod. Faster than `database/sql` + `lib/pq` because it speaks PostgreSQL wire protocol directly, skipping the `database/sql` abstraction overhead. |
| go-redis | `github.com/redis/go-redis/v9` | v9.18.0 | Redis client | Official Redis client (moved to `redis/go-redis` org). v9.18.0 (April 2026) adds native OpenTelemetry metrics, Redis 8.6 support, RESP2/RESP3 protocol. Minimum Go 1.21 — well within our Go 1.25 baseline. |
| MinIO Go SDK | `github.com/minio/minio-go/v7` | v7.0.99 | S3 object storage client | Official MinIO SDK. v7.0.99 (March 2026) supports ~48.83TiB max object size, Go-native iterators for `ListObjects()`, S3 Express zone. v7 is the stable major version since 2020 — no breaking changes expected. |
| demoinfocs-golang | `github.com/markus-wa/demoinfocs-golang/v5` | v5.2.0 | CS2 .dem file parser | The only production-grade Go CS2 demo parser. 70+ typed game events (kills, rounds, bomb, grenades), game-state tracking, automatic Source 1/2 detection. Stars: ~823. License: MIT. v5 targets CS2 and CSTV+ live broadcast parsing. Requires Go 1.24+. |

### Supporting Libraries

| Library | Module Path | Version | Purpose | When to Use |
|---------|------------|---------|---------|-------------|
| golang-migrate | `github.com/golang-migrate/migrate/v4` | v4.19.1 | Database schema migrations | On every schema change. Use pgx v5 driver (`database/pgx/v5`). Supports up/down migrations, CLI + library API. v4.19.1 (Nov 2025) requires Go 1.25 — compatible. |
| Viper | `github.com/spf13/viper` | v1.21.0 | Configuration management | Service startup — reads env vars, YAML config files, and integrates with Cobra flags. Already in the Cobra ecosystem. v1.21.0 (mid-2025) added `pflag.BoolSlice` support. |
| Go standard `log/slog` | stdlib (Go 1.21+) | built-in | Structured logging | All services. Structured JSON logging with levels, handler interface, context propagation. Ships with Go — no dependency. Prefer over zerolog/zap because this project already targets Go 1.25 and adding a third-party logger when stdlib suffices adds maintenance burden. |
| Go standard `testing/synctest` | stdlib (Go 1.25 GA) | built-in | Concurrent code testing | Testing NATS consumers, service orchestration, time-dependent logic. Graduated to GA in Go 1.25 — virtual clock, isolated goroutine bubbles. Eliminates `time.Sleep` in tests. |
| Cobra | `github.com/spf13/cobra` | v1.10.1 (existing) | CLI framework | Already in go.mod. Used for the `dem` CLI. Not needed inside microservices, but retained for the existing command-line tool. |
| goquery | `github.com/PuerkitoBio/goquery` | v1.12.0 (existing) | HTML parsing | Already in go.mod. HLTV scraping. Retained only in the existing `internal/hltv` package — microservices do not import this. |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| Docker Compose | Local infrastructure orchestration | Single `docker-compose.yml` at repo root starts NATS, Postgres, Minio, Redis. Services connect via Docker network. Each service has its own `Dockerfile` for production builds. |
| golang-migrate CLI | Run migrations | `migrate -database "$DATABASE_URL" -path platform/migrations up`. Install via `go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1`. |
| Makefile | Build orchestration | Root Makefile delegates to per-service targets: `make build-poller`, `make run-downloader`. Service directories contain their own Makefiles for independent builds. |
| sqlc (optional, future) | Type-safe SQL codegen | If raw SQL with pgx becomes verbose, sqlc generates type-safe Go from SQL queries. Not needed for Phase 1 — pgx with hand-written queries is sufficient for initial schema. |

## Monorepo Structure

The existing codebase (`cmd/dem/`, `internal/cli/`, `internal/hltv/`, `internal/provider/`, `internal/domain/`, `internal/output/`) is preserved. New microservices are added alongside it:

```
dem/
├── cmd/dem/                   # EXISTING: CLI entrypoint
├── internal/                  # EXISTING: CLI shared code (hltv, cli, provider, domain, output)
├── services/                  # NEW: Microservices
│   ├── poller/                # Tournament polling service
│   │   ├── cmd/main.go        # Service entrypoint
│   │   ├── internal/          # Service-specific code
│   │   └── Dockerfile
│   ├── downloader/            # Demo download service
│   │   ├── cmd/main.go
│   │   ├── internal/
│   │   └── Dockerfile
│   └── parser/                # Demo parsing service
│       ├── cmd/main.go
│       ├── internal/
│       └── Dockerfile
├── platform/                  # NEW: Shared infrastructure
│   ├── migrations/            # Postgres migration files
│   ├── nats/                  # NATS subject schemas, stream configs
│   └── docker-compose.yml     # Local dev infrastructure
├── pkg/                       # NEW: Public shared libraries
│   └── natsutil/              # NATS connection helpers, message types
├── go.mod                     # Single Go module for entire monorepo
├── go.sum
└── Makefile                   # Root orchestrator
```

**Why single `go.mod`:** With 3 services on a small team, a single Go module avoids version-skew between services, simplifies dependency management, and aligns with the existing codebase. If services grow beyond 5-7, split into separate modules — but not yet.

**Why `services/` not `cmd/`:** The existing `cmd/dem/` is the CLI tool. Adding service entrypoints to `cmd/` would pollute the CLI binary namespace. `services/` clearly separates long-running daemons from the CLI tool.

## Installation

```bash
# Core Go dependencies (add to existing go.mod)
go get github.com/nats-io/nats.go@v1.51.0
go get github.com/jackc/pgx/v5@v5.9.2
go get github.com/redis/go-redis/v9@v9.18.0
go get github.com/minio/minio-go/v7@v7.0.99
go get github.com/markus-wa/demoinfocs-golang/v5@v5.2.0

# Supporting
go get github.com/golang-migrate/migrate/v4@v4.19.1
go get github.com/golang-migrate/migrate/v4/database/pgx/v5@v4.19.1
go get github.com/spf13/viper@v1.21.0

# Dev tooling
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1
```

## Alternatives Considered

| Category | Recommended | Alternative | Why Not Alternative |
|----------|-------------|-------------|---------------------|
| Message broker | NATS | RabbitMQ | RabbitMQ requires Erlang runtime, heavier ops burden. NATS is a single Go binary (same language as our services), simpler config, built-in JetStream for persistence. |
| Message broker | NATS | Kafka | Kafka requires ZooKeeper/KRaft, JVM, and is overkill for 3 services. NATS handles our throughput needs (hundreds of messages/day, not millions/sec) with far less operational complexity. |
| Postgres driver | pgx v5 | lib/pq | lib/pq is in maintenance mode (no new features). pgx is 2-5x faster by speaking PostgreSQL wire protocol directly, skipping `database/sql` abstraction. pgxpool provides connection pooling. |
| Postgres driver | pgx v5 | database/sql + pgx stdlib adapter | Using pgx directly (not through `database/sql`) gives access to PostgreSQL-specific features (COPY protocol, LISTEN/NOTIFY, native type support). The adapter layer removes these benefits. |
| Object storage | MinIO (self-hosted) | AWS S3 | Self-hosted MinIO avoids cloud vendor lock-in, works identically in dev (Docker Compose) and production, zero egress costs for internal services. S3 API compatibility means switching to AWS S3 later requires only a config change. |
| Redis client | go-redis v9 | redigo | redigo has less frequent maintenance. go-redis is the official Redis client, has broader type support, cluster/sentinel support, and OpenTelemetry integration. |
| Structured logging | log/slog (stdlib) | zerolog | zerolog is excellent for allocation-sensitive code, but slog ships with Go 1.21+ (we target 1.25), has a handler interface for extensibility, and requires no dependency. The project's performance profile (hundreds of events/day, not millions/sec) does not justify zerolog's allocation optimization. |
| Structured logging | log/slog | zap (Uber) | zap is production-proven but adds a dependency. slog is in stdlib, has comparable features, and is the Go team's recommended approach for new code. |
| Config management | Viper | envconfig/kelseyhightower | envconfig is simpler but only handles env vars. Viper supports YAML config files, env vars, and integrates with Cobra flags — useful when services share configuration patterns with the existing CLI. |
| Config management | Viper | caarlos0/env | caarlos0/env is excellent for 12-factor apps (env vars only) but less useful when we want to share config via YAML files for local Docker Compose development. |
| Migrations | golang-migrate | goose | Both are solid. golang-migrate has broader database support, a simpler migration format (numeric sequence), and a CLI that doesn't require a Go build to run migrations. |
| Demo parsing | demoinfocs-golang v5 | Custom CGO wrapper | demoinfocs-golang is Go-native (no CGO), supports CS2, has 70+ typed events, and is MIT-licensed. A custom wrapper around the C++ demoinfo library would require CGO, complicate cross-compilation, and add maintenance burden for CS2 protocol updates. |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| GORM (ORM) | ORMs generate inefficient queries for complex CS2 event relationships (players, rounds, kills, grenades). They obscure query intent and make debugging slow queries harder. Raw SQL with pgx gives full control over Postgres performance. | pgx with hand-written SQL |
| gRPC | Adds protobuf compilation step, service definitions, and code generation complexity. NATS already handles inter-service messaging with a simpler pub/sub model. The services are fire-and-forget pipelines — no request-response RPC needed. | NATS pub/sub (JetStream for persistence) |
| Kubernetes (k8s) | The milestone explicitly targets Docker Compose for local dev and simple deployment. k8s adds significant operational complexity (pod definitions, services, ingress, configmaps, secrets) that a 3-service platform does not benefit from. | Docker Compose |
| `lib/pq` Postgres driver | In maintenance mode. No support for PostgreSQL protocol 3.0+, SCRAM-SHA-256-PLUS, or the COPY protocol. pgx is the community standard for new Go projects. | pgx v5 |
| Go workspace (`go.work`) | Unnecessary for a single-module monorepo. Workspaces solve multi-module repos. We have one `go.mod` for the whole repo (3 services + existing CLI). | Single `go.mod` at repo root |
| Custom message serialization | Brittle and hard to evolve. JSON is human-debuggable (view in NATS dashboard), self-describing, and adequate for our throughput. | JSON over NATS (message envelope pattern matching existing CLI contract) |
| `database/sql` interface layer | When using pgx, the `database/sql` compatibility layer removes PostgreSQL-specific features. Direct pgx usage gives COPY protocol, LISTEN/NOTIFY, native type mapping, and better performance. | pgx native API (`pgxpool`, `pgx.Conn`) |

## Stack Patterns by Variant

**If service needs at-least-once delivery (download/parse jobs):**
- Use NATS JetStream with durable consumers and explicit `msg.Ack()` after successful processing.
- Configure `MaxAckPending(1)` for download service (sequential downloads respect HLTV rate limits).
- Configure `MaxAckPending(10)` for parser service (CPU-bound parsing can parallelize).

**If service needs idempotency (avoid re-downloading same match):**
- Use Redis `SETNX` with match ID as key before enqueuing a download job.
- Set TTL on dedup keys to auto-expire after reasonable window (e.g., 30 days).
- Alternatively, use Postgres `INSERT ... ON CONFLICT DO NOTHING` with a unique constraint on `match_id`.

**If service needs configuration:**
- Docker Compose: pass config as environment variables in `docker-compose.yml`.
- Local dev: use `.env` files loaded by Viper with `viper.AutomaticEnv()`.
- Production: environment variables override YAML defaults.

**If a service needs to share types with the CLI:**
- Extract shared domain types to `pkg/models/` (e.g., `Match`, `Event`, `DemoLink`).
- Each service defines its own internal types for service-specific concerns.
- Message types on NATS subjects are the contract — keep them in `pkg/natsutil/`.

## Version Compatibility

| Package | Go Version Required | Notes |
|---------|-------------------|-------|
| `nats.go` v1.51.0 | Go 1.25 | Matches our go.mod exactly |
| `pgx/v5` v5.9.2 | Go 1.25+ | Matches our go.mod |
| `go-redis/v9` v9.18.0 | Go 1.21+ | Well within baseline |
| `minio-go/v7` v7.0.99 | Go 1.21+ | Well within baseline |
| `demoinfocs-golang/v5` v5.2.0 | Go 1.24+ | Within baseline (we're on 1.25) |
| `golang-migrate/v4` v4.19.1 | Go 1.25 | Matches our go.mod |
| `viper` v1.21.0 | Go 1.23+ | Within baseline |

**All recommended libraries are compatible with Go 1.25.0.** No version conflicts identified.

## Integration Points

### NATS Subject Namespace

```
dem.events.tier1            # Published by poller: new Tier 1 event detected
dem.matches.new             # Published by poller: new match result found
dem.downloads.jobs          # JetStream stream: demo download jobs
dem.downloads.results       # Published by downloader: demo downloaded (minio key, size, hash)
dem.parse.jobs              # JetStream stream: demo parse jobs
dem.parse.results           # Published by parser: game events persisted, match_id
```

### Data Flow

```
HLTV.org → [Poller] → NATS dem.downloads.jobs → [Downloader] → MinIO (.dem files)
                                                      ↓
                                              NATS dem.parse.jobs
                                                      ↓
MinIO ← [Parser] → PostgreSQL (game_events, rounds, kills, players)
```

### Service Boundaries

| Service | Reads From | Writes To | State |
|---------|-----------|-----------|-------|
| Poller | HLTV.org (HTTP) | NATS, Redis (dedup) | Stateless (Redis for dedup) |
| Downloader | NATS, HLTV.org (HTTP) | MinIO, NATS, Redis (dedup) | Stateless (MinIO for files) |
| Parser | NATS, MinIO | PostgreSQL, NATS | Stateless (Postgres for data) |

## Sources

- [nats-io/nats.go releases](https://github.com/nats-io/nats.go/releases) — verified v1.51.0 (April 14, 2026), Go 1.25 requirement — HIGH confidence
- [jackc/pgx on pkg.go.dev](https://pkg.go.dev/github.com/jackc/pgx/v5) — verified v5.9.2 (April 2026), Go 1.25+ requirement, supported PostgreSQL versions — HIGH confidence
- [minio/minio-go releases](https://github.com/minio/minio-go) — verified v7.0.99 via newreleases.io — MEDIUM confidence (not directly confirmed on pkg.go.dev)
- [redis/go-redis](https://redis.io/docs/latest/integrate/go-redis/) — verified v9.18.0 (April 2026), OpenTelemetry metrics, Redis 8.6 support — HIGH confidence
- [markus-wa/demoinfocs-golang on pkg.go.dev](https://pkg.go.dev/github.com/markus-wa/demoinfocs-golang/v5) — verified v5.2.0, CS2 + CSTV+ support — HIGH confidence
- [golang-migrate/migrate releases](https://github.com/golang-migrate/migrate) — verified v4.19.1 (Nov 2025), Go 1.25, pgx v5 driver available — HIGH confidence
- [spf13/viper on pkg.go.dev](https://pkg.go.dev/github.com/spf13/viper) — verified v1.21.0, 102K+ importers — HIGH confidence
- [Go 1.25 release notes](https://go.dev/doc/go1.25) — verified features (synctest GA, Green Tea GC experimental, container-aware GOMAXPROCS, encoding/json/v2 experimental) — HIGH confidence
- [NATS JetStream Go API patterns](https://qaze.app/blog/nats-batch-publish/) — atomic batch publishing, PushConsumer, durable consumers — MEDIUM confidence (blog source, verified against nats.go release notes)
- [Go monorepo patterns](https://dev.to/aleksei_aleinikov/go-microservices-2025-one-pattern-to-scale-them-all-1448) — services/ layout, three-tier shared code strategy — MEDIUM confidence (community patterns, not official spec)
- Project go.mod at `/Users/base/Documents/dem/go.mod` — verified Go 1.25.0, existing dependencies (Cobra v1.10.1, goquery v1.12.0)

---
*Stack research for: HLTV CLI v1.1 — Microservice Platform*
*Researched: 2026-05-03*

# Requirements: HLTV CLI

**Defined:** 2026-05-03
**Core Value:** Users can reliably fetch HLTV event, result, and demo-link data as stable JSON from a script-friendly CLI, and automatically acquire and parse CS2 demo files at scale.

## v1.1 Requirements

Requirements for the Microservice Platform milestone. Each maps to roadmap phases.

### Infrastructure

- [ ] **INFR-01**: Developer can start all infrastructure with a single `docker-compose up` command — NATS (with JetStream), PostgreSQL 17, MinIO, and Redis all come up with health checks and correct startup ordering.
- [ ] **INFR-02**: Go monorepo uses a single root `go.mod` — existing v1.0 CLI and new services share one module. Services are added as `cmd/<service>/` entrypoints alongside `cmd/dem/`.
- [ ] **INFR-03**: Database schema is managed via migrations (`sql/migrations/`) and creates tables for matches, rounds, kill_events, damage_events, players, and match_players with appropriate constraints and indexes.
- [ ] **INFR-04**: Shared packages (`pkg/natsutil/`, `pkg/minio/`, `pkg/postgres/`) provide connection helpers, stream configuration, and connection pooling with functional options for test doubles.
- [ ] **INFR-05**: NATS JetStream streams are created programmatically at service startup — no manual `nats stream create` commands. Startup fails fast with a clear error if a required stream is missing.
- [ ] **INFR-06**: Postgres connection pooling uses `pgxpool` (not raw `pgx.Conn`) with explicit MaxConns configuration (10-25). The pool is created once at startup and passed to all handlers.

### Tournament Polling

- [ ] **POLL-01**: Poller service checks HLTV for Tier 1 tournaments on a configurable schedule (default: daily, minimum: 6 hours). It reuses the existing v1.0 HLTV parsers without modification.
- [ ] **POLL-02**: Poller discovers matches with available demos from each tournament and publishes a download job to the NATS `dem.download.jobs` subject for each new match.
- [ ] **POLL-03**: Poller deduplicates matches so the same match is never published twice — uses Redis SETNX with match ID and 30-day TTL, or an equivalent Postgres-based approach.

### Demo Download

- [ ] **DWLD-01**: Downloader service consumes jobs from NATS `dem.download.jobs` using a JetStream pull consumer with explicit `defer msg.Ack()` on every message.
- [ ] **DWLD-02**: Downloader streams `.dem.gz` files directly from the HLTV CDN to MinIO without writing to local disk — HTTP response body pipes to MinIO `PutObject` via `io.Reader`.
- [ ] **DWLD-03**: On successful upload, Downloader publishes a parse job to NATS `dem.parse.jobs` with the MinIO object key and match metadata. On transient failure, it uses `msg.NakWithDelay()` for retry.

### Demo Parsing

- [ ] **PARS-01**: Parser service consumes jobs from NATS `dem.parse.jobs` using a JetStream pull consumer with explicit `defer msg.Ack()`.
- [ ] **PARS-02**: Parser streams the `.dem.gz` file from MinIO directly into demoinfocs-golang (`demoinfocs.NewParser(reader)`) — the file is never buffered in memory or written to disk.
- [ ] **PARS-03**: Parser registers event handlers for at minimum: MatchStart, RoundStart, RoundEnd, Kill, PlayerHurt, WeaponFire, BombPlant, BombDefuse, BombExplode, GrenadeProjectileThrow, PlayerConnect, and TeamSideSwitch.
- [ ] **PARS-04**: Parser batch-inserts game events into Postgres per round using pgxpool. Inserts are idempotent — `INSERT ... ON CONFLICT (match_id, event_id) DO NOTHING` — so re-parsing the same demo produces no duplicate data.
- [ ] **PARS-05**: Parser calls `defer p.Close()` on every demoinfocs parser instance to prevent C-memory leaks (~250 MB per unclosed parser).

### Cross-Cutting

- [ ] **CROS-01**: Every service uses structured logging via `log/slog` with `match_id` and `job_id` correlation fields. Services do NOT use the v1.0 CLI JSON envelope format.
- [ ] **CROS-02**: All external dependencies (NATS, Minio, Postgres, Redis) are injected via functional options constructors, consistent with v1.0's provider pattern. Tests use fakes or test containers.
- [ ] **CROS-03**: Existing v1.0 code (`internal/hltv`, `internal/provider`, `internal/cli`, `cmd/dem/`) is NOT modified. Services import v1.0 packages as library code only.

## v2 Requirements

Deferred to future milestone.

### Analytics

- **ANLT-01**: User can query grenade throw patterns — popularity by map, by grenade type (HE, flash, smoke, molotov, decoy), and by team.
- **ANLT-02**: User can view spatial clustering of grenade landing positions on per-map layouts.
- **ANLT-03**: Grenade trajectory data (per-tick position) is stored for heatmap generation.

### Observability

- **OBSV-01**: Prometheus metrics track pipeline throughput, queue depth, error rates, and parse latency.
- **OBSV-02**: Grafana dashboard visualizes pipeline health.

### Query API

- **QAPI-01**: `dem match <id>` returns parsed match stats, round timelines, and player performance as JSON.

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Web UI / dashboard | Connect Metabase/Superset to Postgres instead |
| Live CSTV+ match parsing | Fundamentally different architecture, separate project |
| Multiple parsing engines | demoinfocs-golang v5 is production-proven at HLTV.org scale |
| Aggressive polling (< 6 hours) | Risks HLTV IP blocking, no benefit for Tier 1 |
| Kubernetes deployment | Docker Compose is the right tool for 3 services |
| gRPC between services | NATS pub/sub handles pipeline messages more naturally |
| PostGIS / pgvector / ClickHouse | Deferred until spatial/vector analytics are in-scope |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| INFR-01 | TBD | Pending |
| INFR-02 | TBD | Pending |
| INFR-03 | TBD | Pending |
| INFR-04 | TBD | Pending |
| INFR-05 | TBD | Pending |
| INFR-06 | TBD | Pending |
| POLL-01 | TBD | Pending |
| POLL-02 | TBD | Pending |
| POLL-03 | TBD | Pending |
| DWLD-01 | TBD | Pending |
| DWLD-02 | TBD | Pending |
| DWLD-03 | TBD | Pending |
| PARS-01 | TBD | Pending |
| PARS-02 | TBD | Pending |
| PARS-03 | TBD | Pending |
| PARS-04 | TBD | Pending |
| PARS-05 | TBD | Pending |
| CROS-01 | TBD | Pending |
| CROS-02 | TBD | Pending |
| CROS-03 | TBD | Pending |

**Coverage:**
- v1.1 requirements: 20 total
- Mapped to phases: 0 (roadmap creation will map)

---
*Requirements defined: 2026-05-03*

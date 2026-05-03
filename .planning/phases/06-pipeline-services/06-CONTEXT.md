# Phase 6: Pipeline Services - Context

**Gathered:** 2026-05-03
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase implements the three pipeline services that run on Phase 5's infrastructure foundation. The poller discovers Tier 1 HLTV matches with demos and publishes download jobs to NATS. The downloader consumes those jobs, streams .dem.gz files from HLTV's CDN directly to MinIO, and publishes parse jobs. The parser consumes parse jobs, streams demos from MinIO through demoinfocs-golang v5, and inserts structured game events into Postgres. The system is idempotent end-to-end — re-running any stage produces no duplicate data.

No new infrastructure is created. No analytics, no query API, no metrics. Those are Phase 7+ (v2).
</domain>

<decisions>
## Implementation Decisions

### Poller Scheduling & Deduplication
- **D-01:** Scheduling uses `robfig/cron` v3 with a configurable cron expression (default: daily at 02:00 UTC, minimum interval guard of 6 hours). The poller reuses v1.0 `internal/hltv` parsers without modification.
- **D-02:** Deduplication uses a `processed_matches` table in Postgres with `match_id BIGINT PRIMARY KEY` and `INSERT ... ON CONFLICT DO NOTHING`. More reliable than Redis SETNX with TTL (no expiry edge cases). Add `processed_at TIMESTAMPTZ DEFAULT NOW()` for auditability.

### Downloader Streaming & Retry
- **D-03:** Internal retry loop (3 attempts, exponential backoff: 5s → 25s → 125s) before falling back to `msg.NakWithDelay()`. Transient CDN issues are handled locally; only persistent failures escalate to NATS redelivery.
- **D-04:** Streaming download from HLTV CDN to MinIO: `http.Response.Body` pipes directly to `minio.Client.PutObject()` via `io.Reader`. Zero local disk writes. The existing v1.0 `internal/hltv.Client` is reused for the HTTP transport layer.
- **D-05:** Downloader publishes parse jobs to `dem.parse.jobs` with `{bucket, object_key, match_id, match_url}` as the message payload. Uses JSON-encoded structs for NATS messages.

### Parser Architecture & Performance
- **D-06:** Single parser at a time (`MaxAckPending: 1`), configurable via env var `DEM_PARSER_CONCURRENCY`. The code structure supports bumping to 2+ after memory profiling. ~250 MB peak per parser instance.
- **D-07:** Batch insert per round into Postgres (40-50 kill events per round batched in a single `pgx.Batch` or multi-row INSERT). Deterministic event IDs for idempotency: `{match_id}-{round_number}-{event_type}-{sequence}`. All write paths use `INSERT ... ON CONFLICT DO NOTHING`.
- **D-08:** 12+ event handlers registered: MatchStart, RoundStart, RoundEnd, Kill, PlayerHurt, WeaponFire, BombPlant, BombDefuse, BombExplode, GrenadeProjectileThrow, PlayerConnect, TeamSideSwitch. Handlers collect events into per-round slices and flush on RoundEnd.
- **D-09:** `defer p.Close()` on every demoinfocs parser instance — non-negotiable. 250 MB C-memory leak per unclosed parser.
- **D-10:** `defer msg.Ack()` at the top of every NATS handler — non-negotiable. Only Ack after successful Postgres insert. `msg.NakWithDelay()` on transient failures.

### Service Scaffolding
- **D-11:** A shared `internal/service/` package provides lifecycle primitives: `Service` interface with `Run(ctx) error`, `NewRunner(opts ...RunnerOption)` for dependency wiring, and signal-aware graceful shutdown. Each `cmd/<service>/main.go` is ~30 lines: load config, wire deps via functional options, call `runner.Run()`.
- **D-12:** Config via environment variables with Viper defaults. No config files in containers (12-factor). Each service defines its own `Config` struct with defaults.

### Prior Decisions (binding, from Phase 5 + research)
- All external dependencies (NATS, Minio, Postgres) injected via functional options — consistent with v1.0 and Phase 5.
- Structured logging via `log/slog` with `match_id` and `job_id` correlation fields — no v1.0 JSON envelopes in service output.
- NATS streams (`DEM_DOWNLOAD`, `DEM_PARSE`) already provisioned by `pkg/natsutil`.
- `pgxpool.Pool` with MaxConns=20 already configured via `pkg/postgres`.
- Single `go.mod` monorepo — services compile alongside existing `cmd/dem/`.
- v1.0 code (`internal/hltv`, `internal/domain/models.go`) is read-only — imported as library, never modified.

### Claude's Discretion
- Exact NATS message payload format (JSON struct fields).
- demoinfocs event handler registration order and internal buffering strategy.
- `internal/service/runner.go` API surface and type names.
- Exact backoff jitter implementation for downloader retry.
- `processed_matches` table schema details beyond `match_id` and `processed_at`.
- Plan split — planner decides how to group the three services into plans/waves.
</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase Scope
- `.planning/ROADMAP.md` — Phase 6 boundary, success criteria (6 items), requirements POLL-01 through CROS-03.
- `.planning/REQUIREMENTS.md` — All 14 Phase 6 requirements with descriptions.

### Phase 5 Artifacts (infrastructure this phase runs on)
- `.planning/phases/05-infrastructure-foundation/05-01-SUMMARY.md` — Monorepo scaffolding, go.mod deps, cmd/ entrypoints, domain types.
- `.planning/phases/05-infrastructure-foundation/05-02-SUMMARY.md` — docker-compose.yml, 12 SQL migrations (6 tables).
- `.planning/phases/05-infrastructure-foundation/05-03-SUMMARY.md` — pkg/natsutil, pkg/minio, pkg/postgres with functional options.
- `.planning/phases/05-infrastructure-foundation/05-CONTEXT.md` — Phase 5 decisions: monorepo layout, schema, NATS config, package APIs.

### Research (comprehensive, MUST read)
- `.planning/research/SUMMARY.md` — Pipeline architecture, phase ordering rationale, pitfall-to-phase mapping.
- `.planning/research/ARCHITECTURE.md` — Service boundaries, data flow, NATS subject design, monorepo layout.
- `.planning/research/PITFALLS.md` — Critical pitfalls (msg.Ack, streaming, defer p.Close, idempotency, pgxpool).

### v1.0 Reusable Code (read-only library)
- `internal/hltv/client.go` — `Client.Fetch(ctx, url)` — reused by poller and downloader for HTTP requests.
- `internal/hltv/parser/events.go` — `ParseEvents(io.Reader, sourceURL)` — reused by poller for match discovery.
- `internal/hltv/parser/results.go` — `ParseResults(io.Reader, sourceURL)` — reused by poller for match discovery.
- `internal/hltv/parser/demo.go` — `ParseDemoLink(io.Reader, matchURL)` — reused by poller for demo URL extraction.
- `internal/hltv/urls.go` — `EventsURL()`, `ResultsURL()`, `MatchURL(matchID)` — reused by poller.
- `internal/domain/models.go` — `Event`, `Result`, `DemoLink` types — reused by poller.

### External References
- demoinfocs-golang v5: https://pkg.go.dev/github.com/markus-wa/demoinfocs-golang/v5
- robfig/cron v3: https://pkg.go.dev/github.com/robfig/cron/v3
- NATS JetStream Go: https://github.com/nats-io/nats.go
</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets (from v1.0 + Phase 5)
- `internal/hltv/client.go` — `Client.Fetch(ctx, url)` — HTTP client with timeout, user-agent, TLS config. Used by poller for HLTV pages, downloader for CDN .dem.gz files.
- `internal/hltv/parser/` — All three parsers (events, results, demo) reused as-is by poller for match discovery.
- `internal/domain/models.go` — `Event`, `Result`, `DemoLink` with JSON tags. Poller depends on these.
- `internal/domain/game_events.go` — `MatchMetadata`, `KillEvent`, `RoundInfo`, `DamageEvent`. Parser populates these from demoinfocs events.
- `pkg/natsutil/` — `NewNATSConn`, `CreateStreams`, `VerifyStreams`, subject/stream constants. All three services import this.
- `pkg/minio/` — `NewMinioClient`, `EnsureBucket`, `DefaultBucket`. Downloader and parser import this.
- `pkg/postgres/` — `NewPool` (pgxpool, MaxConns=20). Poller (dedup) and parser (insert) import this.
- `cmd/poller/main.go`, `cmd/downloader/main.go`, `cmd/parser/main.go` — Phase 5 skeleton entrypoints. Phase 6 fills in the actual run logic.

### Established Patterns
- **Functional options**: `NewXxxProvider(opts ...XxxProviderOption)` — carried from v1.0 through Phase 5. Every service constructor uses this.
- **Injectable dependencies**: Services receive NATS, Minio, Postgres connections via constructor, not globals.
- **Structured logging**: `log/slog` with `slog.String("match_id", id)`, `slog.String("job_id", id)`.
- **Skeleton entrypoints**: Existing `cmd/*/main.go` have `log/slog` setup and placeholder comments. Phase 6 replaces `// TODO` with real logic.

### Integration Points
- `cmd/poller/main.go` → imports and wires NATS, Postgres, v1.0 parsers.
- `cmd/downloader/main.go` → imports and wires NATS, Minio, v1.0 HTTP client.
- `cmd/parser/main.go` → imports and wires NATS, Minio, Postgres, demoinfocs-golang.
- All three share `internal/service/runner.go` for lifecycle management.
</code_context>

<specifics>
## Specific Ideas

- The user consistently defers to "best practice" — Go community standards, library author recommendations, production-tested patterns.
- The three services should feel like natural peers — consistent structure, same logging format, same error handling philosophy.
- Conservative defaults preferred: single parser, daily polling, explicit backoff. All tunable via config later.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within Phase 6 scope. All v2 items (grenade analytics, Prometheus, Grafana, query API) already in REQUIREMENTS.md v2 section.
</deferred>

---

*Phase: 06-Pipeline Services*
*Context gathered: 2026-05-03*

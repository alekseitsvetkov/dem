# Feature Research

**Domain:** CS2 demo pipeline — tournament polling, demo download, demo parsing
**Researched:** 2026-05-03
**Confidence:** HIGH

## Feature Landscape

### Table Stakes (Users Expect These)

Features users assume exist. Missing these = product feels incomplete.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Scheduled Tier 1 tournament polling (cron-style daily check) | Any automated demo pipeline must discover new completed matches without manual intervention. The standard pattern (HLTV-match-predictor, OpenDota scanner) is a cron-driven poll against HLTV event/results pages. | MEDIUM | Depends on existing `internal/hltv` Client and `ParseEvents`/`ParseResults` parsers from v1.0. Must deduplicate against already-seen match IDs. Respectful polling with configurable intervals and backoff. |
| Demo download from HLTV to persistent storage | Once a match is discovered and has a demo link, the file must be fetched and stored durably. HLTV serves `.dem.gz` (gzip-compressed GOTV demo, 50-200 MB). This is the basic "grab the file" operation any pipeline needs. | MEDIUM | Depends on v1.0 `ParseDemoLink` for demo URL extraction. Must handle HTTP range requests for resumability, gzip decompression verification, and Minio upload with retry. |
| Demo parsing with demoinfocs-golang into structured game events | The parsed game events (kills, rounds, bomb events, grenades) are the core data product. Without parsing, the downloaded `.dem` is just an opaque binary. demoinfocs-golang is the de facto standard Go parser (used by HLTV.org, Leetify, scope.gg). | HIGH | Depends on download service completing. demoinfocs-golang v5 parses ~25 min gameplay/second, uses ~250 MB RAM per demo. Must stream from Minio (no local temp file required). Outputs typed game events for Postgres insertion. |
| Persist parsed game events to Postgres | Parsed data must survive restarts and support queries. In-memory-only parsing is useless beyond a single session. The postgres schema becomes the source of truth for all analytics queries. | MEDIUM | Schema design must balance query flexibility with write throughput. Batch-insert within transactions per round or per N events. Key tables: matches, rounds, kills, grenades, player_positions. |
| Detection of already-processed matches (idempotency) | Re-downloading and re-parsing the same match wastes bandwidth, CPU, and storage. Any production pipeline must recognize duplicate match IDs and skip them. This is as fundamental as dedup in a message queue. | LOW | Depends on Postgres: check existence by HLTV match_id before enqueuing download job. NATS JetStream also supports idempotency via message deduplication and KV-store state tracking. |
| Error resilience with retry and dead-letter queuing | Network failures (HLTV down, Minio unavailable, parsing panics) are inevitable. A broken pipeline without retry/DLQ silently drops matches forever. Every production event-driven system (OpenDota, Geqo Observer) uses DLQ patterns. | MEDIUM | NATS JetStream: set `MaxDeliver` and `AckWait` for automatic retry with backoff. Failed-after-N-attempts messages route to a DLQ stream for operator inspection. demoinfocs-golang parser must run in a recoverable goroutine to prevent crashes from killing the service. |
| Observability: structured logging + health checks | Operators need to know if the pipeline is running, stuck, or failing. Without visibility, silent failures accumulate. The v1.0 CLI already uses structured JSON errors on stderr — the same discipline must extend to services. | LOW | Each service must expose a `/health` HTTP endpoint and emit structured logs with correlation IDs (match_id, job_id). Prometheus metrics for job latency, success/failure rates, queue depth. |

### Differentiators (Competitive Advantage)

Features that set the product apart. Not required, but valuable.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Event-driven job pipeline with NATS JetStream (not cron-only) | Most CS demo tools are monolithic cron scripts. A NATS-backed pipeline decouples polling, downloading, and parsing — each can scale, fail, and recover independently. This architecture enables horizontal scaling: multiple download workers and parser workers across machines, unlike single-process scripts. | HIGH | Requires NATS cluster setup, JetStream stream/consumer configuration, KV-store for state tracking. The complexity pays off when processing hundreds of matches concurrently. This is the project's architectural differentiator vs tools like HLTVDemoDownloader. |
| Single-command setup with Docker Compose | Most demo pipeline projects (hltv-utility-api, csgo-tools) require manual Go/Python environment setup, database provisioning, and config wrangling. A `docker-compose up` that brings up NATS, Minio, Postgres, and all services reduces the barrier from "system administration" to "developer tool." | MEDIUM | docker-compose.yml with pre-configured services. Environment variables for HLTV endpoints, Minio credentials, Postgres DSN. Health-check dependencies between services. |
| Parse demos directly from Minio stream (no local temp files) | Storing demos locally before parsing wastes disk space and adds I/O overhead. Streaming from Minio via `GetObject()` -> `demoinfocs.NewParser(reader)` -> parse -> Postgres eliminates a whole temp-file lifecycle. Most tools download to disk, parse, then delete. | LOW (once Minio client is wired) | Depends on Minio Go SDK `GetObject()` returning an `io.Reader`. demoinfocs-golang's `NewParser` accepts `io.Reader`. The demo is already gzip-compressed — demoinfocs handles GZIP internally so no separate decompression step is needed. |
| Full grenade trajectory storage (future: grenade analytics) | Most platforms track aggregate grenade stats (flash time, damage). Storing full per-tick trajectory points enables spatial clustering, heatmap generation, team throw pattern analysis — capabilities that only Noesis.gg (9.99/month) offers among consumers. This is the foundation for the planned grenade analytics milestone. | HIGH (deferred to future milestone) | The `grenade_trajectories` table grows large (every tick of every grenade flight). demoinfocs-golang's `GrenadeProjectile.Trajectory` provides all points. Requires careful schema design (partitioning, sampling) for production scale. Deferred: store trajectory points for v1.1, analytics queries in a future milestone. |
| Programmatic JSON API over the parsed data | v1.0 CLI already emits JSON for events/results/demos. Extending this to serve parsed game events via a query API (match stats, player performance, round timeline) preserves the JSON-contract philosophy and enables scripted analysis. This is the data-access differentiator: raw PostgreSQL-level access for power users, not a curated web UI. | MEDIUM | A thin query API over Postgres views. Read-only, no auth (same as v1.0 CLI philosophy). Could be a separate service or integrated into the CLI as `dem match <id>` or `dem stats <match-id>`. |

### Anti-Features (Commonly Requested, Often Problematic)

Features that seem good but create problems.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Real-time match tracking / live GOTV parsing | "Watch live matches and parse stats as they happen" seems like an obvious next step. | HLTV GOTV streams require constant connection, have different URLs, and the v1.0 infrastructure is built around completed-match scraping. Live parsing introduces Websocket management, reconnection logic, partial-match state, and fundamentally different parsing mode (`ParseCSTVBroadcast()` vs `ParseToEnd()`). It also risks aggressive HLTV access patterns that could trigger blocking. | Focus on completed matches only. The milestone explicitly scopes this out. If live parsing is needed later, build it as a separate service with a different parsing path — do not retrofit into the existing pipeline. |
| Web UI or dashboard for viewing parsed data | "Show me a web page with match stats" is the most common request after any data pipeline. | A web UI introduces frontend framework choices, authentication, session management, and a completely different development workflow from the Go CLI/microservice monorepo. It would consume engineering time better spent on pipeline reliability and data quality. | CLI JSON output and a read-only Postgres query API serve the same need for scripters and power users. For visualization, users can point Metabase/Superset/grafana at the Postgres database — these tools already do dashboards better than anything we'd build. |
| Demo file re-download on re-parse request | "If the parser improves, let users re-parse from the original HLTV URL" seems efficient. | HLTV demo links expire. The `/download/demo/{id}` URL may return 404 days later. The downloaded `.dem.gz` in Minio is the only durable copy. Re-parsing should always read from Minio, not re-request from HLTV. | Store every downloaded demo in Minio permanently (or with a configurable retention policy). Re-parse from Minio, not from HLTV. The Minio-stored binary is the source of truth for re-parsing. |
| Store raw .dem files in Postgres as BYTEA | "One database for everything" simplifies operations. | .dem.gz files are 50-200 MB each. Storing them in Postgres bloats the database, degrades query performance, complicates backups, and wastes the object store's purpose. Postgres is for structured queryable data; Minio is for opaque binary blobs. | Store demo metadata (match_id, map, teams, demo_key in Minio) in Postgres. The actual .dem.gz binary lives in Minio. The Postgres row references the Minio object key. |
| Aggressively fast polling intervals (< 1 hour) | "Check for new demos every 5 minutes to be as real-time as possible" appeals to impatience. | HLTV is a publicly scraped website, not an API. Polling too frequently risks IP blocking, degrades the service for everyone, and violates the project's politeness constraint. New Tier 1 matches appear once or twice daily — sub-hour polling provides no real benefit. | Default to daily polling (once per 24 hours at a configurable time). Make the interval configurable but document a minimum of 6 hours. The cron service respects a minimum interval guard. |
| Fan-out to multiple parsing engines | "Use demoinfocs AND demofile (Node.js) AND awpy (Python) for cross-validation" sounds thorough. | Multiple parsing engines means maintaining multiple language runtimes, debugging cross-engine discrepancies, and multiplying the surface area for bugs. demoinfocs-golang is production-proven at HLTV.org scale — cross-validation adds complexity without proportionate value. | Use demoinfocs-golang v5 exclusively. If a second parser is ever needed, run it as a separate optional service that reads from the same Minio bucket — never make it a required pipeline step. |

## Feature Dependencies

```
Tournament Poller (cron)
    |---requires---> v1.0 ParseEvents parser (discovers Tier 1 events)
    |---requires---> v1.0 ParseResults parser (finds new completed matches)
    |---requires---> v1.0 ParseDemoLink parser (extracts demo URLs)
    |---requires---> NATS JetStream (publishes download jobs)
                           |
                           v
                    Demo Download Service
                           |---requires---> NATS JetStream (consumes download jobs)
                           |---requires---> v1.0 HLTV Client (fetches demo files)
                           |---requires---> Minio (stores .dem.gz blobs)
                           |---requires---> Postgres (checks/records match state)
                           |---publishes---> NATS JetStream (parse jobs)
                                               |
                                               v
                                        Demo Parsing Service
                                               |---requires---> NATS JetStream (consumes parse jobs)
                                               |---requires---> Minio (streams .dem.gz for parsing)
                                               |---requires---> demoinfocs-golang v5 (parses .dem)
                                               |---requires---> Postgres (stores game events)

Grenade Analytics (FUTURE milestone)
    |---requires---> Postgres game events (reads grenade_trajectories, kills, rounds)
    |---requires---> demo parsing completion (needs parsed data to exist)

Programmatic Query API (potential v1.1 add-on)
    |---requires---> Postgres game events (reads parsed data)
```

### Dependency Notes

- **Tournament Poller requires NATS JetStream:** The poller doesn't download or parse itself — it publishes jobs to NATS topics. The JetStream server must be running first. This is the entry point of the pipeline.
- **Demo Download Service requires Minio:** Binaries must have a persistent store before parsing can read them. Minio must be provisioned with a bucket before downloads begin.
- **Demo Parsing Service requires both Minio and Postgres:** The parser streams demos from Minio and writes structured events to Postgres. Neither can be optional.
- **v1.0 parsers are reused, not replaced:** The `internal/hltv/parser` package from v1.0 (ParseEvents, ParseResults, ParseDemoLink) provides match discovery. The v1.1 services consume these parsers directly — no rewrite needed.
- **Grenade Analytics is fully dependent on parsing service completion:** All grenade trajectory data originates from the parser writing to Postgres. The analytics layer is read-only over Postgres — no new data ingestion needed.

## MVP Definition

### Launch With (v1.1 — Microservice Platform)

Minimum viable product — what's needed to validate the pipeline concept.

- [ ] **Tournament Polling Service** — Daily cron-style check of Tier 1 HLTV events, detect newly completed matches with demo links, publish download jobs to NATS. No manual intervention for match discovery.
- [ ] **Demo Download Service** — Consume download jobs from NATS, fetch `.dem.gz` from HLTV download URLs, upload to Minio, publish parse jobs on success. Handles transient failures with retry.
- [ ] **Demo Parsing Service** — Consume parse jobs from NATS, stream `.dem.gz` from Minio into demoinfocs-golang, persist kills/rounds/grenades/bomb events to Postgres. Handles parse failures with DLQ routing.
- [ ] **Docker Compose orchestration** — Single-command `docker-compose up` that starts NATS, Minio, Postgres, and all three services with correct startup ordering and health checks.
- [ ] **Idempotency across the pipeline** — No match is downloaded or parsed twice. Poller checks Postgres before publishing. Download/parse workers check before processing.
- [ ] **Structured logging with correlation IDs** — Every log line tagged with match_id and job_id for traceability across services.

### Add After Validation (v1.1.x)

Features to add once core is working.

- [ ] **Programmatic query API** — `dem match <id>` to retrieve parsed match stats, round timelines, and player performance as JSON. Extends the v1.0 JSON-contract philosophy to parsed data.
- [ ] **Grenade trajectory storage** — Full per-tick trajectory points in Postgres (schema exists but write path deferred to avoid bloating MVP).
- [ ] **Prometheus metrics + Grafana dashboard** — Pipeline throughput, queue depth, error rates, parse latency.
- [ ] **Configurable polling interval with minimum guard** — Allow operators to adjust from daily to a minimum of 6 hours, with guardrails in the service config.

### Future Consideration (v2+)

Features to defer until product-market fit is established.

- [ ] **Grenade analytics engine** — Spatial clustering, team throw pattern detection, heatmap generation. Separate milestone.
- [ ] **Web dashboard** — Not planned; users can point BI tools at the Postgres database.
- [ ] **Live match parsing** — Requires fundamentally different architecture; separate project.
- [ ] **Multi-account demo downloading** — Not needed for HLTV public downloads (no Steam GC rate limits for HLTV-hosted demos).

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Tournament Polling Service | HIGH (without this, everything is manual) | MEDIUM (reuses v1.0 parsers, new NATS integration) | P1 |
| Demo Download Service | HIGH (bridges HLTV and local storage) | MEDIUM (HTTP download + Minio upload, retry logic) | P1 |
| Demo Parsing Service | HIGH (the actual data product) | HIGH (demoinfocs integration, Postgres schema, memory management) | P1 |
| Docker Compose orchestration | HIGH (developer experience, reproducibility) | MEDIUM (config files, health checks, startup ordering) | P1 |
| Pipeline idempotency | HIGH (silent duplicates corrupt data quality) | LOW (existence checks at each stage) | P1 |
| Structured logging with correlation IDs | MEDIUM (debuggability) | LOW (log wrapper with context fields) | P1 |
| Programmatic query API | MEDIUM (extends JSON contract) | MEDIUM (new CLI command or HTTP endpoint) | P2 |
| Grenade trajectory storage (full) | MEDIUM (foundation for analytics) | HIGH (schema + write path for high-volume data) | P2 |
| Prometheus + Grafana | MEDIUM (operational visibility) | LOW (standard instrumentation) | P2 |
| Configurable polling interval | LOW (daily is sufficient for most use cases) | LOW (config flag with guard) | P3 |

**Priority key:**
- P1: Must have for launch (no pipeline works without these)
- P2: Should have, add when core pipeline is stable
- P3: Nice to have, future consideration

## Competitor Feature Analysis

| Feature | HLTV Demo Downloader (Python) | hltv-utility-api (Go+Python) | gigobyte/HLTV (Node.js) | Our Approach (v1.1) |
|---------|------------------------------|------------------------------|--------------------------|---------------------|
| Match discovery | Event ID scrape -> regex | GitHub Actions cron + HLTV scrape | JS API wrapper for HLTV pages | Go cron service reusing v1.0 parsers |
| Demo download | Multi-threaded HTTP download to disk | Go download via GitHub Actions | N/A (API-only, no download) | NATS worker -> Minio, streaming (no temp files) |
| Demo parsing | None (download only) | demoinfocs-golang v2 | N/A | demoinfocs-golang v5 (latest) |
| Data storage | Filesystem folders | Static files on Vercel | N/A | Minio (demos) + Postgres (events) |
| Job orchestration | None (single script) | GitHub Actions workflows | N/A | NATS JetStream (at-least-once, retry, DLQ) |
| Service architecture | Monolithic script | GitHub Actions pipeline | Library | Docker Compose microservices |
| Grenade data | None | Full utility trajectory (flat arrays) | None | Event-driven: kills, rounds, grenade events in Postgres |
| Idempotency | None (duplicates on re-run) | None (re-parses everything) | N/A | Postgres-backed dedup at every pipeline stage |
| Error handling | None described | CI pipeline failure | N/A | Per-job retry with DLQ, structured errors |
| Programmability | CLI only | REST API (3 endpoints) | JS function calls | CLI JSON + potential query API |

## Sources

- [demoinfocs-golang v5 — GoDoc](https://pkg.go.dev/github.com/markus-wa/demoinfocs-golang/v5) — Official API reference, event types, parser interface. HIGH confidence.
- [demoinfocs-golang — DeepWiki](https://deepwiki.com/markus-wa/demoinfocs-golang) — Architecture overview, event list, performance benchmarks (0.89s/demo, 250MB RAM). HIGH confidence.
- [markus-wa/demoinfocs-golang — GitHub](https://github.com/markus-wa/demoinfocs-golang) — Primary repository. v5.1.2 released 2026-01-23. Used by HLTV.org, Leetify, scope.gg, noesis.gg, refrag.gg, PureSkill.gg. HIGH confidence.
- [HLTVDemoDownloader — GitHub](https://github.com/hooolius/HLTVDemoDownloader) — Canonical HLTV demo download workflow: Event ID -> Match IDs -> Demo IDs -> bulk download. MEDIUM confidence (Python 2.7, regex-based, fragile).
- [hltv-utility-api — GitHub](https://github.com/hx-w/hltv-utility-api) — Closest reference architecture: Go-based demo download + demoinfocs parsing for utility data. GitHub Actions cron, Vercel deployment. MEDIUM confidence (archived, uses demoinfocs v2).
- [hltv-match-predictor — GitHub](https://github.com/ratx64/hltv-match-predictor) — Dual-cron schedule pattern: morning predictions + evening results. Configurable cron, retry jobs. MEDIUM confidence.
- [OpenDota Architecture Blog](http://blog.opendota.com/2016/05/15/architecture) — Proven pipeline pattern: scanner -> retriever -> parser -> database with distributed workers and rate limiting. MEDIUM confidence (Dota 2 but same architectural challenges).
- [Geqo Observer — GitHub](https://github.com/geqo/cs2-observer) — State-of-the-art CS2 event-driven microservice: Kafka, DLQ, TimescaleDB, stateless workers. MEDIUM confidence.
- [NATS JetStream + Go patterns](https://memphis.dev/blog/building-distributed-systems-with-nats-jetstream-and-golang/) — Push/pull consumers, durable subscriptions, queue groups, KV store, DLQ patterns. HIGH confidence.
- [Minio Go Client SDK v7 — GoDoc](https://pkg.go.dev/github.com/minio/minio-go/v7) — PutObject, GetObject, FGetObject, bucket operations. HIGH confidence.
- [Scope.gg](https://scope.gg/), [Leetify](https://leetify.com/), [Noesis.gg](https://www.noesis.gg/) — Major CS2 demo analytics platforms. Feature comparison via web analysis. MEDIUM confidence (marketing pages, not technical docs).

---
*Feature research for: CS2 demo pipeline — tournament polling, demo download, demo parsing*
*Researched: 2026-05-03*

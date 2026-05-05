# Milestones

## v1.1 Microservice Platform

**Shipped:** 2026-05-06
**Phases:** 5-6 | **Plans:** 7

### What Was Built

Extended the v1.0 CLI into an automated microservice platform: Poller discovers Tier 1 HLTV matches and publishes download jobs to NATS. Downloader consumes jobs, solves Cloudflare challenges via cloudscraper, streams `.rar` demos from HLTV CDN to MinIO, extracts `.dem` files via 7z, and publishes parse jobs. Parser consumes parse jobs, streams demos from MinIO through demoinfocs-golang v5, and inserts structured game events into PostgreSQL with idempotent writes. All services use structured logging via `log/slog`.

### Key Accomplishments

1. **Infrastructure Foundation** — Docker Compose with NATS/PostgreSQL/MinIO/Redis, 8-table schema via golang-migrate, shared packages (pkg/natsutil, pkg/minio, pkg/postgres) with functional options
2. **Poller Service** — cron-driven Tier 1 HLTV match discovery reusing v1.0 parsers, Postgres-based dedup, NATS JetStream job publishing
3. **Downloader Service** — Cloudflare Turnstile bypass via Python cloudscraper, RAR extraction, zero-disk streaming to MinIO, retry with exponential backoff, conditional defer Ack/Nak pattern
4. **Parser Service** — MinIO streaming to demoinfocs-golang, 12 event handlers, pgx.Batch per-round inserts, idempotent `ON CONFLICT (event_id) DO NOTHING`, auto-cleanup of MinIO objects
5. **Cross-Cutting** — Structured slog logging with correlation fields, functional options everywhere, zero v1.0 code modifications

### Stats

- **Source commits:** 42
- **Codebase:** ~16,000 LOC Go
- **Services:** 4 (dem CLI + poller + downloader + parser)
- **Database:** 5 matches, 95 rounds, 616 kills, 2,396 damage events (verified)
- **UAT:** 9/9 passed

### Decisions Made

- Single `go.mod` monorepo for all services
- NATS JetStream with WorkQueue retention and durable pull consumers
- pgxpool.Pool with MaxConns=20
- `INSERT ... ON CONFLICT DO NOTHING` for all write paths
- Conditional defer Ack/Nak pattern (never call both)
- cloudscraper subprocess for Cloudflare challenge bypass
- 7z/unar for RAR extraction
- MinIO auto-cleanup after successful parse
- Native downloader (not Docker) due to R2 CDN IP blocking

### Technical Debt

- Downloader runs natively, not containerized (Docker Desktop VM IP blocked)
- Parser MapName is empty (demoinfocs-golang v5 limitation)
- Poller fetches match pages sequentially (could parallelize with rate limiting)

---

## v1.0 MVP

**Shipped:** 2026-05-03
**Phases:** 1-4 | **Plans:** 9

### What Was Built

Go CLI tool for retrieving Counter-Strike data from HLTV.org. Three commands: `dem events --tier 1`, `dem results --limit N`, `dem demo <match-id>`. JSON stdout contract with `{data, meta}` envelope. Structured errors on stderr.

### Key Accomplishments

1. CLI Foundation — Cobra command tree, JSON output, structured errors
2. HLTV Provider Infrastructure — uTLS HTTP client, provider interfaces, HTML parsers, fixture tests
3. Events and Results Commands — Tier 1 filtering, pagination, result limits
4. Demo Link Lookup — Demo URL extraction from live HLTV match pages

### Stats

- Source commits: ~30
- Codebase: 3,136 LOC Go
- Commands: 3
- Test coverage: 34 tests (fixture-based, no network dependency)

---
*Last updated: 2026-05-06*

# Roadmap: HLTV CLI

## Milestones

- ✅ **v1.0 MVP** — Phases 1-4 (shipped 2026-05-03)
- 🚧 **v1.1 Microservice Platform** — Phases 5-6 (in progress)

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

<details>
<summary>✅ v1.0 MVP (Phases 1-4) — SHIPPED 2026-05-03</summary>

- [x] **Phase 1: CLI Foundation** (2/2 plans) — Go CLI skeleton, Cobra commands, JSON output, structured errors
- [x] **Phase 2: HLTV Provider Infrastructure** (2/2 plans) — HTTP client, provider interfaces, parser, fixtures
- [x] **Phase 3: Events and Results Commands** (3/3 plans) — `dem events`, `dem results` with filtering/limit
- [x] **Phase 4: Demo Link Lookup** (2/2 plans) — `dem demo <match-id>` with live HLTV selectors

See: `.planning/milestones/v1.0-ROADMAP.md` for full phase details.

</details>

### 🚧 v1.1 Microservice Platform (In Progress)

**Milestone Goal:** Extend the v1.0 CLI into an automated microservice platform that discovers Tier 1 tournament matches, downloads `.dem.gz` replay files, parses them into structured game events, and stores everything in a queryable PostgreSQL database.

#### Phase 5: Infrastructure Foundation

- [ ] **Phase 5: Infrastructure Foundation** — Docker Compose, NATS/Postgres/Minio/Redis, DB schema, shared packages, monorepo scaffolding.

#### Phase 6: Pipeline Services

- [ ] **Phase 6: Pipeline Services** — Poller discovers matches and publishes jobs; Downloader streams demos to MinIO; Parser extracts game events into Postgres.

## Phase Details

### Phase 5: Infrastructure Foundation
**Goal**: The microservice platform has a running infrastructure foundation — all backing services healthy, database schema migrated, shared libraries available, and developer tooling in place.
**Depends on**: Phase 4 (v1.0 codebase)
**Requirements**: INFR-01, INFR-02, INFR-03, INFR-04, INFR-05, INFR-06
**Success Criteria** (what must be TRUE):
  1. Developer runs `docker-compose up` and NATS, PostgreSQL, MinIO, and Redis all start with passing health checks.
  2. Developer runs `go build ./cmd/...` and every service entrypoint (poller, downloader, parser, dem) compiles from a single root `go.mod`.
  3. Database migrations (`sql/migrations/`) create the full v1.1 schema — `matches`, `rounds`, `kill_events`, `damage_events`, `players`, `match_players` — all with appropriate constraints and indexes.
  4. Shared packages (`pkg/natsutil`, `pkg/minio`, `pkg/postgres`) are importable and provide connection helpers with functional options. Postgres connections use `pgxpool` with explicit `MaxConns` (10-25).
  5. NATS JetStream streams are created programmatically at service startup — publishing to a subject with no backing stream fails fast with a clear error.
**Plans**: 3 plans in 2 waves

Plans:
- [ ] 05-01-PLAN.md — Monorepo scaffolding: go.mod, cmd/ entrypoints, Makefile, domain types
- [ ] 05-02-PLAN.md — Docker Compose infrastructure + database migrations (6 tables, 12 SQL files)
- [ ] 05-03-PLAN.md — Shared infrastructure packages: pkg/natsutil, pkg/minio, pkg/postgres

Wave structure:
- Wave 1: Plan 01 (scaffolding) + Plan 02 (Docker/migrations) — parallel, no file overlap
- Wave 2: Plan 03 (shared packages) — after Plan 01 (depends on go.mod deps)

### Phase 6: Pipeline Services
**Goal**: The full data pipeline runs end-to-end — Tier 1 tournament matches are automatically discovered, demo files downloaded to MinIO, game events parsed into Postgres, and the system is idempotent and observable.
**Depends on**: Phase 5
**Requirements**: POLL-01, POLL-02, POLL-03, DWLD-01, DWLD-02, DWLD-03, PARS-01, PARS-02, PARS-03, PARS-04, PARS-05, CROS-01, CROS-02, CROS-03
**Success Criteria** (what must be TRUE):
  1. Poller service discovers Tier 1 matches with available demos and publishes download jobs to NATS `dem.download.jobs`. Running the poller twice against the same HLTV data produces no duplicate jobs.
  2. Downloader service consumes a download job, streams the `.dem.gz` file from HLTV CDN directly to MinIO (no local disk write), and publishes a parse job to `dem.parse.jobs` on success.
  3. Parser service consumes a parse job, streams the demo from MinIO through demoinfocs-golang (never buffered in memory), and inserts game events — kills, rounds, damage, player data — into Postgres.
  4. Running the parser twice on the same demo produces identical data with no duplicate rows — idempotent `ON CONFLICT DO NOTHING` inserts work end-to-end.
  5. Every service emits structured log lines via `log/slog` with `match_id` and `job_id` correlation fields. No service uses the v1.0 CLI `{data, meta}` JSON envelope format.
  6. Existing v1.0 CLI commands (`dem events`, `dem results`, `dem demo`) continue to work without modification — the code path under `internal/hltv`, `internal/provider`, and `internal/cli` is untouched.
**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5 → 6

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. CLI Foundation | v1.0 | 2/2 | Complete | 2026-05-02 |
| 2. HLTV Provider Infrastructure | v1.0 | 2/2 | Complete | 2026-05-02 |
| 3. Events and Results Commands | v1.0 | 3/3 | Complete | 2026-05-02 |
| 4. Demo Link Lookup | v1.0 | 2/2 | Complete | 2026-05-03 |
| 5. Infrastructure Foundation | v1.1 | 0/3 | In planning | - |
| 6. Pipeline Services | v1.1 | 0/0 | Not started | - |

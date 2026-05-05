# Roadmap: HLTV CLI

## Milestones

- ✅ **v1.0 MVP** — Phases 1-4 (shipped 2026-05-03)
- ✅ **v1.1 Microservice Platform** — Phases 5-6 (shipped 2026-05-06)

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

<details>
<summary>✅ v1.1 Microservice Platform (Phases 5-6) — SHIPPED 2026-05-06</summary>

- [x] **Phase 5: Infrastructure Foundation** (3/3 plans) — Docker Compose, NATS/Postgres/Minio/Redis, DB schema, shared packages
- [x] **Phase 6: Pipeline Services** (4/4 plans) — Poller, Downloader, Parser — full automated pipeline

See: `.planning/milestones/v1.1-ROADMAP.md` for full phase details.

</details>

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5 → 6

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. CLI Foundation | v1.0 | 2/2 | Complete | 2026-05-02 |
| 2. HLTV Provider Infrastructure | v1.0 | 2/2 | Complete | 2026-05-02 |
| 3. Events and Results Commands | v1.0 | 3/3 | Complete | 2026-05-02 |
| 4. Demo Link Lookup | v1.0 | 2/2 | Complete | 2026-05-03 |
| 5. Infrastructure Foundation | v1.1 | 3/3 | Complete | 2026-05-03 |
| 6. Pipeline Services | v1.1 | 4/4 | Complete | 2026-05-06 |

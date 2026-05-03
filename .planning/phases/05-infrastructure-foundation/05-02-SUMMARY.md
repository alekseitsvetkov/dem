---
phase: 05-infrastructure-foundation
plan: 02
subsystem: infrastructure
tags: [docker-compose, postgresql, nats, minio, redis, migrations, sql]
depends_on:
  provides:
    - docker-compose.yml
    - sql/migrations (6 tables, 12 files)
  requires:
    - Phase 5 planning (05-02-PLAN.md)
  affects:
    - Phase 6 pipeline services (poller, downloader, parser)
tech-stack:
  added:
    - docker-compose.yml (services: nats, postgres, minio, redis)
    - golang-migrate migration convention (6-digit seq, up/down suffixes)
  patterns:
    - Health checks with start_period for all Docker services
    - TEXT PRIMARY KEY for match_id (partition-ready)
    - BIGSERIAL for internal id columns
    - CASCADE drops in DOWN migrations for safe teardown
key-files:
  created:
    - docker-compose.yml
    - sql/migrations/000001_create_matches.up.sql
    - sql/migrations/000001_create_matches.down.sql
    - sql/migrations/000002_create_rounds.up.sql
    - sql/migrations/000002_create_rounds.down.sql
    - sql/migrations/000003_create_kill_events.up.sql
    - sql/migrations/000003_create_kill_events.down.sql
    - sql/migrations/000004_create_damage_events.up.sql
    - sql/migrations/000004_create_damage_events.down.sql
    - sql/migrations/000005_create_players.up.sql
    - sql/migrations/000005_create_players.down.sql
    - sql/migrations/000006_create_match_players.up.sql
    - sql/migrations/000006_create_match_players.down.sql
  modified: []
decisions:
  - "Docker Compose v2 schema (no version field) — per Docker best practice"
  - "Health checks use start_period to prevent race conditions during container startup"
  - "All six tables use TEXT PRIMARY KEY for match_id — partition-ready design per D-04"
  - "Three composite indexes on (match_id, round_number) for kill_events, damage_events, and rounds"
  - "DOWN migrations use CASCADE to handle FK dependency chains on teardown"
metrics:
  duration: "~5 minutes"
  completed_date: "2026-05-03"
---

# Phase 5 Plan 2: Infrastructure Foundation Summary

**One-liner:** Docker Compose orchestrates NATS 2.12, PostgreSQL 17, MinIO, and Redis 8 with health checks; 12 golang-migrate SQL files define the full v1.1 six-table database schema.

## Tasks Completed

### Task 1: Create docker-compose.yml with NATS, PostgreSQL, MinIO, and Redis

**Status:** Complete
**Commit:** `c139980`
**Files:** docker-compose.yml

Created a Docker Compose v2 file with four infrastructure services:

- **nats** (nats:2.12-alpine): JetStream enabled via `-js` flag, monitoring on port 8222, health check via `nats server ping`
- **postgres** (postgres:17-alpine): Database `dem` with user `dem`, health check via `pg_isready -U dem -d dem`
- **minio** (minio/minio:latest): S3 API on 9000, console UI on 9001, health check via `/minio/health/live`
- **redis** (redis:8-alpine): AOF persistence enabled, health check via `redis-cli ping`

Three named volumes: `pgdata`, `miniodata`, `redisdata`. All services have `start_period` grace periods to avoid startup-order race conditions. Go service containers are intentionally excluded — services run natively on host during development per research architecture.

**Verification:** `docker compose config --quiet` exited 0 (valid YAML + Compose schema).

### Task 2: Create all six database migration files (UP + DOWN)

**Status:** Complete
**Commit:** `d108b29`
**Files:** 12 SQL files in `sql/migrations/`

Created all migration files following golang-migrate naming convention (`{6-digit-seq}_{name}.{direction}.sql`):

| Migration | Table | Key Features |
|-----------|-------|--------------|
| 000001 | `matches` | TEXT PRIMARY KEY, all metadata columns |
| 000002 | `rounds` | FK to matches, UNIQUE(match_id, round_number), idx_rounds_match index |
| 000003 | `kill_events` | FK to matches, idx_kill_events_match_round index |
| 000004 | `damage_events` | FK to matches, idx_damage_events_match_round index |
| 000005 | `players` | Standalone, UNIQUE(name) constraint |
| 000006 | `match_players` | FK to matches + players, composite PRIMARY KEY |

All DOWN files use `DROP TABLE IF EXISTS ... CASCADE` for safe teardown of FK dependency chains. Migration ordering ensures parent tables (matches, players) exist before child tables that reference them.

**Verification:** All 12 files confirmed with correct naming, CREATE TABLE in UP files, DROP TABLE CASCADE in DOWN files, 3 CREATE INDEX statements across rounds, kill_events, and damage_events.

## Deviations from Plan

None — plan executed exactly as written. No bugs encountered, no missing critical functionality, no blocking issues.

## Threat Flags

No new threat surface beyond what the plan's `<threat_model>` documented. The docker-compose.yml exposes the exact ports described in the threat register (T-05-01 through T-05-05). All service credentials are development-only defaults as accepted in the threat model.

## Self-Check

- [x] `docker-compose.yml` exists and is valid Compose v2 YAML
- [x] 12 migration files exist with correct golang-migrate naming
- [x] Commit `c139980` (Task 1) present in git log
- [x] Commit `d108b29` (Task 2) present in git log
- [x] No stub patterns found
- [x] No STATE.md or ROADMAP.md modifications (parallel executor constraint)

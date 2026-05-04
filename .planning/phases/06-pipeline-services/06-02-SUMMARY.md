---
phase: 06-pipeline-services
plan: 02
subsystem: poller
tags: [poller, cron, nats, dedup, tier1, hltv]
requires:
  - "06-01 (Service interface, Runner, processed_matches migration)"
provides:
  - "PollerService — cron-driven Tier 1 match discovery with NATS publishing and Postgres dedup"
affects:
  - "NATS dem.download.jobs"
  - "Postgres processed_matches table"
  - "cmd/poller entrypoint"
tech-stack:
  added:
    - "robfig/cron v3.0.1"
  patterns:
    - "Functional options (CROS-02)"
    - "service.Service interface"
    - "slog structured logging (CROS-01)"
    - "Parameterized SQL queries ($1 placeholders)"
key-files:
  created:
    - internal/poller/config.go
    - internal/poller/poller.go
  modified:
    - cmd/poller/main.go
    - go.mod
    - go.sum
decisions:
  - "Used cron.New() without WithSeconds() to match the 5-field default cron expression '0 2 * * *' — plan specified WithSeconds() but the default expression is standard 5-field. Fixed as Rule 1 spec inconsistency."
  - "Replicated tier1Keywords and filterTier1 logic from internal/provider/events.go as in-package code per CROS-03 (no import of v1.0 provider/)."
metrics:
  duration: "~20 min"
  completed_date: "2026-05-04"
  tasks: 2
  files: 5
---

# Phase 6 Plan 2: Poller Service Summary

**One-liner:** Cron-driven HLTV Tier 1 match discovery service with NATS JetStream download job publishing and Postgres-based deduplication, reusing v1.0 parsers without modification.

## Objective Achieved

Implemented the Tournament Poller Service — the entry point of the v1.1 data pipeline. The service discovers Tier 1 HLTV matches with available demos, deduplicates against the `processed_matches` table, and publishes JSON download jobs to NATS `dem.download.jobs`. Runs on a configurable cron schedule (default daily 02:00 UTC) with a 6-hour minimum interval guard.

## Tasks Completed

| # | Task | Type | Commit | Status |
|---|------|------|--------|--------|
| 1 | Create internal/poller/ — config, cron scheduling, HLTV discovery, NATS publishing, dedup | auto | `67285a5` | Done |
| 2 | Wire cmd/poller/main.go with full dependency injection | auto | `2b93ad1` | Done |

## Key Deliverables

**internal/poller/config.go** (66 lines): `Config` struct with Viper environment variable defaults. Reads `DEM_POLLER_CRON`, `DEM_POLLER_MIN_INTERVAL`, `DEM_NATS_URL`, `DEM_DATABASE_URL`, `DEM_HLTV_BASE_URL`, `DEM_LOG_LEVEL`. Exports `LoadConfig()`.

**internal/poller/poller.go** (366 lines): `PollerService` implementing `service.Service`. Key behavior:
- Cron scheduling via `robfig/cron/v3` with configurable expression and UTC timezone
- Minimum interval guard (default 6h) with `sync.Mutex`-protected `lastRun`
- Tier 1 filter: `PrizePool > 250_000` OR name case-insensitive matches keywords (IEM, PGL, Blast, StarLadder, FISSURE, Esports World Cup, Major, BetBoom)
- Demo availability check using `parser.ParseDemoLink` with `ErrorCodeUnavailableData` detection
- Dedup via `INSERT INTO processed_matches (match_id) VALUES ($1) ON CONFLICT DO NOTHING`
- JSON download job published to `natsutil.SubjectDownload` (`dem.download.jobs`)
- Structured slog logging with `match_id` and `event` fields (CROS-01)
- Compile-time interface check: `var _ service.Service = (*PollerService)(nil)`
- Functional options: `WithNATS`, `WithPostgres`, `WithHLTVClient`, `WithLogger` (CROS-02)

**cmd/poller/main.go** (75 lines): Dependency wiring entrypoint. Creates logger, NATS connection, Postgres pool, and HLTV client; injects them into PollerService via functional options; registers with Runner; calls `runner.Run()` for signal-aware lifecycle.

## Verification Results

- `go vet ./cmd/poller/... ./internal/poller/...` — PASSED
- `go build -o /dev/null ./cmd/poller` — PASSED
- Compile-time `service.Service` interface assertion — PASSED
- No v1.0 imports (internal/provider, internal/cli, internal/output) — PASSED (CROS-03)
- All SQL queries use parameterized `$1` placeholders — CONFIRMED (T-06-07 mitigation)
- Minimum interval guard with mutex protection — CONFIRMED (T-06-05 mitigation)
- JSON payload contains only public HLTV data — CONFIRMED (T-06-06 accepted)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Spec inconsistency] Used `cron.New()` instead of `cron.New(cron.WithSeconds())`**
- **Found during:** Task 1 implementation
- **Issue:** Plan specified `cron.WithSeconds()` which expects 6-field cron expressions, but the default `CronExpression` is `"0 2 * * *"` (standard 5-field). Using `WithSeconds()` would cause parse errors at runtime.
- **Fix:** Used `cron.New(cron.WithLocation(time.UTC))` (standard 5-field parser) to match the default expression. Users can still use 6-field expressions by setting `DEM_POLLER_CRON` to a 6-field value with a custom `cron.WithSeconds()` parser option in the future.
- **Files modified:** `internal/poller/poller.go`
- **Commit:** `67285a5`

## Self-Check: PASSED

- `internal/poller/config.go` — EXISTS
- `internal/poller/poller.go` — EXISTS
- `cmd/poller/main.go` — MODIFIED (75 lines)
- `go.mod` — UPDATED (robfig/cron v3.0.1 added)
- `go.sum` — UPDATED
- Commit `67285a5` — EXISTS
- Commit `2b93ad1` — EXISTS

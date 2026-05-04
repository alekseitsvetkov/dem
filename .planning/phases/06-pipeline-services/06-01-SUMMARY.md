---
phase: 06-pipeline-services
plan: 01
subsystem: infra
tags: [go, service-lifecycle, graceful-shutdown, signal-handling, database-migration, processed-matches]

# Dependency graph
requires:
  - phase: 05-01
    provides: go.mod with v1.1 dependencies, cmd/ entrypoint skeletons, domain types
  - phase: 05-03
    provides: pkg/natsutil, pkg/minio, pkg/postgres with functional options patterns
provides:
  - Service interface (Run(ctx) error) contract for all pipeline services
  - Runner struct with signal-aware graceful shutdown (SIGTERM/SIGINT, reverse-order)
  - Functional options (WithLogger, WithShutdownTimeout) for Runner configuration
  - processed_matches migration (000007) with match_id BIGINT PRIMARY KEY
affects:
  - 06-02 (poller) — implements Service interface, queries processed_matches for dedup
  - 06-03 (downloader) — implements Service interface, managed by Runner
  - 06-04 (parser) — implements Service interface, managed by Runner

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Service interface as universal contract for pipeline services (Run(ctx) error)"
    - "Runner orchestrating goroutine-per-service with error channel and WaitGroup"
    - "Signal-aware context via signal.NotifyContext for SIGTERM/SIGINT"
    - "Reverse-order service shutdown with bounded timeout (default 30s)"
    - "Functional options on Runner (matching hltv.ClientOption and pkg/* patterns)"

key-files:
  created:
    - internal/service/runner.go (138 lines, 6 exported symbols)
    - sql/migrations/000007_create_processed_matches.up.sql
    - sql/migrations/000007_create_processed_matches.down.sql
  modified: []

key-decisions:
  - "Service.Run called twice — first with signal-aware context (main loop), second with timeout context (orchestrated shutdown) — per plan spec"
  - "Down migration uses CASCADE to match all 6 existing migration conventions for consistency"
  - "RunnerOption func(*Runner) pattern matches hltv.ClientOption func(*Client) — direct struct modification, no separate Config struct"
  - "No NATS, MinIO, or Postgres imports in service package — pure stdlib + slog, per CROS-03"

patterns-established:
  - "Service interface: a single Run(ctx context.Context) error method is the universal contract"
  - "Runner lifecycle: AddService(appends) → Run(signal-aware, goroutines, error channel) → reverse-order shutdown"
  - "Migration naming: 6-digit zero-padded seq, snake_case name, .up.sql/.down.sql suffixes"

requirements-completed: [POLL-03, CROS-01, CROS-02, CROS-03]

# Metrics
duration: 5min
completed: 2026-05-04
---

# Phase 6 Plan 1: Service Lifecycle Infrastructure Summary

**Service interface and Runner with SIGTERM/SIGINT graceful shutdown, plus processed_matches migration for poller INSERT-ON-CONFLICT deduplication**

## Performance

- **Duration:** ~5 min
- **Started:** 2026-05-04T18:43:59Z
- **Completed:** 2026-05-04T18:49:25Z
- **Tasks:** 2
- **Files created:** 3

## Accomplishments

- Created `internal/service/runner.go` with `Service` interface (`Run(ctx) error`), `Runner` struct for multi-service lifecycle management, and signal-aware graceful shutdown via `signal.NotifyContext`
- Runner starts services in goroutines (order-added), captures errors via buffered channel, and shuts down in reverse order with configurable timeout (default 30s)
- Functional options `WithLogger` and `WithShutdownTimeout` follow the established v1.0 `func(*Runner)` pattern
- Created migration 000007 adding `processed_matches` table with `match_id BIGINT PRIMARY KEY` and `processed_at TIMESTAMPTZ DEFAULT NOW()` for poller deduplication
- Package is pure stdlib + `log/slog` — no NATS, MinIO, or Postgres imports (CROS-03 compliant)

## Task Commits

Each task was committed atomically:

1. **Task 1: Create internal/service/runner.go** — `e817b1b` (feat)
2. **Task 2: Create processed_matches migration (000007)** — `75b989a` (feat)

## Files Created

- `internal/service/runner.go` — Service interface, Runner struct, RunnerOption, NewRunner, WithLogger, WithShutdownTimeout, AddService, Run (138 lines)
- `sql/migrations/000007_create_processed_matches.up.sql` — CREATE TABLE processed_matches with match_id BIGINT PRIMARY KEY
- `sql/migrations/000007_create_processed_matches.down.sql` — DROP TABLE IF EXISTS processed_matches CASCADE

## Decisions Made

- RunnerOption is `func(*Runner)` (direct struct modification), matching `hltv.ClientOption func(*Client)` pattern from v1.0, rather than the `Option func(*Config)` pattern from pkg/*
- Service.Run is called twice per service: first with the signal-aware context for the main loop, then with a timeout context during orchestrated reverse-order shutdown — per the plan's explicit specification
- Down migration includes CASCADE to match the convention established across all 6 existing migrations (000001-000006), even though the plan's example omitted it

## Deviations from Plan

None — plan executed exactly as written. The CASCADE keyword in the down migration is a consistency improvement that matches existing convention and does not conflict with any plan requirement.

## Known Stubs

None — runner.go is a complete implementation with zero placeholder values or mock data. The migration is pure DDL with no stubs.

## Threat Flags

None — no new network endpoints, auth paths, or trust boundaries introduced. The threat model in the plan already covers all surfaces (T-06-01 through T-06-03).

## Next Phase Readiness

- `internal/service/runner.go` is compilable and importable — ready for `cmd/poller/main.go`, `cmd/downloader/main.go`, `cmd/parser/main.go` to wire Runner and implement the Service interface
- Migration 000007 is ready for `make migrate-up` — poller (Plan 02) can `INSERT INTO processed_matches` with `ON CONFLICT DO NOTHING`
- No blockers — all acceptance criteria pass, go vet and go build succeed

---
*Phase: 06-pipeline-services*
*Completed: 2026-05-04*

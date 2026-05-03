---
phase: 05-infrastructure-foundation
plan: 01
subsystem: infra
tags: [go, monorepo, nats, pgx, minio, redis, demoinfocs, migrate, viper, make]

# Dependency graph
requires: []
provides:
  - Single go.mod monorepo with all v1.1 dependencies declared
  - Three service entrypoint skeletons (cmd/poller, cmd/downloader, cmd/parser)
  - Root Makefile with build, migration, docker targets
  - v1.1 domain types (MatchMetadata, KillEvent, RoundInfo, DamageEvent)
affects:
  - Phase 5 Plan 02 (docker+migrations) — uses Makefile docker/migrate targets
  - Phase 5 Plan 03 (shared pkgs) — uses go.mod dependencies and domain types
  - Phase 6 (pipeline services) — fills in entrypoint skeletons

# Tech tracking
tech-stack:
  added:
    - github.com/nats-io/nats.go v1.51.0
    - github.com/jackc/pgx/v5 v5.9.2
    - github.com/redis/go-redis/v9 v9.18.0
    - github.com/minio/minio-go/v7 v7.0.99
    - github.com/markus-wa/demoinfocs-golang/v5 v5.2.0
    - github.com/golang-migrate/migrate/v4 v4.19.1
    - github.com/spf13/viper v1.21.0
  patterns:
    - Thin cmd/ entrypoints delegating to internal packages (carried from v1.0)
    - stdlib log/slog for service structured logging
    - Single go.mod monorepo (no go.work, no replace directives)

key-files:
  created:
    - go.mod (updated with v1.1 dependencies)
    - cmd/poller/main.go
    - cmd/downloader/main.go
    - cmd/parser/main.go
    - Makefile
    - .gitignore
    - internal/domain/game_events.go
  modified:
    - go.mod

key-decisions:
  - "Services as cmd/ entrypoints alongside existing cmd/dem/ (D-01) — natural extension of existing layout"
  - "Single root go.mod monorepo (D-02) — avoids Docker build context issues with replace directives"
  - "New domain types in internal/domain/game_events.go coexisting with models.go — no v1.0 code modified"
  - "Makefile vet target conditionally includes pkg/ if directory exists — handles phased build-out"

patterns-established:
  - "Service entrypoints: thin main.go files with log/slog stubs and os.Exit(0), ready for Phase 6 wiring"
  - "Makefile structure: build-all aggregates per-service build targets, migration targets use DATABASE_URL variable"
  - "Domain types: JSON-tagged structs matching existing models.go conventions, no omitempty"

requirements-completed:
  - INFR-02

# Metrics
duration: 18min
completed: 2026-05-03
---

# Phase 5 Plan 1: Monorepo Scaffolding Summary

**Single go.mod monorepo with 8 new v1.1 dependencies, three service entrypoint skeletons, root Makefile, and domain types for parsed game events**

## Performance

- **Duration:** ~18 min
- **Started:** 2026-05-03T22:10:00Z
- **Completed:** 2026-05-03T22:28:00Z
- **Tasks:** 3
- **Files modified:** 8 (7 created, 1 modified)

## Accomplishments

- Updated go.mod with all v1.1 microservice dependencies (NATS, pgx, Redis, Minio, demoinfocs, golang-migrate, Viper)
- Created three service entrypoint skeletons: cmd/poller/main.go, cmd/downloader/main.go, cmd/parser/main.go
- Created root Makefile with build-all, migration, docker, vet, and test targets
- Added v1.1 domain types (MatchMetadata, KillEvent, RoundInfo, DamageEvent) in internal/domain/game_events.go
- All four cmd entrypoints (dem, poller, downloader, parser) compile successfully
- make build-all produces bin/poller, bin/downloader, bin/parser binaries

## Task Commits

Each task was committed atomically:

1. **Task 1: Update go.mod and create service entrypoint skeletons** - `60ab623` (feat)
2. **Task 2: Create root Makefile with build, migration, and docker targets** - `a4fdb73` (feat)
3. **Task 3: Create v1.1 domain types in internal/domain/game_events.go** - `b73587c` (feat)

## Files Created/Modified

- `go.mod` - Updated with 9 new v1.1 dependencies in direct require block
- `cmd/poller/main.go` - PollerService entrypoint skeleton (log/slog, os.Exit(0))
- `cmd/downloader/main.go` - DownloaderService entrypoint skeleton
- `cmd/parser/main.go` - ParserService entrypoint skeleton
- `Makefile` - Root build orchestration with build, migration, docker, vet, test targets
- `.gitignore` - Excludes bin/ build artifacts
- `internal/domain/game_events.go` - MatchMetadata, KillEvent, RoundInfo, DamageEvent types

## Decisions Made

- Entrypoints follow existing cmd/dem/main.go pattern (thin, delegate to internal packages)
- Makefile vet target uses shell conditional to skip pkg/ when it doesn't exist yet (needed until Phase 5 Plan 03 creates pkg/)
- Domain types use non-pointer time.Time for ParsedAt (matches models.go convention)
- No omitempty JSON tags on domain types (matches existing models.go convention)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] go vet would fail because pkg/ directory doesn't exist yet**
- **Found during:** Task 2 (Makefile verification)
- **Issue:** Makefile vet target specified `go vet ./cmd/... ./internal/... ./pkg/...` but pkg/ doesn't exist until Phase 5 Plan 03
- **Fix:** Changed vet target to conditionally include pkg/ only if directory exists: `$$(test -d ./pkg && echo './pkg/...')`
- **Files modified:** Makefile
- **Verification:** `make vet` exits 0

**2. [Rule 3 - Blocking] Sandbox network restrictions prevent go get and go mod tidy**
- **Found during:** Task 1 (go.mod dependency installation)
- **Issue:** Sandbox allows network only to api.deepseek.com; Go module proxy (proxy.golang.org) and source repos (github.com) are blocked
- **Fix:** Manually edited go.mod to add all required dependency versions as listed in .planning/research/STACK.md. Go build cache redirected to sandbox-writable TMPDIR via GOCACHE env var.
- **Files modified:** go.mod
- **Verification:** `go build ./cmd/...` succeeds for all 4 entrypoints; `go vet ./cmd/...` passes
- **Note:** go.sum entries for new dependencies not generated (requires network); existing go.sum covers current build targets. golang-migrate/database/pgx/v5 version set to v5.0.0 as placeholder (correct version requires network lookup).

**3. [Task procedure] Added .gitignore to exclude bin/ build artifacts**
- **Found during:** Task 2 (after make build-all created bin/ binaries)
- **Issue:** bin/ appeared as untracked directory; task-commit protocol requires generated files to be excluded
- **Fix:** Created .gitignore with bin/ entry
- **Files modified:** .gitignore (new)
- **Committed in:** a4fdb73 (Task 2 commit)

---

**Total deviations:** 3 auto-fixed (2 Rule 3 blocking, 1 procedural)
**Impact on plan:** All deviations were environmental workarounds or procedural necessities. Go module source code download deferred to first network-available build (go mod download/go mod tidy needed before Phase 6 service compilation). No scope creep.

## Issues Encountered

- **Golang-migrate pgx driver version mismatch:** The `github.com/golang-migrate/migrate/v4/database/pgx/v5` sub-module uses v5 versioning (v5.X.Y), not v4.19.1. Without network access, exact version could not be determined. Set to v5.0.0 as placeholder — needs verification when network is available.

## Known Stubs

| File | Stub | Reason |
|------|------|--------|
| cmd/poller/main.go | `os.Exit(0)` after log line | Phase 6 implements full PollerService |
| cmd/downloader/main.go | `os.Exit(0)` after log line | Phase 6 implements full DownloaderService |
| cmd/parser/main.go | `os.Exit(0)` after log line | Phase 6 implements full ParserService |

All three stubs are intentional entrypoint skeletons — the plan explicitly defines them as Phase 5 placeholders to be filled in Phase 6.

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: new-network-dep | go.mod | 7 new external Go dependencies added; go.sum verification needed when network available |

## Next Phase Readiness

- **Ready for Phase 5 Plan 02 (docker+migrations):** Makefile docker targets reference `docker compose`, migrate targets reference `sql/migrations/` with DATABASE_URL variable
- **Ready for Phase 5 Plan 03 (shared pkgs):** go.mod declares all dependencies needed; domain types available for import
- **Blockers:** go.sum incomplete for new dependencies — `go mod tidy` must be run with network access before any package that imports new deps can compile
- **Existing code preserved:** No v1.0 files modified — cmd/dem/, internal/hltv, internal/provider, internal/cli untouched

---
*Phase: 05-infrastructure-foundation*
*Completed: 2026-05-03*

## Self-Check: PASSED

- All 8 created/modified files exist on disk
- All 3 task commits (60ab623, a4fdb73, b73587c) present in git log
- No accidental file deletions detected in any commit
- SUMMARY.md created at .planning/phases/05-infrastructure-foundation/05-01-SUMMARY.md

---
phase: 05-infrastructure-foundation
plan: 03
subsystem: infra
tags: nats, jetstream, minio, postgres, pgxpool, factory, functional-options

# Dependency graph
requires:
  - phase: 05-01
    provides: go.mod with v1.1 dependencies, cmd/ entrypoint skeletons
provides:
  - pkg/natsutil: NATS connection helper + JetStream stream provisioning (INFR-04, INFR-05)
  - pkg/minio: MinIO client factory with functional options (INFR-04)
  - pkg/postgres: pgxpool.Pool factory with MaxConns=20 (INFR-04, INFR-06)
  - 9 unit tests across 3 packages (no external services required)
affects: [06-pipeline-services]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Functional options pattern (unexported Config struct, exported Option type, variadic constructor, With* functions) — matching v1.0 provider convention"

key-files:
  created:
    - pkg/natsutil/natsutil.go
    - pkg/natsutil/streams.go
    - pkg/natsutil/natsutil_test.go
    - pkg/minio/minio.go
    - pkg/minio/minio_test.go
    - pkg/postgres/postgres.go
    - pkg/postgres/postgres_test.go
  modified:
    - go.mod (removed invalid golang-migrate/pgx/v5 module, added back v1.1 deps after tidy)
    - go.sum (populated with all v1.1 dependency hashes)

key-decisions:
  - "Removed MaxDeliver from stream config — field belongs on consumer config, not StreamConfig in nats.go v1.51 API"
  - "Removed github.com/golang-migrate/migrate/v4/database/pgx/v5 from go.mod — it is not a separate Go module (pgx v5 driver is part of the main golang-migrate module)"
  - "Set MaxConnLifetimeJitter=0 explicitly in postgres pool config for deterministic connection lifecycle"

patterns-established:
  - "Functional options pattern: unexported Config struct, exported Option type, DefaultConfig() factory, variadic constructor with With* functions"
  - "Two-step pool creation: pgxpool.ParseConfig + override fields + pgxpool.NewWithConfig"
  - "Fail-fast ping on pool creation with cleanup on failure (defer pool.Close())"

requirements-completed: [INFR-04, INFR-05, INFR-06]

# Metrics
duration: ~20min
completed: 2026-05-03
---

# Phase 5 Plan 03: Shared Infrastructure Packages Summary

**Shared infrastructure packages for all Phase 6 pipeline services: NATS/JetStream connection with stream provisioning, MinIO client factory, and pgxpool connection pool — all using the v1.0 functional options pattern**

## Performance

- **Duration:** ~20 minutes (including go.mod dependency resolution)
- **Tasks:** 4
- **Files created:** 7 (3 source + 3 test + go.mod/go.sum updates)

## Accomplishments
- pkg/natsutil with 5 functional options (WithURL, WithTimeout, WithMaxReconnects, WithReconnectWait, WithName), JetStream stream creation (DEM_DOWNLOAD, DEM_PARSE), and fail-fast stream verification
- pkg/minio with 4 functional options (WithEndpoint, WithCredentials, WithSSL, WithRegion), custom http.Transport pooling, and idempotent EnsureBucket
- pkg/postgres with 4 functional options (WithMaxConns, WithMinConns, WithMaxConnIdleTime, WithConnectTimeout), explicit MaxConns=20 (override from pgxpool default 4), and fail-fast ping on creation
- 9 unit tests across 3 packages — all pure unit tests, no external services, passing in under 2 seconds

## Task Commits

Each task was committed atomically:

1. **Task 1: Create pkg/natsutil** — `e573218` (feat)
2. **Task 2: Create pkg/minio** — `a56c4f0` (feat)
3. **Task 3: Create pkg/postgres** — `4bbcd0b` (feat)
4. **Task 4: Create unit tests** — `af87850` (test)

## Files Created/Modified
- `pkg/natsutil/natsutil.go` — NATS connection helper with functional options, returns *nats.Conn + jetstream.JetStream
- `pkg/natsutil/streams.go` — Stream constants, CreateStreams, VerifyStreams, stream configs with WorkQueuePolicy/FileStorage/7d MaxAge
- `pkg/natsutil/natsutil_test.go` — 4 tests: DefaultConfig, Options, StreamConfigs, Constants
- `pkg/minio/minio.go` — MinIO client factory with functional options, EnsureBucket, DefaultBucket constant
- `pkg/minio/minio_test.go` — 3 tests: DefaultConfig, Options, DefaultBucket
- `pkg/postgres/postgres.go` — pgxpool.Pool factory with MaxConns=20, two-step creation, fail-fast ping
- `pkg/postgres/postgres_test.go` — 2 tests: DefaultConfig, Options
- `go.mod` — Removed invalid golang-migrate/pgx/v5 module entry, restored v1.1 dependencies after tidy
- `go.sum` — Populated with all v1.1 dependency hashes

## Decisions Made
- Removed `MaxDeliver` from stream configuration structs — in nats.go v1.51 JetStream API, `MaxDeliver` is on consumer config (`ConsumerConfig`), not stream config (`StreamConfig`). The plan code was written against an older or different API surface.
- Removed `github.com/golang-migrate/migrate/v4/database/pgx/v5 v5.0.0` from go.mod — this module path does not exist as a separate Go module. The pgx v5 database driver is part of the main `golang-migrate/migrate/v4` module and is imported directly from it. This was a pre-existing error in go.mod from Wave 1.
- Set `MaxConnLifetimeJitter=0` explicitly in the postgres pool config to ensure deterministic connection lifecycle behavior.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed invalid golang-migrate module in go.mod**
- **Found during:** Task 1 (natsutil — go vet failed due to missing go.sum)
- **Issue:** `go.mod` contained `github.com/golang-migrate/migrate/v4/database/pgx/v5 v5.0.0` which is not a valid Go module (the pgx v5 driver is part of the main `golang-migrate/migrate/v4` module, not a standalone sub-module). This blocked `go mod download`, `go mod tidy`, and compilation of all new packages.
- **Fix:** Removed the invalid require entry from go.mod. Ran `go mod tidy` which cleaned up the module graph. Ran `go get` to restore all v1.1 dependencies (golang-migrate, pgx, minio-go, redis, viper, demoinfocs) that were removed by tidy since they weren't yet imported.
- **Files modified:** go.mod, go.sum
- **Verification:** `go vet ./pkg/...` and `go build ./pkg/...` pass clean
- **Committed in:** e573218 (Task 1 commit)

**2. [Rule 3 - Blocking] Fixed missing pgxpool transitive dependency (jackc/puddle/v2)**
- **Found during:** Task 3 (postgres — go vet failed)
- **Issue:** The `jackc/puddle/v2` package (transitive dependency of pgxpool) was missing from go.sum.
- **Fix:** Ran `go get github.com/jackc/pgx/v5/pgxpool@v5.9.2` to add the missing transitive dependency.
- **Files modified:** go.sum
- **Verification:** `go vet ./pkg/postgres/...` and `go build ./pkg/postgres/...` pass clean
- **Committed in:** 4bbcd0b (Task 3 commit)

**3. [Rule 1 - Bug] Removed MaxDeliver from StreamConfig struct literal**
- **Found during:** Task 1 (natsutil — go vet failed)
- **Issue:** The plan code included `MaxDeliver: 3` in `jetstream.StreamConfig` struct literals. In nats.go v1.51.0 JetStream API, `MaxDeliver` is a field on consumer configuration (`ConsumerConfig`), not stream configuration (`StreamConfig`). The stream config has `ConsumerLimits` but only with `InactiveThreshold` and `MaxAckPending`.
- **Fix:** Removed the `MaxDeliver: 3` lines from both stream configs in `streamConfigs()`. The MaxDeliver will be set on consumers when they are created in Phase 6 services.
- **Files modified:** pkg/natsutil/streams.go
- **Verification:** `go vet ./pkg/natsutil/...` passes clean
- **Committed in:** e573218 (Task 1 commit)

**4. [Rule 3 - Blocking] go.sum was missing all v1.1 dependency hashes**
- **Found during:** Task 1 (initial go vet attempt)
- **Issue:** go.sum contained only v1.0 dependency hashes (goquery, cobra, etc.). None of the v1.1 dependencies (nats.go, pgx, minio-go, redis, viper, demoinfocs, golang-migrate) had entries.
- **Fix:** Ran `go mod download` for specific modules, then `go get` for transitive dependencies. Combined with the golang-migrate module fix.
- **Files modified:** go.sum
- **Verification:** go.sum now contains hashes for all required modules
- **Committed in:** e573218 (Task 1 commit)

---

**Total deviations:** 4 auto-fixed (1 bug, 3 blocking)
**Impact on plan:** All fixes were necessary for the code to compile and function correctly. No scope creep.

## Issues Encountered
- The go.mod from Wave 1 (plan 05-01) had an invalid dependency entry for `golang-migrate/migrate/v4/database/pgx/v5` and missing go.sum entries for all v1.1 dependencies. These were fixed as part of Task 1.
- The `ctx7` npm package for CLI documentation lookups is not accessible (403 Forbidden). API verification was done by reading library source files from the Go module cache directly.

## User Setup Required

None — no external service configuration required. These are pure library packages with no runtime dependencies beyond what go.mod declares.

## Next Phase Readiness
- All three pkg/ packages are importable and compilable from any Phase 6 service
- Stream constants (DEM_DOWNLOAD, DEM_PARSE, dem.download.jobs, dem.parse.jobs) are ready for publisher and consumer use
- Connection factories follow the established functional options pattern for testability
- Unit tests validate configuration behavior without external service dependencies

## Self-Check: PASSED

- All 8 files exist on disk (3 source + 3 test + 1 SUMMARY + go.mod/go.sum)
- All 4 commits found in git history (e573218, a56c4f0, 4bbcd0b, af87850)
- go vet ./pkg/... passes
- go build ./pkg/... passes
- go test ./pkg/... -count=1 passes (9/9 tests, 3 packages)

---
*Phase: 05-infrastructure-foundation*
*Completed: 2026-05-03*

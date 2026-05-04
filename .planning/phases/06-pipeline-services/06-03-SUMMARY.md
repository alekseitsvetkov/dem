---
phase: 06-pipeline-services
plan: 03
subsystem: pipeline
tags: [downloader, nats, jetstream, minio, streaming, retry, viper]

# Dependency graph
requires:
  - phase: 06-01
    provides: Service interface + Runner (internal/service/runner.go)
  - phase: 05-03
    provides: pkg/natsutil (stream/subject constants, NATS connection), pkg/minio (MinIO client factory, EnsureBucket)
  - phase: 05-01
    provides: go.mod with v1.1 dependencies, cmd/downloader entrypoint skeleton
  - phase: 05-02
    provides: docker-compose.yml (NATS, MinIO, Postgres)
provides:
  - internal/downloader/config.go: Downloader Config struct with Viper env defaults (13 settings)
  - internal/downloader/downloader.go: DownloaderService implementing service.Service (NATS pull consumer, streaming download, MinIO upload, retry, parse job publish)
  - cmd/downloader/main.go: Entrypoint wiring NATS + MinIO via functional options, Runner lifecycle
affects:
  - 06-04 (Parser Service) — consumes parse jobs from dem.parse.jobs with {bucket, object_key, match_id, ...} payload

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Conditional defer for NATS message acknowledgment (D-10): var msgErr error; defer func() { if msgErr == nil { msg.Ack() } else { msg.NakWithDelay() } }()"
    - "Streaming download via io.Reader pipe (D-04): http.Response.Body -> io.LimitReader -> minio.PutObject"
    - "Exponential backoff with jitter (D-03): 5s -> 25s -> 125s with +/-20% random jitter"
    - "raw *http.Client over hltv.Client for CDN downloads (hltv.Client.Fetch buffers full body)"
    - "Aliased import dmnio for pkg/minio to avoid conflict with minio-go/v7"
    - "NATS publishing via PublishMsg(ctx, &nats.Msg{...}) — matches poller pattern"

key-files:
  created:
    - internal/downloader/config.go
    - internal/downloader/downloader.go
  modified:
    - cmd/downloader/main.go

key-decisions:
  - "Raw *http.Client instead of hltv.Client — hltv.Client.Fetch reads full response body into []byte, incompatible with streaming to MinIO PutObject via io.Reader"
  - "Aliased pkg/minio import as dmnio — avoids naming conflict with minio-go/v7 import (both would be minio)"
  - "NATS PublishMsg uses &nats.Msg{} (not jetstream.PublishMsg) — follows established poller pattern and nats.go v1.51 API"

patterns-established:
  - "Downloader entrypoint pattern matches poller: load config -> wire NATS -> wire storage -> create service with functional options -> create Runner -> runner.Run()"
  - "D-10 conditional defer pattern: single msgErr variable, defer checks it to decide Ack vs NakWithDelay"

requirements-completed: [DWLD-01, DWLD-02, DWLD-03, CROS-01, CROS-02, CROS-03]

# Metrics
duration: ~15min
completed: 2026-05-04
---

# Phase 6 Plan 03: Downloader Service Summary

**Demo downloader service consuming NATS JetStream download jobs, streaming .dem.gz from HLTV CDN directly to MinIO (zero local disk writes), with internal retry and conditional message acknowledgment per D-10**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-05-04T19:22:00Z
- **Completed:** 2026-05-04T19:37:00Z
- **Tasks:** 2
- **Files modified:** 3 (2 created, 1 modified)

## Accomplishments

- **DownloaderService** implementing `service.Service` with full download pipeline
- **Config** with 13 Viper environment variables (DEM_ prefix), duration parsing with fallbacks
- **NATS pull consumer** "download-worker" on DEM_DOWNLOAD stream filtering `dem.download.jobs` with `MaxDeliver: 3`
- **Streaming download** from HLTV CDN to MinIO: `http.Response.Body` -> `io.LimitReader(MaxBytes=500MB)` -> `minio.PutObject(reader, -1)` — zero local disk writes (D-04)
- **Internal retry loop** with exponential backoff: 5s -> 25s -> 125s, with +/-20% jitter (D-03), only escalating to `msg.NakWithDelay()` after 3 failed attempts
- **Parse job publishing** to `dem.parse.jobs` with full metadata payload: bucket, object_key, match_id, match_url, event_name, team1, team2, match_date (D-05)
- **Conditional defer pattern** (D-10): `var msgErr error; defer func() { if msgErr == nil { msg.Ack() } else { msg.NakWithDelay() } }()` — guarantees Ack and Nak are never both called on the same message
- **Safety net**: `io.LimitReader` caps response body at 500 MB (Pitfall 4 / T-06-08); `context.WithTimeout(30min)` prevents hung connections
- **Entrypoint** follows poller pattern: load config, wire NATS + MinIO with functional options, create Runner with Service, call `runner.Run()`

## Task Commits

Each task was committed atomically:

1. **Task 1: Create internal/downloader/ package** — `9996d81` (feat)
2. **Task 2: Wire cmd/downloader/main.go** — `e011d61` (feat)

## Files Created/Modified

- `internal/downloader/config.go` — Config struct with 13 Viper env vars (DEM_ prefix), manual duration parsing with fallbacks
- `internal/downloader/downloader.go` — DownloaderService (200+ lines): NATS pull consumer, streaming download, MinIO upload, retry logic, parse job publishing, conditional defer
- `cmd/downloader/main.go` — Entrypoint (67 lines): NATS + MinIO wiring, DownloaderService creation, Runner lifecycle

## Decisions Made

- **Raw `*http.Client` over `hltv.Client`:** The plan specifies the downloader uses a raw `*http.Client` because `hltv.Client.Fetch` reads the full response body into `[]byte`, which is incompatible with streaming to MinIO. A code comment documents this rationale.
- **Aliased `pkg/minio` import as `dmnio`:** The `pkg/minio` package and `minio-go/v7` import would both use the name `minio`, causing a compilation error. Using `dmnio` for `pkg/minio` resolves the conflict cleanly.
- **NATS `PublishMsg` follows poller pattern:** The poller uses `d.js.PublishMsg(ctx, &nats.Msg{Subject: ..., Data: ...})` — the downloader follows the same pattern for consistency across the codebase.
- **msg.Ack() error checked in defer:** The plan pseudocode called `msg.Ack()` without error checking. Added error logging on Ack/Nak failures per best practice.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing critical functionality] Added error handling for msg.Ack() and msg.NakWithDelay()**
- **Found during:** Task 1 implementation
- **Issue:** The plan's pseudocode called `msg.Ack()` and `msg.NakWithDelay()` without checking returned errors. If Ack or Nak fails, the server may redeliver the message, leading to duplicate downloads.
- **Fix:** Wrapped both calls in error checks within the deferred function; log errors via slog if Ack/Nak fails.
- **Files modified:** internal/downloader/downloader.go
- **Commit:** 9996d81

### Implementation Clarifications

**1. MinIO import aliasing (dmnio):** The plan code assumed `minio.EnsureBucket` and `minio.PutObjectOptions` share a namespace. Since `pkg/minio` and `minio-go/v7` both use `minio`, an alias `dmnio` was introduced for `pkg/minio`. This is standard Go practice, not a plan deviation.

**2. NATS PublishMsg API:** The plan used `jetstream.PublishMsg{}` struct. The nats.go v1.51 API uses `&nats.Msg{}` for PublishMsg (the poller established this pattern). This is an API-level detail, not a behavioral deviation.

---

**Total deviations:** 1 auto-fixed (Rule 2 - error handling)
**Impact on plan:** Error handling added for robustness. No scope creep.

## Known Stubs

None — the downloader is fully implemented with all data-wiring paths complete.

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: streaming-cdn | internal/downloader/downloader.go | Downloader accepts untrusted .dem.gz from HLTV CDN (T-06-08). Mitigated by io.LimitReader (500MB cap) and context.WithTimeout (30min). |

Threats T-06-09 (object key tampering), T-06-10 (credential disclosure), and T-06-11 (repudiation) are handled per the plan's threat model: deterministic object keys from poller-validated match_id, Viper env var credentials, and NATS Ack + MinIO existence as audit trail.

## Requirements Completed

| ID | Description | Status |
|----|-------------|--------|
| DWLD-01 | Downloader consumes jobs from NATS dem.download.jobs with pull consumer | Complete |
| DWLD-02 | Downloader streams .dem.gz from HLTV CDN to MinIO (zero local disk) | Complete |
| DWLD-03 | Downloader retries transient failures (3x, exponential backoff) | Complete |
| CROS-01 | Structured logging via log/slog with match_id and job_id fields | Complete |
| CROS-02 | Functional options for testable dependency injection | Complete |
| CROS-03 | No v1.0 code modified; no internal/provider/ imports | Complete |

## Verification

- `go vet ./internal/downloader/...` — passes
- `go vet ./cmd/downloader/...` — passes
- `go build -o /dev/null ./cmd/downloader` — passes
- No `io.ReadAll`, `os.Create`, or `ioutil.WriteFile` in downloader code (D-04 compliance)
- Conditional defer pattern present: `var msgErr error; defer func() { if msgErr == nil { msg.Ack() } else { msg.NakWithDelay(...) } }()` (D-10 compliance)
- Rationale comment for raw `*http.Client` present at line 32-34 (hltv.Client.Fetch buffers full body)

## Next Phase Readiness

- `dem.download.jobs` messages are consumed and processed — ready for PollerService to publish to
- `dem.parse.jobs` messages are published with full metadata — ready for Parser Service (06-04) to consume
- MinIO bucket `dem-files` is ensured at startup — demo files accessible to Parser Service for streaming read
- All downloader files importable and compilable from any package

---

*Phase: 06-pipeline-services*
*Completed: 2026-05-04*

## Self-Check: PASSED

- VERIFIED: internal/downloader/config.go — exists on disk
- VERIFIED: internal/downloader/downloader.go — exists on disk
- VERIFIED: cmd/downloader/main.go — exists on disk
- VERIFIED: Commit 9996d81 — present in git log (feat: DownloaderService)
- VERIFIED: Commit e011d61 — present in git log (feat: wire entrypoint)
- VERIFIED: No accidental file deletions in either commit
- VERIFIED: SUMMARY.md created at .planning/phases/06-pipeline-services/06-03-SUMMARY.md

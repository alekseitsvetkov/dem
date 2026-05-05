---
phase: 06-pipeline-services
verified: 2026-05-04T23:59:00Z
status: human_needed
score: 6/6 roadmap success criteria verified
overrides_applied: 0
human_verification:
  - test: "Run the full pipeline end-to-end: start docker-compose, run poller, downloader, parser in sequence"
    expected: "Poller discovers Tier 1 matches, publishes to NATS; Downloader streams to MinIO, publishes parse jobs; Parser inserts game events into Postgres. All services emit structured JSON logs with match_id correlation."
    why_human: "Requires running NATS, MinIO, Postgres infrastructure and HLTV.org network access. Cannot be verified from code alone."
  - test: "Run the parser twice on the same demo file and verify no duplicate rows"
    expected: "kill_events and damage_events contain exactly the same rows after both runs — ON CONFLICT (event_id) DO NOTHING prevents duplicates."
    why_human: "Requires a demo file in MinIO and a running Postgres instance to execute queries against."
  - test: "Verify all three entrypoints compile to working binaries"
    expected: "go build -o /dev/null ./cmd/poller && go build -o /dev/null ./cmd/downloader && go build -o /dev/null ./cmd/parser all exit 0."
    why_human: "Sandbox build cache restrictions prevented full compilation check. go vet passed for all packages."
---

# Phase 6: Pipeline Services Verification Report

**Phase Goal:** The full data pipeline runs end-to-end — Tier 1 tournament matches are automatically discovered, demo files downloaded to MinIO, game events parsed into Postgres, and the system is idempotent and observable.
**Verified:** 2026-05-04T23:59:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Roadmap Success Criteria

| # | Success Criterion | Status | Evidence |
|---|-------------------|--------|----------|
| 1 | Poller discovers Tier 1 matches, publishes to `dem.download.jobs`, no duplicate jobs | VERIFIED | `internal/poller/poller.go`: cron scheduling (line 123), Tier 1 filter with PrizePool > 250_000 + keywords (line 349), INSERT ON CONFLICT DO NOTHING dedup (line 284), PublishMsg to SubjectDownload (line 320) |
| 2 | Downloader streams .dem.gz from CDN to MinIO (no disk write), publishes to `dem.parse.jobs` | VERIFIED | `internal/downloader/downloader.go`: streaming via io.LimitReader -> PutObject (line 315-320), no io.ReadAll/os.Create, parse job publish to SubjectParse (line 229-230) |
| 3 | Parser streams demo from MinIO through demoinfocs-golang, inserts into Postgres | VERIFIED | `internal/parser/parser.go`: GetObject -> dem.NewParser(obj) (line 168-177), 12 event handlers (lines 216-394), writer.Flush on RoundEnd (line 255) |
| 4 | Idempotent — re-parsing same demo produces no duplicate rows | VERIFIED | `internal/parser/writer.go`: ON CONFLICT (event_id) DO NOTHING on kill_events (line 156) and damage_events (line 167), ON CONFLICT (match_id, round_number) DO NOTHING on rounds (line 143), ON CONFLICT (match_id) DO UPDATE on matches (line 74); migration 000008 adds event_id TEXT UNIQUE |
| 5 | Structured logging via slog with match_id/job_id, no v1.0 JSON envelopes | VERIFIED | All entrypoints use slog.NewJSONHandler; all services use slog.String("match_id", ...); downloader adds slog.String("job_id", ...); no {data, meta} envelopes found |
| 6 | v1.0 CLI untouched — internal/hltv, internal/provider, internal/cli not modified | VERIFIED | git diff 581bc43..HEAD shows zero changes to v1.0 paths; grep confirms no provider/cli/output/cmd/dem imports in any Phase 6 package |

**Score:** 6/6 roadmap success criteria verified

### Observable Truths from Plan Must-Haves

| # | Truth | Source | Status | Evidence |
|---|-------|--------|--------|----------|
| T1 | Service interface + Runner with signal-aware shutdown | Plan 01 | VERIFIED | `internal/service/runner.go:21-23` (Service interface), line 78 (signal.NotifyContext with SIGTERM/SIGINT), lines 124-127 (reverse-order shutdown) |
| T2 | processed_matches migration with BIGINT PK | Plan 01 | VERIFIED | `sql/migrations/000007_create_processed_matches.up.sql` — match_id BIGINT PRIMARY KEY, processed_at TIMESTAMPTZ DEFAULT NOW() |
| T3 | Poller discovers Tier 1 matches with demos, publishes to NATS with dedup | Plan 02 | VERIFIED | `internal/poller/poller.go` — cron (line 123), Tier 1 filter (line 349), demo check with ParseError (line 266), ON CONFLICT dedup (line 284), PublishMsg (line 320) |
| T4 | Poller runs on configurable cron with 6h min interval guard | Plan 02 | VERIFIED | CronExpression from Viper (config line 13), MinInterval default 6h (config line 17), isTooSoon() guard (line 148) |
| T5 | Poller emits structured slog with match_id per discovered match | Plan 02 | VERIFIED | slog.Info("published download job", slog.String("match_id", ...), slog.String("event", ...)) at line 333 |
| T6 | Downloader consumes from dem.download.jobs with conditional defer Ack/Nak | Plan 03 | VERIFIED | CreateOrUpdateConsumer "download-worker" (line 102), var msgErr conditional defer: nil=Ack, non-nil=NakWithDelay (lines 144-157) |
| T7 | Downloader streams CDN->MinIO, zero disk writes | Plan 03 | VERIFIED | streamDownload: io.LimitReader -> PutObject(reader, -1) (lines 315-324), no io.ReadAll/os.Create anywhere |
| T8 | Downloader retries 3x with exponential backoff before NakWithDelay | Plan 03 | VERIFIED | downloadWithRetry: 5s->25s->125s backoff with jitter (lines 251-285) |
| T9 | Downloader publishes parse jobs to dem.parse.jobs | Plan 03 | VERIFIED | parse job JSON published to SubjectParse (lines 210-243) |
| T10 | Parser consumes from dem.parse.jobs with conditional defer | Plan 04 | VERIFIED | CreateOrUpdateConsumer "parse-worker" (line 93), var msgErr conditional defer (lines 138-152) |
| T11 | Parser streams MinIO->demoinfocs, never buffered | Plan 04 | VERIFIED | GetObject -> dem.NewParser(obj) (lines 168-177), no io.ReadAll on demo data |
| T12 | Parser registers 12 event handlers | Plan 04 | VERIFIED | 12 RegisterEventHandler calls: MatchStart, RoundStart, RoundEnd, Kill, PlayerHurt, WeaponFire, BombPlanted, BombDefused, BombExplode, GrenadeProjectileThrow, PlayerConnect, TeamSideSwitch |
| T13 | Parser batch-inserts per round with ON CONFLICT (event_id) DO NOTHING | Plan 04 | VERIFIED | pgx.Batch + SendBatch (writer.go line 174), event_id UNIQUE constraint (migration 000008), ON CONFLICT (event_id) DO NOTHING on kill_events (line 156) and damage_events (line 167) |
| T14 | Parser calls defer p.Close() on every parser instance | Plan 04 | VERIFIED | `parser.go:179`: defer parser.Close() after every dem.NewParser() |
| T15 | Functional options injection (CROS-02) | Plans 01-04 | VERIFIED | All 3 services use functional options: WithNATS, WithMinio, WithPostgres, WithLogger, WithHLTVClient, WithHTTPClient |
| T16 | No v1.0 code modified (CROS-03) | Plans 01-04 | VERIFIED | git diff shows zero changes; no imports from provider/cli/output/cmd/dem |

**Score:** 16/16 observable truths verified

## Required Artifacts

| Artifact | Expected | Lines | Status | Details |
|----------|----------|-------|--------|---------|
| `internal/service/runner.go` | Service interface, Runner, NewRunner, functional options, SIGTERM/SIGINT | 138 | VERIFIED | 6 exported symbols: Service, Runner, RunnerOption, NewRunner, WithLogger, WithShutdownTimeout. Pure stdlib+slog. |
| `sql/migrations/000007_create_processed_matches.up.sql` | processed_matches table | 4 | VERIFIED | match_id BIGINT PRIMARY KEY, processed_at TIMESTAMPTZ DEFAULT NOW() |
| `sql/migrations/000007_create_processed_matches.down.sql` | Teardown | 1 | VERIFIED | DROP TABLE IF EXISTS processed_matches CASCADE |
| `internal/poller/config.go` | Config with Viper env defaults | 66 | VERIFIED | 6 fields: CronExpression, MinInterval, NATSURL, DatabaseURL, HLTVBaseURL, LogLevel |
| `internal/poller/poller.go` | PollerService implementing service.Service | 366 | VERIFIED | Cron scheduling, Tier 1 filter, HLTV discovery, NATS publishing, dedup, compile-time `var _ service.Service` assertion |
| `cmd/poller/main.go` | Poller entrypoint | 75 | VERIFIED | Full DI wiring: NATS, Postgres, HLTV client -> PollerService -> Runner |
| `internal/downloader/config.go` | Config with 13 Viper env vars | 118 | VERIFIED | 13 fields including MaxRetries, RetryBaseDelay, RetryMaxDelay, NakDelay, DownloadTimeout, MaxBytes |
| `internal/downloader/downloader.go` | DownloaderService implementing service.Service | 330 | VERIFIED | Pull consumer, streaming download, MinIO upload, retry, parse job publish, conditional defer D-10 |
| `cmd/downloader/main.go` | Downloader entrypoint | 76 | VERIFIED | Full DI wiring: NATS, MinIO -> DownloaderService -> Runner |
| `internal/parser/config.go` | Config with Viper env defaults | 66 | VERIFIED | 10 fields including Concurrency (MaxAckPending), ParseTimeout, AckWait |
| `internal/parser/writer.go` | EventWriter with batch insert | 196 | VERIFIED | 7 exported methods, pgx.Batch per-round flush, ON CONFLICT DO NOTHING on all write paths |
| `internal/parser/parser.go` | ParserService with 12 event handlers | 485 | VERIFIED | Pull consumer, MinIO to demoinfocs streaming, 12 event handlers, conditional defer D-10, defer p.Close() |
| `cmd/parser/main.go` | Parser entrypoint | 87 | VERIFIED | Full DI wiring: NATS, MinIO, Postgres -> ParserService -> Runner |
| `sql/migrations/000008_add_event_id_to_kill_damage.up.sql` | event_id UNIQUE on kill_events, damage_events | 5 | VERIFIED | ALTER TABLE ADD COLUMN event_id TEXT NOT NULL, ADD CONSTRAINT UNIQUE |
| `sql/migrations/000008_add_event_id_to_kill_damage.down.sql` | Teardown migration 000008 | 5 | VERIFIED | DROP CONSTRAINT, DROP COLUMN for both tables |

**All 15 expected artifacts exist, are substantive, and are wired.**

## Key Link Verification

| From | To | Via | Status | Evidence |
|------|----|-----|--------|----------|
| runner.go | os/signal.NotifyContext | SIGTERM, SIGINT | WIRED | `runner.go:78`: signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT) |
| runner.go | log/slog.Logger | WithLogger option | WIRED | `runner.go:51`: WithLogger sets slog.Logger on Runner |
| poller.go | processed_matches | INSERT ON CONFLICT DO NOTHING | WIRED | `poller.go:284`: INSERT INTO processed_matches (match_id) VALUES ($1) ON CONFLICT DO NOTHING |
| poller.go | natsutil.SubjectDownload | js.PublishMsg | WIRED | `poller.go:320-322`: PublishMsg with Subject: natsutil.SubjectDownload |
| poller.go | hltv/parser | ParseEvents, ParseResults, ParseDemoLink | WIRED | `poller.go:190,226,263`: parser.ParseEvents, parser.ParseResults, parser.ParseDemoLink |
| downloader.go | StreamDownload/SubjectDownload | CreateOrUpdateConsumer | WIRED | `downloader.go:102-107`: consumer "download-worker" on StreamDownload filtering SubjectDownload |
| downloader.go | MinIO PutObject | resp.Body streaming | WIRED | `downloader.go:315-324`: limitedBody from resp.Body -> PutObject(reader, -1) |
| downloader.go | SubjectParse | PublishMsg parse job | WIRED | `downloader.go:229-231`: PublishMsg with Subject: natsutil.SubjectParse |
| parser.go | MinIO GetObject -> dem.NewParser | Streaming reader | WIRED | `parser.go:168-177`: minio.GetObject -> dem.NewParser(obj) |
| parser.go | RoundEnd -> writer.Flush | Batch insert trigger | WIRED | `parser.go:255`: writer.Flush(context.Background()) in RoundEnd handler |
| parser.go | writer.go INSERT -> migration 000008 | ON CONFLICT (event_id) DO NOTHING | WIRED | `writer.go:152-157`: eventID + INSERT ... ON CONFLICT (event_id) DO NOTHING |
| All services | slog.NewJSONHandler | Structured logging | WIRED | All 3 cmd entrypoints use slog.NewJSONHandler(os.Stdout) |

**All 12 key links are VERIFIED as WIRED.**

## Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|--------------|--------|--------------------|--------|
| PollerService.poll() | events, results, demoLink | hltv.Client.Fetch -> parser.Parse* -> hltv.org HTML | Yes (real HLTV HTTP + goquery parsing) | FLOWING |
| DownloaderService.streamDownload() | resp.Body | http.Get(CDN URL) -> io.LimitReader -> MinIO PutObject | Yes (real HTTP streaming, no buffering) | FLOWING |
| ParserService.processMessage() | obj (io.ReadCloser) | minio.GetObject -> dem.NewParser(obj) | Yes (real MinIO streaming, no buffering) | FLOWING |
| EventWriter.Flush() | kills, damages, round | demoinfocs event handlers -> per-round buffers -> pgx.Batch | Yes (real game events from demo parse) | FLOWING |

**All 4 data-flow traces confirm real data flows — no hardcoded empty returns, no static fallbacks that never get populated.**

## Behavioral Spot-Checks

Step 7b: SKIPPED (no runnable entry points without infrastructure — NATS, MinIO, and Postgres must be running; HLTV.org must be accessible)

## Requirements Coverage

| Requirement | Source Plans | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| POLL-01 | 06-02 | Poller checks HLTV for Tier 1 tournaments on configurable schedule, reuses v1.0 parsers | SATISFIED | poller.go: cron with configurable CronExpression (line 123), min interval guard (line 148), imports hltv/parser (line 21) |
| POLL-02 | 06-02 | Poller discovers matches with demos, publishes to dem.download.jobs | SATISFIED | poller.go: ParseDemoLink check (line 263), ErrorCodeUnavailableData skip (line 266), PublishMsg to SubjectDownload (line 320) |
| POLL-03 | 06-01, 06-02 | Poller deduplicates matches — Postgres-based approach | SATISFIED | poller.go: INSERT INTO processed_matches ON CONFLICT DO NOTHING (line 284), RowsAffected()==0 skip (line 294); migration 000007 provides the table |
| DWLD-01 | 06-03 | Downloader consumes jobs from dem.download.jobs with pull consumer, explicit Ack | SATISFIED | downloader.go: "download-worker" consumer (line 102-107), conditional defer Ack/Nak (lines 144-157) |
| DWLD-02 | 06-03 | Downloader streams .dem.gz from CDN to MinIO, no disk writes | SATISFIED | downloader.go: io.LimitReader->PutObject (lines 315-324), no io.ReadAll/os.Create anywhere |
| DWLD-03 | 06-03 | Downloader publishes parse job on success, retries on failure | SATISFIED | downloader.go: parse job to SubjectParse on success (line 229-231), 3x retry with backoff (line 253), NakWithDelay on exhaustion (line 153) |
| PARS-01 | 06-04 | Parser consumes jobs from dem.parse.jobs with pull consumer, explicit Ack | SATISFIED | parser.go: "parse-worker" consumer (line 93-100), conditional defer Ack/Nak (lines 138-152) |
| PARS-02 | 06-04 | Parser streams demo from MinIO to demoinfocs, never buffered | SATISFIED | parser.go: GetObject -> dem.NewParser(obj) (lines 168-177), no io.ReadAll on demo data |
| PARS-03 | 06-04 | Parser registers 12 event handlers | SATISFIED | parser.go: 12 RegisterEventHandler calls — MatchStart, RoundStart, RoundEnd, Kill, PlayerHurt, WeaponFire, BombPlanted, BombDefused, BombExplode, GrenadeProjectileThrow, PlayerConnect, TeamSideSwitch (lines 216-394) |
| PARS-04 | 06-04 | Parser batch-inserts per round, idempotent ON CONFLICT | SATISFIED | writer.go: pgx.Batch per round (line 174), ON CONFLICT (event_id) DO NOTHING on kill_events (line 156) and damage_events (line 167), ON CONFLICT (match_id, round_number) DO NOTHING on rounds (line 143); migration 000008 enables event_id UNIQUE |
| PARS-05 | 06-04 | Parser calls defer p.Close() on every demoinfocs parser | SATISFIED | parser.go:179: defer parser.Close() after every dem.NewParser() |
| CROS-01 | 06-01, 06-02, 06-03, 06-04 | Structured logging via slog with match_id/job_id | SATISFIED | All entrypoints use slog.NewJSONHandler; all services use slog.String("match_id", ...); downloader adds slog.String("job_id", ...); no {data, meta} envelopes |
| CROS-02 | 06-01, 06-02, 06-03, 06-04 | Functional options for dependency injection | SATISFIED | All 3 services: WithNATS, WithMinio, WithPostgres, WithLogger, WithHLTVClient, WithHTTPClient patterns; Runner: WithLogger, WithShutdownTimeout |
| CROS-03 | 06-01, 06-02, 06-03, 06-04 | No v1.0 code modified | SATISFIED | git diff shows zero changes to internal/hltv, internal/provider, internal/cli, internal/output, cmd/dem; grep confirms no imports from those packages |

**14/14 requirements SATISFIED. No orphaned requirements.**

## Anti-Patterns Found

| File | Pattern | Severity | Impact |
|------|---------|----------|--------|
| `internal/parser/parser.go:221` | `MapName: ""` hardcoded | INFO | demoinfocs-golang v5 does not expose MapName through public API (private field). Map name is nullable in schema. Can be populated from job payload in future enhancement. |
| `internal/parser/config.go` | Viper keys use UPPER_CASE (e.g., `NATS_URL`) vs lowercase in poller/downloader (e.g., `nats_url`) | INFO | Functionally equivalent — both map to same `DEM_NATS_URL` env var. Style inconsistency only. |

**No blockers or warnings. Two informational items only.**

## Human Verification Required

### 1. End-to-End Pipeline Execution
**Test:** Start infrastructure (`docker-compose up`), then run all three services (`go run ./cmd/poller`, `./cmd/downloader`, `./cmd/parser`)
**Expected:** Poller discovers Tier 1 matches from HLTV and publishes download jobs. Downloader streams .dem.gz files to MinIO and publishes parse jobs. Parser streams demos from MinIO, extracts game events, and inserts into Postgres.
**Why human:** Requires running NATS, MinIO, Postgres infrastructure and HLTV.org network access.

### 2. Idempotency Verification
**Test:** Run the parser twice on the same demo file, then query `SELECT COUNT(*) FROM kill_events WHERE match_id = '<id>'` and `SELECT COUNT(*) FROM damage_events WHERE match_id = '<id>'` after each run.
**Expected:** Identical row counts after both runs — ON CONFLICT (event_id) DO NOTHING prevents duplicates.
**Why human:** Requires a demo file in MinIO and a running Postgres instance.

### 3. Compilation Check
**Test:** Run `go build -o /dev/null ./cmd/poller && go build -o /dev/null ./cmd/downloader && go build -o /dev/null ./cmd/parser`
**Expected:** All three exit 0.
**Why human:** Sandbox build cache restrictions prevented full compilation. go vet passed for all packages; go build passed for internal/service and internal/poller.

## Gaps Summary

No blockers found. The pipeline is fully wired end-to-end in code:

```
Poller (cron + HLTV parsers)
  -> NATS JetStream dem.download.jobs
    -> Downloader (streaming CDN->MinIO)
      -> NATS JetStream dem.parse.jobs
        -> Parser (MinIO->demoinfocs->Postgres)
```

All 6 roadmap success criteria and all 14 requirements are satisfied with substantive code evidence. All artifacts exist, are substantive, and are wired through verified key links. Data flows trace from real sources (HLTV HTTP, CDN streaming, MinIO objects) through to Postgres inserts — no hardcoded empty data anywhere in the pipeline.

Two informational items noted:
1. `MapName` is empty in parser (demoinfocs-golang v5 API limitation) — schema allows NULL
2. Parser config Viper key casing differs from poller/downloader — functionally equivalent

Three human verification items identified for runtime behavior that cannot be verified from code alone.

---

_Verified: 2026-05-04T23:59:00Z_
_Verifier: Claude (gsd-verifier)_

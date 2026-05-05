---
status: complete
phase: 06-pipeline-services
source:
  - 06-01-SUMMARY.md
  - 06-02-SUMMARY.md
  - 06-03-SUMMARY.md
  - 06-04-SUMMARY.md
started: 2026-05-06T00:00:00Z
updated: 2026-05-06T02:20:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Infrastructure starts with docker compose
expected: |
  `docker compose up -d` starts NATS (port 4222), PostgreSQL (5432), MinIO (9000/9001), and Redis (6379).
  All four services report healthy within 30 seconds.
  `docker compose ps` shows all services Up and healthy.
result: pass

### 2. Poller discovers matches and publishes jobs
expected: |
  Running the poller (one-shot mode) fetches HLTV events, filters to Tier 1 tournaments,
  and publishes download jobs to NATS. Log shows "fetched events", "filtered to Tier 1",
  and "published download job" lines with match_id and event_name.
result: pass

### 3. Downloader fetches and stores demo in MinIO
expected: |
  Downloader consumes jobs from NATS, uses cloudscraper to download the .rar demo,
  extracts the .dem file, and uploads it to MinIO. Log shows "download completed"
  and "published parse job" with the object key.
result: pass

### 4. Parser processes demo into Postgres
expected: |
  Parser consumes parse jobs from NATS, streams the .dem from MinIO through demoinfocs,
  and populates Postgres. Log shows "match row written", no FK errors, and "parse complete".
  Querying Postgres shows non-zero counts in matches, rounds, kill_events, and damage_events tables.
result: pass

### 5. Deduplication prevents duplicate jobs
expected: |
  Running the poller twice produces no duplicate download jobs for already-processed matches.
  The processed_matches table has count(*) = count(DISTINCT match_id).
result: pass

### 6. Idempotent parsing — re-parsing same demo produces no duplicates
expected: |
  The parser ON CONFLICT (event_id) DO NOTHING ensures re-parsing the same demo
  produces identical row counts. SELECT count(*), count(DISTINCT event_id) FROM kill_events
  shows both counts equal.
result: pass

### 7. v1.0 CLI continues to work
expected: |
  Running dem events --tier 1, dem results --limit 5, or dem demo <match-id>
  returns valid JSON output without errors. Existing v1.0 code is untouched.
result: pass

### 8. Demos deleted from MinIO after successful parse
expected: |
  After the parser successfully completes, the demo object is removed from MinIO.
  The MinIO bucket only contains demos that are currently being parsed or that failed parsing.
result: pass

### 9. Structured logging with correlation fields
expected: |
  All three services (poller, downloader, parser) emit JSON log lines via log/slog.
  Download and parse logs include match_id and job_id fields.
  No service uses the v1.0 CLI {data, meta} JSON envelope format.
result: pass

## Summary

total: 9
passed: 9
issues: 0
pending: 0
skipped: 0

## Gaps

[none yet]

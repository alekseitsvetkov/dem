# Architecture Research: Microservice Platform (v1.1)

**Domain:** Go monorepo with NATS-based microservices for CS2 demo pipeline
**Researched:** 2026-05-03
**Confidence:** HIGH (all technology choices verified against current documentation)

## System Overview

The v1.1 milestone adds four components around the existing v1.0 CLI:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Docker Compose (local dev)                         │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌──────────────────────┐       ┌──────────────────────┐                    │
│  │  Tournament Polling   │       │   Demo Download       │                    │
│  │  Service              │       │   Service             │                    │
│  │  (Cron: daily/interval)│      │  (NATS consumer)      │                    │
│  │ ┌──────────────────┐  │       │ ┌──────────────────┐  │                    │
│  │ │ HLTV Client/     │  │       │ │ HLTV Downloader │  │                    │
│  │ │ Parser (reused)  │  │       │ │ (http GET .dem)  │  │                    │
│  │ └────────┬─────────┘  │       │ └────────┬─────────┘  │                    │
│  │          │            │       │          │            │                    │
│  │          ▼            │       │          ▼            │                    │
│  │   Discover new        │       │   Store .dem in       │                    │
│  │   Tier 1 matches      │       │   Minio bucket        │                    │
│  │   → publish jobs      │       │   → publish parse job │                    │
│  └──────────┬────────────┘       └──────────┬────────────┘                    │
│             │                               │                                 │
│  ┌──────────▼───────────────────────────────▼──────────────────────────┐     │
│  │                         NATS (JetStream)                             │     │
│  │  Stream: DEM_JOBS (retention: work queue, ack: explicit)           │     │
│  │  Subjects: dem.download.jobs, dem.parse.jobs                        │     │
│  └──────────┬───────────────────────────────┬──────────────────────────┘     │
│             │                               │                                 │
│  ┌──────────▼────────────┐      ┌───────────▼───────────────────────────┐   │
│  │  Demo Parsing Service  │      │  Infrastructure                       │   │
│  │  (NATS consumer)       │      │                                       │   │
│  │ ┌────────────────────┐ │      │  ┌──────────┐  ┌──────────────────┐  │   │
│  │ │ demoinfocs-golang  │ │      │  │  Minio   │  │   Postgres       │  │   │
│  │ │ (v5.2.0)           │ │      │  │  :9000   │  │   :5432          │  │   │
│  │ └────────┬───────────┘ │      │  │  (dem     │  │   (game_events,  │  │   │
│  │          │             │      │  │  files)   │  │    matches, etc)  │  │   │
│  │          ▼             │      │  └──────────┘  └──────────────────┘  │   │
│  │  Extract game events   │      │                                       │   │
│  │  → write to Postgres   │      │  ┌──────────────────────────────────┐│   │
│  └────────────────────────┘      │  │  v1.0 CLI (UNCHANGED)            ││   │
│                                   │  │  dem events/results/demo         ││   │
│                                   │  │  (direct HLTV, no services)      ││   │
│                                   │  └──────────────────────────────────┘│   │
│                                   └──────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Component Responsibilities

| Component | Responsibility | Implementation | Status |
|-----------|----------------|----------------|--------|
| `v1.0 CLI` (`cmd/dem`) | Existing HLTV commands (`events`, `results`, `demo`) | Cobra + Provider + HLTV Client + goquery | **Existing, unchanged** |
| `tournament-poller` | Cron-driven: fetch Tier 1 events, discover matches with demos, publish download jobs to NATS | Reuses `internal/hltv` Client + Parser; new scheduler + NATS publisher | **New** |
| `demo-downloader` | NATS consumer: receive match IDs, download .dem from HLTV, store in Minio, publish parse jobs | New HLTV download endpoint; Minio `minio-go/v7` client | **New** |
| `demo-parser` | NATS consumer: receive parse jobs, download .dem from Minio, parse with demoinfocs, write game events to Postgres | `demoinfocs-golang/v5`; `pgx/v5` for Postgres | **New** |
| `NATS` | Message queue for download/parse job distribution. Durable, at-least-once delivery, no lost jobs. | `nats-io/nats.go` + `nats-io/nats.go/jetstream` (new API) | **New infra** |
| `Minio` | S3-compatible object storage for .dem binary files (~30-200 MB each) | `minio/minio-go/v7` | **New infra** |
| `Postgres` | Relational store for parsed game events, matches, players, rounds | `jackc/pgx/v5` | **New infra** |

## Monorepo Layout

```
dem/                                # Root module (github.com/alekseitsvetkov/dem)
├── go.work                         # Go workspace (added in v1.1)
├── go.mod                          # Root: v1.0 CLI + shared libs
├── go.sum
├── cmd/
│   └── dem/
│       └── main.go                 # v1.0 CLI entry (UNCHANGED)
├── internal/
│   ├── cli/                        # Cobra commands (UNCHANGED)
│   ├── domain/                     # Domain models (EXTENDED with new types)
│   │   └── models.go
│   ├── hltv/                       # HLTV Client + Parser (REUSED by poller)
│   │   ├── client.go
│   │   ├── urls.go
│   │   ├── errors.go
│   │   └── parser/
│   ├── provider/                   # Provider interfaces (UNCHANGED)
│   └── output/                     # JSON helpers (UNCHANGED)
├── services/                       # NEW: microservice binaries
│   ├── poller/
│   │   ├── cmd/
│   │   │   └── main.go            # Entry point
│   │   └── internal/
│   │       ├── scheduler/          # Cron/interval scheduling
│   │       ├── discover/           # Tier 1 event + match discovery
│   │       └── publisher/          # NATS job publisher
│   ├── downloader/
│   │   ├── cmd/
│   │   │   └── main.go
│   │   └── internal/
│   │       ├── consumer/           # NATS download job consumer
│   │       ├── fetcher/            # HLTV .dem download
│   │       └── storage/            # Minio upload
│   └── parser/
│       ├── cmd/
│       │   └── main.go
│       └── internal/
│           ├── consumer/           # NATS parse job consumer
│           ├── extractor/          # demoinfocs event extraction
│           └── repository/         # Postgres writer
├── pkg/                            # NEW: shared libraries
│   ├── natsutil/                   # NATS connection helpers, stream config
│   ├── minio/                      # Minio client config, presigned URLs
│   └── postgres/                   # Connection pool, migrations
├── sql/                            # NEW: database migrations
│   └── migrations/
│       ├── 001_create_matches.up.sql
│       ├── 001_create_matches.down.sql
│       └── ...
├── docker-compose.yml              # NEW: local infra (NATS, Minio, Postgres)
├── Dockerfile.poller               # NEW
├── Dockerfile.downloader           # NEW
└── Dockerfile.parser               # NEW
```

### Structure Rationale

- **`services/*/cmd/` + `services/*/internal/`**: Each service is a separate binary with its own `main.go`. `internal/` enforces Go compiler-level encapsulation between services. Services are NOT separate Go modules initially -- they compile against the root module (shared `internal/hltv`, `internal/domain`, `pkg/*`). This avoids versioning overhead until services need independent release cycles.

- **`pkg/`**: Libraries intentionally shared across services or with external consumers. NATS connection setup, Minio client config, and Postgres pool management are shared infrastructure -- but each service chooses which `pkg/` to import. This is deliberate: the downloader imports `pkg/minio` but not `pkg/postgres`; the parser imports both.

- **`internal/domain/`** (EXTENDED): New domain types for parsed game events (`GameEvent`, `KillEvent`, `RoundInfo`, `MatchMetadata`) coexist with v1.0 types (`Event`, `Result`, `DemoLink`). No v1.0 types are modified.

- **`sql/migrations/`**: Versioned SQL migrations using `golang-migrate/migrate` or raw SQL with a versioning convention. Run at service startup or via a dedicated migration tool.

- **`docker-compose.yml`** at root: Single command starts the entire local platform (`docker compose up`). Go services run natively on the host (not containerized during development) for fast iteration; infrastructure (NATS, Minio, Postgres) runs in containers.

## Integration Points: Existing vs. New

### Existing Code Reused (no modifications)

| Existing Code | Reused By | How |
|---------------|-----------|-----|
| `internal/hltv/client.go` | Poller, Downloader | Poller calls `Client.Fetch()` for events/results pages. Downloader uses `net/http` directly for .dem binaries (different URL pattern, streaming download). HLTV client's browser-impersonation TLS config is not needed for .dem downloads. |
| `internal/hltv/urls.go` | Poller | Poller uses `URLs.EventsURL()`, `URLs.ResultsURLForEvent()`, `URLs.MatchURL()`. Added URL for .dem download (HLTV demo CDN) but at a different layer. |
| `internal/hltv/parser/` | Poller | Poller uses `ParseEvents`, `ParseResults`, `ParseDemoLink` to discover matches with available demos. |
| `internal/domain/models.go` | All services | v1.0 types unchanged. New types added to the same file (or a new `domain/game_events.go` file) for parsed event data. |
| `internal/output/` | None | v1.0 only. Services write structured logs, not CLI JSON output. |

### New Code (no existing code touched)

| New Code | Purpose | Dependencies |
|----------|---------|--------------|
| `services/poller/` | Tournament polling service | `internal/hltv`, `internal/domain`, `pkg/natsutil` |
| `services/downloader/` | Demo file download service | `pkg/natsutil`, `pkg/minio`, `internal/domain` |
| `services/parser/` | Demo parsing service | `pkg/natsutil`, `pkg/minio`, `pkg/postgres`, `internal/domain`, `demoinfocs-golang/v5` |
| `pkg/natsutil/` | NATS stream/consumer setup | `nats-io/nats.go/jetstream` |
| `pkg/minio/` | Minio client factory | `minio/minio-go/v7` |
| `pkg/postgres/` | Postgres pool + migrations | `jackc/pgx/v5` |
| `sql/migrations/` | Database schema | None (pure SQL) |
| `internal/domain/game_events.go` | New domain types for parsed events | None (pure structs) |

### Key Decision: No v1.0 Code Modified

The v1.0 CLI path (`cmd/dem/main.go` through `internal/cli/*` through `internal/provider/*` through `internal/hltv/*`) is completely untouched. Services import shared packages (`internal/hltv`, `internal/domain`) as library code -- they do not modify it. This preserves the CLI's JSON contract, test suite, and independent deployability.

### Key Decision: Go Workspace, Not Multi-Module

The root `go.mod` remains the single module. A `go.work` file is NOT needed initially because all services compile against the same module. Multi-module separation (with `go.work`) is deferred until a service needs independent versioning or a different Go version requirement. This avoids the complexity of `go.work` and cross-module dependency management for the v1.1 scope.

## Data Flow

### Flow 1: Tournament Discovery (daily / on-demand)

```
[Cron trigger / manual run]
        │
        ▼
  Poller Service
        │
        ├─► hltv.Client.Fetch(EventsURL()) ──► parser.ParseEvents()
        │       │
        │       ▼
        │   Filter Tier 1 events (reuse tier1Keywords from provider/events.go)
        │       │
        │       ▼
        ├─► For each Tier 1 event: Fetch results page ──► parser.ParseResults()
        │       │
        │       ▼
        │   For each result with a match ID:
        │       │
        │       ├─► Fetch match page ──► parser.ParseDemoLink()
        │       │       │
        │       │       ▼
        │       │   If demo_url is present (demo available):
        │       │       │
        │       │       ▼
        │       │   Publish to NATS: dem.download.jobs
        │       │   Payload: {match_id, demo_url, event_name, teams, match_date}
        │       │
        │       └─► Record processed match IDs (in-memory or Postgres)
        │               to avoid duplicate publishing on next poll cycle.
        │
        ▼
    Log summary: "Discovered N new demos, published M download jobs"
```

### Flow 2: Demo Download (triggered by NATS message)

```
NATS Message: dem.download.jobs
        │
        ▼
  Downloader Service (consumes via JetStream pull consumer)
        │
        ├─► Parse job payload: {match_id, demo_url, ...}
        │
        ├─► GET demo_url (streaming HTTP download; .dem files are 30-200 MB)
        │       │
        │       ▼
        │   Stream response body directly to Minio PutObject
        │   (no local disk write -- pipe body to minio-go PutObject with io.Reader)
        │       │
        │       ▼
        │   Minio key: demos/{match_id}.dem
        │   Minio bucket: dem-files
        │       │
        │       ▼
        │   On success: Ack() NATS message
        │       │
        │       ▼
        │   Publish to NATS: dem.parse.jobs
        │   Payload: {match_id, minio_key: "demos/{match_id}.dem", demo_url, metadata...}
        │
        ▼
    On failure: Nak() with delay (retry), log error, increment failure counter
```

### Flow 3: Demo Parsing (triggered by NATS message)

```
NATS Message: dem.parse.jobs
        │
        ▼
  Parser Service (consumes via JetStream pull consumer)
        │
        ├─► Parse job payload: {match_id, minio_key, ...}
        │
        ├─► Get .dem from Minio: minioClient.GetObject(bucket, key)
        │       │
        │       ▼
        │   Pipe response body to demoinfocs.NewParser(reader)
        │       │
        │       ▼
        │   Register event handlers:
        │       - events.MatchStart        → upsert match row
        │       - events.RoundStart         → insert round row
        │       - events.RoundEnd           → update round with winner/reason
        │       - events.Kill               → insert kill event
        │       - events.PlayerHurt         → insert damage event
        │       - events.WeaponFire         → insert weapon fire event
        │       - events.BombPlanted        → insert bomb plant event
        │       - events.BombDefused        → insert bomb defuse event
        │       - events.BombExplode        → insert bomb explode event
        │       - events.GrenadeProjectileThrow → insert grenade throw event
        │       - events.PlayerConnect      → upsert player row
        │       - events.TeamSideSwitch     → update round side info
        │       - events.Footstep           → (optional) for future clustering
        │       │
        │       ▼
        │   p.ParseToEnd()
        │       │
        │       ▼
        │   On success: Ack() NATS message
        │       │
        │       ▼
        │   Insert/update match metadata in Postgres (parsed_at, duration, tick_count)
        │
        ▼
    On failure: Nak() with delay, log error
```

## NATS Topic Design

### Stream Configuration

```
Stream: DEM_JOBS
├── Subjects:
│   ├── dem.download.jobs      # Poller publishes, Downloader consumes
│   └── dem.parse.jobs         # Downloader publishes, Parser consumes
├── Retention: WorkQueue       # Messages deleted after acknowledgment
├── Storage: File              # Persistent to disk
├── MaxAge: 7 days             # Unacknowledged jobs expire after 7 days
├── Replicas: 1                # Single server (dev); 3 for production
└── Discard: Old               # Old messages discarded if stream full
```

### Consumer Configuration

| Consumer | Stream | Durable Name | Filter Subject | Pattern | Notes |
|----------|--------|-------------|----------------|---------|-------|
| `download-worker` | `DEM_JOBS` | `download-worker` | `dem.download.jobs` | Pull, `AckExplicit`, `MaxDeliver: 3` | At-least-once. Fails after 3 delivery attempts → DLQ. |
| `parse-worker` | `DEM_JOBS` | `parse-worker` | `dem.parse.jobs` | Pull, `AckExplicit`, `MaxDeliver: 3` | At-least-once. Parsing is CPU-intensive. |

### Subject Naming Convention

```
dem.{domain}.{action}
 │     │        │
 │     │        └── jobs (command), events (notification)
 │     └── download, parse, cluster
 └── Project namespace
```

This uses a **domain-first, action-last** convention. The project namespace (`dem`) is the first token, enabling NATS multi-tenancy if needed later. `jobs` suffix indicates a work-queue message; `events` suffix would indicate a notification/fanout message (used for future clustering results).

### Message Payload Schemas

**dem.download.jobs:**
```json
{
  "match_id": "2378941",
  "demo_url": "https://demos.hltv.org/demo123.zip",
  "event_name": "IEM Katowice 2026",
  "team1": "FaZe",
  "team2": "G2",
  "match_date": "2026-04-15",
  "discovered_at": "2026-05-03T10:00:00Z"
}
```

**dem.parse.jobs:**
```json
{
  "match_id": "2378941",
  "minio_key": "demos/2378941.dem",
  "minio_bucket": "dem-files",
  "event_name": "IEM Katowice 2026",
  "team1": "FaZe",
  "team2": "G2",
  "match_date": "2026-04-15",
  "downloaded_at": "2026-05-03T10:05:00Z"
}
```

## Domain Models (Extended for v1.1)

New types in `internal/domain/game_events.go` (coexisting with v1.0 types in `models.go`):

```go
// MatchMetadata represents a parsed demo's match-level information.
type MatchMetadata struct {
    MatchID   string    `json:"match_id"`
    Team1     string    `json:"team1"`
    Team2     string    `json:"team2"`
    MapName   string    `json:"map_name"`
    TickRate  float64   `json:"tick_rate"`
    Duration  float64   `json:"duration_seconds"`
    ParsedAt  time.Time `json:"parsed_at"`
}

// KillEvent represents a single kill during a match.
type KillEvent struct {
    MatchID      string `json:"match_id"`
    RoundNumber  int    `json:"round_number"`
    Tick         int    `json:"tick"`
    Killer       string `json:"killer"`
    Victim       string `json:"victim"`
    Weapon       string `json:"weapon"`
    IsHeadshot   bool   `json:"is_headshot"`
    Wallbang     bool   `json:"wallbang"`
    KillerTeam   string `json:"killer_team"`
    VictimTeam   string `json:"victim_team"`
}

// RoundInfo represents a round's start and end state.
type RoundInfo struct {
    MatchID     string `json:"match_id"`
    RoundNumber int    `json:"round_number"`
    StartTick   int    `json:"start_tick"`
    EndTick     int    `json:"end_tick"`
    Winner      string `json:"winner"`
    EndReason   string `json:"end_reason"`
    TTeam       string `json:"t_team"`
    CTTeam      string `json:"ct_team"`
}

// DamageEvent represents player damage (for future clustering analysis).
type DamageEvent struct {
    MatchID      string `json:"match_id"`
    RoundNumber  int    `json:"round_number"`
    Tick         int    `json:"tick"`
    Attacker     string `json:"attacker"`
    Victim       string `json:"victim"`
    Weapon       string `json:"weapon"`
    Damage       int    `json:"damage"`
    HitGroup     string `json:"hit_group"`
}
```

## Database Schema (Postgres)

```sql
-- Matches table: one row per parsed demo
CREATE TABLE matches (
    match_id       TEXT PRIMARY KEY,
    team1          TEXT NOT NULL,
    team2          TEXT NOT NULL,
    map_name       TEXT,
    event_name     TEXT,
    match_date     DATE,
    tick_rate      DOUBLE PRECISION,
    duration_secs  DOUBLE PRECISION,
    demo_url       TEXT,
    minio_key      TEXT,
    parsed_at      TIMESTAMPTZ DEFAULT NOW(),
    created_at     TIMESTAMPTZ DEFAULT NOW()
);

-- Rounds: one row per round in a match
CREATE TABLE rounds (
    id             BIGSERIAL PRIMARY KEY,
    match_id       TEXT NOT NULL REFERENCES matches(match_id),
    round_number   INTEGER NOT NULL,
    start_tick     INTEGER NOT NULL,
    end_tick       INTEGER,
    winner         TEXT,        -- 'CT' or 'T'
    end_reason     TEXT,        -- 'bomb_detonated', 'ct_eliminated', etc.
    t_team         TEXT,
    ct_team        TEXT,
    UNIQUE (match_id, round_number)
);

-- Kill events: one row per kill
CREATE TABLE kill_events (
    id             BIGSERIAL PRIMARY KEY,
    match_id       TEXT NOT NULL REFERENCES matches(match_id),
    round_number   INTEGER NOT NULL,
    tick           INTEGER NOT NULL,
    killer         TEXT NOT NULL,
    victim         TEXT NOT NULL,
    weapon         TEXT,
    is_headshot    BOOLEAN DEFAULT FALSE,
    wallbang       BOOLEAN DEFAULT FALSE,
    killer_team    TEXT,
    victim_team    TEXT
);

-- Damage events: for future spatial analysis
CREATE TABLE damage_events (
    id             BIGSERIAL PRIMARY KEY,
    match_id       TEXT NOT NULL REFERENCES matches(match_id),
    round_number   INTEGER NOT NULL,
    tick           INTEGER NOT NULL,
    attacker       TEXT NOT NULL,
    victim         TEXT NOT NULL,
    weapon         TEXT,
    damage         INTEGER NOT NULL,
    hit_group      TEXT
);

-- Players: deduplicated across matches
CREATE TABLE players (
    id             BIGSERIAL PRIMARY KEY,
    name           TEXT NOT NULL,
    UNIQUE (name)
);

-- Match-Player junction: which players played which matches
CREATE TABLE match_players (
    match_id       TEXT NOT NULL REFERENCES matches(match_id),
    player_id      BIGINT NOT NULL REFERENCES players(id),
    team           TEXT,  -- 'CT' or 'T' or 'both'
    PRIMARY KEY (match_id, player_id)
);

-- Indexes for common queries
CREATE INDEX idx_kill_events_match_round ON kill_events(match_id, round_number);
CREATE INDEX idx_damage_events_match_round ON damage_events(match_id, round_number);
CREATE INDEX idx_rounds_match ON rounds(match_id);
```

## Architectural Patterns

### Pattern 1: Provider/Tier Isolation (carried from v1.0)

**What:** Each service's business logic is behind an interface. External dependencies (NATS, Minio, Postgres) are injected via functional options.

**When:** Every service constructor.

**Example:**
```go
type PollerService struct {
    client  *hltv.Client
    urls    hltv.URLs
    js      jetstream.JetStream
}

type PollerOption func(*PollerService)

func NewPollerService(opts ...PollerOption) *PollerService {
    p := &PollerService{
        client: hltv.NewClient(),
        urls:   hltv.NewURLs(""),
    }
    for _, opt := range opts {
        opt(p)
    }
    return p
}

func WithNATS(js jetstream.JetStream) PollerOption {
    return func(p *PollerService) { p.js = js }
}
```

### Pattern 2: Streaming Upload (no disk intermediate)

**What:** .dem files (30-200 MB) should never be written to local disk. Pipe the HTTP response body directly to Minio `PutObject` via `io.Reader`.

**When:** Download service.

**Trade-offs:** Pro: no disk I/O, no temp file cleanup, lower memory (streaming). Con: cannot resume partial downloads. Mitigated by NATS `MaxDeliver` retries -- a failed download retries from scratch.

**Example:**
```go
resp, err := http.Get(demoURL)
// ...
info, err := minioClient.PutObject(ctx, bucket, key, resp.Body, resp.ContentLength,
    minio.PutObjectOptions{ContentType: "application/octet-stream"})
```

### Pattern 3: JetStream Work Queue with Explicit Ack

**What:** Durable pull consumers with `AckExplicit`. Each message is `Ack()`'d only after successful processing. `MaxDeliver: 3` with `Nak()` for transient failures. Unprocessable messages exceed `MaxDeliver` → dead-letter handling (future: DLQ stream).

**When:** All NATS consumers (`download-worker`, `parse-worker`).

**Trade-offs:** At-least-once semantics means duplicate processing is possible (e.g., service crashes after processing but before ack). Services must be idempotent: `INSERT ... ON CONFLICT DO NOTHING` for Postgres writes, `PutObject` with `If-None-Match` or pre-check for Minio.

**Example:**
```go
stream, _ := js.CreateStream(ctx, jetstream.StreamConfig{
    Name:      "DEM_JOBS",
    Subjects:  []string{"dem.download.jobs", "dem.parse.jobs"},
    Retention: jetstream.WorkQueuePolicy,
})

cons, _ := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
    Durable:    "download-worker",
    AckPolicy:  jetstream.AckExplicitPolicy,
    MaxDeliver: 3,
})

iter, _ := cons.Messages(jetstream.PullMaxMessages(1))
for {
    msg, err := iter.Next()
    if err != nil {
        break
    }
    if err := processDownload(msg.Data()); err != nil {
        msg.Nak()          // Retry later
        continue
    }
    msg.Ack()
}
```

### Pattern 4: Config via Environment

**What:** Each service reads configuration from environment variables, NOT from config files. 12-factor app style. Docker Compose sets env vars on service containers.

**When:** All services at startup.

**Example:**
```go
type Config struct {
    NATSURL       string        // NATS_URL (default: nats://localhost:4222)
    MinioEndpoint string        // MINIO_ENDPOINT
    MinioAccessKey string       // MINIO_ACCESS_KEY
    MinioSecretKey string       // MINIO_SECRET_KEY
    MinioBucket   string        // MINIO_BUCKET (default: dem-files)
    DatabaseURL   string        // DATABASE_URL
    PollInterval  time.Duration // POLL_INTERVAL (default: 6h)
}

func LoadConfig() Config {
    return Config{
        NATSURL:        env("NATS_URL", "nats://localhost:4222"),
        MinioEndpoint:  env("MINIO_ENDPOINT", "localhost:9000"),
        MinioAccessKey: env("MINIO_ACCESS_KEY", ""),
        MinioSecretKey: env("MINIO_SECRET_KEY", ""),
        MinioBucket:    env("MINIO_BUCKET", "dem-files"),
        DatabaseURL:    env("DATABASE_URL", ""),
        PollInterval:   envDuration("POLL_INTERVAL", 6*time.Hour),
    }
}
```

## Docker Compose (Local Development)

```yaml
version: "3.9"

services:
  nats:
    image: nats:2.10-alpine
    command: ["-js", "-m", "8222"]
    ports:
      - "4222:4222"
      - "8222:8222"
    restart: unless-stopped

  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: dem
      POSTGRES_PASSWORD: dem
      POSTGRES_DB: dem
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U dem"]
      interval: 5s
      timeout: 3s
      retries: 5

  minio:
    image: minio/minio:latest
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    ports:
      - "9000:9000"
      - "9001:9001"
    volumes:
      - miniodata:/data
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 10s
      timeout: 5s
      retries: 3

volumes:
  pgdata:
  miniodata:
```

Go services run natively (not containerized) during development:
```bash
# Terminal 1: Start infra
docker compose up -d

# Terminal 2: Run poller
export NATS_URL=nats://localhost:4222
go run ./services/poller/cmd/main.go

# Terminal 3: Run downloader
export NATS_URL=nats://localhost:4222 MINIO_ENDPOINT=localhost:9000 MINIO_ACCESS_KEY=minioadmin MINIO_SECRET_KEY=minioadmin
go run ./services/downloader/cmd/main.go

# Terminal 4: Run parser
export NATS_URL=nats://localhost:4222 DATABASE_URL=postgres://dem:dem@localhost:5432/dem?sslmode=disable
go run ./services/parser/cmd/main.go
```

## Anti-Patterns to Avoid

### Anti-Pattern 1: Tight Coupling via Shared Database
**What:** Multiple services reading/writing the same Postgres tables directly.
**Why wrong:** Schema changes in one service break others. No clear data ownership.
**Instead:** Only the Parser Service writes to Postgres. Other services that need game event data (future: clustering service) read through a defined API or consume NATS event messages, not direct SQL.

### Anti-Pattern 2: Blocking NATS Handlers with 30-Minute Operations
**What:** Processing a .dem download or parse inside the NATS message handler without heartbeats, causing `no heartbeat received` errors from the consumer.
**Why wrong:** NATS assumes handlers are fast. Long operations cause consumer disconnects and message redelivery.
**Instead:** Use `Consume()` with a buffered channel. The message handler posts to the channel and acks immediately after work is queued. Or use `Messages()` with `PullMaxMessages(1)` and extended heartbeat intervals. Set `AckWait` to a value longer than the maximum expected processing time (e.g., 30 minutes for parsing).

### Anti-Pattern 3: Writing .dem Files to Disk
**What:** Downloading .dem to a local temp file, then uploading to Minio from disk.
**Why wrong:** Wastes disk I/O, requires temp space for 200 MB files, leaves garbage on crashes.
**Instead:** Stream from HTTP response body directly to Minio `PutObject` using `io.Reader`. The downloader never touches disk.

### Anti-Pattern 4: Modifying v1.0 Provider Interfaces
**What:** Adding service-related methods to existing `EventsProvider`, `ResultsProvider`, `DemoProvider` interfaces.
**Why wrong:** Breaks the v1.0 CLI contract and tests. Forces CLI to depend on service infrastructure.
**Instead:** Services import `internal/hltv` directly (Client + Parser) as library code. They do not use the Provider layer. The Provider layer remains CLI-specific middleware.

## Scalability Considerations

| Scale | Architecture Adjustment |
|-------|------------------------|
| 10 matches/day | Single instance of each service. NATS, Minio, Postgres on one machine. Current design works as-is. |
| 100 matches/day | Run multiple downloader instances (competing consumers on same durable queue). Parser remains single-instance (CPU-bound) but can scale horizontally if demos are independent. |
| 1,000+ matches/day | Add a `dem.parse.jobs` sharding key (e.g., by match_id) to distribute across multiple parser instances. Add Redis for deduplication cache in poller. Consider S3/Minio tiered storage for old demos. |

### First Bottleneck

**Parser Service CPU.** demoinfocs parses 25 minutes of game time per second (single-threaded). A 60-minute match requires ~2.4 seconds of CPU plus I/O overhead. At 100 matches/day, this is still under a minute of compute. The real bottleneck is concurrent parsing -- demoinfocs consumes ~250 MB RAM per demo. Run 2-3 parser instances with `MaxAckPending` limiting concurrency per instance.

### Second Bottleneck

**Postgres write volume.** Kill events alone produce ~40-50 rows per round, ~1,200 rows per match (24 rounds). With damage events, that triples. Batch inserts (not single-row) are essential. Use `pgx` CopyFrom for bulk inserts after each round completes, not after each individual event.

## Build Order Implications

Services depend on infrastructure being available. Recommended build order:

1. **Infrastructure**: Docker Compose, `pkg/natsutil`, `pkg/minio`, `pkg/postgres`, database migrations (`sql/`)
   - Nothing else can run without NATS, Minio, Postgres.
   - Database schema must exist before parser inserts data.

2. **Domain models**: `internal/domain/game_events.go`
   - Shared by all services. Must exist before any service implementation.

3. **Poller Service**: `services/poller/`
   - Depends on: infrastructure, domain models, `internal/hltv` (already exists)
   - First service because it produces messages. No consumers yet means messages queue up (NATS handles this gracefully).

4. **Downloader Service**: `services/downloader/`
   - Depends on: infrastructure, domain models, `pkg/natsutil`, `pkg/minio`
   - Second service because it consumes poller messages and produces parse job messages.

5. **Parser Service**: `services/parser/`
   - Depends on: infrastructure, domain models, `pkg/natsutil`, `pkg/minio`, `pkg/postgres`, `demoinfocs-golang/v5`
   - Final service: consumes downloader messages, writes to Postgres.

### Why This Order

- Each service depends only on infrastructure and shared packages, never on another service. This is by design: NATS decouples them.
- Infrastructure first because `docker compose up` must work before any service starts.
- Poller first because it produces the messages that downstream services consume. While building the downloader, you can manually publish test messages to `dem.download.jobs` using `nats pub`.
- Parser last because it has the most external dependencies (demoinfocs, Postgres) and is the hardest to test end-to-end.

## Sources

- [NATS JetStream Go Client (nats.go)](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream) — HIGH confidence
- [NATS by Example: Pull Consumers (Go)](https://examples.nats.io/examples/jetstream/pull-consumer/go) — HIGH confidence
- [NATS Subject Naming Conventions](https://docs.nats.io/nats-concepts/subjects) — HIGH confidence
- [demoinfocs-golang v5.2.0](https://pkg.go.dev/github.com/markus-wa/demoinfocs-golang/v5) — HIGH confidence
- [demoinfocs-golang Events Package](https://pkg.go.dev/github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events) — HIGH confidence
- [Minio Go SDK v7](https://pkg.go.dev/github.com/minio/minio-go/v7) — HIGH confidence
- [evrone/go-service-template (Go Clean Architecture with NATS)](https://github.com/evrone/go-service-template) — MEDIUM confidence
- Existing codebase: `internal/hltv/client.go`, `internal/provider/events.go`, `internal/provider/demo.go`, `internal/domain/models.go` — HIGH confidence

---
*Architecture research for: HLTV CLI v1.1 Microservice Platform*
*Researched: 2026-05-03*

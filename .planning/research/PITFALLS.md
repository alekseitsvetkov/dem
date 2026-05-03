# Pitfalls Research

**Domain:** Go microservice platform with NATS, Minio, Postgres, and demo file pipelines
**Researched:** 2026-05-03
**Confidence:** HIGH

## Critical Pitfalls

These are the mistakes that cause rewrites, data loss, or multi-day debugging sessions. Every one is drawn from production incidents, maintainer discussions, or documented failure modes.

---

### Pitfall 1: JetStream Messages Published Without a Backing Stream — Silent Data Loss

**What goes wrong:**
Publishing to a JetStream subject without first creating the stream succeeds without error — but the message is silently discarded. No logs, no errors, no way to know it was lost without external monitoring. This is particularly dangerous in the tournament polling pipeline where missing a match means the demo is never downloaded or parsed.

**Why it happens:**
The NATS Go client's `Publish` method does not validate that a stream exists for the subject. It returns success immediately. The NATS server routes the message to JetStream, finds no matching stream, and drops it. This is by design: Core NATS subjects are not required to have JetStream streams.

**How to avoid:**
Create all streams explicitly at service startup (not lazily on first publish). Use `StreamConfig` with explicit `Subjects` arrays — never publish before stream creation completes:

```go
func initStreams(js jetstream.JetStream) error {
    // Order matters: create streams before any publisher starts
    _, err := js.CreateStream(ctx, jetstream.StreamConfig{
        Name: "DEMO_DOWNLOADS",
        Subjects: []string{"demo.downloads.>"},
        Retention: jetstream.WorkQueuePolicy,
        MaxAge: 7 * 24 * time.Hour,
    })
    return err
}

// Then start publishers. Never the other way around.
```

For safety, add a startup health check that verifies all expected streams exist via `js.Stream()` before marking the service as ready.

**Warning signs:**
- Publishing returns no error but consumers never see the message.
- `nats consumer info` shows 0 pending messages despite publishers reporting success.
- The `Nats-Msg-Id` header doesn't prevent duplicates (because duplicates imply the first was received).

**Phase to address:**
Phase 1 (Infrastructure Foundation) — stream provisioning must be the first thing that happens in the platform bootstrap.

---

### Pitfall 2: Missing `msg.Ack()` — Infinite Redelivery Loops

**What goes wrong:**
A worker pulls a message, begins processing, but never calls `msg.Ack()`. After `AckWait` expires (typically 30 seconds), the message is redelivered. If the consumer is configured without `MaxDeliver`, this loops forever, burning CPU and potentially re-executing side effects (downloading the same demo twice, inserting duplicate game events).

**Why it happens:**
JetStream requires explicit, manual acknowledgements. Unlike Kafka, there is no auto-commit. Three common failure paths:
1. Handler panics or has an early `return` before reaching `Ack()`.
2. `Ack()` is called inside a conditional block that isn't always reached.
3. The consumer or NATS connection is deleted/closed before `Ack()` returns — the Ack silently succeeds (returns no error) but the message is never actually removed from the stream. It will be redelivered on consumer recreation. (Documented in nats.go issue #1793.)

**How to avoid:**

```go
// ALWAYS use defer. This is the single most important pattern in NATS workers.
cons.Consume(func(msg jetstream.Msg) {
    defer msg.Ack()  // fires even on panic

    // For error cases, Nak with a delay:
    if err := processMessage(msg); err != nil {
        msg.NakWithDelay(5 * time.Second)
        return  // defer still fires Ack — but Nak takes priority
    }
    // all good, defer fires Ack
})
```

Key rules:
- `defer msg.Ack()` at the TOP of every handler. Non-negotiable.
- Use `msg.InProgress()` for jobs exceeding 30 seconds to extend the ack window without finalizing.
- Set `MaxDeliver: 5` on every consumer. A poison message looping forever is worse than a lost message.
- Implement consumer-side idempotency (Postgres upsert, unique constraint on match_id + event_id).

**Warning signs:**
- Messages appear in `nats consumer info` as "Unprocessed" with delivery count climbing.
- Application logs show the same match/demo being processed over and over.
- Memory/CPU steady growth from repeated re-processing.

**Phase to address:**
Phase 2 (Download/Parse Services) — correct Ack patterns must be baked into the first worker implementation, not retrofitted.

---

### Pitfall 3: Building Docker Images with `COPY . .` in a Monorepo with `go.mod` Replace Directives

**What goes wrong:**
Each service's `go.mod` has a `replace` directive pointing to `../../internal/shared` or `../../internal/domain`. During local `go build`, this resolves fine. Inside Docker, the build context is limited to the service directory (`./services/downloader/`), so `COPY ../../internal/...` fails with "Forbidden path outside the build context." The build breaks.

Even if you fix the COPY by setting context to the repo root, `replace` directives reference local paths that don't exist in the Docker builder stage unless you copy the entire monorepo — which destroys layer caching (any file change invalidates `go mod download`).

**Why it happens:**
Go's `replace` directive is a local development tool, not a deployment mechanism. Docker's build model and Go's module resolution have a fundamental impedance mismatch in monorepos with multiple `go.mod` files.

**How to avoid:**

For this project specifically, the cleanest path is a **single root `go.mod`** for the entire repository:

```
dem/
├── go.mod              # module github.com/user/dem
├── cmd/
│   ├── dem/            # existing CLI binary (main.go)
│   ├── poller/         # new: tournament polling service (main.go)
│   ├── downloader/     # new: demo download service (main.go)
│   └── parser/         # new: demo parsing service (main.go)
└── internal/
    ├── cli/            # existing
    ├── hltv/           # existing
    ├── provider/       # existing
    ├── domain/         # existing models + new domain types
    ├── nats/           # new: NATS connection, stream setup, JetStream helpers
    ├── storage/        # new: Minio client wrapper
    └── database/       # new: Postgres/pgx pool and queries
```

Then each service Dockerfile:

```dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download           # cached unless go.mod changes
COPY . .                      # only needed files via .dockerignore
RUN go build -o /app ./cmd/downloader

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /app /app
ENTRYPOINT ["/app"]
```

```yaml
# docker-compose.yml — context at repo root, dockerfile points to service
services:
  downloader:
    build:
      context: .
      dockerfile: ./deploy/docker/downloader/Dockerfile
```

**Why this works for this project:** The existing code is already 3,136 LOC with one primary module. Adding services as additional `cmd/` entrypoints under one module avoids the `replace` directive problem entirely. Each binary can import from `internal/` packages without hacks.

**Warning signs:**
- `docker build` fails with "Forbidden path outside the build context."
- CI passes but Docker builds fail (because CI runs `go test ./...` locally without Docker).
- Developers add increasingly complex `COPY` commands and `.dockerignore` rules to work around it.

**Phase to address:**
Phase 1 (Infrastructure Foundation) — the monorepo structure decision (single vs. multiple go.mod) must be made before any service scaffolding begins.

---

### Pitfall 4: Loading 100MB+ .dem Files Entirely Into Memory

**What goes wrong:**
A 100MB+ demo file is downloaded from HLTV, read entirely into a `[]byte` via `io.ReadAll(resp.Body)`, then uploaded to Minio and passed to demoinfocs-golang. Memory usage spikes to file size plus parser overhead (~250 MB per demo). With multiple concurrent downloads or parsers, the service OOM-kills.

**Why it happens:**
The "default" Go pattern of `io.ReadAll` is correct for small payloads but lethal for 100MB+ files. Developers gravitating to the simplest API (read all, then process) hit this when demos arrive.

**How to avoid:**

Download streaming — never buffer the entire file:

```go
// Stream download directly to Minio without buffering in memory
resp, err := http.Get(demoURL)
defer resp.Body.Close()

// Stream directly to Minio PutObject — no intermediate []byte
_, err = minioClient.PutObject(ctx, bucket, objectKey, resp.Body, -1, minio.PutObjectOptions{
    PartSize:    128 * 1024 * 1024,  // 128 MiB parts for 100MB+ files
    ContentType: "application/octet-stream",
})
```

Parse directly from Minio without buffering:

```go
// Stream from Minio to demoinfocs-golang
obj, err := minioClient.GetObject(ctx, bucket, objectKey, minio.GetObjectOptions{})
defer obj.Close()

p := dem.NewParser(obj)  // demoinfocs-golang accepts io.Reader
defer p.Close()

// Register handlers, then parse
p.ParseToEnd()
```

Key rules:
- **Never** `io.ReadAll()` a demo file.
- **Never** `os.ReadFile()` a demo file.
- **Always** stream: HTTP response body -> Minio PutObject -> Minio GetObject -> demoinfocs.
- Add `http.MaxBytesReader` as a safety net to fail fast on oversized responses.

**Warning signs:**
- Memory graphs show spikes correlated with demo downloads.
- OOM kills when multiple demos are being processed concurrently.
- `runtime.MemStats.Alloc` climbing during `PutObject` or `GetObject` calls.

**Phase to address:**
Phase 2 (Download/Parse Services) — the download and parse implementations must be stream-oriented from the first line of code.

---

### Pitfall 5: Using `pgx.Conn` Instead of `pgxpool` in Concurrent Services

**What goes wrong:**
`pgx.Conn` is not concurrency-safe. If multiple goroutines (NATS message handlers, concurrent demo parsers) share a single `*pgx.Conn`, race conditions occur. Worse, a developer creates a new connection per request via `pgx.Connect()` instead of using a pool — each connection holds a PostgreSQL backend process, and connections aren't reused. Under load, PostgreSQL hits `max_connections` and rejects all new connections.

**Why it happens:**
The `pgx` library has two tiers: `pgx.Conn` (low-level, single-connection, NOT concurrency-safe) and `pgxpool.Pool` (connection pool, concurrency-safe). Developers familiar with `database/sql` assume all connection types are pooled. pgx is explicit: you must choose the pool type.

**How to avoid:**

```go
// Create pool ONCE at startup, reuse for service lifetime
func main() {
    ctx := context.Background()
    config, _ := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
    config.MaxConns = 20                      // default is 4 — way too low
    config.MinConns = 2                       // warm connections
    config.MaxConnIdleTime = 5 * time.Minute
    config.HealthCheckPeriod = 1 * time.Minute

    pool, err := pgxpool.NewWithConfig(ctx, config)
    if err != nil {
        log.Fatalf("pgxpool: %v", err)
    }
    defer pool.Close()

    // Pass pool — not a single Conn — to all handlers
    svc := &Service{db: pool}
}
```

Critical pgxpool defaults to override:
| Setting | Default | Recommended | Why |
|---------|---------|-------------|-----|
| `MaxConns` | 4 | 10-25 | 4 is too low for any concurrent workload |
| `MinConns` | 0 | 2-4 | Avoid cold-start latency on first query |
| `ConnectTimeout` | none | 5s | Fail fast if Postgres is unreachable |

**Warning signs:**
- Panic with "conn is closed" or race detector failures.
- PostgreSQL logs show `FATAL: too many connections for role`.
- `pool.Stat().AcquireCount()` grows without bound.

**Phase to address:**
Phase 1 (Infrastructure Foundation) — the database connection pool is foundational infrastructure, shared by all services.

---

### Pitfall 6: At-Least-Once Delivery Without Idempotency — Duplicate Game Events

**What goes wrong:**
A demo download completes, a parse job is queued on NATS. The worker pulls the job, begins parsing, and crashes (or exceeds `AckWait`) before calling `msg.Ack()`. NATS redelivers the message. The next worker re-downloads the demo and re-parses, inserting duplicate game events (kills, round starts, grenade throws) into Postgres.

This is guaranteed to happen in production — at-least-once semantics means redelivery is a feature, not a bug. Without idempotency at the storage layer, you get duplicate data that silently corrupts analytical queries.

**Why it happens:**
Developers conflate "delivery" with "exactly-once processing." NATS delivers at-least-once. The consumer must make processing idempotent. The most tempting shortcut is to just INSERT game events (fast), which creates duplicates, instead of UPSERT (slightly slower, correct).

**How to avoid:**

1. **Use Postgres `ON CONFLICT` for idempotent inserts:**

```go
const insertKill = `
INSERT INTO game_events (match_id, event_id, tick, type, payload)
VALUES ($1, $2, $3, 'kill', $4)
ON CONFLICT (match_id, event_id) DO NOTHING
`
```

2. **Design domain models with deterministic IDs:**
Each game event gets a deterministic ID: `{match_id}:{tick}:{event_sequence}`. Re-parsing the same demo produces the same IDs, and `ON CONFLICT DO NOTHING` skips duplicates.

3. **Use a processing state table for the demo itself:**

```sql
CREATE TABLE demo_processing (
    match_id BIGINT PRIMARY KEY,
    status TEXT NOT NULL,  -- 'downloading', 'parsing', 'complete', 'failed'
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error_message TEXT
);
```

```go
// Claim ownership atomically before starting work
result, err := pool.Exec(ctx, `
    INSERT INTO demo_processing (match_id, status, started_at)
    VALUES ($1, 'parsing', NOW())
    ON CONFLICT (match_id) DO NOTHING
`, matchID)
if result.RowsAffected() == 0 {
    // Another worker already claimed this match
    msg.Ack()
    return
}
```

4. **The queue is transport; Postgres is the source of truth.** Never determine "should I process this" from the queue — always check the database.

**Warning signs:**
- COUNT(*) on game_events table shows more events than expected per match.
- The same kill/round event appears multiple times with identical match_id and event_id.
- Debugging reveals mismatched message delivery vs. processing counts.

**Phase to address:**
Phase 2 (Download/Parse Services) — idempotency must be designed into the schema and the parsing pipeline from the start. Retrofitting idempotency requires data migration to de-duplicate, which is far more expensive than doing it right initially.

---

### Pitfall 7: Not Closing demoinfocs-golang Parser — Memory Leak of ~250 MB Per Demo

**What goes wrong:**
A parser is created for each demo, `ParseToEnd()` is called, results are extracted, but `p.Close()` is never called (or called only on the happy path). Each unclosed parser leaks ~250 MB of heap memory from the internal entity system, protocol buffers, and game state tracking. After parsing a few demos, the service OOMs even though it appears to be done with the work.

**Why it happens:**
demoinfocs-golang uses cgo and internal C memory for low-level parsing. The Go garbage collector cannot track or free C allocations. `p.Close()` explicitly releases these resources. The library's documentation doesn't prominently warn about this, and Go developers are conditioned to trust GC.

**How to avoid:**

```go
func parseDemo(reader io.Reader) (*ParseResult, error) {
    p := dem.NewParser(reader)
    defer p.Close()  // ALWAYS close, even on panic

    result := &ParseResult{}

    p.RegisterEventHandler(func(e events.Kill) {
        result.Kills = append(result.Kills, KillEvent{
            Killer: e.Killer.Name,
            Victim: e.Victim.Name,
            Weapon: e.Weapon.String(),
            Tick:   p.GameState().IngameTick(),
        })
    })

    err := p.ParseToEnd()
    if err != nil {
        return nil, fmt.Errorf("parse demo: %w", err)
    }
    return result, nil
}
```

Key rules:
- `defer p.Close()` immediately after `NewParser()`. Non-negotiable.
- Do not reuse parser instances across demos. Create new, Close, discard.
- For concurrent parsing, set GOMAXPROCS appropriate to memory available (each parser ~250MB + ~100MB for the demo file if buffered). Plan for peak concurrent parse count.

**Warning signs:**
- RSS grows steadily even after `ParseToEnd()` returns.
- `pprof` heap profile shows `C.malloc` or demoinfocs internal objects dominating allocations.
- Service restarts after OOM, but each run processes fewer demos before crashing.

**Phase to address:**
Phase 2 (Parse Service) — this is parser lifecycle management, specific to the parse service.

---

### Pitfall 8: The Demarcation Line — CLI Contract vs. Microservice Behavior

**What goes wrong:**
The existing v1.0 CLI has a hard JSON stdout contract: `{data, meta}` on stdout, structured errors on stderr. When adding microservices, developers mix these patterns: a service writes JSON to its own stdout for debugging, polluting the container logs; or a service uses the same error envelope but sends it to NATS, creating confusion about where errors come from.

Additionally, the existing `internal/hltv`, `internal/provider`, and `internal/domain` packages were designed for a CLI with synchronous, request-response semantics. If microservices import these directly, they inherit the CLI's HTTP timeout settings, fixture-test dependencies, and the assumption of single-command execution.

**Why it happens:**
The codebase already has 3,136 LOC of working, tested code. The natural instinct is to reuse it by importing existing packages. But these packages were designed for a different execution model (synchronous CLI). Blind reuse without adaptation creates coupling where the microservice depends on CLI-specific behavior.

**How to avoid:**

1. **Preserve the CLI contract as-is.** The `dem` binary is the CLI. Microservices are separate binaries in `cmd/` that share `internal/` packages but have their own execution contracts.

2. **Structured logging, not stdout JSON.** Microservices use `slog` (Go 1.21+ standard library) for all output. JSON logging to stdout is fine for container log aggregation, but the `{data, meta}` envelope is CLI-only.

3. **Create distinct infrastructure packages.** Keep `internal/hltv` and `internal/provider` as CLI dependencies. Add `internal/nats`, `internal/storage`, `internal/database` for microservice infrastructure. If the polling service needs HLTV access, it either imports `internal/hltv` consciously (with awareness of the HTTP model) or gets its own HTTP client with service-appropriate timeouts.

4. **Explicit adapter layer if reusing CLI packages in services:**

```go
// internal/nats/hltvclient/adapter.go
// Adapts internal/hltv.Client for long-running service use
type ServiceClient struct {
    cli *hltv.Client
}

func NewServiceClient(baseURL string) *ServiceClient {
    return &ServiceClient{
        cli: hltv.NewClient(
            hltv.WithHTTPClient(&http.Client{Timeout: 60 * time.Second}),
            hltv.WithUserAgent("DemMicroservice/1.0"),
        ),
    }
}
```

**Warning signs:**
- Microservice binaries import `internal/cli` or `cmd/dem` packages.
- Service log output contains `{"data": ..., "meta": ...}` envelopes (CLI contract leaked into services).
- Changing a CLI flag parser breaks a microservice.

**Phase to address:**
Phase 1 (Infrastructure Foundation) — the package structure decision (where do microservice packages live, what do they import) must be established before any service coding begins.

---

## Technical Debt Patterns

Shortcuts that seem reasonable but create long-term problems.

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Hardcoding Minio endpoint as `localhost:9000` | Works immediately in Docker Compose | Breaks in any non-local deployment; every service needs code change to reconfigure | Never — use env vars from day one |
| Using `replace` in `go.mod` to share internal packages | Avoids restructuring code | Docker builds break; CI complexity explodes; every new service adds another `replace` to maintain | Never for production services — use single root `go.mod` |
| Skipping `MaxDeliver` on NATS consumers | One less config field to think about | Poison message loops forever, saturating CPU and filling logs | Only during initial NATS exploration on a dev machine |
| Using `PublishAsync` without tracking the `PubAckFuture` | Slightly faster publishes (fire-and-forget) | Messages silently lost if NATS connection drops before persistence; no way to know which messages made it | Only for non-critical telemetry/analytics events where occasional loss is acceptable |
| Parsing demos into memory then batch-inserting everything | Simpler insert logic (one transaction) | If parsing crashes mid-demo, ALL events from that demo are lost (not just the unparsed remainder) | Never — insert in batches during parsing via event handlers |
| One `go.mod` per service in a monorepo | Services "feel independent" | Every shared library change requires N `go.mod` updates; diamond dependency hell; `go mod tidy` runs N times in CI | Only if you have dedicated tooling for cross-module version bumps and a private module proxy |
| `depends_on` without health checks in Docker Compose | Simpler compose file | Services start before Postgres/NATS/Minio accept connections; crash loops on startup | Only for local dev where you manually wait; never in CI or shared environments |

---

## Integration Gotchas

Common mistakes when connecting to external services.

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| **NATS / JetStream** | Not checking `js, err := jetstream.New(nc)` — panics on first publish if JetStream is not enabled | Validate JetStream context at startup; fail fast with clear error if NATS server lacks `-js` flag |
| **NATS / JetStream** | Publishing before creating the stream — silent data loss | Create all streams at service startup; add startup health check verifying stream existence |
| **NATS / JetStream** | Using legacy `nats.Conn.JetStream()` API mixed with new `jetstream` package | Use the `jetstream` package (v2 API) exclusively for all new development |
| **NATS / JetStream** | `FetchBatch` + checking `len(msgs.Messages())` immediately (channel may be empty) | Range over `msgs.Messages()` channel; never check length before consuming |
| **Minio / S3** | `endpoint` with `http://` prefix in minio-go v7 — connection fails silently | Endpoint is bare host:port (e.g., `minio:9000`). Set `Secure: false` explicitly for HTTP |
| **Minio / S3** | Creating a new `minio.Client` per request — FD exhaustion under load | Single global client instance with custom `http.Transport` (MaxIdleConns: 100, IdleConnTimeout: 90s) |
| **Minio / S3** | `PutObject` with entire file in `[]byte` — OOM on 100MB+ demos | Stream from `io.Reader` directly; set `PartSize: 128 * 1024 * 1024` for large files |
| **Minio / S3** | Not calling `object.Close()` after `GetObject` — connection pool leak | Always `defer object.Close()` immediately after `GetObject` |
| **Minio / S3** | AWS SDK v2 `aws-chunked` encoding produces 64 MiB chunks; Minio rejects >16 MiB | Set `RequestChecksumCalculation = WhenRequired` or use minio-go SDK instead of AWS SDK |
| **Minio / S3** | Presigned URL 403 errors — clock skew between container and Minio server | Ensure containers have NTP-synced clocks; use `time.Now().UTC()` for all signature operations |
| **Minio / S3** | `ListObjects` truncation — only first 1,000 objects returned | Use the channel-based API that handles pagination internally: `for obj := range client.ListObjects(...)` |
| **Postgres / pgx** | Creating `pgx.Connect()` per request instead of a pool | Create `pgxpool` once at startup; pass to handlers |
| **Postgres / pgx** | Default `MaxConns=4` in pgxpool | Explicitly set `MaxConns=10-25` depending on service concurrency needs |
| **Postgres / pgx** | Raw `BEGIN`/`COMMIT` SQL strings instead of `pool.Begin(ctx)` | Use `pool.Begin(ctx)` — it tracks the transaction lifecycle and returns connections cleanly to the pool |
| **Docker Compose** | `localhost` in connection strings — resolves to the container, not the host | Use Docker service names: `nats://nats:4222`, `postgres:5432`, `minio:9000` |
| **Docker Compose** | `depends_on` without `condition: service_healthy` | Add HEALTHCHECK to each Dockerfile; use `condition: service_healthy` in depends_on |
| **Docker Compose** | Rebuilding Go services in Docker for every code change during dev | Run Go services natively on host; use Docker Compose only for infrastructure (NATS, Minio, Postgres) |
| **demoinfocs-golang** | Not calling `p.Close()` after parsing — ~250 MB C-memory leak per demo | `defer p.Close()` immediately after `NewParser()` |
| **demoinfocs-golang** | Reusing parser instances across demos | Create new parser for each demo; Close after `ParseToEnd()` |
| **demoinfocs-golang** | Assuming CS2-only demos — CS:GO legacy demos need v3 module | Version-detect demo header before choosing parser version; or only support CS2 demos explicitly |
| **HLTV HTTP (existing)** | Reusing CLI HTTP timeouts (30s) for service polling (needs 60s+) | Create service-specific HTTP clients with appropriate timeouts; don't reuse CLI client config |

---

## Performance Traps

Patterns that work at small scale but fail as usage grows.

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Single NATS pull consumer for all demo downloads | Downloads queue up; throughput limited by one-at-a-time consumption | Use concurrent pull consumers with `MaxAckPending` tuned to worker count | >10 demos queued simultaneously |
| Not setting `MaxAckPending` explicitly | Default (1000) may overwhelm slow parsers; or too-low values throttle throughput | Tune per service: download service ~50, parse service ~5 (heavy work) | When demo volume increases to multiple per hour |
| Parsing demos with `ParseToEnd()` without streaming events to DB | 250 MB parser + 100 MB demo in memory, then massive batch INSERT at end — spike in both memory and DB write load | Register event handlers that insert in batches (every 100 events, flush to DB); or use `ParseNextFrame()` for chunked processing with intermediate commits | Demos >100 MB with >10,000 game events each |
| No `AckWait` tuning for long-running parse jobs | 30-second default `AckWait` expires while parsing a 60-minute demo — message redelivered mid-parse | Set `AckWait` to 2-3x the p99 parse time; use `msg.InProgress()` for jobs exceeding `AckWait`; or ack first, then parse (move ownership to DB) | Demos longer than ~20 minutes of gameplay |
| Inserting each game event individually | Thousands of round-trips per demo; Postgres connection saturation | Batch insert (100-500 events per INSERT); or use `pgx.CopyFrom` for bulk loading | >1,000 events per demo |
| No connection pool limits on Minio client | FD exhaustion from too many concurrent uploads/downloads | Configure `http.Transport.MaxIdleConnsPerHost: 10`; use a semaphore to limit concurrent Minio operations per service | >10 concurrent Minio operations per service |
| Running all services with full Docker Compose during development | 30+ second rebuild cycles on every code change; no debugger | Hybrid: Docker for infrastructure, `go run` on host for Go services | First day of active development |

---

## "Looks Done But Isn't" Checklist

Things that appear complete but are missing critical pieces.

- [ ] **NATS Stream Creation:** Often created manually via CLI during testing but not automated. Verify: does the service create streams programmatically at startup if they don't exist? Does it verify stream existence before publishing?
- [ ] **NATS Message Ack:** Often works on happy path but not on error/crash/panic. Verify: `defer msg.Ack()` in every handler? `MaxDeliver` set on every consumer? DLQ handling implemented?
- [ ] **Minio Connection:** Often works with default settings. Verify: custom `http.Transport` with idle connection pooling? `object.Close()` deferred everywhere? Streaming (not buffering) for all Put/Get operations?
- [ ] **Postgres Pool:** Often "works" with `pgx.Connect` in a prototype. Verify: using `pgxpool.Pool`? `MaxConns` explicitly set? Pool created once at startup, not per-request?
- [ ] **Docker Multi-Stage Builds:** Often works in dev but produces 800MB images. Verify: multi-stage build? `CGO_ENABLED=0` for static binary? Final stage uses `gcr.io/distroless/static` or `scratch`? Image size <20 MB?
- [ ] **Demo Parser Lifecycle:** Often works on one demo. Verify: `defer p.Close()` on every parser? New parser per demo? `ParseToEnd()` errors checked?
- [ ] **Service Health Checks:** Often "the service starts" but no health endpoint. Verify: does each service expose a health check that validates its dependencies (NATS connected, Postgres reachable, Minio accessible)? Does the Dockerfile have `HEALTHCHECK`?
- [ ] **Graceful Shutdown:** Often missing (kill -9 works, right?). Verify: on SIGTERM, does the service drain NATS subscriptions, finish in-flight work, close DB pool, and exit cleanly? Timeout for forced shutdown if graceful exceeds limit?
- [ ] **Idempotency:** Often "it works if nothing crashes." Verify: can the same demo be submitted for parsing twice without duplicate events? Do deterministic event IDs exist? Is `ON CONFLICT DO NOTHING` used on all insert paths?
- [ ] **Environment Configuration:** Often hardcoded for local dev. Verify: all connection strings from env vars? Reasonable defaults for dev? Validation at startup with clear error messages for missing required vars?
- [ ] **Logging:** Often `fmt.Println` or `log.Println` scattered everywhere. Verify: structured logging (`log/slog`) with consistent keys? Log levels respected? No `{data, meta}` CLI envelopes in service output?
- [ ] **Concurrent Parsing Safety:** Often "it worked with one demo." Verify: is the demoinfocs parser instance per-goroutine (not shared)? Are game event handlers thread-safe? Is the DB pool shared safely? What happens when 3 demos are parsed concurrently?

---

## Recovery Strategies

When pitfalls occur despite prevention, how to recover.

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Messages published without a backing stream | MEDIUM | 1. Create the stream. 2. Re-publish messages from source of truth (HLTV polling data). 3. Add startup stream verification to prevent recurrence. Lost messages during the gap are unrecoverable unless you can re-poll HLTV. |
| Missing `Ack` causing redelivery loops | LOW | 1. Fix the handler to `defer msg.Ack()`. 2. Restart the consumer. 3. Run deduplication query on Postgres to remove duplicate events. Cost: a few duplicate rows, cleaned up with a one-off DELETE. |
| Docker build failures from `go.mod` replace directives | HIGH | 1. Migrate to single root `go.mod`. 2. Update all import paths. 3. Update all Dockerfiles to use repo-root context. 4. Test all builds. Cost: significant refactor, but a one-time cost that prevents perpetual build issues. |
| OOM from buffering 100MB+ demo files in memory | MEDIUM | 1. Refactor download/parse to stream. 2. Add `http.MaxBytesReader` guards. 3. Add memory limits to Docker Compose services. Cost: moderate code change to the download/parse pipeline. Existing data in Minio is fine (it was stored correctly; the issue was in-memory buffering). |
| pgx connection exhaustion | LOW | 1. Switch from `pgx.Connect` to `pgxpool`. 2. Set explicit `MaxConns`. 3. Add connection timeout. Cost: swapping connection creation calls. Data is intact. |
| Duplicate game events from non-idempotent parsing | MEDIUM | 1. Add deterministic event IDs to the schema. 2. Write deduplication migration (DELETE duplicates keeping first by created_at). 3. Add `ON CONFLICT DO NOTHING` to insert paths. Cost: one-off data migration; no permanent data loss. |
| demoinfocs-golang memory leak (missing Close) | LOW | 1. Add `defer p.Close()`. 2. Restart service. Cost: code fix only. The leak is per-process; restarting reclaims memory. |
| Clock skew causing Minio presigned URL failures | LOW | 1. Add NTP sync to containers. 2. Use `time.Now().UTC()` for all timestamps. 3. Regenerate any affected URLs. Cost: operational fix, no code change needed. |

---

## Pitfall-to-Phase Mapping

How roadmap phases should address these pitfalls.

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| 1 — Silent message loss (no stream) | Phase 1: Infrastructure Foundation | Startup health check: verify all expected streams exist before accepting work |
| 2 — Missing ACK / redelivery loops | Phase 2: Download/Parse Services | Code review: every handler has `defer msg.Ack()` at top; `MaxDeliver` set on every consumer |
| 3 — Docker build context + go.mod replace | Phase 1: Infrastructure Foundation | `docker compose build` succeeds for all services from clean checkout |
| 4 — OOM from buffering .dem files | Phase 2: Download/Parse Services | Memory profiling: parse 100MB demo, verify RSS < 300 MB total |
| 5 — pgx.Conn instead of pgxpool | Phase 1: Infrastructure Foundation | Code review: all services use `pgxpool.Pool` created once at startup |
| 6 — Duplicate game events (no idempotency) | Phase 2: Parse Service | Integration test: submit same match_id twice, verify COUNT(DISTINCT event_id) = expected |
| 7 — demoinfocs Close leak | Phase 2: Parse Service | Code review: `defer p.Close()` after every `NewParser()`. Heap profile after 10 parses shows no growth |
| 8 — CLI contract leaked into services | Phase 1: Infrastructure Foundation | Service stdout/stderr contains only structured log output, no `{data, meta}` envelopes |

---

## Sources

### NATS / JetStream
- nats.io Go client discussions: [PublishAsync failure scenarios](https://github.com/nats-io/nats.go/discussions/1184), [legacy vs new JetStream API](https://github.com/nats-io/nats.go/discussions/1634)
- NATS maintainers: [CRE-2025-0082 — JetStream HA failure vectors](http://docs.prequel.dev/cres/public/cre-2025-0082) (MaxAckPending + LastPerSubject deadlock, JetStream before ReadyForConnections)
- nats.go issues: [Ack after consumer delete silent failure (#1793)](https://github.com/nats-io/nats.go/issues/1793), [AckWait + dead letter queue (#4994)](https://github.com/nats-io/nats-server/discussions/4994)
- Worker heartbeat deep dive: [slepp.ca — ownership vs. delivery](https://slepp.ca/2026/02/01/deep-dive-worker-heartbeat/)
- Community wisdom: [slow consumer patterns (#888)](https://github.com/nats-io/nats.go/discussions/888), [question in regards to slow consumer (#1950)](https://github.com/nats-io/nats.go/discussions/1950), [WorkQueuePolicy overlapping subjects (#3639)](https://github.com/nats-io/nats-server/issues/3639)

### Minio / S3
- minio/minio-go: [multipart upload issues (#1025)](https://github.com/minio/minio-go/issues/1025)
- minio/minio: [aws-chunked 16 MiB limit (#21611)](https://github.com/minio/minio/issues/21611)
- Stack Overflow: [GetObject multipart download behavior](https://stackoverflow.com/questions/72394248)
- Community: [Minio S3 Go tutorials, large file practices](https://www.php.cn/faq/2191605.html)

### demoinfocs-golang
- DeepWiki: [architecture, performance benchmarks, memory usage (~250 MB/demo)](https://deepwiki.com/markus-wa/demoinfocs-golang)
- pkg.go.dev: [v5 documentation](https://pkg.go.dev/github.com/markus-wa/demoinfocs-golang/v5)
- Performance: ~25 min gameplay/sec, ~900k allocations/demo, concurrent: 1h25m gameplay/sec (8 demos)

### Go + PostgreSQL / pgx
- jackc/pgx maintainer: [pool creation pattern (#1989)](https://github.com/jackc/pgx/discussions/1989)
- quantmHQ: [pgx.Conn concurrency safety bug (#210)](https://github.com/quantmHQ/quantm/issues/210)
- DBA StackExchange: [pgxpool + PgBouncer interaction](https://dba.stackexchange.com/q/346183)
- Production guides: [Go with PostgreSQL best practices](https://dev.to/mx_tech/go-with-postgresql-best-practices-for-performance-and-safety-47d7), [Kite Metric guide](https://kitemetric.com/blogs/go-with-postgresql-best-practices-for-performance-and-safety)

### Docker Compose Monorepo
- Stack Overflow: [Deploying Go services from monorepo](https://stackoverflow.com/questions/76742716)
- docker/compose: [build context restrictions (#11594)](https://github.com/docker/compose/issues/11594)
- OneUptime: [Monorepo Docker structure](https://oneuptime.com/blog/post/2026-02-08-how-to-structure-a-monorepo-with-docker)
- helpwave/services: [go mod replace breaks Docker build (#25)](https://github.com/helpwave/services/issues/25)
- Archbee: [Dependency handling: monorepo vs multirepo](https://blog.devgenius.io/dependency-handling-for-microservices-monorepo-vs-multirepo-c10ffd1ee970)

### Go Large File Streaming
- Production guides: [Large file download streaming](https://www.php.cn/faq/1515077.html), [HTTP client large files](https://www.php.cn/faq/1763469.html)
- dev.to: [Streaming large files between microservices](https://dev.to/dialaeke/streaming-large-files-between-microservices-a-go-implementation-4n62)

### General Microservice Platform Patterns
- go-coffeeshop: [Real-world Go microservice demo](https://github.com/thangchung/go-coffeeshop) — Docker Compose hybrid pattern
- servicepack: [Go framework for monorepo microservices](https://ciprian.51k.eu/servicepack-the-go-framework-that-actually-understands-how-real-development-works/)
- Corgi: [CLI for local microservice management](https://dev.to/andriiklymiuk/corgi-the-cli-that-tames-your-local-microservices-chaos-45nd)

### Message Broker Reliability
- devbytes: [Message brokers can lose persistent messages](https://devbytes.co.in/news/message-brokers-can-lose-persistent-messages-heres-why)
- Community patterns: [Message acknowledgement with external subsystems](https://github.com/nats-io/nats.go/discussions/907)

---
*Pitfalls research for: HLTV CLI v1.1 — Microservice Platform*
*Researched: 2026-05-03*

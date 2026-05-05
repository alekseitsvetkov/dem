package poller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/robfig/cron/v3"

	"github.com/alekseitsvetkov/dem/internal/domain"
	"github.com/alekseitsvetkov/dem/internal/hltv"
	"github.com/alekseitsvetkov/dem/internal/hltv/parser"
	"github.com/alekseitsvetkov/dem/internal/service"
	"github.com/alekseitsvetkov/dem/pkg/natsutil"
)

// Compile-time interface check: PollerService implements service.Service.
var _ service.Service = (*PollerService)(nil)

// tier1Keywords are event name keywords that identify Tier 1 tournaments.
// Replicated from v1.0 internal/provider/events.go per CROS-03 (do not import provider/).
var tier1Keywords = []string{
	"IEM",
	"PGL",
	"Blast",
	"StarLadder",
	"FISSURE",
	"Esports World Cup",
	"Major",
	"BetBoom",
}

// DownloadJobPayload is the JSON payload published to NATS dem.download.jobs.
type DownloadJobPayload struct {
	MatchID      string `json:"match_id"`
	DemoURL      string `json:"demo_url"`
	MatchURL     string `json:"match_url"`
	EventName    string `json:"event_name"`
	Team1        string `json:"team1"`
	Team2        string `json:"team2"`
	MatchDate    string `json:"match_date"`
	DiscoveredAt string `json:"discovered_at"`
}

// PollerService discovers Tier 1 HLTV matches with available demos,
// deduplicates them against the processed_matches table, and publishes
// download jobs to NATS dem.download.jobs. It runs on a configurable
// cron schedule with a minimum interval guard.
type PollerService struct {
	cfg    Config
	logger *slog.Logger
	js     jetstream.JetStream
	pool   *pgxpool.Pool
	client *hltv.Client
	urls   hltv.URLs

	mu      sync.Mutex
	lastRun time.Time
}

// PollerOption is a functional option for configuring PollerService.
// Per CROS-02: all external dependencies are injected via functional options.
type PollerOption func(*PollerService)

// NewPollerService creates a new PollerService with the given Config and options.
func NewPollerService(cfg Config, opts ...PollerOption) *PollerService {
	p := &PollerService{
		cfg:    cfg,
		urls:   hltv.NewURLs(cfg.HLTVBaseURL),
		logger: slog.Default(),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// WithNATS injects the JetStream context for publishing download jobs.
func WithNATS(js jetstream.JetStream) PollerOption {
	return func(p *PollerService) {
		p.js = js
	}
}

// WithPostgres injects the pgxpool connection pool for dedup queries.
func WithPostgres(pool *pgxpool.Pool) PollerOption {
	return func(p *PollerService) {
		p.pool = pool
	}
}

// WithHLTVClient injects the HLTV HTTP client for fetching pages.
func WithHLTVClient(client *hltv.Client) PollerOption {
	return func(p *PollerService) {
		p.client = client
	}
}

// WithLogger injects the structured logger.
func WithLogger(logger *slog.Logger) PollerOption {
	return func(p *PollerService) {
		p.logger = logger
	}
}

// Run starts the cron scheduler and blocks until ctx is cancelled.
// Implements service.Service.
func (p *PollerService) Run(ctx context.Context) error {
	// Ensure JetStream streams exist before publishing (idempotent).
	if err := natsutil.CreateStreams(ctx, p.js); err != nil {
		return fmt.Errorf("create streams: %w", err)
	}

	p.logger.Info("poller service starting",
		slog.String("cron", p.cfg.CronExpression),
		slog.Duration("min_interval", p.cfg.MinInterval),
		slog.Bool("oneshot", p.cfg.OneShot),
	)

	// One-shot mode: run once immediately and return.
	if p.cfg.OneShot {
		p.poll(ctx)
		return nil
	}

	c := cron.New(cron.WithLocation(time.UTC))
	_, err := c.AddFunc(p.cfg.CronExpression, func() {
		pollCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
		p.poll(pollCtx)
	})
	if err != nil {
		return err
	}

	c.Start()
	p.logger.Info("poller cron scheduler started")

	<-ctx.Done()
	p.logger.Info("poller service stopping")

	cronCtx := c.Stop()
	<-cronCtx.Done()
	p.logger.Info("poller cron scheduler stopped")

	return nil
}

// isTooSoon checks if the minimum interval has not elapsed since the last successful run.
// Protected by p.mu.
func (p *PollerService) isTooSoon() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.lastRun.IsZero() {
		return false
	}
	return time.Since(p.lastRun) < p.cfg.MinInterval
}

// markRun records the current time as the last successful run start.
// Protected by p.mu.
func (p *PollerService) markRun() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastRun = time.Now()
}

// poll performs one full discovery cycle: fetch events, filter Tier 1,
// fetch results per event, fetch demo links per match, dedup, and publish.
func (p *PollerService) poll(ctx context.Context) {
	if p.isTooSoon() {
		p.logger.Warn("skipping poll — minimum interval not elapsed",
			slog.Duration("min_interval", p.cfg.MinInterval),
		)
		return
	}
	p.markRun()

	p.logger.Info("starting poll cycle")

	// Step 1: Fetch and parse events
	eventsURL := p.urls.EventsURL()
	body, err := p.client.Fetch(ctx, eventsURL)
	if err != nil {
		p.logger.Error("failed to fetch events page",
			slog.String("url", eventsURL),
			slog.String("error", err.Error()),
		)
		return
	}

	events, err := parser.ParseEvents(bytes.NewReader(body), eventsURL)
	if err != nil {
		p.logger.Error("failed to parse events page",
			slog.String("error", err.Error()),
		)
		return
	}

	p.logger.Info("fetched events", slog.Int("total", len(events)))

	// Step 2: Filter to Tier 1
	tier1Events := filterTier1(events)
	p.logger.Info("filtered to Tier 1", slog.Int("count", len(tier1Events)))

	// Step 3: For each Tier 1 event, fetch results
	for _, event := range tier1Events {
		eventID, err := strconv.Atoi(event.ID)
		if err != nil {
			p.logger.Warn("skipping event with non-numeric ID",
		slog.String("event_id", event.ID),
		slog.String("event_name", event.Name),
			)
			continue
		}

		resultsURL := p.urls.ResultsURLForEvent(eventID)
		body, err := p.client.Fetch(ctx, resultsURL)
		if err != nil {
			p.logger.Warn("failed to fetch results for event",
		slog.Int("event_id", eventID),
		slog.String("event_name", event.Name),
		slog.String("error", err.Error()),
			)
			continue
		}

		results, err := parser.ParseResults(bytes.NewReader(body), resultsURL)
		if err != nil {
			p.logger.Warn("failed to parse results for event",
		slog.Int("event_id", eventID),
		slog.String("event_name", event.Name),
		slog.String("error", err.Error()),
			)
			continue
		}

		p.logger.Info("fetched results for event",
			slog.Int("event_id", eventID),
			slog.String("event_name", event.Name),
			slog.Int("result_count", len(results)),
		)

		// Step 4: For each result, check demo availability, dedup, and publish
		for _, result := range results {
			matchIDInt, err := strconv.ParseInt(result.MatchID, 10, 64)
			if err != nil {
		p.logger.Warn("skipping result with non-numeric match ID",
			slog.String("match_id", result.MatchID),
			slog.String("event_name", event.Name),
		)
		continue
			}

			matchURL := p.urls.MatchURL(int(matchIDInt))
		// Rate limit: HLTV blocks rapid requests. 2s between match page fetches.
		time.Sleep(2 * time.Second)

			body, err := p.client.Fetch(ctx, matchURL)
			if err != nil {
		p.logger.Warn("failed to fetch match page",
			slog.Int64("match_id", matchIDInt),
			slog.String("error", err.Error()),
		)
		continue
			}

			demoLink, err := parser.ParseDemoLink(bytes.NewReader(body), matchURL)
			if err != nil {
		var pe *parser.ParseError
		if errors.As(err, &pe) && pe.Code == parser.ErrorCodeUnavailableData {
			// No demo available — expected for many matches, skip silently.
			continue
		}
		p.logger.Warn("failed to parse demo link",
			slog.Int64("match_id", matchIDInt),
			slog.String("error", err.Error()),
		)
		continue
			}

			if demoLink.DemoURL == "" {
		continue
			}

			// Dedup: INSERT INTO processed_matches ON CONFLICT DO NOTHING.
			// Per D-02: RowsAffected() == 0 means this match was already processed.
			tag, err := p.pool.Exec(ctx,
		"INSERT INTO processed_matches (match_id) VALUES ($1) ON CONFLICT DO NOTHING",
		matchIDInt,
			)
			if err != nil {
		p.logger.Error("failed to insert processed_matches",
			slog.Int64("match_id", matchIDInt),
			slog.String("error", err.Error()),
		)
		continue
			}
			if tag.RowsAffected() == 0 {
		// Already processed (D-02).
		continue
			}

			// Publish download job to NATS dem.download.jobs.
			payload := DownloadJobPayload{
		MatchID:      result.MatchID,
		DemoURL:      demoLink.DemoURL,
		MatchURL:     demoLink.MatchURL,
		EventName:    event.Name,
		Team1:        result.Team1,
		Team2:        result.Team2,
		MatchDate:    result.Date,
		DiscoveredAt: time.Now().UTC().Format(time.RFC3339),
			}

			jsonBytes, err := json.Marshal(payload)
			if err != nil {
		p.logger.Error("failed to marshal download job payload",
			slog.Int64("match_id", matchIDInt),
			slog.String("error", err.Error()),
		)
		continue
			}

			_, err = p.js.PublishMsg(ctx, &nats.Msg{
		Subject: natsutil.SubjectDownload,
		Data:    jsonBytes,
			})
			if err != nil {
		p.logger.Error("failed to publish download job",
			slog.Int64("match_id", matchIDInt),
			slog.String("subject", natsutil.SubjectDownload),
			slog.String("error", err.Error()),
		)
		continue
			}

			p.logger.Info("published download job",
		slog.String("match_id", result.MatchID),
		slog.String("event", event.Name),
			)
		}
	}

	p.logger.Info("poll cycle complete")
}

// filterTier1 returns events matching the Tier 1 heuristic:
// prize pool > $250,000 OR name contains a known Tier 1 keyword.
// Replicated from v1.0 internal/provider/events.go per CROS-03.
func filterTier1(events []domain.Event) []domain.Event {
	var filtered []domain.Event
	for _, e := range events {
		if e.PrizePool > 250_000 || matchesKeyword(e.Name) {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// matchesKeyword checks if the event name contains any known Tier 1 keyword
// (case-insensitive). Replicated from v1.0 internal/provider/events.go per CROS-03.
func matchesKeyword(name string) bool {
	upper := strings.ToUpper(name)
	for _, kw := range tier1Keywords {
		if strings.Contains(upper, strings.ToUpper(kw)) {
			return true
		}
	}
	return false
}

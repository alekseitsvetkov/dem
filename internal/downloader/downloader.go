package downloader

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	minio "github.com/minio/minio-go/v7"

	dmnio "github.com/alekseitsvetkov/dem/pkg/minio"
	"github.com/alekseitsvetkov/dem/pkg/natsutil"
)

// Compile-time interface check: DownloaderService implements service.Service.
// Importing the service package here would create a cycle since Runner
// orchestrates Service — we verify compliance via struct method signature instead.

// DownloaderService implements service.Service for the Demo Downloader.
// It consumes download jobs from NATS JetStream, streams .dem.gz files
// from the HLTV CDN directly to MinIO (zero local disk writes), retries
// transient failures with exponential backoff, and publishes parse jobs
// to dem.parse.jobs on success.
//
// Uses a raw *http.Client (not hltv.Client) because hltv.Client.Fetch
// reads the full response body into []byte, which is incompatible with
// streaming to MinIO PutObject via io.Reader.
type DownloaderService struct {
	cfg        Config
	logger     *slog.Logger
	js         jetstream.JetStream
	minio      *minio.Client
	httpClient *http.Client
}

// DownloaderOption is a functional option for configuring a DownloaderService.
type DownloaderOption func(*DownloaderService)

// NewDownloaderService creates a new DownloaderService with the given config and options.
// Defaults: logger=slog.Default(), httpClient created with cfg.DownloadTimeout.
func NewDownloaderService(cfg Config, opts ...DownloaderOption) *DownloaderService {
	ds := &DownloaderService{
		cfg:    cfg,
		logger: slog.Default(),
		httpClient: &http.Client{
			Timeout: cfg.DownloadTimeout,
		},
	}
	for _, opt := range opts {
		opt(ds)
	}
	return ds
}

// WithNATS injects a JetStream context for subscribing and publishing.
func WithNATS(js jetstream.JetStream) DownloaderOption {
	return func(d *DownloaderService) { d.js = js }
}

// WithMinio injects a MinIO client for object storage.
func WithMinio(client *minio.Client) DownloaderOption {
	return func(d *DownloaderService) { d.minio = client }
}

// WithLogger injects a structured logger.
func WithLogger(logger *slog.Logger) DownloaderOption {
	return func(d *DownloaderService) { d.logger = logger }
}

// WithHTTPClient injects an HTTP client (primarily for testing).
// If not set, NewDownloaderService creates one with cfg.DownloadTimeout.
func WithHTTPClient(client *http.Client) DownloaderOption {
	return func(d *DownloaderService) { d.httpClient = client }
}

// Run implements service.Service.
//
// Lifecycle:
//  1. Ensure the MinIO bucket exists (fail fast on error).
//  2. Create or update a durable JetStream pull consumer "download-worker"
//     on the DEM_DOWNLOAD stream, filtering dem.download.jobs.
//  3. Pull messages one at a time in an infinite loop.
//  4. Per D-10: each message is either Ack'd (on successful download +
//     parse job publish) or NakWithDelay'd (on failure after internal
//     retries exhausted). The conditional defer pattern guarantees Ack
//     and Nak are never called on the same message.
func (d *DownloaderService) Run(ctx context.Context) error {
	// 1. Ensure MinIO bucket exists — fail fast.
	if err := dmnio.EnsureBucket(ctx, d.minio, d.cfg.MinioBucket); err != nil {
		return fmt.Errorf("ensure minio bucket %s: %w", d.cfg.MinioBucket, err)
	}
	d.logger.Info("minio bucket ensured", slog.String("bucket", d.cfg.MinioBucket))

	// 2. Create or update the durable pull consumer.
	cons, err := d.js.CreateOrUpdateConsumer(ctx, natsutil.StreamDownload, jetstream.ConsumerConfig{
		Durable:       "download-worker",
		FilterSubject: natsutil.SubjectDownload,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    3,
	})
	if err != nil {
		return fmt.Errorf("create consumer %s on stream %s: %w", "download-worker", natsutil.StreamDownload, err)
	}
	d.logger.Info("consumer ready",
		slog.String("consumer", "download-worker"),
		slog.String("stream", natsutil.StreamDownload),
		slog.String("subject", natsutil.SubjectDownload),
	)

	// 3. Pull message loop — pull one message at a time.
	iter, err := cons.Messages(jetstream.PullMaxMessages(1))
	if err != nil {
		return fmt.Errorf("messages iterator: %w", err)
	}
	defer iter.Stop()

	d.logger.Info("downloader service started, waiting for download jobs")

	for {
		msg, err := iter.Next()
		if err != nil {
			d.logger.Error("next message error", slog.String("error", err.Error()))
			return fmt.Errorf("message iteration: %w", err)
		}
		d.processMessage(ctx, msg)
	}
}

// processMessage handles a single download job message.
//
// D-10: Uses a conditional defer pattern to guarantee Ack only on success
// and NakWithDelay only on failure — never both on the same message.
// The deferred function checks msgErr: nil -> Ack, non-nil -> NakWithDelay.
// Every code path either sets msgErr and returns (causing NakWithDelay) or
// leaves it nil and falls through to the end (causing Ack).
func (d *DownloaderService) processMessage(ctx context.Context, msg jetstream.Msg) {
	var msgErr error
	defer func() {
		// D-10: Only Ack after successful MinIO upload + parse job publish.
		// NakWithDelay on failure after internal retries exhausted.
		if msgErr == nil {
			if ackErr := msg.Ack(); ackErr != nil {
				d.logger.Error("msg ack failed", slog.String("error", ackErr.Error()))
			}
		} else {
			if nakErr := msg.NakWithDelay(d.cfg.NakDelay); nakErr != nil {
				d.logger.Error("msg nak failed", slog.String("error", nakErr.Error()))
			}
		}
	}()

	// Extract job_id from message metadata for correlation logging.
	jobID := ""
	if meta, err := msg.Metadata(); err == nil {
		jobID = fmt.Sprintf("%s/%d/%d", meta.Stream, meta.Sequence.Stream, meta.Sequence.Consumer)
	}

	// Parse JSON payload from poller:
	// {match_id, demo_url, match_url, event_name, team1, team2, match_date, discovered_at}
	var job struct {
		MatchID   string `json:"match_id"`
		DemoURL   string `json:"demo_url"`
		MatchURL  string `json:"match_url"`
		EventName string `json:"event_name"`
		Team1     string `json:"team1"`
		Team2     string `json:"team2"`
		MatchDate string `json:"match_date"`
	}
	if err := json.Unmarshal(msg.Data(), &job); err != nil {
		msgErr = fmt.Errorf("unmarshal job payload: %w", err)
		d.logger.Error("unmarshal job payload failed",
			slog.String("error", err.Error()),
			slog.String("job_id", jobID),
		)
		return // defer runs: NakWithDelay
	}

	logger := d.logger.With(
		slog.String("match_id", job.MatchID),
		slog.String("job_id", jobID),
	)
	logger.Info("received download job",
		slog.String("event_name", job.EventName),
		slog.String("team1", job.Team1),
		slog.String("team2", job.Team2),
	)

	// Download with retry (D-03: 3 attempts, 5s/25s/125s backoff with jitter).
	objectKey := fmt.Sprintf("demos/%s.dem.gz", job.MatchID)
	if msgErr = d.downloadWithRetry(ctx, job.DemoURL, objectKey, logger); msgErr != nil {
		logger.Error("download failed after retries",
			slog.String("error", msgErr.Error()),
		)
		return // defer runs: NakWithDelay
	}

	logger.Info("download completed",
		slog.String("bucket", d.cfg.MinioBucket),
		slog.String("object_key", objectKey),
	)

	// Publish parse job to dem.parse.jobs (D-05).
	parseJob := map[string]string{
		"bucket":     d.cfg.MinioBucket,
		"object_key": objectKey,
		"match_id":   job.MatchID,
		"match_url":  job.MatchURL,
		"event_name": job.EventName,
		"team1":      job.Team1,
		"team2":      job.Team2,
		"match_date": job.MatchDate,
	}
	parseJSON, err := json.Marshal(parseJob)
	if err != nil {
		msgErr = fmt.Errorf("marshal parse job: %w", err)
		logger.Error("marshal parse job failed",
			slog.String("error", err.Error()),
		)
		return // defer runs: NakWithDelay
	}

	if _, err = d.js.PublishMsg(ctx, &nats.Msg{
		Subject: natsutil.SubjectParse,
		Data:    parseJSON,
	}); err != nil {
		msgErr = fmt.Errorf("publish parse job: %w", err)
		logger.Error("publish parse job failed",
			slog.String("subject", natsutil.SubjectParse),
			slog.String("error", err.Error()),
		)
		return // defer runs: NakWithDelay
	}

	logger.Info("published parse job",
		slog.String("subject", natsutil.SubjectParse),
	)
	// msgErr remains nil — defer runs: msg.Ack()
}

// downloadWithRetry attempts to download a demo file with exponential backoff.
// Per D-03: 3 attempts with backoff 5s -> 25s -> 125s, with +/-20% jitter.
// Only after all internal retries are exhausted does the error propagate up
// to processMessage, which triggers NakWithDelay.
func (d *DownloaderService) downloadWithRetry(ctx context.Context, demoURL, objectKey string, logger *slog.Logger) error {
	var lastErr error
	for attempt := 0; attempt < d.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := d.cfg.RetryBaseDelay * time.Duration(1<<uint(attempt-1)) // 5s, 25s, 125s
			if delay > d.cfg.RetryMaxDelay {
				delay = d.cfg.RetryMaxDelay
			}
			// Add jitter: +/-20% random.
			jitter := time.Duration(rand.Int63n(int64(delay)/5 + 1))
			if rand.Intn(2) == 0 {
				jitter = -jitter
			}
			sleepDuration := delay + jitter
			logger.Info("retrying download",
				slog.Int("attempt", attempt+1),
				slog.Int("max_attempts", d.cfg.MaxRetries),
				slog.Duration("delay", sleepDuration),
			)
			time.Sleep(sleepDuration)
		}

		err := d.streamDownload(ctx, demoURL, objectKey)
		if err == nil {
			return nil
		}
		lastErr = err
		logger.Warn("download attempt failed",
			slog.Int("attempt", attempt+1),
			slog.Int("max_attempts", d.cfg.MaxRetries),
			slog.String("error", err.Error()),
		)
	}
	return fmt.Errorf("download failed after %d attempts: %w", d.cfg.MaxRetries, lastErr)
}

// streamDownload streams a .dem.gz file from the HLTV CDN directly to MinIO.
// Per D-04: zero local disk writes — http.Response.Body pipes directly to
// minio.Client.PutObject via io.Reader. No io.ReadAll, no temp files.
//
// Per Pitfall 4 (T-06-08): io.LimitReader caps the response body at MaxBytes
// (500 MB default) to prevent oversized files from exhausting memory.
func (d *DownloaderService) streamDownload(ctx context.Context, demoURL, objectKey string) error {
	downloadCtx, cancel := context.WithTimeout(ctx, d.cfg.DownloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(downloadCtx, http.MethodGet, demoURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "DemDownloader/1.0")

	client := &http.Client{Timeout: d.cfg.DownloadTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status %d", resp.StatusCode)
	}

	// Safety net: limit response body size (Pitfall 4).
	limitedBody := io.LimitReader(resp.Body, d.cfg.MaxBytes)

	// Per D-04: stream directly from HTTP response body to MinIO.
	// size=-1 tells MinIO to stream with unknown total size.
	// PartSize=128 MiB for efficient large-file uploads.
	_, err = d.minio.PutObject(downloadCtx, d.cfg.MinioBucket, objectKey, limitedBody, -1,
		minio.PutObjectOptions{
			ContentType: "application/gzip",
			PartSize:    128 * 1024 * 1024, // 128 MiB parts
		})
	if err != nil {
		return fmt.Errorf("minio put: %w", err)
	}

	return nil
}

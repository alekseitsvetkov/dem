package natsutil

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// Stream names and subject patterns.
const (
	// StreamDownload is the JetStream stream for demo download jobs.
	StreamDownload = "DEM_DOWNLOAD"

	// StreamParse is the JetStream stream for demo parse jobs.
	StreamParse = "DEM_PARSE"

	// SubjectDownload is the subject for download job messages.
	SubjectDownload = "dem.download.jobs"

	// SubjectParse is the subject for parse job messages.
	SubjectParse = "dem.parse.jobs"
)

// streamConfigs returns the canonical stream configurations for this platform.
// Per D-05: each stream uses WorkQueue retention, file storage, MaxAge 7 days.
func streamConfigs() []jetstream.StreamConfig {
	return []jetstream.StreamConfig{
		{
			Name:      StreamDownload,
			Subjects:  []string{SubjectDownload},
			Retention: jetstream.WorkQueuePolicy,
			Storage:   jetstream.FileStorage,
			MaxAge:    7 * 24 * time.Hour,
			Discard:   jetstream.DiscardOld,
		},
		{
			Name:      StreamParse,
			Subjects:  []string{SubjectParse},
			Retention: jetstream.WorkQueuePolicy,
			Storage:   jetstream.FileStorage,
			MaxAge:    7 * 24 * time.Hour,
			Discard:   jetstream.DiscardOld,
		},
	}
}

// CreateStreams creates all required JetStream streams.
// Per D-06: streams are created programmatically at startup.
// Per Pitfall 1: this MUST be called before any publisher starts.
// If a stream already exists, CreateStream is a no-op (idempotent).
func CreateStreams(ctx context.Context, js jetstream.JetStream) error {
	for _, cfg := range streamConfigs() {
		_, err := js.CreateStream(ctx, cfg)
		if err != nil {
			return fmt.Errorf("create stream %s: %w", cfg.Name, err)
		}
	}
	return nil
}

// VerifyStreams checks that all required JetStream streams exist.
// Per Pitfall 1: call this at startup to fail fast with a clear error
// if a required stream was not created.
func VerifyStreams(ctx context.Context, js jetstream.JetStream) error {
	for _, cfg := range streamConfigs() {
		_, err := js.Stream(ctx, cfg.Name)
		if err != nil {
			return fmt.Errorf("required stream %s not found: %w", cfg.Name, err)
		}
	}
	return nil
}

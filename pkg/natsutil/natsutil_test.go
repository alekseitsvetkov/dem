package natsutil

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.URL != "nats://localhost:4222" {
		t.Errorf("expected default URL nats://localhost:4222, got %s", cfg.URL)
	}
	if cfg.Timeout != 5*time.Second {
		t.Errorf("expected default timeout 5s, got %v", cfg.Timeout)
	}
	if cfg.MaxReconnects != 10 {
		t.Errorf("expected default MaxReconnects 10, got %d", cfg.MaxReconnects)
	}
}

func TestOptions(t *testing.T) {
	cfg := DefaultConfig()
	WithURL("nats://example.com:4222")(&cfg)
	WithTimeout(10 * time.Second)(&cfg)
	WithMaxReconnects(5)(&cfg)
	WithReconnectWait(3 * time.Second)(&cfg)
	WithName("test-service")(&cfg)

	if cfg.URL != "nats://example.com:4222" {
		t.Errorf("URL = %s", cfg.URL)
	}
	if cfg.Timeout != 10*time.Second {
		t.Errorf("Timeout = %v", cfg.Timeout)
	}
	if cfg.MaxReconnects != 5 {
		t.Errorf("MaxReconnects = %d", cfg.MaxReconnects)
	}
	if cfg.ReconnectWait != 3*time.Second {
		t.Errorf("ReconnectWait = %v", cfg.ReconnectWait)
	}
	if cfg.Name != "test-service" {
		t.Errorf("Name = %s", cfg.Name)
	}
}

func TestStreamConfigs(t *testing.T) {
	configs := streamConfigs()
	if len(configs) != 2 {
		t.Fatalf("expected 2 stream configs, got %d", len(configs))
	}
	names := map[string]bool{}
	for _, c := range configs {
		names[c.Name] = true
	}
	if !names["DEM_DOWNLOAD"] {
		t.Error("missing DEM_DOWNLOAD stream")
	}
	if !names["DEM_PARSE"] {
		t.Error("missing DEM_PARSE stream")
	}
}

func TestConstants(t *testing.T) {
	if StreamDownload != "DEM_DOWNLOAD" {
		t.Errorf("StreamDownload = %s", StreamDownload)
	}
	if StreamParse != "DEM_PARSE" {
		t.Errorf("StreamParse = %s", StreamParse)
	}
	if SubjectDownload != "dem.download.jobs" {
		t.Errorf("SubjectDownload = %s", SubjectDownload)
	}
	if SubjectParse != "dem.parse.jobs" {
		t.Errorf("SubjectParse = %s", SubjectParse)
	}
}

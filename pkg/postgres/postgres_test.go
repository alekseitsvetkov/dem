package postgres

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig("postgres://dem:dem@localhost:5432/dem?sslmode=disable")
	if cfg.DatabaseURL != "postgres://dem:dem@localhost:5432/dem?sslmode=disable" {
		t.Errorf("DatabaseURL = %s", cfg.DatabaseURL)
	}
	if cfg.MaxConns != 20 {
		t.Errorf("expected default MaxConns 20, got %d", cfg.MaxConns)
	}
	if cfg.MinConns != 2 {
		t.Errorf("expected default MinConns 2, got %d", cfg.MinConns)
	}
	if cfg.ConnectTimeout != 5*time.Second {
		t.Errorf("expected default ConnectTimeout 5s, got %v", cfg.ConnectTimeout)
	}
}

func TestOptions(t *testing.T) {
	cfg := DefaultConfig("")
	WithMaxConns(15)(&cfg)
	WithMinConns(1)(&cfg)
	WithMaxConnIdleTime(10 * time.Minute)(&cfg)
	WithConnectTimeout(10 * time.Second)(&cfg)

	if cfg.MaxConns != 15 {
		t.Errorf("MaxConns = %d", cfg.MaxConns)
	}
	if cfg.MinConns != 1 {
		t.Errorf("MinConns = %d", cfg.MinConns)
	}
	if cfg.MaxConnIdleTime != 10*time.Minute {
		t.Errorf("MaxConnIdleTime = %v", cfg.MaxConnIdleTime)
	}
	if cfg.ConnectTimeout != 10*time.Second {
		t.Errorf("ConnectTimeout = %v", cfg.ConnectTimeout)
	}
}

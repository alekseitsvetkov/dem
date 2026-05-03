package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config holds PostgreSQL connection pool configuration.
type Config struct {
	DatabaseURL       string
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
	ConnectTimeout    time.Duration
}

// Option is a functional option for configuring the PostgreSQL connection pool.
type Option func(*Config)

// DefaultConfig returns the default PostgreSQL pool configuration.
// Per D-08 and PITFALLS.md: MaxConns is overridden to 20 (not the default 4).
func DefaultConfig(databaseURL string) Config {
	return Config{
		DatabaseURL:       databaseURL,
		MaxConns:          20,
		MinConns:          2,
		MaxConnLifetime:   30 * time.Minute,
		MaxConnIdleTime:   5 * time.Minute,
		HealthCheckPeriod: 1 * time.Minute,
		ConnectTimeout:    5 * time.Second,
	}
}

// WithMaxConns sets the maximum number of connections in the pool.
// Per D-08: recommended range is 10-25. Default is 20.
func WithMaxConns(n int32) Option {
	return func(c *Config) { c.MaxConns = n }
}

// WithMinConns sets the minimum number of connections in the pool.
// Default is 2 (warm connections to avoid cold-start latency).
func WithMinConns(n int32) Option {
	return func(c *Config) { c.MinConns = n }
}

// WithMaxConnIdleTime sets the maximum time a connection can be idle.
// Default is 5 minutes (frees connections during low activity).
func WithMaxConnIdleTime(d time.Duration) Option {
	return func(c *Config) { c.MaxConnIdleTime = d }
}

// WithConnectTimeout sets the timeout for establishing new connections.
// Default is 5 seconds (fail fast if Postgres is unreachable).
func WithConnectTimeout(d time.Duration) Option {
	return func(c *Config) { c.ConnectTimeout = d }
}

// NewPool creates a new pgxpool.Pool with the given configuration.
// Per D-08: pool is created once at startup and passed to all handlers.
// Per INFR-06: uses pgxpool (not pgx.Conn) with explicit MaxConns (10-25).
//
// The returned pool is concurrency-safe and should be shared across the service.
// Caller must defer pool.Close() on the returned pool.
func NewPool(ctx context.Context, databaseURL string, opts ...Option) (*pgxpool.Pool, error) {
	cfg := DefaultConfig(databaseURL)
	for _, opt := range opts {
		opt(&cfg)
	}

	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("postgres parse config: %w", err)
	}

	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolCfg.HealthCheckPeriod = cfg.HealthCheckPeriod
	poolCfg.MaxConnLifetimeJitter = 0

	ctx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("pgxpool new: %w", err)
	}

	// Verify connectivity with a ping
	pingCtx, pingCancel := context.WithTimeout(context.Background(), cfg.ConnectTimeout)
	defer pingCancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}

	return pool, nil
}

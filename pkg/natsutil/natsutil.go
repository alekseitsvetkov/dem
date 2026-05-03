package natsutil

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Config holds NATS connection configuration.
type Config struct {
	URL           string
	Timeout       time.Duration
	MaxReconnects int
	ReconnectWait time.Duration
	Name          string
}

// Option is a functional option for configuring NATS connections.
type Option func(*Config)

// DefaultConfig returns the default NATS connection configuration.
func DefaultConfig() Config {
	return Config{
		URL:           "nats://localhost:4222",
		Timeout:       5 * time.Second,
		MaxReconnects: 10,
		ReconnectWait: 2 * time.Second,
		Name:          "dem-service",
	}
}

// WithURL sets the NATS server URL.
func WithURL(url string) Option {
	return func(c *Config) { c.URL = url }
}

// WithTimeout sets the NATS connection timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Config) { c.Timeout = d }
}

// WithMaxReconnects sets the maximum number of reconnection attempts.
func WithMaxReconnects(n int) Option {
	return func(c *Config) { c.MaxReconnects = n }
}

// WithReconnectWait sets the delay between reconnection attempts.
func WithReconnectWait(d time.Duration) Option {
	return func(c *Config) { c.ReconnectWait = d }
}

// WithName sets the NATS client connection name.
func WithName(name string) Option {
	return func(c *Config) { c.Name = name }
}

// NewNATSConn creates a NATS connection and JetStream context.
// Returns the connection, JetStream interface, and any error.
// Caller must defer nc.Close() on the returned connection.
//
// Per D-06: connection is established before JetStream context creation.
// Per Pitfall 1: callers must call CreateStreams or VerifyStreams before publishing.
func NewNATSConn(opts ...Option) (*nats.Conn, jetstream.JetStream, error) {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	nc, err := nats.Connect(cfg.URL,
		nats.Timeout(cfg.Timeout),
		nats.MaxReconnects(cfg.MaxReconnects),
		nats.ReconnectWait(cfg.ReconnectWait),
		nats.Name(cfg.Name),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("nats connect: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, nil, fmt.Errorf("jetstream new: %w", err)
	}

	return nc, js, nil
}

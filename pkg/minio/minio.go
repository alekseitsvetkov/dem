package minio

import (
	"context"
	"fmt"
	"net/http"
	"time"

	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Config holds MinIO client configuration.
type Config struct {
	Endpoint        string
	AccessKey       string
	SecretKey       string
	UseSSL          bool
	Region          string
	RequestTimeout  time.Duration
	MaxIdleConns    int
	IdleConnTimeout time.Duration
}

// Option is a functional option for configuring MinIO clients.
type Option func(*Config)

// DefaultConfig returns the default MinIO client configuration (local dev defaults).
func DefaultConfig() Config {
	return Config{
		Endpoint:        "localhost:9000",
		AccessKey:       "minioadmin",
		SecretKey:       "minioadmin",
		UseSSL:          false,
		Region:          "us-east-1",
		RequestTimeout:  30 * time.Second,
		MaxIdleConns:    100,
		IdleConnTimeout: 90 * time.Second,
	}
}

// WithEndpoint sets the MinIO server endpoint (host:port, no http:// prefix).
func WithEndpoint(e string) Option {
	return func(c *Config) { c.Endpoint = e }
}

// WithCredentials sets the MinIO access key and secret key.
func WithCredentials(accessKey, secretKey string) Option {
	return func(c *Config) {
		c.AccessKey = accessKey
		c.SecretKey = secretKey
	}
}

// WithSSL enables HTTPS for MinIO connections.
func WithSSL() Option {
	return func(c *Config) { c.UseSSL = true }
}

// WithRegion sets the MinIO/S3 region.
func WithRegion(r string) Option {
	return func(c *Config) { c.Region = r }
}

// NewMinioClient creates a new MinIO client with the given configuration options.
// Per PITFALLS.md: endpoint is bare host:port (no http:// prefix).
// Per PITFALLS.md: uses a custom http.Transport with connection pooling config.
//
// Caller should call EnsureBucket after creating the client to verify connectivity.
func NewMinioClient(opts ...Option) (*minio.Client, error) {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	transport := &http.Transport{
		MaxIdleConns:       cfg.MaxIdleConns,
		IdleConnTimeout:    cfg.IdleConnTimeout,
		DisableCompression: true,
	}

	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:     credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure:    cfg.UseSSL,
		Region:    cfg.Region,
		Transport: transport,
	})
	if err != nil {
		return nil, fmt.Errorf("minio new client: %w", err)
	}

	return client, nil
}

// DefaultBucket is the default MinIO bucket name for demo files.
const DefaultBucket = "dem-files"

// EnsureBucket creates the bucket if it doesn't already exist.
// This is safe to call multiple times — bucket creation is idempotent.
func EnsureBucket(ctx context.Context, client *minio.Client, bucket string) error {
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("minio bucket exists check: %w", err)
	}
	if exists {
		return nil
	}
	err = client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: ""})
	if err != nil {
		return fmt.Errorf("minio make bucket: %w", err)
	}
	return nil
}

package downloader

import (
	"time"

	"github.com/spf13/viper"
)

// Config holds the DownloaderService configuration sourced from environment variables.
// Per D-12: all values use Viper with DEM_ prefix and sensible defaults.
type Config struct {
	// NATSURL is the NATS server connection URL.
	NATSURL string

	// MinioEndpoint is the MinIO server endpoint (host:port, no http:// prefix).
	MinioEndpoint string

	// MinioAccessKey is the MinIO access key for authentication.
	MinioAccessKey string

	// MinioSecretKey is the MinIO secret key for authentication.
	MinioSecretKey string

	// MinioBucket is the MinIO bucket name for demo file storage.
	MinioBucket string

	// MinioUseSSL enables HTTPS for MinIO connections.
	MinioUseSSL bool

	// LogLevel controls the structured log level (debug, info, warn, error).
	LogLevel string

	// MaxRetries is the maximum number of internal download retry attempts (default: 3 per D-03).
	MaxRetries int

	// RetryBaseDelay is the initial delay between retries (default: 5s per D-03).
	RetryBaseDelay time.Duration

	// RetryMaxDelay is the maximum delay between retry attempts (default: 125s per D-03).
	RetryMaxDelay time.Duration

	// NakDelay is the duration to delay NATS redelivery via NakWithDelay.
	NakDelay time.Duration

	// DownloadTimeout is the maximum duration for a single download attempt.
	DownloadTimeout time.Duration

	// MaxBytes is the maximum download size in bytes (safety net per Pitfall 4).
	MaxBytes int64
}

// LoadConfig reads configuration from environment variables with Viper defaults.
// Environment variable prefix is DEM.
//
// Environment variables:
//
//	DEM_NATS_URL            (default: "nats://localhost:4222")
//	DEM_MINIO_ENDPOINT      (default: "localhost:9000")
//	DEM_MINIO_ACCESS_KEY    (default: "minioadmin")
//	DEM_MINIO_SECRET_KEY    (default: "minioadmin")
//	DEM_MINIO_BUCKET        (default: "dem-files")
//	DEM_MINIO_USE_SSL       (default: "false")
//	DEM_LOG_LEVEL           (default: "info")
//	DEM_DOWNLOADER_MAX_RETRIES  (default: "3")
//	DEM_DOWNLOADER_RETRY_BASE   (default: "5s")
//	DEM_DOWNLOADER_RETRY_MAX    (default: "125s")
//	DEM_DOWNLOADER_NAK_DELAY    (default: "5m")
//	DEM_DOWNLOADER_TIMEOUT      (default: "30m")
//	DEM_DOWNLOADER_MAX_BYTES    (default: "524288000" — 500 MB)
func LoadConfig() Config {
	v := viper.New()
	v.SetEnvPrefix("DEM")
	v.AutomaticEnv()

	v.SetDefault("nats_url", "nats://localhost:4222")
	v.SetDefault("minio_endpoint", "localhost:9000")
	v.SetDefault("minio_access_key", "minioadmin")
	v.SetDefault("minio_secret_key", "minioadmin")
	v.SetDefault("minio_bucket", "dem-files")
	v.SetDefault("minio_use_ssl", false)
	v.SetDefault("log_level", "info")
	v.SetDefault("downloader_max_retries", 3)
	v.SetDefault("downloader_retry_base", "5s")
	v.SetDefault("downloader_retry_max", "125s")
	v.SetDefault("downloader_nak_delay", "5m")
	v.SetDefault("downloader_timeout", "30m")
	v.SetDefault("downloader_max_bytes", int64(500*1024*1024)) // 500 MB

	return Config{
		NATSURL:        v.GetString("nats_url"),
		MinioEndpoint:  v.GetString("minio_endpoint"),
		MinioAccessKey: v.GetString("minio_access_key"),
		MinioSecretKey: v.GetString("minio_secret_key"),
		MinioBucket:    v.GetString("minio_bucket"),
		MinioUseSSL:    v.GetBool("minio_use_ssl"),
		LogLevel:       v.GetString("log_level"),
		MaxRetries:     v.GetInt("downloader_max_retries"),
		RetryBaseDelay: parseDownloaderDuration(v, "downloader_retry_base", 5*time.Second),
		RetryMaxDelay:  parseDownloaderDuration(v, "downloader_retry_max", 125*time.Second),
		NakDelay:       parseDownloaderDuration(v, "downloader_nak_delay", 5*time.Minute),
		DownloadTimeout: parseDownloaderDuration(v, "downloader_timeout", 30*time.Minute),
		MaxBytes:       v.GetInt64("downloader_max_bytes"),
	}
}

// parseDownloaderDuration parses a duration string from viper with a fallback.
// Follows the poller's pattern of manual duration parsing for consistency.
func parseDownloaderDuration(v *viper.Viper, key string, fallback time.Duration) time.Duration {
	raw := v.GetString(key)
	d, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	if d <= 0 {
		return fallback
	}
	return d
}

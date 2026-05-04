package parser

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds parser service configuration.
// All values are loaded from environment variables with DEM_ prefix via Viper.
type Config struct {
	NATSURL        string        // DEM_NATS_URL (default: "nats://localhost:4222")
	DatabaseURL    string        // DEM_DATABASE_URL (default: "postgres://dem:dem@localhost:5432/dem?sslmode=disable")
	MinioEndpoint  string        // DEM_MINIO_ENDPOINT (default: "localhost:9000")
	MinioAccessKey string        // DEM_MINIO_ACCESS_KEY (default: "minioadmin")
	MinioSecretKey string        // DEM_MINIO_SECRET_KEY (default: "minioadmin")
	MinioUseSSL    bool          // DEM_MINIO_USE_SSL (default: false)
	LogLevel       string        // DEM_LOG_LEVEL (default: "info")
	Concurrency    int           // DEM_PARSER_CONCURRENCY (default: 1 per D-06 — MaxAckPending)
	ParseTimeout   time.Duration // DEM_PARSER_TIMEOUT (default: 60 * time.Minute — long demos)
	AckWait        time.Duration // DEM_PARSER_ACK_WAIT (default: 90 * time.Minute — exceeds max parse time per PITFALLS.md Pitfall 2/5)
}

// LoadConfig loads parser configuration from environment variables with DEM_ prefix.
// Uses Viper for automatic environment variable binding with defaults for all fields.
func LoadConfig() (Config, error) {
	v := viper.New()
	v.SetEnvPrefix("DEM")
	v.AutomaticEnv()

	// Set defaults
	v.SetDefault("NATS_URL", "nats://localhost:4222")
	v.SetDefault("DATABASE_URL", "postgres://dem:dem@localhost:5432/dem?sslmode=disable")
	v.SetDefault("MINIO_ENDPOINT", "localhost:9000")
	v.SetDefault("MINIO_ACCESS_KEY", "minioadmin")
	v.SetDefault("MINIO_SECRET_KEY", "minioadmin")
	v.SetDefault("MINIO_USE_SSL", false)
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("PARSER_CONCURRENCY", 1)           // D-06: single parser by default
	v.SetDefault("PARSER_TIMEOUT", "60m")           // ParseTimeout: 60 minutes
	v.SetDefault("PARSER_ACK_WAIT", "90m")          // AckWait: 90 minutes

	parseTimeout, err := time.ParseDuration(v.GetString("PARSER_TIMEOUT"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid DEM_PARSER_TIMEOUT: %w", err)
	}

	ackWait, err := time.ParseDuration(v.GetString("PARSER_ACK_WAIT"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid DEM_PARSER_ACK_WAIT: %w", err)
	}

	return Config{
		NATSURL:        v.GetString("NATS_URL"),
		DatabaseURL:    v.GetString("DATABASE_URL"),
		MinioEndpoint:  v.GetString("MINIO_ENDPOINT"),
		MinioAccessKey: v.GetString("MINIO_ACCESS_KEY"),
		MinioSecretKey: v.GetString("MINIO_SECRET_KEY"),
		MinioUseSSL:    v.GetBool("MINIO_USE_SSL"),
		LogLevel:       v.GetString("LOG_LEVEL"),
		Concurrency:    v.GetInt("PARSER_CONCURRENCY"),
		ParseTimeout:   parseTimeout,
		AckWait:        ackWait,
	}, nil
}

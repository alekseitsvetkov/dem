package poller

import (
	"time"

	"github.com/spf13/viper"
)

// Config holds the PollerService configuration sourced from environment variables.
// Per D-01 and D-12: all values use Viper with DEM_ prefix and sensible defaults.
type Config struct {
	// CronExpression is the cron schedule for poll runs (default: daily at 02:00 UTC).
	CronExpression string

	// MinInterval is the minimum interval between successive poll runs.
	// If the last successful run was less than MinInterval ago, the poll is skipped.
	MinInterval time.Duration

	// NATSURL is the NATS server connection URL.
	NATSURL string

	// DatabaseURL is the PostgreSQL connection URL.
	DatabaseURL string

	// HLTVBaseURL is the base URL for HLTV.org (used for URL construction).
	HLTVBaseURL string

	// LogLevel controls the structured log level (debug, info, warn, error).
	LogLevel string

	// OneShot runs the poll once immediately and exits instead of starting the cron scheduler.
	OneShot bool
}

// LoadConfig reads configuration from environment variables with Viper defaults.
// Environment variable prefix is DEM. Map: DEM_POLLER_CRON → CronExpression, etc.
func LoadConfig() Config {
	v := viper.New()
	v.SetEnvPrefix("DEM")
	v.AutomaticEnv()

	v.SetDefault("poller_cron", "0 2 * * *")
	v.SetDefault("poller_min_interval", "6h")
	v.SetDefault("nats_url", "nats://localhost:4222")
	v.SetDefault("database_url", "postgres://dem:dem@localhost:5432/dem?sslmode=disable")
	v.SetDefault("hltv_base_url", "https://www.hltv.org")
	v.SetDefault("log_level", "info")
	v.SetDefault("poller_oneshot", false)

	return Config{
		CronExpression: v.GetString("poller_cron"),
		MinInterval:    parseMinInterval(v.GetString("poller_min_interval")),
		NATSURL:        v.GetString("nats_url"),
		DatabaseURL:    v.GetString("database_url"),
		HLTVBaseURL:    v.GetString("hltv_base_url"),
		LogLevel:       v.GetString("log_level"),
		OneShot:        v.GetBool("poller_oneshot"),
	}
}

// parseMinInterval parses a duration string with a fallback to 6 hours.
func parseMinInterval(raw string) time.Duration {
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 6 * time.Hour
	}
	if d <= 0 {
		return 6 * time.Hour
	}
	return d
}

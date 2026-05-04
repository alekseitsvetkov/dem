package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/alekseitsvetkov/dem/internal/hltv"
	"github.com/alekseitsvetkov/dem/internal/poller"
	"github.com/alekseitsvetkov/dem/internal/service"
	"github.com/alekseitsvetkov/dem/pkg/natsutil"
	"github.com/alekseitsvetkov/dem/pkg/postgres"
)

func main() {
	cfg := poller.LoadConfig()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(cfg.LogLevel),
	}))

	// Wire NATS
	nc, js, err := natsutil.NewNATSConn(
		natsutil.WithURL(cfg.NATSURL),
		natsutil.WithName("poller"),
	)
	if err != nil {
		logger.Error("nats connect failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer nc.Close()

	// Wire Postgres
	pool, err := postgres.NewPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Error("postgres connect failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	// Wire HLTV Client (reuse v1.0 client — per D-01)
	hltvClient := hltv.NewClient()

	// Create PollerService with functional options (per CROS-02)
	svc := poller.NewPollerService(cfg,
		poller.WithNATS(js),
		poller.WithPostgres(pool),
		poller.WithHLTVClient(hltvClient),
		poller.WithLogger(logger),
	)

	// Create Runner and add service
	runner := service.NewRunner(
		service.WithLogger(logger),
	)
	runner.AddService(svc)

	if err := runner.Run(context.Background()); err != nil {
		logger.Error("poller service exited with error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

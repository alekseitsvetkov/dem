package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/alekseitsvetkov/dem/internal/parser"
	"github.com/alekseitsvetkov/dem/internal/service"
	"github.com/alekseitsvetkov/dem/pkg/minio"
	"github.com/alekseitsvetkov/dem/pkg/natsutil"
	"github.com/alekseitsvetkov/dem/pkg/postgres"
)

func main() {
	cfg, err := parser.LoadConfig()
	if err != nil {
		slog.Error("config load failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(cfg.LogLevel),
	}))

	// Wire NATS
	nc, js, err := natsutil.NewNATSConn(
		natsutil.WithURL(cfg.NATSURL),
		natsutil.WithName("parser"),
	)
	if err != nil {
		logger.Error("nats connect failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer nc.Close()

	// Wire MinIO
	minioOpts := []minio.Option{
		minio.WithEndpoint(cfg.MinioEndpoint),
		minio.WithCredentials(cfg.MinioAccessKey, cfg.MinioSecretKey),
	}
	if cfg.MinioUseSSL {
		minioOpts = append(minioOpts, minio.WithSSL())
	}
	minioClient, err := minio.NewMinioClient(minioOpts...)
	if err != nil {
		logger.Error("minio connect failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Wire Postgres
	pool, err := postgres.NewPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Error("postgres connect failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	// Create ParserService with dependency injection per CROS-02.
	svc := parser.NewParserService(cfg,
		parser.WithNATS(js),
		parser.WithMinio(minioClient),
		parser.WithPostgres(pool),
		parser.WithLogger(logger),
	)

	runner := service.NewRunner(service.WithLogger(logger))
	runner.AddService(svc)

	if err := runner.Run(context.Background()); err != nil {
		logger.Error("parser service exited with error", slog.String("error", err.Error()))
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

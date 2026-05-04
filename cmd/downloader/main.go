package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/alekseitsvetkov/dem/internal/downloader"
	"github.com/alekseitsvetkov/dem/internal/service"
	dmnio "github.com/alekseitsvetkov/dem/pkg/minio"
	"github.com/alekseitsvetkov/dem/pkg/natsutil"
)

func main() {
	cfg := downloader.LoadConfig()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(cfg.LogLevel),
	}))

	// Wire NATS
	nc, js, err := natsutil.NewNATSConn(
		natsutil.WithURL(cfg.NATSURL),
		natsutil.WithName("downloader"),
	)
	if err != nil {
		logger.Error("nats connect failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer nc.Close()

	// Wire MinIO
	minioOpts := []dmnio.Option{
		dmnio.WithEndpoint(cfg.MinioEndpoint),
		dmnio.WithCredentials(cfg.MinioAccessKey, cfg.MinioSecretKey),
	}
	if cfg.MinioUseSSL {
		minioOpts = append(minioOpts, dmnio.WithSSL())
	}
	minioClient, err := dmnio.NewMinioClient(minioOpts...)
	if err != nil {
		logger.Error("minio connect failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Create DownloaderService with functional options (per CROS-02)
	svc := downloader.NewDownloaderService(cfg,
		downloader.WithNATS(js),
		downloader.WithMinio(minioClient),
		downloader.WithLogger(logger),
	)

	// Create Runner and add service
	runner := service.NewRunner(
		service.WithLogger(logger),
	)
	runner.AddService(svc)

	if err := runner.Run(context.Background()); err != nil {
		logger.Error("downloader service exited with error", slog.String("error", err.Error()))
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
